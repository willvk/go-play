package commits

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name, msg string
		want      Commit
	}{
		{
			name: "simple feat",
			msg:  "feat: add widget",
			want: Commit{Conventional: true, Type: "feat", Description: "add widget"},
		},
		{
			name: "scope and bang",
			msg:  "Fix(api)!: drop v1 endpoint",
			want: Commit{Conventional: true, Type: "fix", Scope: "api", Description: "drop v1 endpoint", Breaking: true, BreakingNotes: []string{"drop v1 endpoint"}},
		},
		{
			name: "breaking footer spanning lines",
			msg:  "refactor: rework config\n\nSome body.\n\nBREAKING CHANGE: config moved\nto config.yaml\nRefs: #12",
			want: Commit{Conventional: true, Type: "refactor", Description: "rework config", Breaking: true, BreakingNotes: []string{"config moved to config.yaml"}},
		},
		{
			name: "release-as footer",
			msg:  "chore: release 2.0.0\n\nRelease-As: v2.0.0",
			want: Commit{Conventional: true, Type: "chore", Description: "release 2.0.0", ReleaseAs: "2.0.0"},
		},
		{
			name: "not conventional",
			msg:  "Merge branch 'main'\n\nBREAKING CHANGE: ignored",
			want: Commit{Description: "Merge branch 'main'"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse("", tc.msg)
			got.Message = ""
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
}
