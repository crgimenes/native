// Package power provides cgo-free control over system power behavior.
//
// Today it does one thing: keep the system awake. PreventSleep asks the OS not
// to idle-sleep until the returned Token is released — the keep-awake you want
// around a long copy, a build, or media playback. Each platform binds the API
// the OS already ships, with no cgo and no bundled native libraries.
//
//	tok, err := power.PreventSleep("ripping a disc")
//	if err != nil {
//		// errors.Is(err, power.ErrUnsupported) on platforms with no backend
//	}
//	defer tok.Release()
//
// Backends: macOS uses an IOKit power-management assertion, Windows uses
// SetThreadExecutionState. Linux has no portable cgo-free path (the systemd
// login1 inhibitor needs D-Bus, which this module deliberately avoids), so it
// returns ErrUnsupported.
//
// Battery / power-source state is intentionally out of scope for now; it is a
// separate, side-effect-free concern and is not needed yet.
package power

import (
	"errors"
	"sync"
)

// ErrUnsupported is returned by PreventSleep on a platform that has no backend.
var ErrUnsupported = errors.New("power: not supported on this platform")

// Token represents one active sleep inhibition. Release it to let the system
// idle-sleep normally again.
type Token struct {
	mu       sync.Mutex
	released bool
	h        handle // platform-specific, produced by preventSleep
}

// PreventSleep asks the OS to keep the system from idle-sleeping until the
// returned Token is released.
//
// reason is a short human-readable label some platforms surface in their power
// tooling (it becomes the macOS assertion name) and others ignore. On a platform
// with no backend it returns ErrUnsupported and a nil Token.
func PreventSleep(reason string) (*Token, error) {
	h, err := preventSleep(reason)
	if err != nil {
		return nil, err
	}
	return &Token{h: h}, nil
}

// Release ends the inhibition. It is idempotent: calling it more than once (or
// concurrently) is safe, and only the first call does any work.
func (t *Token) Release() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.released {
		return nil
	}
	err := release(t.h)
	if err != nil {
		return err
	}
	t.released = true
	return nil
}
