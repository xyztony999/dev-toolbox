// Command icongen renders the DevToolbox application icon: a warm paper
// rounded square and the copper key (ring bow, shaft, two teeth). The
// geometry matches assets/logo.svg and the in-app BrandMark. Run from the
// repo root:
//
//	go run ./tools/icongen
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// The output path is a literal so the generator can only ever overwrite
// this one asset inside the repo.
const masterSize = 512.0

type pt struct{ x, y float64 }

// geometry in master units (0..512)
const (
	inset      = 40.0
	squareSize = 432.0
	corner     = 96.0
)

var (
	paper  = color.RGBA{244, 239, 230, 255}
	copper = color.RGBA{163, 75, 42, 255}
)

// Key geometry in the same 116×58 space as BrandMark, placed in the 512 tile.
const (
	markScale = 400.0 / 116.0
	markX     = 56.0
	markY     = 156.0
)

func toMark(p pt) pt {
	return pt{(p.x - markX) / markScale, (p.y - markY) / markScale}
}

func inRoundRect(p pt, x, y, w, h, r float64) bool {
	if p.x < x || p.x > x+w || p.y < y || p.y > y+h {
		return false
	}
	cx := math.Max(x+r, math.Min(p.x, x+w-r))
	cy := math.Max(y+r, math.Min(p.y, y+h-r))
	return math.Hypot(p.x-cx, p.y-cy) <= r || (p.x >= x+r && p.x <= x+w-r) || (p.y >= y+r && p.y <= y+h-r)
}

func paintKey(p pt) bool {
	m := toMark(p)
	d := math.Hypot(m.x-28, m.y-28)
	if d <= 4 {
		return true
	}
	if d <= 10 {
		return false
	}
	if d <= 22 {
		return true
	}
	if inRoundRect(m, 42, 21, 62, 14, 3) || inRoundRect(m, 82, 30, 8, 16, 2) || inRoundRect(m, 96, 30, 8, 20, 2) {
		return true
	}
	return false
}

func inSquircle(p pt) bool {
	x0, y0 := inset, inset
	x1, y1 := inset+squareSize, inset+squareSize
	if p.x < x0 || p.x > x1 || p.y < y0 || p.y > y1 {
		return false
	}
	cx := math.Max(x0+corner, math.Min(p.x, x1-corner))
	cy := math.Max(y0+corner, math.Min(p.y, y1-corner))
	return math.Hypot(p.x-cx, p.y-cy) <= corner
}

func sample(p pt) color.RGBA {
	if !inSquircle(p) {
		return color.RGBA{0, 0, 0, 0}
	}
	if paintKey(p) {
		return copper
	}
	return paper
}

func render(size int) *image.RGBA {
	super := size * 2
	img := image.NewRGBA(image.Rect(0, 0, super, super))
	for py := 0; py < super; py++ {
		for px := 0; px < super; px++ {
			p := pt{x: (float64(px) + 0.5) / float64(super) * masterSize, y: (float64(py) + 0.5) / float64(super) * masterSize}
			img.Set(px, py, sample(p))
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r0 := img.RGBAAt(2*x, 2*y)
			r1 := img.RGBAAt(2*x+1, 2*y)
			r2 := img.RGBAAt(2*x, 2*y+1)
			r3 := img.RGBAAt(2*x+1, 2*y+1)
			avg := func(values ...uint8) uint8 {
				sum := uint32(0)
				for _, value := range values {
					sum += uint32(value)
				}
				return uint8(sum / uint32(len(values)))
			}
			out.SetRGBA(x, y, color.RGBA{
				avg(r0.R, r1.R, r2.R, r3.R),
				avg(r0.G, r1.G, r2.G, r3.G),
				avg(r0.B, r1.B, r2.B, r3.B),
				avg(r0.A, r1.A, r2.A, r3.A),
			})
		}
	}
	return out
}

func pngBytes(size int) []byte {
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, render(size)); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func writeICO(sizes []int) []byte {
	images := make([][]byte, len(sizes))
	for i, s := range sizes {
		images[i] = pngBytes(s)
	}
	out := &bytes.Buffer{}
	header := make([]byte, 6)
	// reserved(2)=0, type(2)=1 (icon), count(2), all little-endian
	header[2] = 1
	header[4] = byte(len(sizes))
	out.Write(header)
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		entry := make([]byte, 16)
		if s < 256 {
			entry[0], entry[1] = byte(s), byte(s)
		}
		entry[4], entry[5] = 1, 32
		size := len(images[i])
		entry[8] = byte(size)
		entry[9] = byte(size >> 8)
		entry[10] = byte(size >> 16)
		entry[11] = byte(size >> 24)
		entry[12] = byte(offset)
		entry[13] = byte(offset >> 8)
		entry[14] = byte(offset >> 16)
		entry[15] = byte(offset >> 24)
		out.Write(entry)
		offset += size
	}
	for _, data := range images {
		out.Write(data)
	}
	return out.Bytes()
}

func main() {
	ico := writeICO([]int{16, 24, 32, 48, 64, 128, 256})
	if err := os.WriteFile("internal/desktop/icon.ico", ico, 0o600); err != nil {
		panic(err)
	}
}
