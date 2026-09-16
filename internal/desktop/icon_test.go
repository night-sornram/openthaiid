package desktop

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestIconPNGAndICO(t *testing.T) {
	p := IconPNG()
	img, err := png.Decode(bytes.NewReader(p))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("png size %v", img.Bounds())
	}
	ico := IconICO()
	if len(ico) < 22+40 {
		t.Fatalf("ico too small %d", len(ico))
	}
	if binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		t.Fatal("ico type")
	}
	assertNoPII(t, string(p))
}
