package desktop

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

const iconSize = 32

// IconPNG is a 32×32 helper glyph (no text, no government marks).
func IconPNG() []byte {
	img := iconRGBA()
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// IconICO is a 32×32 Windows icon built from the same glyph.
func IconICO() []byte {
	img := iconRGBA()
	const w, h = iconSize, iconSize
	xorSize := w * h * 4
	andRow := ((w + 31) / 32) * 4
	andSize := andRow * h
	dibSize := 40 + xorSize + andSize

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	buf.WriteByte(byte(w))
	buf.WriteByte(byte(h))
	buf.WriteByte(0)
	buf.WriteByte(0)
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(buf, binary.LittleEndian, uint32(dibSize))
	_ = binary.Write(buf, binary.LittleEndian, uint32(22))
	_ = binary.Write(buf, binary.LittleEndian, uint32(40))
	_ = binary.Write(buf, binary.LittleEndian, int32(w))
	_ = binary.Write(buf, binary.LittleEndian, int32(h*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(xorSize+andSize))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			buf.WriteByte(byte(b >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(r >> 8))
			buf.WriteByte(byte(a >> 8))
		}
	}
	buf.Write(make([]byte, andSize))
	return buf.Bytes()
}

func iconRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	teal := color.RGBA{R: 0x0f, G: 0x76, B: 0x6e, A: 0xff}
	card := color.RGBA{R: 0xf8, G: 0xfa, B: 0xfc, A: 0xff}
	chip := color.RGBA{R: 0xcb, G: 0xd5, B: 0xe1, A: 0xff}
	cx, cy, rad := 16, 16, 14
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= rad*rad {
				img.SetRGBA(x, y, teal)
			}
		}
	}
	for y := 10; y <= 22; y++ {
		for x := 8; x <= 23; x++ {
			img.SetRGBA(x, y, card)
		}
	}
	for y := 13; y <= 16; y++ {
		for x := 10; x <= 14; x++ {
			img.SetRGBA(x, y, chip)
		}
	}
	return img
}
