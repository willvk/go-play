// Command release-please is a Go port of the core of googleapis/release-please.
//
// Usage:
//
//	release-please [flags] release-pr       open or update the release PR
//	release-please [flags] github-release   tag releases for merged release PRs
//	release-please [flags] all              github-release, then release-pr (like the GitHub Action)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"releasePlease/internal/github"
	"releasePlease/internal/release"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("release-please", flag.ContinueOnError)
	var (
		token      = fs.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token (default $GITHUB_TOKEN)")
		repo       = fs.String("repo", os.Getenv("GITHUB_REPOSITORY"), "owner/name (default $GITHUB_REPOSITORY)")
		apiURL     = fs.String("api-url", envOr("GITHUB_API_URL", "https://api.github.com"), "GitHub API URL")
		webURL     = fs.String("server-url", envOr("GITHUB_SERVER_URL", "https://github.com"), "GitHub web URL, used for changelog links")
		branch     = fs.String("branch", "", "target branch (default: the repository's default branch)")
		tagPrefix  = fs.String("tag-prefix", "v", "prefix for release tags")
		initial    = fs.String("initial-version", "1.0.0", "version to use when there is no previous release")
		changelog  = fs.String("changelog-path", "CHANGELOG.md", "changelog file; empty to disable")
		versionTxt = fs.String("version-file", "version.txt", "plain-text version file; empty to disable")
		extra      = fs.String("extra-files", "", "comma-separated files containing x-release-please-version annotations")
		minorPre   = fs.Bool("bump-minor-pre-major", false, "breaking changes bump minor, not major, before 1.0.0")
		patchPre   = fs.Bool("bump-patch-for-minor-pre-major", false, "features bump patch, not minor, before 1.0.0")
		dryRun     = fs.Bool("dry-run", false, "print what would happen without writing to GitHub")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: release-please [flags] release-pr|github-release|all\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one command")
	}
	cmd := fs.Arg(0)

	owner, name, ok := strings.Cut(*repo, "/")
	if !ok || owner == "" || name == "" {
		return fmt.Errorf("--repo must be owner/name, got %q", *repo)
	}
	if *token == "" && !*dryRun {
		return fmt.Errorf("a GitHub token is required (--token or $GITHUB_TOKEN)")
	}

	ctx := context.Background()
	gh := &github.Client{BaseURL: *apiURL, Token: *token, Owner: owner, Repo: name}
	if *branch == "" {
		b, err := gh.DefaultBranch(ctx)
		if err != nil {
			return fmt.Errorf("looking up default branch: %w", err)
		}
		*branch = b
	}

	var extraFiles []string
	for _, f := range strings.Split(*extra, ",") {
		if f = strings.TrimSpace(f); f != "" {
			extraFiles = append(extraFiles, f)
		}
	}

	r := &release.Runner{
		GH:  gh,
		Log: os.Stderr,
		Cfg: release.Config{
			Branch:                    *branch,
			TagPrefix:                 *tagPrefix,
			InitialVersion:            *initial,
			BumpMinorPreMajor:         *minorPre,
			BumpPatchForMinorPreMajor: *patchPre,
			ChangelogPath:             *changelog,
			VersionFile:               *versionTxt,
			ExtraFiles:                extraFiles,
			WebURL:                    *webURL,
			DryRun:                    *dryRun,
		},
	}

	out := map[string]string{}
	switch cmd {
	case "release-pr":
		return releasePR(ctx, r, out)
	case "github-release":
		return githubRelease(ctx, r, out)
	case "all":
		if err := githubRelease(ctx, r, out); err != nil {
			return err
		}
		return releasePR(ctx, r, out)
	}
	fs.Usage()
	return fmt.Errorf("unknown command %q", cmd)
}

func releasePR(ctx context.Context, r *release.Runner, out map[string]string) error {
	res, err := r.ReleasePR(ctx)
	if err != nil {
		return err
	}
	if res != nil && res.PR.Number != 0 {
		out["pr"] = strconv.Itoa(res.PR.Number)
		out["pr_url"] = res.PR.HTMLURL
	}
	return writeOutputs(out)
}

func githubRelease(ctx context.Context, r *release.Runner, out map[string]string) error {
	created, err := r.GitHubRelease(ctx)
	if err != nil {
		return err
	}
	out["release_created"] = strconv.FormatBool(len(created) > 0)
	if len(created) > 0 {
		out["tag_name"] = created[0].Tag
		out["version"] = created[0].Version
		out["html_url"] = created[0].URL
	}
	return writeOutputs(out)
}

// writeOutputs appends step outputs when running inside GitHub Actions.
func writeOutputs(out map[string]string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" || len(out) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range out {
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	clear(out)
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
