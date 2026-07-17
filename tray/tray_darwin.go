// macOS backend: an NSStatusItem in the menu bar with an NSMenu, via purego's
// Objective-C runtime (no cgo). The status item and its menu are AppKit objects,
// so everything here runs on the main thread; Run drives NSApplication's run
// loop and Stop wakes it from any thread through -performSelectorOnMainThread:.
//
// Menu clicks come back through a small Objective-C target class registered once
// (NativeTrayTarget): each NSMenuItem carries its index as its tag, and the
// action looks the Go callback up by tag.

package tray

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	targetClassName = "NativeTrayTarget"

	nsApplicationActivationPolicyAccessory = 1    // menu-bar app, no Dock icon
	nsEventTypeApplicationDefined          = 15   // for the run-loop wake event
	nsVariableStatusItemLength             = -1.0 // NSVariableStatusItemLength
	statusIconSize                         = 18   // menu-bar icon side, in points
)

type cgPoint struct{ X, Y float64 }
type nsSize struct{ W, H float64 }

var (
	mu          sync.Mutex
	running     bool
	activeItems []Item
	trayTarget  objc.ID

	initOnce sync.Once
	initErr  error
	selCache sync.Map // string -> objc.SEL
)

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
		panic(fmt.Sprintf("tray: objc class %q not found", name))
	}
	return objc.ID(c)
}

func nsstr(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

// ensureInit loads AppKit (for NSStatusBar/NSMenu/NSImage) and registers the
// menu-action target class once. Foundation and AppKit are usually already
// mapped inside a GUI app, but dlopen'ing them makes a bare CLI binary work too.
func ensureInit() error {
	initOnce.Do(func() {
		for _, fw := range []string{
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/System/Library/Frameworks/AppKit.framework/AppKit",
		} {
			_, err := purego.Dlopen(fw, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err != nil {
				initErr = fmt.Errorf("tray: load %s: %w", fw, err)
				return
			}
		}
		_, err := objc.RegisterClass(
			targetClassName, objc.GetClass("NSObject"), nil, nil,
			[]objc.MethodDef{
				{
					Cmd: sel("trayItemClicked:"),
					Fn: func(self objc.ID, _cmd objc.SEL, sender objc.ID) {
						// #nosec G115 -- tag is an item index we set (0..len-1)
						onItemClicked(int(sender.Send(sel("tag"))))
					},
				},
				{
					Cmd: sel("trayStop"),
					Fn: func(self objc.ID, _cmd objc.SEL) {
						stopRunLoop()
					},
				},
			})
		if err != nil {
			initErr = fmt.Errorf("tray: register target class: %w", err)
		}
	})
	return initErr
}

func onItemClicked(tag int) {
	mu.Lock()
	var fn func()
	if tag >= 0 && tag < len(activeItems) {
		fn = activeItems[tag].OnClick
	}
	mu.Unlock()
	if fn != nil {
		fn()
	}
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
	activeItems = cfg.Items
	mu.Unlock()

	runtime.LockOSThread()

	app := class("NSApplication").Send(sel("sharedApplication"))
	app.Send(sel("setActivationPolicy:"), nsApplicationActivationPolicyAccessory)

	target := class(targetClassName).Send(sel("alloc")).Send(sel("init"))
	target.Send(sel("retain"))

	bar := class("NSStatusBar").Send(sel("systemStatusBar"))
	item := bar.Send(sel("statusItemWithLength:"), float64(nsVariableStatusItemLength))
	item.Send(sel("retain")) // the status bar does not keep it alive for us

	applyButton(item.Send(sel("button")), cfg)
	item.Send(sel("setMenu:"), buildMenu(cfg.Items, target))

	mu.Lock()
	trayTarget = target
	mu.Unlock()

	app.Send(sel("run")) // blocks until trayStop stops the loop

	bar.Send(sel("removeStatusItem:"), item)
	item.Send(sel("release"))
	target.Send(sel("release"))

	mu.Lock()
	running = false
	trayTarget = 0
	activeItems = nil
	mu.Unlock()
	return nil
}

// applyButton sets the status item's icon, title, and tooltip. It guarantees a
// visible button: with no icon and no title it falls back to a bullet glyph.
func applyButton(button objc.ID, cfg Config) {
	if button == 0 {
		return
	}
	if len(cfg.Icon) > 0 {
		data := class("NSData").Send(sel("dataWithBytes:length:"),
			unsafe.Pointer(&cfg.Icon[0]), uint(len(cfg.Icon))) // #nosec G103 -- NSData copies the bytes synchronously
		img := class("NSImage").Send(sel("alloc")).Send(sel("initWithData:"), data)
		if img != 0 {
			img.Send(sel("setSize:"), nsSize{statusIconSize, statusIconSize})
			button.Send(sel("setImage:"), img)
			img.Send(sel("release")) // the button retains it
		}
	}
	switch {
	case cfg.Title != "":
		button.Send(sel("setTitle:"), nsstr(cfg.Title))
	case len(cfg.Icon) == 0:
		button.Send(sel("setTitle:"), nsstr("●")) // ● so an iconless tray is still clickable
	}
	if cfg.Tooltip != "" {
		button.Send(sel("setToolTip:"), nsstr(cfg.Tooltip))
	}
}

// buildMenu builds an NSMenu whose clickable items target the shared target and
// carry their index as tag. autoenablesItems is off so Disabled is honored and
// enabled items stay clickable without a validateMenuItem: implementation.
func buildMenu(items []Item, target objc.ID) objc.ID {
	menu := class("NSMenu").Send(sel("alloc")).Send(sel("init"))
	menu.Send(sel("setAutoenablesItems:"), false)
	for i, it := range items {
		if it.Separator {
			menu.Send(sel("addItem:"), class("NSMenuItem").Send(sel("separatorItem")))
			continue
		}
		mi := class("NSMenuItem").Send(sel("alloc")).Send(
			sel("initWithTitle:action:keyEquivalent:"), nsstr(it.Title), sel("trayItemClicked:"), nsstr(""))
		mi.Send(sel("setTarget:"), target)
		mi.Send(sel("setTag:"), i)
		if it.Disabled {
			mi.Send(sel("setEnabled:"), false)
		}
		menu.Send(sel("addItem:"), mi)
		mi.Send(sel("release")) // the menu retains it
	}
	return menu
}

// stopRunLoop stops NSApplication and posts a dummy event so the run loop wakes
// and returns even when idle. Must run on the main thread.
func stopRunLoop() {
	app := class("NSApplication").Send(sel("sharedApplication"))
	app.Send(sel("stop:"), objc.ID(0))
	event := class("NSEvent").Send(
		sel("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:"),
		nsEventTypeApplicationDefined, cgPoint{0, 0}, uint(0), float64(0), 0, objc.ID(0), int16(0), 0, 0)
	app.Send(sel("postEvent:atStart:"), event, true)
}

func stop() {
	mu.Lock()
	t := trayTarget
	r := running
	mu.Unlock()
	if !r || t == 0 {
		return
	}
	// Route the AppKit teardown onto the main thread; Stop may be called from a
	// menu callback (already main) or any other goroutine.
	t.Send(sel("performSelectorOnMainThread:withObject:waitUntilDone:"), sel("trayStop"), objc.ID(0), false)
}
