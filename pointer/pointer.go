// Package pointer reports pointer events that GUI toolkits built on GLFW do
// not pass on, starting with the trackpad pinch, cgo-free.
//
// Threading: Watch must be called on the main thread, and fn runs there, in
// the middle of the host's event handling. Keep fn short; an Ebitengine app
// queues the event for its Update.
//
// Implemented on macOS, through a local NSEvent monitor: events are seen and
// passed on unchanged, so the host still gets them. Elsewhere Watch returns
// ErrUnsupported.
package pointer

import "errors"

var ErrUnsupported = errors.New("pointer: not supported on this platform")

type Kind int

const (
	// Magnify is a pinch on the trackpad.
	Magnify Kind = iota + 1
)

type Event struct {
	Kind Kind
	// Magnification, for Magnify, is the change in scale this event adds:
	// 0.1 is ten percent larger, negative is smaller.
	Magnification float64
}

// Watch calls fn with every event of the kinds this package reports until
// stop is called. stop is safe to call more than once.
func Watch(fn func(Event)) (stop func(), err error) {
	if fn == nil {
		return nil, errNoCallback
	}
	return watch(fn)
}

var errNoCallback = errors.New("pointer: Watch needs a callback")
