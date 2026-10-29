// Package github is a minimal GitHub REST client covering the endpoints
// release-please needs. Like upstream, everything goes through the API, so no
// local clone is required.
package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL string // e.g. https://api.github.com
	Token   string
	Owner   string
	Repo    string
	HTTP    *http.Client
}

type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github: %s %s: %d %s", e.Method, e.URL, e.StatusCode, strings.TrimSpace(e.Body))
}

func IsNotFound(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}

type Label struct {
	Name string `json:"name"`
}

type PullRequest struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	State          string     `json:"state"`
	HTMLURL        string     `json:"html_url"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	Labels         []Label    `json:"labels"`
	Head           struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func (p PullRequest) HasLabel(name string) bool {
	for _, l := range p.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

type Release struct {
	ID         int64  `json:"id"`
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	HTMLURL    string `json:"html_url"`
}

type Commit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

func (c *Client) repoPath(format string, args ...any) string {
	return fmt.Sprintf("/repos/%s/%s", url.PathEscape(c.Owner), url.PathEscape(c.Repo)) + fmt.Sprintf(format, args...)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	base := c.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	u := strings.TrimRight(base, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{StatusCode: resp.StatusCode, Method: method, URL: u, Body: string(b)}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

const perPage = 100

// paged fetches up to maxPages pages of a list endpoint.
func paged[T any](ctx context.Context, c *Client, path string, q url.Values, maxPages int) ([]T, error) {
	if q == nil {
		q = url.Values{}
	}
	q.Set("per_page", strconv.Itoa(perPage))
	var all []T
	for page := 1; page <= maxPages; page++ {
		q.Set("page", strconv.Itoa(page))
		var batch []T
		if err := c.do(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) DefaultBranch(ctx context.Context) (string, error) {
	var r struct {
		DefaultBranch string `json:"default_branch"`
	}
	err := c.do(ctx, http.MethodGet, c.repoPath(""), nil, &r)
	return r.DefaultBranch, err
}

func (c *Client) ListReleases(ctx context.Context, maxPages int) ([]Release, error) {
	return paged[Release](ctx, c, c.repoPath("/releases"), nil, maxPages)
}

// ListCommits returns commits reachable from ref, newest first.
func (c *Client) ListCommits(ctx context.Context, ref string, maxPages int) ([]Commit, error) {
	return paged[Commit](ctx, c, c.repoPath("/commits"), url.Values{"sha": {ref}}, maxPages)
}

// CommitsBetween returns commits in head that are not in base, newest first.
func (c *Client) CommitsBetween(ctx context.Context, base, head string) ([]Commit, error) {
	var all []Commit
	path := c.repoPath("/compare/%s...%s", url.PathEscape(base), url.PathEscape(head))
	for page := 1; page <= 50; page++ {
		var r struct {
			TotalCommits int      `json:"total_commits"`
			Commits      []Commit `json:"commits"`
		}
		q := url.Values{"per_page": {strconv.Itoa(perPage)}, "page": {strconv.Itoa(page)}}
		if err := c.do(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &r); err != nil {
			return nil, err
		}
		all = append(all, r.Commits...)
		if len(r.Commits) < perPage || len(all) >= r.TotalCommits {
			break
		}
	}
	// The compare API lists oldest first.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all, nil
}

// GetFile returns a file's contents at ref; found is false if it does not exist.
func (c *Client) GetFile(ctx context.Context, path, ref string) (content string, found bool, err error) {
	var r struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	err = c.do(ctx, http.MethodGet, c.repoPath("/contents/%s?ref=%s", strings.Join(segs, "/"), url.QueryEscape(ref)), nil, &r)
	if IsNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if r.Encoding != "base64" {
		return "", false, fmt.Errorf("github: %s: unsupported encoding %q", path, r.Encoding)
	}
	b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(r.Content, "\n", ""))
	return string(b), true, err
}

func (c *Client) BranchSHA(ctx context.Context, branch string) (string, error) {
	var r struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	err := c.do(ctx, http.MethodGet, c.repoPath("/git/ref/heads/%s", branch), nil, &r)
	return r.Object.SHA, err
}

// CommitFiles creates a single commit on top of parent that writes files,
// and returns its SHA.
func (c *Client) CommitFiles(ctx context.Context, parent, message string, files map[string]string) (string, error) {
	var pc struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := c.do(ctx, http.MethodGet, c.repoPath("/git/commits/%s", parent), nil, &pc); err != nil {
		return "", err
	}

	type entry struct {
		Path    string `json:"path"`
		Mode    string `json:"mode"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	tree := struct {
		BaseTree string  `json:"base_tree"`
		Tree     []entry `json:"tree"`
	}{BaseTree: pc.Tree.SHA}
	for p, content := range files {
		tree.Tree = append(tree.Tree, entry{Path: p, Mode: "100644", Type: "blob", Content: content})
	}
	var tr struct {
		SHA string `json:"sha"`
	}
	if err := c.do(ctx, http.MethodPost, c.repoPath("/git/trees"), tree, &tr); err != nil {
		return "", err
	}

	var cr struct {
		SHA string `json:"sha"`
	}
	err := c.do(ctx, http.MethodPost, c.repoPath("/git/commits"), map[string]any{
		"message": message,
		"tree":    tr.SHA,
		"parents": []string{parent},
	}, &cr)
	return cr.SHA, err
}

