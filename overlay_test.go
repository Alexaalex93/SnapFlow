package main

import (
	"testing"
)

func renderTest(w, h int, square bool) []byte {
	st := overlayStyle(1, [3]float32{0.2, 0.4, 0.9}, 0.3, square)
	buf := make([]byte, w*h*4)
	renderRoundRect(buf, w*4, w, h, st)
	return buf
}

func px(buf []byte, w, x, y int) [4]byte {
	i := (y*w + x) * 4
	return [4]byte{buf[i], buf[i+1], buf[i+2], buf[i+3]}
}

// The preview covers the whole target rect (no inset): edges are border,
// the centre is the translucent fill.
func TestRenderRoundRect_FillsWholeRect(t *testing.T) {
	w, h := 400, 300
	for _, square := range []bool{true, false} {
		buf := renderTest(w, h, square)
		for _, p := range [][2]int{{w / 2, 0}, {w / 2, h - 1}, {0, h / 2}, {w - 1, h / 2}} {
			if a := px(buf, w, p[0], p[1])[3]; a < 200 {
				t.Fatalf("square=%v: edge pixel %v should be opaque border, alpha=%d", square, p, a)
			}
		}
		if c := px(buf, w, w/2, h/2); c[3] < 70 || c[3] > 84 {
			t.Fatalf("square=%v: centre should be the translucent fill, got %v", square, c)
		}
		for i := 0; i < len(buf); i += 4 {
			if buf[i] > buf[i+3] || buf[i+1] > buf[i+3] || buf[i+2] > buf[i+3] {
				t.Fatalf("pixel %d not premultiplied: %v", i/4, buf[i:i+4])
			}
		}
		for y := 0; y < h; y += 7 {
			for x := 0; x < 30; x++ {
				if px(buf, w, x, y) != px(buf, w, w-1-x, y) {
					t.Fatalf("square=%v: asymmetric at (%d,%d)", square, x, y)
				}
			}
		}
	}
}

func TestRenderRoundRect_Corners(t *testing.T) {
	w, h := 400, 300
	sq := renderTest(w, h, true)
	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}} {
		if a := px(sq, w, p[0], p[1])[3]; a < 200 {
			t.Fatalf("square preview: corner %v should be opaque, alpha=%d", p, a)
		}
	}
	rd := renderTest(w, h, false)
	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}} {
		if a := px(rd, w, p[0], p[1])[3]; a > 40 {
			t.Fatalf("rounded preview: corner %v should be (almost) transparent, alpha=%d", p, a)
		}
	}
}

func TestRenderRoundRect_TinySizesDontPanic(t *testing.T) {
	for _, s := range [][2]int{{1, 1}, {2, 50}, {50, 2}, {7, 7}, {20, 20}} {
		renderTest(s[0], s[1], true)
		renderTest(s[0], s[1], false)
	}
}

func BenchmarkRenderRoundRect_1440p(b *testing.B) {
	w, h := 2560, 1400
	st := overlayStyle(1.5, [3]float32{0.2, 0.4, 0.9}, 0.3, false)
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
