package main

import (
	"testing"
)

func renderTest(w, h int) ([]byte, roundRectStyle) {
	st := overlayStyle(1, [3]float32{0.2, 0.4, 0.9}, 0.3)
	buf := make([]byte, w*h*4)
	renderRoundRect(buf, w*4, w, h, st)
	return buf, st
}

func px(buf []byte, w, x, y int) [4]byte {
	i := (y*w + x) * 4
	return [4]byte{buf[i], buf[i+1], buf[i+2], buf[i+3]}
}

func TestRenderRoundRect_Regions(t *testing.T) {
	w, h := 400, 300
	buf, st := renderTest(w, h)

	// Outside the inset → fully transparent.
	if p := px(buf, w, 0, 0); p[3] != 0 {
		t.Fatalf("corner pixel should be transparent, got %v", p)
	}
	// Rounded corner: just inside the inset at the very corner is still outside the arc.
	in := int(st.inset)
	if p := px(buf, w, in, in); p[3] > 40 {
		t.Fatalf("pixel at the rounded corner should be (almost) transparent, got %v", p)
	}
	// Middle of the top border → border alpha (~0.92).
	if p := px(buf, w, w/2, in); p[3] < 200 {
		t.Fatalf("top border should be nearly opaque, got %v", p)
	}
	// Centre → fill alpha (0.3 ≈ 77).
	c := px(buf, w, w/2, h/2)
	if c[3] < 70 || c[3] > 84 {
		t.Fatalf("centre should be the translucent fill, got %v", c)
	}
	// Premultiplied: no colour channel may exceed alpha.
	for i := 0; i < len(buf); i += 4 {
		if buf[i] > buf[i+3] || buf[i+1] > buf[i+3] || buf[i+2] > buf[i+3] {
			t.Fatalf("pixel %d not premultiplied: %v", i/4, buf[i:i+4])
		}
	}
	// Symmetry left/right (bands are rendered separately from the middle).
	for y := 0; y < h; y += 7 {
		for x := 0; x < 30; x++ {
			if px(buf, w, x, y) != px(buf, w, w-1-x, y) {
				t.Fatalf("asymmetric at (%d,%d)", x, y)
			}
		}
	}
}

func TestRenderRoundRect_TinySizesDontPanic(t *testing.T) {
	for _, s := range [][2]int{{1, 1}, {2, 50}, {50, 2}, {7, 7}, {20, 20}} {
		renderTest(s[0], s[1])
	}
}

func BenchmarkRenderRoundRect_1440p(b *testing.B) {
	w, h := 2560, 1400
	st := overlayStyle(1.5, [3]float32{0.2, 0.4, 0.9}, 0.3)
	buf := make([]byte, w*h*4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderRoundRect(buf, w*4, w, h, st)
	}
}

func TestEaseOutCubic(t *testing.T) {
	if easeOutCubic(0) != 0 || easeOutCubic(1) != 1 {
		t.Fatal("ease must map 0→0 and 1→1")
	}
	if easeOutCubic(0.5) <= 0.5 {
		t.Fatal("ease-out should be ahead of linear at the midpoint")
	}
}