// ForceBranch points branch at sha, creating the branch if needed.
func (c *Client) ForceBranch(ctx context.Context, branch, sha string) error {
	_, err := c.BranchSHA(ctx, branch)
	if IsNotFound(err) {
		return c.do(ctx, http.MethodPost, c.repoPath("/git/refs"), map[string]string{"ref": "refs/heads/" + branch, "sha": sha}, nil)
	}
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPatch, c.repoPath("/git/refs/heads/%s", branch), map[string]any{"sha": sha, "force": true}, nil)
}

func (c *Client) ListPulls(ctx context.Context, q url.Values) ([]PullRequest, error) {
	return paged[PullRequest](ctx, c, c.repoPath("/pulls"), q, 1)
}

func (c *Client) CreatePull(ctx context.Context, title, head, base, body string) (PullRequest, error) {
	var pr PullRequest
	err := c.do(ctx, http.MethodPost, c.repoPath("/pulls"), map[string]string{"title": title, "head": head, "base": base, "body": body}, &pr)
	return pr, err
}

func (c *Client) UpdatePull(ctx context.Context, number int, title, body string) (PullRequest, error) {
	var pr PullRequest
	err := c.do(ctx, http.MethodPatch, c.repoPath("/pulls/%d", number), map[string]string{"title": title, "body": body}, &pr)
	return pr, err
}

func (c *Client) AddLabels(ctx context.Context, number int, labels ...string) error {
	return c.do(ctx, http.MethodPost, c.repoPath("/issues/%d/labels", number), map[string][]string{"labels": labels}, nil)
}

func (c *Client) RemoveLabel(ctx context.Context, number int, label string) error {
	err := c.do(ctx, http.MethodDelete, c.repoPath("/issues/%d/labels/%s", number, url.PathEscape(label)), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// ReleaseByTag returns nil if no release exists for tag.
func (c *Client) ReleaseByTag(ctx context.Context, tag string) (*Release, error) {
	var r Release
	err := c.do(ctx, http.MethodGet, c.repoPath("/releases/tags/%s", url.PathEscape(tag)), nil, &r)
	if IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRelease creates a release and, if missing, its tag at target.
func (c *Client) CreateRelease(ctx context.Context, tag, target, name, body string, prerelease bool) (Release, error) {
	var r Release
	err := c.do(ctx, http.MethodPost, c.repoPath("/releases"), map[string]any{
		"tag_name":         tag,
		"target_commitish": target,
		"name":             name,
		"body":             body,
		"prerelease":       prerelease,
	}, &r)
	return r, err
}
