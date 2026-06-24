// macOS backend: NSWorkspace via purego's Objective-C runtime. Open uses
// -[NSWorkspace openURL:]; Reveal uses -[NSWorkspace activateFileViewerSelectingURLs:]
// (the "Reveal in Finder" call), which highlights the file in a Finder window.

package openurl

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	initOnce sync.Once
	initErr  error
	selCache sync.Map // string -> objc.SEL
)

// ensureInit loads the frameworks that vend NSURL/NSArray/NSWorkspace. They are
// usually already mapped, but dlopen'ing them is cheap and makes the package
// self-sufficient when used from a bare CLI binary.
func ensureInit() error {
	initOnce.Do(func() {
		for _, fw := range []string{
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/System/Library/Frameworks/AppKit.framework/AppKit",
		} {
			if _, err := purego.Dlopen(fw, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				initErr = fmt.Errorf("openurl: load %s: %w", fw, err)
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
		return 0, fmt.Errorf("openurl: objc class %q not found", name)
	}
	return objc.ID(c), nil
}

// nsstr builds an autoreleased NSString from a Go string.
func nsstr(s string) objc.ID {
	cls, _ := class("NSString") // NSString always exists once Foundation is loaded.
	return cls.Send(sel("stringWithUTF8String:"), s)
}

// autorelease wraps f in an NSAutoreleasePool, draining it afterward.
func autorelease(f func()) {
	cls, _ := class("NSAutoreleasePool")
	pool := cls.Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))
	f()
}

func openURL(rawurl string) error {
	if err := ensureInit(); err != nil {
		return err
	}
	wsCls, err := class("NSWorkspace")
	if err != nil {
		return err
	}
	urlCls, err := class("NSURL")
	if err != nil {
		return err
	}
	var ok bool
	autorelease(func() {
		ws := wsCls.Send(sel("sharedWorkspace"))
		nsurl := urlCls.Send(sel("URLWithString:"), nsstr(rawurl))
		if nsurl != 0 {
			ok = ws.Send(sel("openURL:"), nsurl) != 0
		}
	})
	if !ok {
		return fmt.Errorf("openurl: NSWorkspace openURL: failed for %q", rawurl)
	}
	return nil
}

func revealFile(absPath string) error {
	if err := ensureInit(); err != nil {
		return err
	}
	wsCls, err := class("NSWorkspace")
	if err != nil {
		return err
	}
	urlCls, err := class("NSURL")
	if err != nil {
		return err
	}
	arrCls, err := class("NSArray")
	if err != nil {
		return err
	}
	autorelease(func() {
		ws := wsCls.Send(sel("sharedWorkspace"))
		fileURL := urlCls.Send(sel("fileURLWithPath:"), nsstr(absPath))
		urls := arrCls.Send(sel("arrayWithObject:"), fileURL)
		ws.Send(sel("activateFileViewerSelectingURLs:"), urls)
	})
	return nil
}
