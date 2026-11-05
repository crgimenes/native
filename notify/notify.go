// Package notify posts a desktop notification, cgo-free.
//
//	err := notify.Send(notify.Message{Title: "Build done", Body: "all green"})
//
// Send returns once the OS has accepted the notification; it does not wait for
// the user to see or dismiss it. Safe to call from any goroutine.
//
// Only macOS has a backend; elsewhere Send returns ErrUnsupported (check it
// with errors.Is).
package notify

import (
	"errors"
	"fmt"
)

var ErrUnsupported = fmt.Errorf("notify: not supported on this platform: %w", errors.ErrUnsupported)

var errEmpty = errors.New("notify: message has neither title nor body")

type Message struct {
	Title string
	Body  string
}

func Send(m Message) error {
	if m.Title == "" && m.Body == "" {
		return errEmpty
	}
	return send(m)
}
