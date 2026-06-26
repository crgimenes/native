package power_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/crgimenes/native/power"
)

// TestPreventSleepRoundTrip exercises the real backend where there is one: it
// creates an inhibition and releases it, then confirms Release is idempotent.
// CI runs this for real on macOS (IOKit assertion) and Windows
// (SetThreadExecutionState); platforms without a backend skip. This proves the
// binding works end to end. Whether the machine actually stays awake is not
// observable from a test and is verified by hand (macOS: `pmset -g assertions`,
// Windows: `powercfg /requests`).
func TestPreventSleepRoundTrip(t *testing.T) {
	tok, err := power.PreventSleep("native power test")
	if errors.Is(err, power.ErrUnsupported) {
		t.Skipf("power unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("PreventSleep: %v", err)
	}

	err = tok.Release()
	if err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Releasing again must be a harmless no-op.
	err = tok.Release()
	if err != nil {
		t.Fatalf("second Release: %v", err)
	}
}

// TestUnsupportedReturnsSentinel gives the backend-less platforms (Linux on the
// CI matrix) something real to assert: PreventSleep reports ErrUnsupported rather
// than a vague failure or a hang.
func TestUnsupportedReturnsSentinel(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("platform has a backend")
	}
	_, err := power.PreventSleep("x")
	if !errors.Is(err, power.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}
