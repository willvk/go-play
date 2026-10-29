package release

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"releasePlease/internal/commits"
	"releasePlease/internal/github"
	"releasePlease/internal/semver"
)

// fakeGitHub is an in-memory stand-in for the subset of the REST API the
// runner uses. Commits are stored newest first.
type fakeGitHub struct {
	mu       sync.Mutex
	files    map[string]string // path -> content on main
	commits  []github.Commit   // history of main, newest first
	tags     map[string]string // tag -> sha
	releases []github.Release
	refs     map[string]string // branch -> sha
	trees    map[string]map[string]string
	pulls    []*github.PullRequest
	nextID   int
}

func newFake() *fakeGitHub {
	return &fakeGitHub{files: map[string]string{}, tags: map[string]string{}, refs: map[string]string{}, trees: map[string]map[string]string{}}
}

func (f *fakeGitHub) commit(sha, msg string) {
	c := github.Commit{SHA: sha}
	c.Commit.Message = msg
	f.commits = append([]github.Commit{c}, f.commits...)
	f.refs["main"] = sha
}

func (f *fakeGitHub) id() int { f.nextID++; return f.nextID }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fakeGitHub) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	const p = "/repos/o/r"
	lock := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { f.mu.Lock(); defer f.mu.Unlock(); h(w, r) }
	}

	mux.HandleFunc("GET "+p+"/releases", lock(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			writeJSON(w, []github.Release{})
			return
		}
		writeJSON(w, f.releases)
	}))
	mux.HandleFunc("GET "+p+"/releases/tags/{tag}", lock(func(w http.ResponseWriter, r *http.Request) {
		for _, rel := range f.releases {
			if rel.TagName == r.PathValue("tag") {
				writeJSON(w, rel)
				return
			}
		}
		http.NotFound(w, r)
	}))
	mux.HandleFunc("POST "+p+"/releases", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			TagName string `json:"tag_name"`
			Target  string `json:"target_commitish"`
			Name    string `json:"name"`
			Body    string `json:"body"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.tags[in.TagName] = in.Target
		rel := github.Release{ID: int64(f.id()), TagName: in.TagName, Name: in.Name, HTMLURL: "https://github.com/o/r/releases/tag/" + in.TagName}
		f.releases = append(f.releases, rel)
		t.Logf("release body for %s:\n%s", in.TagName, in.Body)
		if strings.Contains(in.Body, "## [") || !strings.Contains(in.Body, "### Features") {
			t.Errorf("unexpected release body: %q", in.Body)
		}
		writeJSON(w, rel)
	}))
	mux.HandleFunc("GET "+p+"/compare/{basehead}", lock(func(w http.ResponseWriter, r *http.Request) {
		base, _, _ := strings.Cut(r.PathValue("basehead"), "...")
		var out []github.Commit
		for _, c := range f.commits {
			if c.SHA == f.tags[base] {
				break
			}
			out = append(out, c)
		}
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		writeJSON(w, map[string]any{"total_commits": len(out), "commits": out})
	}))
	mux.HandleFunc("GET "+p+"/commits", lock(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			writeJSON(w, []github.Commit{})
			return
		}
		writeJSON(w, f.commits)
	}))
	mux.HandleFunc("GET "+p+"/contents/{path...}", lock(func(w http.ResponseWriter, r *http.Request) {
		content, ok := f.files[r.PathValue("path")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]string{"encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(content))})
	}))
	mux.HandleFunc("GET "+p+"/git/ref/heads/{branch...}", lock(func(w http.ResponseWriter, r *http.Request) {
		sha, ok := f.refs[r.PathValue("branch")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{"object": map[string]string{"sha": sha}})
	}))
	mux.HandleFunc("GET "+p+"/git/commits/{sha}", lock(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"tree": map[string]string{"sha": "tree-" + r.PathValue("sha")}})
	}))
	mux.HandleFunc("POST "+p+"/git/trees", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Tree []struct{ Path, Content string } `json:"tree"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		files := map[string]string{}
		for _, e := range in.Tree {
			files[e.Path] = e.Content
		}
		sha := "newtree"
		f.trees[sha] = files
		writeJSON(w, map[string]string{"sha": sha})
	}))
	mux.HandleFunc("POST "+p+"/git/commits", lock(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"sha": "releasecommit"})
	}))
	mux.HandleFunc("POST "+p+"/git/refs", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Ref, SHA string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.refs[strings.TrimPrefix(in.Ref, "refs/heads/")] = in.SHA
		w.WriteHeader(http.StatusCreated)
	}))
	mux.HandleFunc("PATCH "+p+"/git/refs/heads/{branch...}", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ SHA string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.refs[r.PathValue("branch")] = in.SHA
	}))
	mux.HandleFunc("GET "+p+"/pulls", lock(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := []github.PullRequest{}
		for _, pr := range f.pulls {
			if pr.State == q.Get("state") && (q.Get("head") == "" || "o:"+pr.Head.Ref == q.Get("head")) {
				out = append(out, *pr)
			}
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("POST "+p+"/pulls", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Title, Body, Head, Base string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		pr := &github.PullRequest{Number: f.id(), Title: in.Title, Body: in.Body, State: "open"}
		pr.Head.Ref = in.Head
		f.pulls = append(f.pulls, pr)
		writeJSON(w, pr)
	}))
	mux.HandleFunc("PATCH "+p+"/pulls/{n}", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Title, Body string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		pr := f.pull(r.PathValue("n"))
		pr.Title, pr.Body = in.Title, in.Body
		writeJSON(w, pr)
	}))
	mux.HandleFunc("POST "+p+"/issues/{n}/labels", lock(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Labels []string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		pr := f.pull(r.PathValue("n"))
		for _, l := range in.Labels {
			if !pr.HasLabel(l) {
				pr.Labels = append(pr.Labels, github.Label{Name: l})
			}
		}
		writeJSON(w, pr.Labels)
	}))
	mux.HandleFunc("DELETE "+p+"/issues/{n}/labels/{label}", lock(func(w http.ResponseWriter, r *http.Request) {
		pr := f.pull(r.PathValue("n"))
		var kept []github.Label
		for _, l := range pr.Labels {
			if l.Name != r.PathValue("label") {
				kept = append(kept, l)
			}
		}
		pr.Labels = kept
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
		http.NotFound(w, r)
	})
	return mux
}

