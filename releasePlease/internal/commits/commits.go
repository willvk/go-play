// Package commits parses Conventional Commits (https://www.conventionalcommits.org).
package commits

import (
	"regexp"
	"strings"
)

type Commit struct {
	SHA     string
	Message string

	Conventional bool
	Type         string
	Scope        string
	Description  string

	Breaking      bool
	BreakingNotes []string

	// ReleaseAs is set from a "Release-As: x.y.z" footer and forces the
	// next version.
	ReleaseAs string
}

var (
	headerRe    = regexp.MustCompile(`^([A-Za-z]+)(?:\(([^()\r\n]*)\))?(!)?:[ \t]+(.+)$`)
	breakingRe  = regexp.MustCompile(`^BREAKING[ -]CHANGE:[ \t]*(.*)$`)
	releaseAsRe = regexp.MustCompile(`(?i)^release-as:[ \t]*(\S+)[ \t]*$`)
	footerRe    = regexp.MustCompile(`^(?:BREAKING[ -]CHANGE|[A-Za-z][A-Za-z-]*)(?:: | #)`)
)

func Parse(sha, message string) Commit {
	c := Commit{SHA: sha, Message: message}
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	header := strings.TrimSpace(lines[0])
	c.Description = header

	if m := headerRe.FindStringSubmatch(header); m != nil {
		c.Conventional = true
		c.Type = strings.ToLower(m[1])
		c.Scope = strings.TrimSpace(m[2])
		c.Breaking = m[3] == "!"
		c.Description = strings.TrimSpace(m[4])
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if m := releaseAsRe.FindStringSubmatch(line); m != nil {
			c.ReleaseAs = strings.TrimPrefix(m[1], "v")
			continue
		}
		m := breakingRe.FindStringSubmatch(line)
		if m == nil || !c.Conventional {
			continue
		}
		note := strings.TrimSpace(m[1])
		// A footer value continues until a blank line or the next footer.
		for i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next == "" || footerRe.MatchString(next) {
				break
			}
			note = strings.TrimSpace(note + " " + next)
			i++
		}
		c.Breaking = true
		if note != "" {
			c.BreakingNotes = append(c.BreakingNotes, note)
		}
	}

	if c.Breaking && len(c.BreakingNotes) == 0 {
		c.BreakingNotes = []string{c.Description}
	}
	return c
}
