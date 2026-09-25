// macOS: +[NSEvent addLocalMonitorForEventsMatchingMask:handler:] with a Go
// block, via purego's Objective-C runtime.

package pointer

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const nsEventTypeMagnify = 30

var loadAppKit = sync.OnceValue(func() error {
	_, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	return err
})

func watch(fn func(Event)) (func(), error) {
	err := loadAppKit()
	if err != nil {
		return nil, fmt.Errorf("pointer: %w", err)
	}
	cls := objc.ID(objc.GetClass("NSEvent"))
	if cls == 0 {
		return nil, fmt.Errorf("pointer: objc class NSEvent not found")
	}
	typeSel := objc.RegisterName("type")
	magSel := objc.RegisterName("magnification")
	block := objc.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID {
		if objc.Send[uint](ev, typeSel) == nsEventTypeMagnify {
			fn(Event{Kind: Magnify, Magnification: objc.Send[float64](ev, magSel)})
		}
		// Returning the event passes it on: the host still sees it.
		return ev
	})
	monitor := cls.Send(objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1)<<nsEventTypeMagnify, block)
	if monitor == 0 {
		block.Release()
		return nil, fmt.Errorf("pointer: NSEvent refused the monitor")
	}
	monitor.Send(objc.RegisterName("retain"))
	var once sync.Once
	return func() {
		once.Do(func() {
			cls.Send(objc.RegisterName("removeMonitor:"), monitor)
			monitor.Send(objc.RegisterName("release"))
			block.Release()
		})
	}, nil
}