func (f *fakeGitHub) pull(n string) *github.PullRequest {
	for _, pr := range f.pulls {
		if strconv.Itoa(pr.Number) == n {
			return pr
		}
	}
	return &github.PullRequest{}
}

func TestEndToEnd(t *testing.T) {
	f := newFake()
	f.commit("c0", "chore: initial")
	f.tags["v1.1.0"] = "c0"
	f.releases = []github.Release{{ID: 1, TagName: "v1.1.0"}, {ID: 2, TagName: "v1.0.0"}}
	f.commit("c1aaaaaaaa", "feat(api): add search endpoint")
	f.commit("c2bbbbbbbb", "fix: handle empty input\n\nCloses #4")
	f.commit("c3cccccccc", "docs: typo")
	f.files["CHANGELOG.md"] = "# Changelog\n\n## [1.1.0](https://github.com/o/r/compare/v1.0.0...v1.1.0) (2026-01-01)\n\n* old\n"
	f.files["version.go"] = "package main\n\nconst Version = \"1.1.0\" // x-release-please-version\n"

	srv := httptest.NewServer(f.handler(t))
	defer srv.Close()

	var log strings.Builder
	r := &Runner{
		GH: &github.Client{BaseURL: srv.URL, Owner: "o", Repo: "r", HTTP: srv.Client()},
		Cfg: Config{
			Branch:        "main",
			TagPrefix:     "v",
			ChangelogPath: "CHANGELOG.md",
			VersionFile:   "version.txt",
			ExtraFiles:    []string{"version.go"},
		},
		Log: &log,
		Now: func() time.Time { return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC) },
	}
	ctx := context.Background()

	// 1. Release PR is opened.
	res, err := r.ReleasePR(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Version != "1.2.0" {
		t.Fatalf("got %+v, want version 1.2.0", res)
	}
	if f.refs["release-please--branches--main"] != "releasecommit" {
		t.Errorf("release branch not pushed: %v", f.refs)
	}
	tree := f.trees["newtree"]
	wantCL := "# Changelog\n\n## [1.2.0](https://github.com/o/r/compare/v1.1.0...v1.2.0) (2026-10-29)\n\n\n" +
		"### Features\n\n* **api:** add search endpoint ([c1aaaaa](https://github.com/o/r/commit/c1aaaaaaaa))\n\n" +
		"### Bug Fixes\n\n* handle empty input ([c2bbbbb](https://github.com/o/r/commit/c2bbbbbbbb))\n\n" +
		"## [1.1.0](https://github.com/o/r/compare/v1.0.0...v1.1.0) (2026-01-01)\n\n* old\n"
	if tree["CHANGELOG.md"] != wantCL {
		t.Errorf("CHANGELOG.md:\n%s\nwant:\n%s", tree["CHANGELOG.md"], wantCL)
	}
	if tree["version.txt"] != "1.2.0\n" {
		t.Errorf("version.txt = %q", tree["version.txt"])
	}
	if !strings.Contains(tree["version.go"], `"1.2.0" // x-release-please-version`) {
		t.Errorf("version.go = %q", tree["version.go"])
	}
	if len(f.pulls) != 1 || f.pulls[0].Title != "chore(main): release 1.2.0" || !f.pulls[0].HasLabel(LabelPending) {
		t.Fatalf("pull not created as expected: %+v", f.pulls)
	}

	// 2. A second run updates the same PR instead of opening another.
	f.commit("c4dddddddd", "fix: another bug")
	if _, err := r.ReleasePR(ctx); err != nil {
		t.Fatal(err)
	}
	if len(f.pulls) != 1 || !strings.Contains(f.pulls[0].Body, "another bug") {
		t.Fatalf("expected updated PR, got %+v", f.pulls)
	}

	// 3. Merge it; github-release tags it and flips the labels.
	now := time.Now()
	pr := f.pulls[0]
	pr.State, pr.MergedAt, pr.MergeCommitSHA = "closed", &now, "merge1"
	f.commit("merge1", pr.Title)

	created, err := r.GitHubRelease(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0].Tag != "v1.2.0" || f.tags["v1.2.0"] != "merge1" {
		t.Fatalf("release not created: %+v tags=%v", created, f.tags)
	}
	if pr.HasLabel(LabelPending) || !pr.HasLabel(LabelTagged) {
		t.Errorf("labels not updated: %+v", pr.Labels)
	}

	// 4. Nothing new since the release: no PR, no release.
	if res, err := r.ReleasePR(ctx); err != nil || res != nil {
		t.Errorf("expected no-op, got %+v, %v", res, err)
	}
	if created, err := r.GitHubRelease(ctx); err != nil || len(created) != 0 {
		t.Errorf("expected no-op, got %+v, %v", created, err)
	}
	t.Log(log.String())
}

