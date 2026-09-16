package card

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
)

// Portrait is the optional JPEG face photo assembled from the cited 20-part
// READ BINARY blocks (dotnetthailand APDU.md Photo_Part1/20 … Photo_Part20/20).
//
// Protocol encoding is explicit Base64 so JavaScript consumers do not depend
// on Go's []byte JSON convention. Applications typically turn this into a
// data URL (`data:<mimeType>;base64,<data>`) or a Blob.
type Portrait struct {
	MimeType string `json:"mimeType"`
	Encoding string `json:"encoding"`
	Data     string `json:"data"`
}

const (
	PortraitMimeJPEG       = "image/jpeg"
	PortraitEncodingBase64 = "base64"

	// FictionalPortraitSentinel is embedded as a JPEG COM marker in the
	// synthetic fixture. It must never appear in logs or acceptance reports.
	FictionalPortraitSentinel = "FICTIONAL-OPENTHAIID-PORTRAIT"
)

// Portrait layout from the cited APDU table (independent reconstruction):
//
//	Photo_Part1/20  CLA=80 INS=B0 P1=01 P2=7B Lc=02 Data=00 Le=FF  offset 0x017B
//	Photo_PartN/20  offset = 0x017B + (N-1)*0xFF, Le=0xFF
//	Photo_Part20/20 P1=14 P2=68  offset 0x1468
const (
	offPortrait       = 0x017B
	lenPortraitPart   = 0xFF
	portraitPartCount = 20
	lenPortrait       = portraitPartCount * lenPortraitPart // 5100
)

// PortraitSize is the concatenated 20×255-byte photo region.
const PortraitSize = lenPortrait

func portraitPartOffset(i int) int {
	return offPortrait + i*lenPortraitPart
}

func portraitCommandName(offset int) string {
	return "PHOTO@" + hexOffset(offset)
}

func hexOffset(offset int) string {
	const digits = "0123456789ABCDEF"
	return "0x" + string([]byte{
		digits[(offset>>12)&0xF],
		digits[(offset>>8)&0xF],
		digits[(offset>>4)&0xF],
		digits[offset&0xF],
	})
}

// FictionalPortraitJPEG returns a tiny geometric JPEG that is obviously not a
// person. Used only by the synthetic backend and tests.
func FictionalPortraitJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	teal := color.RGBA{R: 0x20, G: 0x90, B: 0x88, A: 0xFF}
	cream := color.RGBA{R: 0xE0, G: 0xE8, B: 0xD8, A: 0xFF}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if (x/4+y/4)%2 == 0 {
				img.Set(x, y, teal)
			} else {
				img.Set(x, y, cream)
			}
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		// encoding a 16×16 RGBA never fails in practice; keep tests deterministic
		return []byte{0xFF, 0xD8, 0xFF, 0xD9}
	}
	return insertJPEGComment(buf.Bytes(), FictionalPortraitSentinel)
}

func insertJPEGComment(jpegBytes []byte, comment string) []byte {
	if len(jpegBytes) < 2 || jpegBytes[0] != 0xFF || jpegBytes[1] != 0xD8 {
		return jpegBytes
	}
	c := []byte(comment)
	length := len(c) + 2
	com := []byte{0xFF, 0xFE, byte(length >> 8), byte(length)}
	com = append(com, c...)
	out := make([]byte, 0, 2+len(com)+len(jpegBytes)-2)
	out = append(out, 0xFF, 0xD8)
	out = append(out, com...)
	out = append(out, jpegBytes[2:]...)
	return out
}

// PadPortraitRegion copies jpeg into a 5100-byte photo region (trailing NULs).
func PadPortraitRegion(jpeg []byte) []byte {
	out := make([]byte, lenPortrait)
	copy(out, jpeg)
	return out
}
