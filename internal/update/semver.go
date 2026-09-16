package update

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseVersion splits a SemVer-ish string (optional leading "v").
// Pre-release is kept; build metadata is discarded.
func ParseVersion(s string) (major, minor, patch int, pre []string, err error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" {
		return 0, 0, 0, nil, fmt.Errorf("empty version")
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	core, preStr, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return 0, 0, 0, nil, fmt.Errorf("invalid version")
	}
	nums := [3]int{}
	for i, p := range parts {
		n, convErr := strconv.Atoi(p)
		if convErr != nil || n < 0 {
			return 0, 0, 0, nil, fmt.Errorf("invalid version")
		}
		nums[i] = n
	}
	var preParts []string
	if preStr != "" {
		preParts = strings.Split(preStr, ".")
	}
	return nums[0], nums[1], nums[2], preParts, nil
}

// CompareVersions returns -1 if a<b, 0 if equal, 1 if a>b.
// A version with a pre-release is older than the same core without one
// (0.1.0-rc.1 < 0.1.0). Unparseable strings compare as equal to themselves
// and older than a parsed release.
func CompareVersions(a, b string) int {
	am, ai, ap, apre, aerr := ParseVersion(a)
	bm, bi, bp, bpre, berr := ParseVersion(b)
	if aerr != nil && berr != nil {
		if a == b {
			return 0
		}
		if a < b {
			return -1
		}
		return 1
	}
	if aerr != nil {
		return -1
	}
	if berr != nil {
		return 1
	}
	if am != bm {
		return cmpInt(am, bm)
	}
	if ai != bi {
		return cmpInt(ai, bi)
	}
	if ap != bp {
		return cmpInt(ap, bp)
	}
	return comparePre(apre, bpre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func comparePre(a, b []string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1
	}
	if len(b) == 0 {
		return -1
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if c := compareIdent(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(a), len(b))
}

func compareIdent(a, b string) int {
	an, aNum := strconv.Atoi(a)
	bn, bNum := strconv.Atoi(b)
	if aNum == nil && bNum == nil {
		return cmpInt(an, bn)
	}
	if aNum == nil {
		return -1
	}
	if bNum == nil {
		return 1
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
