// Windows backend: SetThreadExecutionState (kernel32) via purego. Windows has no
// dlopen, so the symbol is resolved with LoadLibrary/GetProcAddress.
//
// SetThreadExecutionState is THREAD-scoped: the continuous flag is cleared when
// the thread that set it exits, and a Go goroutine may migrate OS threads. So a
// dedicated goroutine locks its OS thread, sets the flag, parks until released,
// then clears the flag on that same thread. One parked OS thread per active
// inhibition is the cost.

package power

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"

	"github.com/ebitengine/purego"
)

const (
	esSystemRequired = 0x00000001 // ES_SYSTEM_REQUIRED
	esContinuous     = 0x80000000 // ES_CONTINUOUS
)

var (
	initOnce sync.Once
	initErr  error

	setThreadExecutionState func(esFlags uint32) uint32
)

func ensureInit() error {
	initOnce.Do(func() {
		k32, err := syscall.LoadLibrary("kernel32.dll")
		if err != nil {
			initErr = fmt.Errorf("power: load kernel32.dll: %w", err)
			return
		}
		addr, err := syscall.GetProcAddress(k32, "SetThreadExecutionState")
		if err != nil {
			initErr = fmt.Errorf("power: resolve SetThreadExecutionState: %w", err)
			return
		}
		purego.RegisterFunc(&setThreadExecutionState, addr)
	})
	return initErr
}

// inhibitor owns the thread-pinned goroutine holding the execution-state flag.
type inhibitor struct {
	stop chan struct{} // closed by release to wake the goroutine
	done chan struct{} // closed by the goroutine once the flag is cleared
}

type handle = *inhibitor

func preventSleep(_ string) (handle, error) {
	err := ensureInit()
	if err != nil {
		return nil, err
	}

	in := &inhibitor{stop: make(chan struct{}), done: make(chan struct{})}
	started := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// A zero return means the call failed; otherwise it is the previous state.
		prev := setThreadExecutionState(esContinuous | esSystemRequired)
		if prev == 0 {
			started <- errors.New("power: SetThreadExecutionState failed")
			return
		}
		started <- nil

		<-in.stop
		setThreadExecutionState(esContinuous) // drop the requirement on this thread
		close(in.done)
	}()

	err = <-started
	if err != nil {
		return nil, err
	}
	return in, nil
}

func release(h handle) error {
	close(h.stop)
	<-h.done // wait until the flag is actually cleared
	return nil
}
