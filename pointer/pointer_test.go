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
