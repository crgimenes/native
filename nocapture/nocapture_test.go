package nocapture

import (
	"errors"
	"testing"
)

// TestProtectNilWindow guards the one input mistake every caller can make.
// The real behaviour needs a live native window, which a unit test does not
// have; consumers exercise it (inro's screen-capture protection).
func TestProtectNilWindow(t *testing.T) {
	err := Protect(nil)
	if err == nil {
		t.Fatal("Protect(nil) succeeded; want an error")
	}
}

// Callers treat an unsupported platform as a normal answer, so the sentinel
// wraps the stdlib one: one errors.Is covers every optional capability here.
func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", ErrUnsupported)
	}
}
