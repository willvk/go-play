// Package changelog renders release notes in the same markdown shape as
// release-please (which follows conventional-changelog).
package changelog

import (
	"fmt"
	"strings"
	"time"

	"releasePlease/internal/commits"
)

type Section struct {
	Type   string
	Title  string
	Hidden bool
}

// DefaultSections mirrors release-please's default changelog-sections.
var DefaultSections = []Section{
	{Type: "feat", Title: "Features"},
	{Type: "feature", Title: "Features"},
	{Type: "fix", Title: "Bug Fixes"},
	{Type: "perf", Title: "Performance Improvements"},
	{Type: "revert", Title: "Reverts"},
	{Type: "deps", Title: "Dependencies"},
	{Type: "docs", Title: "Documentation", Hidden: true},
	{Type: "style", Title: "Styles", Hidden: true},
	{Type: "chore", Title: "Miscellaneous Chores", Hidden: true},
	{Type: "refactor", Title: "Code Refactoring", Hidden: true},
	{Type: "test", Title: "Tests", Hidden: true},
	{Type: "build", Title: "Build System", Hidden: true},
	{Type: "ci", Title: "Continuous Integration", Hidden: true},
}

type Options struct {
	Version     string
	CurrentTag  string
	PreviousTag string
	Date        time.Time
	// RepoURL is the web URL of the repository, e.g. https://github.com/o/r.
	// Leave empty to render notes without links.
	RepoURL  string
	Sections []Section
}

// Releasable reports whether any commit would show up in the changelog or
// otherwise forces a release.
func Releasable(sections []Section, cs []commits.Commit) bool {
	for _, c := range cs {
		if c.Breaking || c.ReleaseAs != "" {
			return true
		}
		if s, ok := lookup(sections, c.Type); ok && !s.Hidden {
			return true
		}
	}
	return false
}

// Build renders the notes for one release. Commits are listed in the order
// given, which is newest first when they come from the GitHub API.
func Build(opts Options, cs []commits.Commit) string {
	sections := opts.Sections
	if sections == nil {
		sections = DefaultSections
	}

	var b strings.Builder
	date := opts.Date.Format("2006-01-02")
	if opts.PreviousTag != "" && opts.RepoURL != "" {
		fmt.Fprintf(&b, "## [%s](%s/compare/%s...%s) (%s)\n", opts.Version, opts.RepoURL, opts.PreviousTag, opts.CurrentTag, date)
	} else {
		fmt.Fprintf(&b, "## %s (%s)\n", opts.Version, date)
	}

	var titles []string
	entries := map[string][]string{}
	add := func(title, line string) {
		if _, ok := entries[title]; !ok {
			titles = append(titles, title)
		}
		entries[title] = append(entries[title], line)
	}

	const breakingTitle = "⚠ BREAKING CHANGES"
	for _, c := range cs {
		for _, note := range c.BreakingNotes {
			add(breakingTitle, entry(c, note, opts.RepoURL))
		}
	}
	for _, s := range sections {
		if s.Hidden {
			continue
		}
		for _, c := range cs {
			if c.Conventional && c.Type == s.Type {
				add(s.Title, entry(c, c.Description, opts.RepoURL))
			}
		}
	}

	for _, t := range titles {
		fmt.Fprintf(&b, "\n\n### %s\n\n%s", t, strings.Join(entries[t], "\n"))
	}
	b.WriteString("\n")
	return b.String()
}

func entry(c commits.Commit, text, repoURL string) string {
	s := "* "
	if c.Scope != "" {
		s += "**" + c.Scope + ":** "
	}
	s += text
	if c.SHA != "" {
		short := c.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		if repoURL != "" {
			s += fmt.Sprintf(" ([%s](%s/commit/%s))", short, repoURL, c.SHA)
		} else {
			s += " (" + short + ")"
		}
	}
	return s
}

func lookup(sections []Section, typ string) (Section, bool) {
	if sections == nil {
		sections = DefaultSections
	}
	for _, s := range sections {
		if s.Type == typ {
			return s, true
		}
	}
	return Section{}, false
}
