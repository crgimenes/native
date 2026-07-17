// Windows backend: a notification-area icon via Shell_NotifyIconW, with a hidden
// helper window whose procedure receives the icon's mouse messages and shows the
// popup menu (TrackPopupMenu). Windows has no dlopen, so symbols are resolved
// with LoadLibrary/GetProcAddress and bound with purego.RegisterFunc; the window
// procedure is a purego.NewCallback.
//
// Run owns the thread's message loop and blocks; Stop posts WM_CLOSE to the
// helper window from any thread, which tears the icon down and ends the loop.
//
// The icon is currently the application's default icon (LoadIcon IDI_APPLICATION);
// honoring Config.Icon (a PNG) would need a GDI+ PNG->HICON conversion and is a
// follow-up. Config.Tooltip (falling back to Config.Title) becomes the hover tip.

package tray

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	wmDestroy   = 0x0002
	wmClose     = 0x0010
	wmNull      = 0x0000
	wmApp       = 0x8000
	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205

	trayCallbackMsg = wmApp + 1

	nimAdd    = 0x00000000
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	idiApplication = 32512
	idcArrow       = 32512

	mfString    = 0x0000
	mfGrayed    = 0x0001
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100
)

type point struct{ X, Y int32 }

// wndClassExW mirrors WNDCLASSEXW (80 bytes on x64). Fields the package does not
// set are blanked to keep the layout without tripping the unused-field linter.
type wndClassExW struct {
	cbSize        uint32
	_             uint32 // style
	lpfnWndProc   uintptr
	_             int32 // cbClsExtra
	_             int32 // cbWndExtra
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	_             uintptr // hbrBackground
	_             *uint16 // lpszMenuName
	lpszClassName *uint16
	_             uintptr // hIconSm
}

// msgStruct is an opaque MSG buffer (48 bytes on x64): its address is handed to
// Get/Translate/DispatchMessage and its fields are never read here.
type msgStruct struct{ _ [6]uintptr }

// notifyIconData mirrors NOTIFYICONDATAW; its size (976 bytes on x64) is written
// into cbSize so the shell accepts it. Only the icon/message/tip fields are used;
// the tail is blanked to preserve the size without unused-field warnings.
type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	_                uint32      // dwState
	_                uint32      // dwStateMask
	_                [256]uint16 // szInfo
	_                uint32      // uVersion / uTimeout
	_                [64]uint16  // szInfoTitle
	_                uint32      // dwInfoFlags
	_                [16]byte    // guidItem
	_                uintptr     // hBalloonIcon
}

// Compile-time guard that the mirror matches sizeof(NOTIFYICONDATAW) on x64/arm64
// (both LLP64) — a miscounted field would otherwise only surface as a rejected
// cbSize on Windows hardware. Either array goes negative if the size is wrong.
var (
	_ [unsafe.Sizeof(notifyIconData{}) - 976]byte
	_ [976 - unsafe.Sizeof(notifyIconData{})]byte
)

var (
	initOnce sync.Once
	initErr  error

	mu       sync.Mutex
	running  bool
	trayHwnd uintptr
	hMenu    uintptr
	nid      notifyIconData

	cbMu      sync.Mutex
	callbacks = map[int]func(){}

	trayWndProcCB uintptr
	classNamePtr  *uint16

	registerClassExW    func(*wndClassExW) uint16
	createWindowExW     func(exStyle uint32, className, windowName *uint16, style uint32, x, y, width, height int32, parent, menu, instance, param uintptr) uintptr
	defWindowProcW      func(hwnd, msg, wParam, lParam uintptr) uintptr
	getMessageW         func(msg *msgStruct, hwnd uintptr, filterMin, filterMax uint32) int32
	translateMessage    func(*msgStruct) int32
	dispatchMessageW    func(*msgStruct) uintptr
	postQuitMessage     func(exitCode int32)
	destroyWindow       func(hwnd uintptr) int32
	loadIconW           func(instance, name uintptr) uintptr
	loadCursorW         func(instance, name uintptr) uintptr
	getCursorPos        func(*point) int32
	setForegroundWindow func(hwnd uintptr) int32
	trackPopupMenu      func(menu uintptr, flags uint32, x, y, reserved int32, hwnd, rect uintptr) int32
	createPopupMenu     func() uintptr
	appendMenuW         func(menu uintptr, flags uint32, id uintptr, item *uint16) int32
	destroyMenu         func(menu uintptr) int32
	postMessageW        func(hwnd, msg, wParam, lParam uintptr) int32
	getModuleHandleW    func(name *uint16) uintptr
	shellNotifyIconW    func(msg uint32, data *notifyIconData) int32
)

