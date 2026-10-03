package pointer

import (
	"errors"
	"runtime"
	"testing"
)

// A pinch cannot be synthesized, so the event path is checked by hand; this
// checks that installing and removing the monitor works and is idempotent.
func TestWatchAndStop(t *testing.T) {
	_, err := Watch(nil)
	if !errors.Is(err, errNoCallback) {
		t.Fatalf("err %v, want errNoCallback", err)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	stop, err := Watch(func(Event) {})
	if errors.Is(err, ErrUnsupported) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	stop()
	stop()
}

// Callers treat an unsupported platform as a normal answer, so the sentinel
// wraps the stdlib one: one errors.Is covers every optional capability here.
func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", ErrUnsupported)
	}
}
