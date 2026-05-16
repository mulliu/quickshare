package qr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"

	"rsc.io/qr"
)

// GeneratePNG encodes text as a QR code and returns PNG bytes.
func GeneratePNG(text string, scale int) ([]byte, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, err
	}
	size := code.Size
	ps := size * scale
	img := image.NewGray(image.Rect(0, 0, ps, ps))
	for y := 0; y < ps; y++ {
		for x := 0; x < ps; x++ {
			if code.Black(x/scale, y/scale) {
				img.SetGray(x, y, color.Gray{Y: 0})
			} else {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderASCII writes a QR code to w using only ASCII characters.
// Each module is 2 characters wide: "##" for black, "  " for white.
// Fully compatible with all terminal encodings (no Unicode needed).
func RenderASCII(text string, w io.Writer) error {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return err
	}
	size := code.Size
	padded := size + 4

	for y := 0; y < padded; y++ {
		for x := 0; x < padded; x++ {
			px := x - 2
			py := y - 2
			black := false
			if px >= 0 && py >= 0 && px < size && py < size {
				black = code.Black(px, py)
			}
			if black {
				w.Write([]byte("@@"))
			} else {
				w.Write([]byte("  "))
			}
		}
		w.Write([]byte("\n"))
	}
	return nil
}
