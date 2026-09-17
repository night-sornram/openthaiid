package bindstress

import "testing"

func TestFirstAPDUDump(t *testing.T) {
	if hit := firstAPDUDump([]byte(`{"ok":true}`)); hit != "" {
		t.Fatalf("clean json hit %q", hit)
	}
	if hit := firstAPDUDump([]byte("00A4040008A000000054480001")); hit == "" {
		t.Fatal("expected SELECT hex")
	}
	if hit := firstAPDUDump([]byte("00 a4 04 00 08 a0 00 00 00 54 48 00 01")); hit == "" {
		t.Fatal("expected spaced SELECT hex")
	}
	if hit := firstAPDUDump([]byte("80b00004")); hit == "" {
		t.Fatal("expected CID READ BINARY prefix")
	}
}
