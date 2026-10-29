package semver

import "testing"

func TestParseRoundTrip(t *testing.T) {
	for _, s := range []string{"0.0.1", "1.2.3", "1.2.3-beta.1", "1.2.3-rc.1+build.5", "10.20.30+meta"} {
		v, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		if v.String() != s {
			t.Errorf("round trip %q -> %q", s, v.String())
		}
	}
	for _, s := range []string{"", "1.2", "01.2.3", "v1.2.3", "1.2.3-"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should fail", s)
		}
	}
}

func TestBump(t *testing.T) {
	v, _ := Parse("1.2.3-beta.1")
	cases := map[Bump]string{Major: "2.0.0", Minor: "1.3.0", Patch: "1.2.4", None: "1.2.3-beta.1"}
	for b, want := range cases {
		if got := v.Bump(b).String(); got != want {
			t.Errorf("Bump(%d) = %s, want %s", b, got, want)
		}
	}
}

func TestCompare(t *testing.T) {
	// Ordered example from semver.org section 11.
	order := []string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0"}
	for i := 0; i+1 < len(order); i++ {
		a, _ := Parse(order[i])
		b, _ := Parse(order[i+1])
		if a.Compare(b) != -1 || b.Compare(a) != 1 {
			t.Errorf("expected %s < %s", a, b)
		}
	}
}
