package alert

import (
	"errors"
	"testing"
)

// The alert itself is modal UI that needs a display and the main thread;
// what runs here is the argument check every platform shares.
func TestShowNeedsAButton(t *testing.T) {
	_, err := Show(Options{Title: "x"})
	if !errors.Is(err, errNoButtons) {
		t.Fatalf("err %v, want errNoButtons", err)
	}
}
