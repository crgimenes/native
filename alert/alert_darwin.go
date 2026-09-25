// macOS: NSAlert via purego's Objective-C runtime. runModal is
// application-modal, so Show must be called on the main thread.

package alert

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// AppKit is loaded here rather than assumed: a host without a GUI toolkit
// of its own has not loaded it, and NSAlert would not be found.
var loadAppKit = sync.OnceValue(func() error {
	_, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	return err
})

// NSAlertFirstButtonReturn; the n-th button added answers 1000+n.
const firstButtonReturn = 1000

type cgPoint struct{ X, Y float64 }
type cgSize struct{ Width, Height float64 }
type cgRect struct {
	Origin cgPoint
	Size   cgSize
}

var selCache sync.Map

func sel(name string) objc.SEL {
	v, ok := selCache.Load(name)
	if ok {
		return v.(objc.SEL)
	}
	s := objc.RegisterName(name)
	selCache.Store(name, s)
	return s
}

func class(name string) objc.ID {
	c := objc.GetClass(name)
	if c == 0 {
		panic(fmt.Sprintf("alert: objc class %q not found", name))
	}
	return objc.ID(c)
}

func nsstr(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

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

func show(opts Options) (Result, error) {
	var res Result
	err := loadAppKit()
	if err != nil {
		return res, fmt.Errorf("alert: %w", err)
	}
	pool := class("NSAutoreleasePool").Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))

	app := class("NSApplication").Send(sel("sharedApplication"))
	prev := app.Send(sel("keyWindow"))
	// AppKit does not reliably give focus back to the window that had it.
	defer func() {
		if prev != 0 {
			prev.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		}
		app.Send(sel("activateIgnoringOtherApps:"), true)
	}()

	a := class("NSAlert").Send(sel("alloc")).Send(sel("init"))
	defer a.Send(sel("release"))
	a.Send(sel("setMessageText:"), nsstr(opts.Title))
	if opts.Message != "" {
		a.Send(sel("setInformativeText:"), nsstr(opts.Message))
	}
	for _, b := range opts.Buttons {
		btn := a.Send(sel("addButtonWithTitle:"), nsstr(b.Title))
		if b.Cancel {
			btn.Send(sel("setKeyEquivalent:"), nsstr("\x1b"))
		}
		// hasDestructiveAction arrived in macOS 11.
		if b.Destructive && objc.Send[bool](btn, sel("respondsToSelector:"), sel("setHasDestructiveAction:")) {
			btn.Send(sel("setHasDestructiveAction:"), true)
		}
	}
	var field objc.ID
	if opts.Input {
		field = class("NSTextField").Send(sel("alloc")).Send(sel("initWithFrame:"),
			cgRect{Size: cgSize{Width: 240, Height: 24}})
		defer field.Send(sel("release"))
		field.Send(sel("setStringValue:"), nsstr(opts.Text))
		a.Send(sel("setAccessoryView:"), field)
		a.Send(sel("layout"))
		a.Send(sel("window")).Send(sel("setInitialFirstResponder:"), field)
	}
	res.Button = int(a.Send(sel("runModal"))) - firstButtonReturn // #nosec G115 -- small modal response
	if field != 0 {
		res.Text = cstr(field.Send(sel("stringValue")).Send(sel("UTF8String")))
	}
	return res, nil
}
