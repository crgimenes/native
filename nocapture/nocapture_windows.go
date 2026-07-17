// Windows backend: SetWindowDisplayAffinity with WDA_MONITOR. Captures and
// screen sharing show the window as a black rectangle; the user keeps seeing
// it normally on the monitor.
//
// WDA_MONITOR (blacked out) is deliberately chosen over WDA_EXCLUDEFROMCAPTURE
// (window omitted, background shows through): the black box tells the viewer
// "there is a window here and it is protected", which is the honest and the
// expected rendering — it is how DRM video players appear in a shared screen.

package nocapture

import (
	"fmt"
	"syscall"
	"unsafe"
)

// wdaMonitor is WDA_MONITOR (winuser.h): content displayed only on a monitor.
const wdaMonitor = 0x00000001

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
)

func protect(window unsafe.Pointer) error {
	r, _, callErr := procSetWindowDisplayAffinity.Call(
		uintptr(window),
		wdaMonitor,
	)
	if r == 0 {
		return fmt.Errorf("nocapture: SetWindowDisplayAffinity: %w", callErr)
	}
	return nil
}
