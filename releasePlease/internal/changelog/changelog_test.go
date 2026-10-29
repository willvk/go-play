package changelog

import (
	"testing"
	"time"

	"releasePlease/internal/commits"
)

func TestBuild(t *testing.T) {
	cs := []commits.Commit{
		commits.Parse("aaaaaaaaaa", "feat(api)!: new auth flow"),
		commits.Parse("bbbbbbbbbb", "fix: off by one"),
		commits.Parse("cccccccccc", "chore: tidy"),
		commits.Parse("dddddddddd", "feat: dark mode"),
	}
	got := Build(Options{
		Version:     "2.0.0",
		CurrentTag:  "v2.0.0",
		PreviousTag: "v1.4.0",
		Date:        time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		RepoURL:     "https://github.com/o/r",
	}, cs)

	want := `## [2.0.0](https://github.com/o/r/compare/v1.4.0...v2.0.0) (2026-10-29)


### ⚠ BREAKING CHANGES

* **api:** new auth flow ([aaaaaaa](https://github.com/o/r/commit/aaaaaaaaaa))

### Features

* **api:** new auth flow ([aaaaaaa](https://github.com/o/r/commit/aaaaaaaaaa))
* dark mode ([ddddddd](https://github.com/o/r/commit/dddddddddd))

### Bug Fixes

* off by one ([bbbbbbb](https://github.com/o/r/commit/bbbbbbbbbb))
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReleasable(t *testing.T) {
	hidden := []commits.Commit{commits.Parse("", "chore: x"), commits.Parse("", "docs: y"), commits.Parse("", "random")}
	if Releasable(DefaultSections, hidden) {
		t.Error("chore/docs only should not be releasable")
	}
	if !Releasable(DefaultSections, append(hidden, commits.Parse("", "chore!: drop node 14"))) {
		t.Error("breaking chore should be releasable")
	}
	if !Releasable(DefaultSections, []commits.Commit{commits.Parse("", "fix: y")}) {
		t.Error("fix should be releasable")
	}
}
