package updater

import "testing"

func TestPrependChangelog(t *testing.T) {
	notes := "## 1.1.0 (2026-10-29)\n\n\n### Features\n\n* x\n"
	cases := map[string]string{
		"":                          "# Changelog\n\n" + notes,
		"# Changelog\n\n## 1.0.0\n": "# Changelog\n\n" + notes + "\n## 1.0.0\n",
		"## 1.0.0\n":                notes + "\n## 1.0.0\n",
		"# Changelog\n":             "# Changelog\n\n" + notes,
	}
	for in, want := range cases {
		if got := PrependChangelog(in, notes); got != want {
			t.Errorf("PrependChangelog(%q) =\n%q\nwant\n%q", in, got, want)
		}
	}
}

func TestGeneric(t *testing.T) {
	in := `package main

const Version = "1.2.3" // x-release-please-version
const Other = "1.2.3"

// x-release-please-start-version
var image = "app:1.2.3-beta.1"
// x-release-please-end
var after = "1.2.3"
`
	want := `package main

const Version = "2.0.0" // x-release-please-version
const Other = "1.2.3"

// x-release-please-start-version
var image = "app:2.0.0"
// x-release-please-end
var after = "1.2.3"
`
	if got := Generic(in, "2.0.0"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
