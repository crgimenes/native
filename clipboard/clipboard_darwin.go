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

// Pasteboard types (UTIs).
const (
	nsPasteboardTypeString = "public.utf8-plain-text"
	nsPasteboardTypePNG    = "public.png"
	nsPasteboardTypeTIFF   = "public.tiff"

	nsBitmapImageFileTypePNG = 4
)

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
			_, err := purego.Dlopen(fw, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err != nil {
				initErr = fmt.Errorf("clipboard: load %s: %w", fw, err)
				return
			}
		}
	})
	return initErr
}

func sel(name string) objc.SEL {
	v, ok := selCache.Load(name)
	if ok {
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
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&id)) // #nosec G103 -- C string memory, not a Go pointer
	var n int
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n)) // #nosec G103 -- slice over the C string buffer
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
	err := ensureInit()
	if err != nil {
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
	err := ensureInit()
	if err != nil {
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

// goBytes copies an NSData's contents into Go memory.
func goBytes(data objc.ID) []byte {
	n := int(data.Send(sel("length"))) // #nosec G115 -- NSUInteger length of pasteboard data
	if n == 0 {
		return nil
	}
	p := data.Send(sel("bytes"))
	src := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), n) // #nosec G103 -- NSData buffer, not a Go pointer
	return append([]byte(nil), src...)
}

func readImage() ([]byte, error) {
	err := ensureInit()
	if err != nil {
		return nil, err
	}
	pbCls, err := class("NSPasteboard")
	if err != nil {
		return nil, err
	}
	repCls, err := class("NSBitmapImageRep")
	if err != nil {
		return nil, err
	}
	var out []byte
	autorelease(func() {
		pb := pbCls.Send(sel("generalPasteboard"))
		data := pb.Send(sel("dataForType:"), nsstr(nsPasteboardTypePNG))
		if data != 0 {
			out = goBytes(data)
			return
		}
		// Screenshots and Preview put TIFF here; hand back PNG all the same.
		data = pb.Send(sel("dataForType:"), nsstr(nsPasteboardTypeTIFF))
		if data == 0 {
			return
		}
		rep := repCls.Send(sel("imageRepWithData:"), data)
		if rep == 0 {
			return
		}
		png := rep.Send(sel("representationUsingType:properties:"), uint(nsBitmapImageFileTypePNG), objc.ID(0))
		if png != 0 {
			out = goBytes(png)
		}
	})
	return out, nil
}

func writeImage(png []byte) error {
	if len(png) == 0 {
		return errors.New("clipboard: empty image")
	}
	err := ensureInit()
	if err != nil {
		return err
	}
	pbCls, err := class("NSPasteboard")
	if err != nil {
		return err
	}
	dataCls, err := class("NSData")
	if err != nil {
		return err
	}
	var ok bool
	autorelease(func() {
		data := dataCls.Send(sel("dataWithBytes:length:"), unsafe.Pointer(&png[0]), uint(len(png))) // #nosec G103 -- NSData copies the bytes before returning
		pb := pbCls.Send(sel("generalPasteboard"))
		pb.Send(sel("clearContents"))
		ok = pb.Send(sel("setData:forType:"), data, nsstr(nsPasteboardTypePNG)) != 0
	})
	if !ok {
		return errors.New("clipboard: NSPasteboard setData:forType: failed")
	}
	return nil
}
