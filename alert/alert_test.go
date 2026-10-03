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

// Callers treat an unsupported platform as a normal answer, so the sentinel
// wraps the stdlib one: one errors.Is covers every optional capability here.
func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", ErrUnsupported)
	}
}
