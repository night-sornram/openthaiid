package bindstress

import "bytes"

// APDU dumps must never appear in reports. The Thai applet SELECT AID
// and READ BINARY prefix are the traces this harness is tempted to log;
// block them. Personal fields are already refused by qualify.FirstForbidden.
func firstAPDUDump(p []byte) string {
	lower := bytes.ToLower(p)
	compact := make([]byte, 0, len(lower))
	for _, b := range lower {
		if b == ' ' || b == '-' || b == ':' {
			continue
		}
		compact = append(compact, b)
	}
	needles := [][]byte{
		[]byte("00a4040008a000000054480001"),
		[]byte("a000000054480001"),
		[]byte("80b00004"), // CID READ BINARY — never a bind-stress field
		[]byte("\\x00\\xa4\\x04"),
	}
	for _, n := range needles {
		if bytes.Contains(lower, n) || bytes.Contains(compact, n) {
			return string(n)
		}
	}
	return ""
}
