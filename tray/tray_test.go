package tray_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/crgimenes/native/tray"
)

// TestRunUnsupported checks that on a platform with no backend Run fails fast
// with ErrUnsupported instead of blocking. The macOS and Windows backends own
// the main thread and its run loop and need a display, so they are not started
// from a unit test; examples/tray is their manual (and CI smoke-test) vehicle.
func TestRunUnsupported(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skipf("interactive tray backend on %s is exercised via examples/tray", runtime.GOOS)
	}
	err := tray.Run(tray.Config{Title: "test"})
	if !errors.Is(err, tray.ErrUnsupported) {
		t.Fatalf("Run on %s: got %v, want ErrUnsupported", runtime.GOOS, err)
	}
}

// TestStopWhenIdle is a no-op safety check: Stop must not panic or block when no
// tray is running, on every platform.
func TestStopWhenIdle(t *testing.T) {
	tray.Stop()
}
