package singleinstance_test

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/crgimenes/native/singleinstance"
)

// uniqueID keeps parallel CI jobs and reruns from colliding on the same lock.
func uniqueID(name string) string {
	return fmt.Sprintf("native-singleinstance-test-%s-%d", name, os.Getpid())
}

// TestAcquireSendRoundTrip runs the whole flow in one process: the first Acquire
// wins, a second Acquire is rejected with ErrAlreadyRunning, and Send forwards
// arguments that arrive at the primary's OnMessage. flock denies a second lock
// even from the same process (it is per open file description), and a Windows
// named pipe with FILE_FLAG_FIRST_PIPE_INSTANCE likewise rejects the second
// create — so the round trip is fully exercised on CI without spawning a child.
func TestAcquireSendRoundTrip(t *testing.T) {
	id := uniqueID("roundtrip")
	got := make(chan []string, 1)

	inst, err := singleinstance.Acquire(id, singleinstance.Options{
		OnMessage: func(args []string) {
			select {
			case got <- args:
			default:
			}
		},
	})
	if errors.Is(err, singleinstance.ErrUnsupported) {
		t.Skipf("singleinstance unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("Acquire (primary): %v", err)
	}
	defer func() { _ = inst.Release() }()

	second, err := singleinstance.Acquire(id, singleinstance.Options{})
	if !errors.Is(err, singleinstance.ErrAlreadyRunning) {
		if second != nil {
			_ = second.Release()
		}
		t.Fatalf("second Acquire = %v, want ErrAlreadyRunning", err)
	}

	want := []string{"open", "/tmp/a b.txt", "café ✓"}
	if err := singleinstance.Send(id, want); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case args := <-got:
		if !slices.Equal(args, want) {
			t.Fatalf("forwarded args = %v, want %v", args, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for forwarded args")
	}
}

// TestReleaseAllowsReacquire confirms Release frees the lock so a later Acquire
// succeeds again.
func TestReleaseAllowsReacquire(t *testing.T) {
	id := uniqueID("reacquire")
	inst, err := singleinstance.Acquire(id, singleinstance.Options{})
	if errors.Is(err, singleinstance.ErrUnsupported) {
		t.Skipf("singleinstance unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := inst.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	again, err := singleinstance.Acquire(id, singleinstance.Options{})
	if err != nil {
		t.Fatalf("re-Acquire after Release: %v", err)
	}
	_ = again.Release()
}

// TestSendWithoutInstance reports an error rather than blocking when nothing is
// listening.
func TestSendWithoutInstance(t *testing.T) {
	id := uniqueID("noinstance")
	err := singleinstance.Send(id, []string{"x"})
	if errors.Is(err, singleinstance.ErrUnsupported) {
		t.Skipf("singleinstance unsupported here: %v", err)
	}
	if err == nil {
		t.Fatal("Send with no running instance should fail")
	}
}
