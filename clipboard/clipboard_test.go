package clipboard_test

import (
	"errors"
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
