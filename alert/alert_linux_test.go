package alert

import (
	"runtime"
	"testing"

	"github.com/ebitengine/purego"
)

// TestGTKDialog shows real dialogs and answers them through testHook: the
// entry's text comes back with the chosen button, and Escape (a negative
// response) maps to the Cancel button, or to the first when none is Cancel.
// It skips without GTK or a display; under xvfb it runs for real.
func TestGTKDialog(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := gtkReady()
	if err != nil {
		t.Skipf("GTK unavailable: %v", err)
	}
	name := "libgtk-3.so.0"
	if gtk4 {
		name = "libgtk-4.so.1"
	}
	lib, err := purego.Dlopen(name, purego.RTLD_LAZY|rtldNoload)
	if err != nil {
		t.Fatal(err)
	}
	var respond func(dialog uintptr, response int32)
	purego.RegisterLibFunc(&respond, lib, "gtk_dialog_response")
	t.Cleanup(func() { testHook = nil })

	openCancel := []Button{{Title: "Open"}, {Title: "Cancel", Cancel: true}}
	cases := []struct {
		name     string
		buttons  []Button
		text     string
		response int32
		want     Result
	}{
		{"typed and accepted", openCancel, "go.dev", 0, Result{Button: 0, Text: "go.dev"}},
		{"escape picks cancel", openCancel, "ignored", -4, Result{Button: 1, Text: "ignored"}},
		{"escape without cancel", []Button{{Title: "OK"}}, "", -4, Result{Button: 0}},
	}
	for _, c := range cases {
		testHook = func(dialog, entry uintptr) {
			if entry != 0 {
				setEntryText(entry, c.text)
			}
			respond(dialog, c.response)
		}
		got, err := Show(Options{Title: "Scorcio", Message: c.name, Input: c.text != "", Buttons: c.buttons})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
