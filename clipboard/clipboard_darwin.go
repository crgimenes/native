// macOS clipboard backend: NSPasteboard via purego's Objective-C runtime.

package clipboard

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// NSPasteboardTypeString is the UTI for plain UTF-8 text on the pasteboard.
const nsPasteboardTypeString = "public.utf8-plain-text"

var (
	initOnce sync.Once
	initErr  error
	selCache sync.Map // string -> objc.SEL
)

// ensureInit loads the frameworks that vend NSString/NSPasteboard. AppKit and
// Foundation are usually already mapped, but dlopen'ing them is cheap and makes
// the package self-sufficient when used from a bare CLI binary.
func ensureInit() error {
	initOnce.Do(func() {
		for _, fw := range []string{
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/System/Library/Frameworks/AppKit.framework/AppKit",
		} {
			if _, err := purego.Dlopen(fw, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				initErr = fmt.Errorf("clipboard: load %s: %w", fw, err)
				return
			}
		}
	})
	return initErr
}

func sel(name string) objc.SEL {
	if v, ok := selCache.Load(name); ok {
		return v.(objc.SEL)
	}
	s := objc.RegisterName(name)
	selCache.Store(name, s)
	return s
}

func class(name string) (objc.ID, error) {
	c := objc.GetClass(name)
	if c == 0 {
		return 0, fmt.Errorf("clipboard: objc class %q not found", name)
	}
	return objc.ID(c), nil
}

// nsstr builds an autoreleased NSString from a Go string.
func nsstr(s string) objc.ID {
	cls, _ := class("NSString") // NSString always exists once Foundation is loaded.
	return cls.Send(sel("stringWithUTF8String:"), s)
}

// cstr reads a NUL-terminated C string returned as an objc.ID (e.g. -UTF8String).
func cstr(id objc.ID) string {
	if id == 0 {
		return ""
	}
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&id)) // circumvent go vet
	var n int
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n))
}

// autorelease wraps f in an NSAutoreleasePool, draining it afterward.
//
// LockOSThread pins the goroutine for the whole pool lifetime: an
// NSAutoreleasePool is thread-local, so if the goroutine migrated between
// creating the pool and the deferred drain (which Go's scheduler is free to do),
// the pool would be drained on the wrong thread and corrupt the autorelease
// stack — an intermittent SIGSEGV. The defers run LIFO, so drain happens before
// UnlockOSThread, i.e. while still on the creating thread.
func autorelease(f func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cls, _ := class("NSAutoreleasePool")
	pool := cls.Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))
	f()
}

func readText() (string, error) {
	if err := ensureInit(); err != nil {
		return "", err
	}
	pbCls, err := class("NSPasteboard")
	if err != nil {
		return "", err
	}
	var out string
	autorelease(func() {
		pb := pbCls.Send(sel("generalPasteboard"))
		s := pb.Send(sel("stringForType:"), nsstr(nsPasteboardTypeString))
		if s != 0 {
			out = cstr(s.Send(sel("UTF8String")))
		}
	})
	return out, nil
}

func writeText(s string) error {
	if err := ensureInit(); err != nil {
		return err
	}
	pbCls, err := class("NSPasteboard")
	if err != nil {
		return err
	}
	var ok bool
	autorelease(func() {
		pb := pbCls.Send(sel("generalPasteboard"))
		pb.Send(sel("clearContents"))
		ok = pb.Send(sel("setString:forType:"), nsstr(s), nsstr(nsPasteboardTypeString)) != 0
	})
	if !ok {
		return errors.New("clipboard: NSPasteboard setString:forType: failed")
	}
	return nil
}
