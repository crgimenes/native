// Package alert shows the operating system's modal alert: a message, a row of
// buttons and, when asked, one line of text to type, cgo-free.
//
// Threading: the alert is platform UI and must be shown from the main thread,
// as with filedialog; an Ebitengine app wraps the call in
// ebiten.RunOnMainThread.
//
// Implemented on macOS (NSAlert). Elsewhere Show returns ErrUnsupported, and
// the caller keeps a dialog of its own.
package alert

import "errors"

var ErrUnsupported = errors.New("alert: not supported on this platform")

var errNoButtons = errors.New("alert: at least one button is required")

// Button is one choice. Cancel makes Escape choose it; Destructive draws it
// the platform's way for an action that loses something.
type Button struct {
	Title       string
	Cancel      bool
	Destructive bool
}

// Options describes one alert. Buttons go most important first: the first
// is the default, answered by Return, and the system lays them out (on macOS
// the first is rightmost).
type Options struct {
	Title   string
	Message string
	Buttons []Button
	Input   bool   // show a one-line text field
	Text    string // what the field starts with
}

// Result is the index of the chosen button in Options.Buttons and, with
// Input, what the field held when it was chosen.
type Result struct {
	Button int
	Text   string
}

func Show(opts Options) (Result, error) {
	if len(opts.Buttons) == 0 {
		return Result{}, errNoButtons
	}
	return show(opts)
}
