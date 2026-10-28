package main

import (
	"image/color"
	"testing"
)

func TestRenderProbe(t *testing.T) {
	img := render(256)
	at := func(x, y int) color.RGBA { return img.RGBAAt(x, y) }
	t.Logf("top node (84,70): %+v", at(84, 70))
	t.Logf("bg top (128,30): %+v", at(128, 30))
	t.Logf("bg bottom (128,230): %+v", at(128, 230))
	t.Logf("token (174,165): %+v", at(174, 165))
	t.Logf("corner (2,2): %+v", at(2, 2))
	t.Logf("mint expected: {128 224 192 255}")
}
