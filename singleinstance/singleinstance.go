// Package singleinstance enforces a single running instance of an application
// and hands the launch arguments of later starts to the one already running.
//
// The typical flow: on startup call Acquire. If it succeeds you are the primary
// instance — hold the returned *Instance for the app's lifetime. If it returns
// ErrAlreadyRunning, another instance owns the lock, so forward your arguments
// to it with Send and exit; the primary receives them through Options.OnMessage.
//
//	inst, err := singleinstance.Acquire("com.example.app", singleinstance.Options{
//		OnMessage: func(args []string) { /* a second launch passed these */ },
//	})
//	if errors.Is(err, singleinstance.ErrAlreadyRunning) {
//		_ = singleinstance.Send("com.example.app", os.Args[1:])
//		return
//	}
//	defer inst.Release()
//
// Backends use only what the OS ships: flock + a Unix-domain socket on
// macOS/Linux (no purego), a named pipe on Windows (via purego). No cgo.
package singleinstance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
)

// ErrUnsupported is returned on a platform with no backend wired up.
var ErrUnsupported = errors.New("singleinstance: not supported on this platform")

// ErrAlreadyRunning is returned by Acquire when another instance already holds
// the lock for the given id.
var ErrAlreadyRunning = errors.New("singleinstance: another instance is already running")

// Options configures Acquire.
type Options struct {
	// OnMessage is invoked on the primary instance with the arguments a later
	// launch passed to Send. It runs on its own goroutine, so guard shared state
	// (or hand off to your UI thread). Optional — nil discards incoming messages.
	OnMessage func(args []string)
}

// Instance is a held single-instance lock. Keep it alive for as long as the
// application runs; Release it to let another process take over.
type Instance struct {
	once    sync.Once
	relErr  error
	release func() error
}

// Release relinquishes the lock and stops listening for hand-offs. Safe to call
// more than once; later calls are no-ops that return the first result.
func (i *Instance) Release() error {
	i.once.Do(func() { i.relErr = i.release() })
	return i.relErr
}

// Acquire tries to become the single instance identified by id (a stable
// application identifier such as "com.example.app"). It returns the owning
// *Instance when no instance is running, or ErrAlreadyRunning when one already
// is — in which case the caller should Send its arguments and exit.
func Acquire(id string, opts Options) (*Instance, error) { return acquire(id, opts) }

// Send delivers args to the instance already running under id. Use it after
// Acquire returned ErrAlreadyRunning. It returns an error when no instance is
// listening.
func Send(id string, args []string) error { return send(id, args) }

// keyFor derives a short, filesystem- and pipe-name-safe key from an arbitrary
// id, so the lock/socket/pipe names stay bounded in length and collision-free.
func keyFor(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:8])
}
