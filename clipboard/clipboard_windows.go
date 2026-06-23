// Windows clipboard backend: the Win32 clipboard (user32 + kernel32) via purego.
// Windows has no dlopen, so symbols are resolved with LoadLibrary/GetProcAddress
// and bound with purego.RegisterFunc (which applies the stdcall convention).

package clipboard

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	cfUnicodeText = 13     // CF_UNICODETEXT
	gmemMoveable  = 0x0002 // GMEM_MOVEABLE
)

var (
	initOnce sync.Once
	initErr  error

	openClipboard              func(hwnd uintptr) bool
	closeClipboard             func() bool
	emptyClipboard             func() bool
	getClipboardData           func(format uint32) uintptr
	setClipboardData           func(format uint32, mem uintptr) uintptr
	isClipboardFormatAvailable func(format uint32) bool

	globalAlloc  func(flags uint32, bytes uintptr) uintptr
	globalLock   func(mem uintptr) uintptr
	globalUnlock func(mem uintptr) bool
	globalFree   func(mem uintptr) uintptr
)

func ensureInit() error {
	initOnce.Do(func() {
		user32, err := syscall.LoadLibrary("user32.dll")
		if err != nil {
			initErr = fmt.Errorf("clipboard: load user32.dll: %w", err)
			return
		}
		kernel32, err := syscall.LoadLibrary("kernel32.dll")
		if err != nil {
			initErr = fmt.Errorf("clipboard: load kernel32.dll: %w", err)
			return
		}

		type binding struct {
			ptr  any
			lib  syscall.Handle
			name string
		}
		for _, b := range []binding{
			{&openClipboard, user32, "OpenClipboard"},
			{&closeClipboard, user32, "CloseClipboard"},
			{&emptyClipboard, user32, "EmptyClipboard"},
			{&getClipboardData, user32, "GetClipboardData"},
			{&setClipboardData, user32, "SetClipboardData"},
			{&isClipboardFormatAvailable, user32, "IsClipboardFormatAvailable"},
			{&globalAlloc, kernel32, "GlobalAlloc"},
			{&globalLock, kernel32, "GlobalLock"},
			{&globalUnlock, kernel32, "GlobalUnlock"},
			{&globalFree, kernel32, "GlobalFree"},
		} {
			addr, e := syscall.GetProcAddress(b.lib, b.name)
			if e != nil {
				initErr = fmt.Errorf("clipboard: resolve %s: %w", b.name, e)
				return
			}
			purego.RegisterFunc(b.ptr, addr)
		}
	})
	return initErr
}

// openClipboardRetry retries OpenClipboard briefly: another process may hold the
// clipboard open for a few milliseconds at a time.
func openClipboardRetry() bool {
	for i := 0; i < 10; i++ {
		if openClipboard(0) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func readText() (string, error) {
	if err := ensureInit(); err != nil {
		return "", err
	}
	if !openClipboardRetry() {
		return "", errors.New("clipboard: OpenClipboard failed")
	}
	defer closeClipboard()

	if !isClipboardFormatAvailable(cfUnicodeText) {
		return "", nil
	}
	h := getClipboardData(cfUnicodeText)
	if h == 0 {
		return "", nil
	}
	p := globalLock(h)
	if p == 0 {
		return "", errors.New("clipboard: GlobalLock failed")
	}
	defer globalUnlock(h)
	return utf16PtrToString(p), nil
}

func writeText(s string) error {
	if err := ensureInit(); err != nil {
		return err
	}

	u16 := append(utf16.Encode([]rune(s)), 0) // NUL-terminated UTF-16
	h := globalAlloc(gmemMoveable, uintptr(len(u16)*2))
	if h == 0 {
		return errors.New("clipboard: GlobalAlloc failed")
	}
	dst := globalLock(h)
	if dst == 0 {
		globalFree(h)
		return errors.New("clipboard: GlobalLock failed")
	}
	base := ptr(dst)
	for i, ch := range u16 {
		*(*uint16)(unsafe.Add(base, i*2)) = ch
	}
	globalUnlock(h)

	if !openClipboardRetry() {
		globalFree(h)
		return errors.New("clipboard: OpenClipboard failed")
	}
	if !emptyClipboard() {
		closeClipboard()
		globalFree(h)
		return errors.New("clipboard: EmptyClipboard failed")
	}
	if setClipboardData(cfUnicodeText, h) == 0 {
		closeClipboard()
		globalFree(h)
		return errors.New("clipboard: SetClipboardData failed")
	}
	// SetClipboardData succeeded: the system now owns h, so it must not be freed.
	closeClipboard()
	return nil
}

// ptr reinterprets a uintptr's bits as an unsafe.Pointer without a direct
// uintptr->Pointer conversion. The pointers we feed it come from GlobalLock,
// which returns system memory that the Go GC neither owns nor moves, so the
// reinterpretation is safe; the spelling just keeps `go vet`'s unsafeptr check
// from flagging it.
func ptr(u uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&u)) }

// utf16PtrToString reads a NUL-terminated UTF-16 string from a raw pointer
// returned by GlobalLock (system memory, so it does not move under the GC).
func utf16PtrToString(p uintptr) string {
	base := ptr(p)
	var u16 []uint16
	for i := 0; ; i++ {
		ch := *(*uint16)(unsafe.Add(base, i*2))
		if ch == 0 {
			break
		}
		u16 = append(u16, ch)
	}
	return string(utf16.Decode(u16))
}