func ensureInit() error {
	initOnce.Do(func() {
		user32, err := syscall.LoadLibrary("user32.dll")
		if err != nil {
			initErr = fmt.Errorf("tray: load user32.dll: %w", err)
			return
		}
		shell32, err := syscall.LoadLibrary("shell32.dll")
		if err != nil {
			initErr = fmt.Errorf("tray: load shell32.dll: %w", err)
			return
		}
		kernel32, err := syscall.LoadLibrary("kernel32.dll")
		if err != nil {
			initErr = fmt.Errorf("tray: load kernel32.dll: %w", err)
			return
		}

		reg := func(p any, lib syscall.Handle, name string) {
			if initErr != nil {
				return
			}
			addr, e := syscall.GetProcAddress(lib, name)
			if e != nil {
				initErr = fmt.Errorf("tray: resolve %s: %w", name, e)
				return
			}
			purego.RegisterFunc(p, addr)
		}
		reg(&registerClassExW, user32, "RegisterClassExW")
		reg(&createWindowExW, user32, "CreateWindowExW")
		reg(&defWindowProcW, user32, "DefWindowProcW")
		reg(&getMessageW, user32, "GetMessageW")
		reg(&translateMessage, user32, "TranslateMessage")
		reg(&dispatchMessageW, user32, "DispatchMessageW")
		reg(&postQuitMessage, user32, "PostQuitMessage")
		reg(&destroyWindow, user32, "DestroyWindow")
		reg(&loadIconW, user32, "LoadIconW")
		reg(&loadCursorW, user32, "LoadCursorW")
		reg(&getCursorPos, user32, "GetCursorPos")
		reg(&setForegroundWindow, user32, "SetForegroundWindow")
		reg(&trackPopupMenu, user32, "TrackPopupMenu")
		reg(&createPopupMenu, user32, "CreatePopupMenu")
		reg(&appendMenuW, user32, "AppendMenuW")
		reg(&destroyMenu, user32, "DestroyMenu")
		reg(&postMessageW, user32, "PostMessageW")
		reg(&getModuleHandleW, kernel32, "GetModuleHandleW")
		reg(&shellNotifyIconW, shell32, "Shell_NotifyIconW")
		if initErr != nil {
			return
		}

		trayWndProcCB = purego.NewCallback(trayWndProc)
		classNamePtr = utf16Ptr("NativeTrayWindow")
		hInst := getModuleHandleW(nil)
		wc := wndClassExW{
			lpfnWndProc:   trayWndProcCB,
			hInstance:     hInst,
			hIcon:         loadIconW(0, idiApplication),
			hCursor:       loadCursorW(0, idcArrow),
			lpszClassName: classNamePtr,
		}
		wc.cbSize = uint32(unsafe.Sizeof(wc)) // #nosec G115 -- fixed small struct size
		if registerClassExW(&wc) == 0 {
			initErr = errors.New("tray: RegisterClassExW failed")
		}
	})
	return initErr
}

// trayWndProc handles the icon's mouse callback (show the menu) and window
// teardown; everything else goes to DefWindowProc.
func trayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case trayCallbackMsg:
		ev := lParam & 0xFFFF
		if ev == wmLButtonUp || ev == wmRButtonUp {
			showMenu(hwnd)
		}
		return 0
	case wmClose:
		destroyWindow(hwnd)
		return 0
	case wmDestroy:
		postQuitMessage(0)
		return 0
	}
	return defWindowProcW(hwnd, msg, wParam, lParam)
}

