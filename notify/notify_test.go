package notify_test

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/crgimenes/native/notify"
)

// A real notification lands on the screen of whoever runs the tests, so the
// backend runs only where CI sets NATIVE_NOTIFY_TEST.
func TestSend(t *testing.T) {
	if os.Getenv("NATIVE_NOTIFY_TEST") == "" {
		t.Skip("posts a visible notification; set NATIVE_NOTIFY_TEST=1")
	}
	err := notify.Send(notify.Message{Title: "native/notify", Body: "test"})
	if errors.Is(err, notify.ErrUnsupported) {
		t.Skipf("notify unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestUnsupportedReturnsSentinel(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("platform has a backend")
	}
	err := notify.Send(notify.Message{Title: "x"})
	if !errors.Is(err, notify.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestEmptyMessage(t *testing.T) {
	err := notify.Send(notify.Message{})
	if err == nil {
		t.Fatal("empty message accepted")
	}
}

func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(notify.ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("notify.ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", notify.ErrUnsupported)
	}
}
