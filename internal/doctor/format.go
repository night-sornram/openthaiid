package doctor

import (
	"fmt"
	"strconv"
	"strings"
)

// Format prints a bilingual technician snapshot. It never includes personal
// card fields, tokens, or private-key paths. The line `personal fields: not read`
// is always present. Hardware kit B3 looks for `pcsc: OK` on a healthy desk.
func Format(s Snapshot) string {
	s = Finalize(s)
	var b strings.Builder
	ver := s.Version
	if ver == "" {
		ver = "dev"
	}
	fmt.Fprintf(&b, "openthaiid doctor (%s)\n", ver)
	if s.HardFail {
		code := s.FailCode
		if code == "" {
			code = "FAIL"
		}
		fmt.Fprintf(&b, "result: FAIL (%s)\n", code)
	} else {
		b.WriteString("result: OK\n")
	}
	fmt.Fprintf(&b, "desk: %s\n", deskLine(s))
	helperVer := ver
	if s.Helper == HelperRunning && s.Version != "" {
		helperVer = s.Version
	}
	fmt.Fprintf(&b, "helper: %s  version=%s\n", s.Helper, helperVer)
	fmt.Fprintf(&b, "listen: %s  tls=%s", orDash(s.HTTP), orDash(s.TLS))
	if s.TLS == TLSYes && s.TLSListen != "" {
		fmt.Fprintf(&b, "  https=%s", s.TLSListen)
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "ready: %s\n", orDash(s.ReadyURL))
	fmt.Fprintf(&b, "pcsc: %s", s.PCSC)
	if s.Backend != "" {
		fmt.Fprintf(&b, " backend=%s", s.Backend)
	}
	if s.BackendStatus != "" {
		fmt.Fprintf(&b, " status=%s", s.BackendStatus)
	}
	fmt.Fprintf(&b, " readers=%s\n", strconv.Itoa(len(s.Readers)))
	for _, r := range s.Readers {
		fmt.Fprintf(&b, "  reader: %s", r.Name)
		if r.Selected {
			b.WriteString("  selected")
		}
		if r.Presence != "" {
			fmt.Fprintf(&b, "  presence=%s", r.Presence)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "probe: %s\n", probeLine(s.Probe))
	fmt.Fprintf(&b, "update: %s\n", updateLine(s.Update))
	b.WriteString("personal fields: not read\n")
	if s.Hint != "" {
		fmt.Fprintf(&b, "hint: %s\n", s.Hint)
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "ถัดไป: %s\n", s.NextTH)
	fmt.Fprintf(&b, "Next: %s\n", s.NextEN)
	out := b.String()
	if ContainsForbidden(out) {
		return "openthaiid doctor\nresult: FAIL\npersonal fields: not read\n\nถัดไป: เปิด Technician tools แล้วใช้ Copy diagnostics\nNext: Open Technician tools and use Copy diagnostics.\n"
	}
	return out
}

func deskLine(s Snapshot) string {
	if s.HardFail {
		return "ยังไม่พร้อม / Not ready yet"
	}
	if s.Helper != HelperRunning || s.PCSC != PCSCOK {
		return "ยังไม่พร้อม / Not ready yet"
	}
	if len(s.Readers) == 0 {
		return "ยังไม่พร้อม / Not ready yet"
	}
	selected := 0
	present := false
	for _, r := range s.Readers {
		if r.Selected {
			selected++
			if r.Presence == "present" {
				present = true
			}
		}
	}
	if len(s.Readers) > 1 && selected != 1 {
		return "ยังไม่พร้อม / Not ready yet"
	}
	if selected == 0 && len(s.Readers) == 1 {
		present = s.Readers[0].Presence == "present"
	}
	if !present {
		return "ยังไม่พร้อม / Not ready yet"
	}
	if s.Practice {
		return "พร้อม (บัตรฝึกซ้อม) / Ready (practice card)"
	}
	return "พร้อม / Ready"
}

func probeLine(p Probe) string {
	if p.Status == HelperDown {
		return "skipped (helper down)"
	}
	if p.OK && p.Backend != "" {
		return "OK backend=" + p.Backend
	}
	if p.Backend != "" {
		return "backend=" + p.Backend
	}
	if p.Status != "" {
		return p.Status
	}
	if p.OK {
		return "OK"
	}
	return "unknown"
}

func updateLine(u Update) string {
	state := u.State
	if state == "" {
		state = UpdateNotChecked
	}
	switch state {
	case UpdateNotChecked:
		return "not checked (offline-safe; openthaiid update check)"
	case UpdateUnavailable:
		reason := u.Reason
		if reason == "" {
			reason = "offline"
		}
		return "unavailable reason=" + reason + "  (soft; Fill still works)"
	case UpdateNewer:
		line := "newer"
		if u.Tag != "" {
			line += " tag=" + u.Tag
		}
		return line + "  (soft; unsigned; not installed)"
	case UpdateUpToDate:
		return "up to date  (soft; last GitHub check)"
	case UpdateNone:
		return "none  (soft; no GitHub Releases yet)"
	default:
		line := state
		if u.Reason != "" {
			line += " reason=" + u.Reason
		}
		if u.Tag != "" {
			line += " tag=" + u.Tag
		}
		return line + "  (soft)"
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
