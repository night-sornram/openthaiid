package bindstress

import "strings"

// BindingIDs is the acceptance-matrix P-series this harness targets.
var BindingIDs = []string{"P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10"}

// HostEvidence lists which P-ids a hardware run on this GOOS/GOARCH may
// evidence. Synthetic never closes any of them.
func HostEvidence(goos, goarch string, synthetic bool) Evidence {
	ev := Evidence{
		NeedsACR39U: true,
	}
	if synthetic {
		ev.SyntheticDoesNot = append([]string(nil), BindingIDs...)
		return ev
	}
	may := platformIDs(goos, goarch)
	ev.ThisHostMayEvidence = may
	ev.StillNeedsOtherHost = complement(may)
	return ev
}

func platformIDs(goos, goarch string) []string {
	goos = strings.ToLower(goos)
	goarch = strings.ToLower(goarch)
	out := []string{"P7", "P8", "P9"}
	switch {
	case goos == "darwin" && goarch == "arm64":
		out = append([]string{"P2"}, out...)
		out = append(out, "P10")
	case goos == "darwin" && goarch == "amd64":
		out = append([]string{"P3"}, out...)
		out = append(out, "P10")
	case goos == "linux" && goarch == "amd64":
		out = append([]string{"P4"}, out...)
		out = append(out, "P10")
	case goos == "linux" && goarch == "arm64":
		out = append([]string{"P5"}, out...)
		out = append(out, "P10")
	case goos == "windows":
		out = append([]string{"P6"}, out...)
		// WinSCard is not cgo; P10 stays a darwin/linux item.
	}
	return out
}

func complement(have []string) []string {
	set := map[string]bool{}
	for _, id := range have {
		set[id] = true
	}
	var missing []string
	for _, id := range BindingIDs {
		if !set[id] {
			missing = append(missing, id)
		}
	}
	return missing
}
