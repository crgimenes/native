//go:build !windows

package nocapture

import (
	"errors"
	"testing"
	"unsafe"
)

// TestProtectUnsupported pins the contract on platforms without a working
// backend (macOS included, since Apple removed the capability): a clear
// sentinel, not a silent no-op that callers would mistake for protection.
func TestProtectUnsupported(t *testing.T) {
	var dummy int
	err := Protect(unsafe.Pointer(&dummy))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}