// showMenu pops the context menu at the cursor and dispatches the chosen item.
// SetForegroundWindow + the trailing WM_NULL are the documented workaround so
// the menu dismisses when the user clicks elsewhere.
func showMenu(hwnd uintptr) {
	var pt point
	getCursorPos(&pt)
	setForegroundWindow(hwnd)
	cmd := trackPopupMenu(hMenu, tpmRightButton|tpmReturnCmd|tpmNoNotify, pt.X, pt.Y, 0, hwnd, 0)
	postMessageW(hwnd, wmNull, 0, 0)
	if cmd > 0 {
		cbMu.Lock()
		fn := callbacks[int(cmd)]
		cbMu.Unlock()
		if fn != nil {
			fn()
		}
	}
}

// buildMenu creates the popup once and records each clickable item's command id.
func buildMenu(items []Item) {
	cbMu.Lock()
	callbacks = map[int]func(){}
	cbMu.Unlock()

	h := createPopupMenu()
	seq := 0
	for _, it := range items {
		if it.Separator {
			appendMenuW(h, mfSeparator, 0, nil)
			continue
		}
		flags := uint32(mfString)
		if it.Disabled {
			flags |= mfGrayed
		}
		var id uintptr
		if it.OnClick != nil && !it.Disabled {
			seq++
			id = uintptr(seq)
			cbMu.Lock()
			callbacks[seq] = it.OnClick
			cbMu.Unlock()
		}
		appendMenuW(h, flags, id, utf16Ptr(it.Title))
	}
	hMenu = h
}

func run(cfg Config) error {
	mu.Lock()
	if running {
		mu.Unlock()
		return ErrAlreadyRunning
	}
	err := ensureInit()
	if err != nil {
		mu.Unlock()
		return err
	}
	running = true
	mu.Unlock()

	runtime.LockOSThread()

	hInst := getModuleHandleW(nil)
	hwnd := createWindowExW(0, classNamePtr, utf16Ptr("native tray"), 0, 0, 0, 0, 0, 0, 0, hInst, 0)
	if hwnd == 0 {
		mu.Lock()
		running = false
		mu.Unlock()
		return errors.New("tray: CreateWindowExW failed")
	}

	buildMenu(cfg.Items)

	nid = notifyIconData{}
	nid.cbSize = uint32(unsafe.Sizeof(nid)) // #nosec G115 -- fixed struct size (976 on x64)
	nid.hWnd = hwnd
	nid.uID = 1
	nid.uFlags = nifMessage | nifIcon | nifTip
	nid.uCallbackMessage = trayCallbackMsg
	nid.hIcon = loadIconW(0, idiApplication)
	setTip(&nid, tipText(cfg))
	shellNotifyIconW(nimAdd, &nid)

	mu.Lock()
	trayHwnd = hwnd
	mu.Unlock()

	var msg msgStruct
	for {
		r := getMessageW(&msg, 0, 0, 0)
		if r == 0 || r == -1 { // WM_QUIT or error
			break
		}
		translateMessage(&msg)
		dispatchMessageW(&msg)
	}

	shellNotifyIconW(nimDelete, &nid)
	destroyMenu(hMenu)

	mu.Lock()
	running = false
	trayHwnd = 0
	hMenu = 0
	mu.Unlock()
	return nil
}

func stop() {
	mu.Lock()
	h := trayHwnd
	r := running
	mu.Unlock()
	if !r || h == 0 {
		return
	}
	postMessageW(h, wmClose, 0, 0) // PostMessage is safe from any thread
}

// tipText is the hover tooltip: Tooltip, or Title as a fallback.
func tipText(cfg Config) string {
	if cfg.Tooltip != "" {
		return cfg.Tooltip
	}
	return cfg.Title
}

// setTip copies s into the fixed szTip buffer, NUL-terminated and truncated to
// fit. An embedded NUL (invalid) leaves the tip empty.
func setTip(n *notifyIconData, s string) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	m := len(u)
	if m > len(n.szTip) {
		m = len(n.szTip)
	}
	copy(n.szTip[:m], u[:m])
	n.szTip[len(n.szTip)-1] = 0
}

// utf16Ptr returns a NUL-terminated UTF-16 pointer for s ("" on an embedded NUL,
// never nil).
func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		empty, _ := syscall.UTF16PtrFromString("")
		return empty
	}
	return p
}
