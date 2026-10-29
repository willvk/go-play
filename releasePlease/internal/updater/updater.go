// Package updater rewrites files for a new release.
package updater

import (
	"regexp"
	"strings"
)

// PrependChangelog inserts notes above the newest existing release entry,
// keeping any preamble (such as the "# Changelog" heading) at the top.
func PrependChangelog(existing, notes string) string {
	notes = strings.TrimRight(notes, "\n") + "\n"
	if strings.TrimSpace(existing) == "" {
		return "# Changelog\n\n" + notes
	}
	idx := -1
	if strings.HasPrefix(existing, "## ") {
		idx = 0
	} else if i := strings.Index(existing, "\n## "); i >= 0 {
		idx = i + 1
	}
	if idx < 0 {
		return strings.TrimRight(existing, "\n") + "\n\n" + notes
	}
	return existing[:idx] + notes + "\n" + existing[idx:]
}

var versionRe = regexp.MustCompile(`\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?`)

// Generic replaces versions using release-please's annotation comments:
// any line containing "x-release-please-version", and every line between
// "x-release-please-start-version" and "x-release-please-end".
func Generic(content, version string) string {
	lines := strings.SplitAfter(content, "\n")
	inBlock := false
	for i, l := range lines {
		switch {
		case strings.Contains(l, "x-release-please-start-version"):
			inBlock = true
		case strings.Contains(l, "x-release-please-end"):
			inBlock = false
		case inBlock || strings.Contains(l, "x-release-please-version"):
			lines[i] = versionRe.ReplaceAllString(l, version)
		}
	}
	return strings.Join(lines, "")
}
