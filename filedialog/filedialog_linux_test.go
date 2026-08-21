package filedialog

import (
	"runtime"
	"testing"
)

// TestGTKSmoke exercises the real ABI risk headlessly: the GTK stack loads, the
// symbols resolve, gtk_init succeeds, and a GtkFileChooserNative can be created
// and released — everything short of showing the modal (which needs a user).
// It skips when the environment has no GTK or no display, so `go test ./...`
// stays green on a headless box; under xvfb it runs for real.
func TestGTKSmoke(t *testing.T) {
	// GTK calls must stay on one thread from gtk_init onward.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !gtkReady() {
		t.Skipf("GTK unavailable or no display (init error: %v)", initErr)
	}
	t.Logf("gtk4=%v", gtk4)

	dlg := gtkFileChooserNativeNew("smoke", 0, gtkFileChooserActionOpen, "_Open", "_Cancel")
	if dlg == 0 {
		t.Fatal("gtk_file_chooser_native_new returned nil")
	}
	applyChooserFilter(dlg, []string{"png", ".jpg"})
	setChooserFolder(dlg, t.TempDir())
	gObjectUnref(dlg)
}
