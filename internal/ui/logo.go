package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strings"

	"github.com/disintegration/imaging"
)

// RenderImage converts an embedded PNG to half-block ANSI art.
// Uses ▀ (upper half block) to pack two pixel rows into one terminal row.
// width is the desired terminal column width.
func RenderImage(data []byte, width int) string {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	// Resize to target width, height auto-scaled
	resized := imaging.Resize(img, width, 0, imaging.Lanczos)
	bounds := resized.Bounds()
	w := bounds.Max.X
	h := bounds.Max.Y

	// Ensure even height for half-block pairing
	if h%2 != 0 {
		h--
	}

	var sb strings.Builder

	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x++ {
			upper := resized.At(x, y)
			lower := resized.At(x, y+1)

			ua := alphaOf(upper)
			la := alphaOf(lower)

			switch {
			case ua > 128 && la > 128:
				// Both pixels visible — upper fg, lower bg, use ▀
				r1, g1, b1 := rgbOf(upper)
				r2, g2, b2 := rgbOf(lower)
				sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀\033[0m", r1, g1, b1, r2, g2, b2))
			case ua > 128:
				// Only upper pixel
				r, g, b := rgbOf(upper)
				sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▀\033[0m", r, g, b))
			case la > 128:
				// Only lower pixel
				r, g, b := rgbOf(lower)
				sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▄\033[0m", r, g, b))
			default:
				// Both transparent
				sb.WriteRune(' ')
			}
		}
		if y+2 < h {
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}

func rgbOf(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

func alphaOf(c color.Color) uint8 {
	_, _, _, a := c.RGBA()
	return uint8(a >> 8)
}
