package update

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.0.1-dev", "0.1.0-rc.1", -1},
		{"v0.1.0-rc.1", "0.1.0-rc.1", 0},
		{"0.1.0-rc.1", "0.1.0", -1},
		{"0.1.0", "0.1.0-rc.1", 1},
		{"0.1.0-rc.1", "0.1.0-rc.2", -1},
		{"0.1.0-rc.2", "0.1.0-rc.3", -1},
		{"0.1.0-rc.3", "0.1.0-rc.4", -1},
		{"0.1.0-rc.4", "0.1.0-rc.5", -1},
		{"0.1.0-rc.5", "0.1.0-rc.6", -1},
		{"0.1.0-rc.6", "0.1.0-rc.7", -1},
		{"1.2.3", "1.2.4", -1},
		{"v1.0.0", "1.0.0+build", 0},
	}
	for _, c := range cases {
		got := CompareVersions(c.a, c.b)
		if got != c.want {
			t.Fatalf("CompareVersions(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}
