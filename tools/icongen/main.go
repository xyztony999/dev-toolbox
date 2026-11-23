// Command icongen renders the DevToolbox application icon (navy rounded
// square, mint "identity key assembled from git commit nodes" glyph: a
// commit-ring bow with a light core, a shaft, and two teeth ending in
// commit nodes) and writes the multi-size ICO to its fixed asset path.
// The geometry mirrors the SVG design 1:1 in 512-unit space. Run from the
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
	strokeHalf = 14.4
)

var (
	gradTop    = color.RGBA{26, 58, 92, 255}
	gradBottom = color.RGBA{11, 28, 46, 255}
	mint       = color.RGBA{128, 224, 192, 255}
	token      = color.RGBA{234, 252, 244, 255}
)

var (
	// bow: commit-ring with a light core (dual-identity accent)
	bow                        = pt{176, 256}
	ringInner, ringOuter, dotR = 46.0, 78.0, 24.0
	// shaft and teeth ending in commit nodes
	shaftA, shaftB   = pt{254, 256}, pt{384, 256}
	shaftHalf        = 16.0
	tooth1A, tooth1B = pt{320, 256}, pt{320, 312}
	tooth2A, tooth2B = pt{376, 256}, pt{376, 336}
	toothHalf        = 14.0
	node1, node2     = pt{320, 318}, pt{376, 342}
	nodeR            = 22.0
)

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func lerpColor(t float64) color.RGBA {
	return color.RGBA{
		uint8(lerp(float64(gradTop.R), float64(gradBottom.R), t)),
		uint8(lerp(float64(gradTop.G), float64(gradBottom.G), t)),
		uint8(lerp(float64(gradTop.B), float64(gradBottom.B), t)),
		255,
	}
}

func distToSegment(p, a, b pt) float64 {
	abx, aby := b.x-a.x, b.y-a.y
	apx, apy := p.x-a.x, p.y-a.y
	length := abx*abx + aby*aby
	t := 0.0
	if length > 0 {
		t = (apx*abx + apy*aby) / length
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
	}
	cx, cy := a.x+abx*t, a.y+aby*t
	return math.Hypot(p.x-cx, p.y-cy)
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

func dist(p, c pt) float64 { return math.Hypot(p.x-c.x, p.y-c.y) }

func paintGlyph(p pt) bool {
	if d := dist(p, bow); d >= ringInner && d <= ringOuter {
		return true
	}
	if distToSegment(p, shaftA, shaftB) <= shaftHalf {
		return true
	}
	if distToSegment(p, tooth1A, tooth1B) <= toothHalf || dist(p, node1) <= nodeR {
		return true
	}
	if distToSegment(p, tooth2A, tooth2B) <= toothHalf || dist(p, node2) <= nodeR {
		return true
	}
	return false
}

func sample(p pt) color.RGBA {
	if !inSquircle(p) {
		return color.RGBA{0, 0, 0, 0}
	}
	t := (p.y - inset) / squareSize
	base := lerpColor(math.Min(math.Max(t, 0), 1))
	if dist(p, bow) <= dotR {
		return token
	}
	if paintGlyph(p) {
		return mint
	}
	return base
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
