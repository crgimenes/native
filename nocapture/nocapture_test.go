package nocapture

import "testing"

// TestProtectNilWindow guards the one input mistake every caller can make.
// The real behaviour needs a live native window, which a unit test does not
// have; consumers exercise it (inro's screen-capture protection).
func TestProtectNilWindow(t *testing.T) {
	err := Protect(nil)
	if err == nil {
		t.Fatal("Protect(nil) succeeded; want an error")
	}
}
