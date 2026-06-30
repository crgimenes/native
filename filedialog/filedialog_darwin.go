// macOS open/save panels: NSOpenPanel and NSSavePanel via purego's Objective-C
// runtime (no cgo). The panels are application-modal (runModal), so callers must
// invoke Open/Save on the main thread.

package filedialog

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

const nsModalResponseOK = 1

var selCache sync.Map

func sel(name string) objc.SEL {
	if v, ok := selCache.Load(name); ok {
		return v.(objc.SEL)
	}
	s := objc.RegisterName(name)
	selCache.Store(name, s)
	return s
}

func class(name string) objc.ID {
	c := objc.GetClass(name)
	if c == 0 {
		panic(fmt.Sprintf("filedialog: objc class %q not found", name))
	}
	return objc.ID(c)
}

func nsstr(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

// cstr reads a Go string from a C string returned by -UTF8String.
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

func autorelease(f func()) {
	pool := class("NSAutoreleasePool").Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))
	f()
}

// restoreFocus re-activates the app and the window that was key before a modal
// panel. AppKit does not reliably hand focus back to the host window after the
// panel closes, so callers (e.g. an Ebitengine window) would be left unfocused.
// It restores whatever window was key beforehand, so it stays window-agnostic.
func restoreFocus(app, prev objc.ID) {
	if prev != 0 {
		prev.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
	}
	app.Send(sel("activateIgnoringOtherApps:"), true)
}

// allowedFileTypes builds an NSArray<NSString*> of bare extensions, or 0 (no
// restriction) when the list is empty or contains a wildcard.
func allowedFileTypes(exts []string) objc.ID {
	clean := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.TrimPrefix(e, ".")
		if e == "" || e == "*" {
			return 0
		}
		clean = append(clean, e)
	}
	if len(clean) == 0 {
		return 0
	}
	arr := class("NSMutableArray").Send(sel("array"))
	for _, e := range clean {
		arr.Send(sel("addObject:"), nsstr(e))
	}
	return arr
}

// applyCommon sets the options shared by the open and save panels.
func applyCommon(panel objc.ID, opts Options) {
	if opts.Title != "" {
		panel.Send(sel("setMessage:"), nsstr(opts.Title))
	}
	if opts.Directory != "" {
		url := class("NSURL").Send(sel("fileURLWithPath:"), nsstr(opts.Directory))
		panel.Send(sel("setDirectoryURL:"), url)
	}
	if types := allowedFileTypes(opts.Extensions); types != 0 {
		panel.Send(sel("setAllowedFileTypes:"), types)
	}
}

// Open shows a modal open-file panel and returns the chosen path ("" if cancelled).
func Open(opts Options) string {
	var path string
	autorelease(func() {
		app := class("NSApplication").Send(sel("sharedApplication"))
		prev := app.Send(sel("keyWindow"))
		defer restoreFocus(app, prev)
		panel := class("NSOpenPanel").Send(sel("openPanel"))
		panel.Send(sel("setCanChooseFiles:"), true)
		panel.Send(sel("setCanChooseDirectories:"), false)
		panel.Send(sel("setAllowsMultipleSelection:"), false)
		applyCommon(panel, opts)
		if int(panel.Send(sel("runModal"))) != nsModalResponseOK { // #nosec G115 -- small int response
			return
		}
		urls := panel.Send(sel("URLs"))
		if urls == 0 || int(urls.Send(sel("count"))) == 0 { // #nosec G115 -- small count
			return
		}
		u := urls.Send(sel("objectAtIndex:"), uint(0))
		path = cstr(u.Send(sel("path")).Send(sel("UTF8String")))
	})
	return path
}

// Save shows a modal save-file panel and returns the chosen path ("" if cancelled).
func Save(opts Options) string {
	var path string
	autorelease(func() {
		app := class("NSApplication").Send(sel("sharedApplication"))
		prev := app.Send(sel("keyWindow"))
		defer restoreFocus(app, prev)
		panel := class("NSSavePanel").Send(sel("savePanel"))
		if opts.Filename != "" {
			panel.Send(sel("setNameFieldStringValue:"), nsstr(opts.Filename))
		}
		applyCommon(panel, opts)
		if int(panel.Send(sel("runModal"))) != nsModalResponseOK { // #nosec G115 -- small int response
			return
		}
		if u := panel.Send(sel("URL")); u != 0 {
			path = cstr(u.Send(sel("path")).Send(sel("UTF8String")))
		}
	})
	return path
}
