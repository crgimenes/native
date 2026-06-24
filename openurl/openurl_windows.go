// Windows backend: ShellExecuteW (shell32) via purego. Windows has no dlopen, so
// the symbol is resolved with LoadLibrary/GetProcAddress and bound with
// purego.RegisterFunc (which applies the stdcall convention). Reveal launches
// Explorer with "/select," so the file is highlighted in its folder.

package openurl

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/ebitengine/purego"
)

const swShowNormal = 1 // SW_SHOWNORMAL

var (
	initOnce sync.Once
	initErr  error

	shellExecuteW func(hwnd uintptr, op, file, params, dir *uint16, showCmd int32) uintptr
)

func ensureInit() error {
	initOnce.Do(func() {
		shell32, err := syscall.LoadLibrary("shell32.dll")
		if err != nil {
			initErr = fmt.Errorf("openurl: load shell32.dll: %w", err)
			return
		}
		addr, err := syscall.GetProcAddress(shell32, "ShellExecuteW")
		if err != nil {
			initErr = fmt.Errorf("openurl: resolve ShellExecuteW: %w", err)
			return
		}
		purego.RegisterFunc(&shellExecuteW, addr)
	})
	return initErr
}

func openURL(rawurl string) error {
	if err := ensureInit(); err != nil {
		return err
	}
	op, _ := syscall.UTF16PtrFromString("open") // constant, never contains NUL
	file, err := syscall.UTF16PtrFromString(rawurl)
	if err != nil {
		return fmt.Errorf("openurl: %q: %w", rawurl, err)
	}
	// ShellExecuteW returns a value > 32 on success, an error code otherwise.
	if r := shellExecuteW(0, op, file, nil, nil, swShowNormal); r <= 32 {
		return fmt.Errorf("openurl: ShellExecuteW(%q) failed (code %d)", rawurl, r)
	}
	return nil
}

func revealFile(absPath string) error {
	if err := ensureInit(); err != nil {
		return err
	}
	file, _ := syscall.UTF16PtrFromString("explorer.exe") // constant
	// "explorer /select,<path>" opens the containing folder and highlights the
	// file. The path is quoted so spaces don't split it into two arguments.
	params, err := syscall.UTF16PtrFromString(`/select,"` + absPath + `"`)
	if err != nil {
		return fmt.Errorf("openurl: %q: %w", absPath, err)
	}
	if r := shellExecuteW(0, nil, file, params, nil, swShowNormal); r <= 32 {
		return fmt.Errorf("openurl: reveal %q failed (code %d)", absPath, r)
	}
	return nil
}
