package clipboard_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/crgimenes/native/clipboard"
)

// TestRoundTrip writes text and reads it back. It mutates the real system
// clipboard, so it saves and restores whatever text was there. On platforms
// with no backend yet (Linux, *BSD) it skips instead of failing.
func TestRoundTrip(t *testing.T) {
	_, err := clipboard.ReadText()
	if errors.Is(err, clipboard.ErrUnsupported) {
		t.Skipf("clipboard backend unavailable here: %v", err)
	}

	orig, _ := clipboard.ReadText()
	t.Cleanup(func() { _ = clipboard.WriteText(orig) })

	cases := []string{
		"",
		"hello",
		"line one\nline two — café ✓ 日本語",
	}
	for _, want := range cases {
		err := clipboard.WriteText(want)
		if err != nil {
			t.Fatalf("WriteText(%q): %v", want, err)
		}
		got, err := clipboard.ReadText()
		if err != nil {
			t.Fatalf("ReadText after %q: %v", want, err)
		}
		if got != want {
			t.Fatalf("round-trip mismatch: got %q, want %q", got, want)
		}
	}
}

// TestImageRoundTrip writes a PNG and reads it back pixel for pixel, then
// restores the text that was there (an image on the clipboard before the test
// is lost; the text case is what a developer's clipboard usually holds).
func TestImageRoundTrip(t *testing.T) {
	_, err := clipboard.ReadImage()
	if errors.Is(err, clipboard.ErrUnsupported) {
		t.Skipf("image clipboard unavailable here: %v", err)
	}
	orig, _ := clipboard.ReadText()
	t.Cleanup(func() { _ = clipboard.WriteText(orig) })

	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 0xff, A: 0xff})
	src.SetNRGBA(2, 1, color.NRGBA{G: 0x80, B: 0x40, A: 0x80})
	var buf bytes.Buffer
	err = png.Encode(&buf, src)
	if err != nil {
		t.Fatal(err)
	}
	err = clipboard.WriteImage(buf.Bytes())
	if err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	data, err := clipboard.ReadImage()
	if err != nil {
		t.Fatalf("ReadImage: %v", err)
	}
	got, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode what came back: %v", err)
	}
	for y := range 2 {
		for x := range 3 {
			g := color.NRGBAModel.Convert(got.At(x, y))
			if g != src.NRGBAAt(x, y) {
				t.Fatalf("(%d,%d) = %v, want %v", x, y, g, src.NRGBAAt(x, y))
			}
		}
	}
	_ = clipboard.WriteText("text")
	data, err = clipboard.ReadImage()
	if err != nil || data != nil {
		t.Fatalf("with only text on the clipboard: %d bytes, %v; want none", len(data), err)
	}
}