func TestNextVersion(t *testing.T) {
	v := func(s string) *semver.Version { x, _ := semver.Parse(s); return &x }
	parse := func(msgs ...string) []commits.Commit {
		var cs []commits.Commit
		for _, m := range msgs {
			cs = append(cs, commits.Parse("", m))
		}
		return cs
	}
	cases := []struct {
		name string
		prev *semver.Version
		cs   []commits.Commit
		cfg  Config
		want string
	}{
		{"fix", v("1.2.3"), parse("fix: a"), Config{}, "1.2.4"},
		{"feat", v("1.2.3"), parse("fix: a", "feat: b"), Config{}, "1.3.0"},
		{"breaking", v("1.2.3"), parse("feat: b", "fix!: c"), Config{}, "2.0.0"},
		{"pre-major breaking", v("0.4.1"), parse("feat!: x"), Config{BumpMinorPreMajor: true}, "0.5.0"},
		{"pre-major feat", v("0.4.1"), parse("feat: x"), Config{BumpPatchForMinorPreMajor: true}, "0.4.2"},
		{"release-as", v("1.2.3"), parse("chore: x\n\nRelease-As: 3.0.0"), Config{}, "3.0.0"},
		{"initial", nil, parse("feat: x"), Config{InitialVersion: "0.1.0"}, "0.1.0"},
		{"initial default", nil, parse("feat: x"), Config{}, "1.0.0"},
	}
	for _, tc := range cases {
		got, err := NextVersion(tc.prev, tc.cs, tc.cfg)
		if err != nil || got.String() != tc.want {
			t.Errorf("%s: got %s (%v), want %s", tc.name, got, err, tc.want)
		}
	}
}

func TestNotesFromBody(t *testing.T) {
	body := bodyHeader + "## [1.2.0](x) (2026-10-29)\n\n\n### Features\n\n* a\n\n---\nin a code block? no" + bodyFooter
	got := notesFromBody(body)
	if !strings.HasPrefix(got, "### Features") || strings.Contains(got, "Release Please") {
		t.Errorf("notesFromBody = %q", got)
	}
}
