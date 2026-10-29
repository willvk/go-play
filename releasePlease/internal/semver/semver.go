// Package semver implements the subset of Semantic Versioning 2.0.0 that
// release-please needs: parsing, ordering and bumping.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Prerelease          string
	Build               string
}

type Bump int

const (
	None Bump = iota
	Patch
	Minor
	Major
)

var re = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

func Parse(s string) (Version, error) {
	m := re.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Version{}, fmt.Errorf("invalid semver %q", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return Version{Major: maj, Minor: min, Patch: pat, Prerelease: m[4], Build: m[5]}, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Bump returns the next version for the given bump, dropping any
// prerelease or build metadata.
func (v Version) Bump(b Bump) Version {
	switch b {
	case Major:
		return Version{Major: v.Major + 1}
	case Minor:
		return Version{Major: v.Major, Minor: v.Minor + 1}
	case Patch:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
	return v
}

// Compare returns -1, 0 or 1. Build metadata is ignored, as per the spec.
func (v Version) Compare(o Version) int {
	for _, d := range []int{v.Major - o.Major, v.Minor - o.Minor, v.Patch - o.Patch} {
		if d != 0 {
			return sign(d)
		}
	}
	switch {
	case v.Prerelease == o.Prerelease:
		return 0
	case v.Prerelease == "":
		return 1
	case o.Prerelease == "":
		return -1
	}
	a, b := strings.Split(v.Prerelease, "."), strings.Split(o.Prerelease, ".")
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := compareIdent(a[i], b[i]); c != 0 {
			return c
		}
	}
	return sign(len(a) - len(b))
}

func compareIdent(a, b string) int {
	an, aErr := strconv.Atoi(a)
	bn, bErr := strconv.Atoi(b)
	switch {
	case aErr == nil && bErr == nil:
		return sign(an - bn)
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	}
	return strings.Compare(a, b)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
