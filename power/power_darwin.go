// macOS backend: IOKit power-management assertions via purego. PreventSleep
// creates a PreventUserIdleSystemSleep assertion and Release drops it by id.
//
// The assertion is process-global, not thread-scoped, so no thread pinning is
// needed for create/release. The two CFStrings are created with a Create call
// and released explicitly with CFRelease, so there is no NSAutoreleasePool in
// play (and therefore none of its thread-affinity hazard).

package power

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

const (
	// kIOPMAssertionTypePreventUserIdleSystemSleep: keep the system from
	// idle-sleeping while still letting the display sleep.
	assertionTypePreventIdleSystemSleep = "PreventUserIdleSystemSleep"
	assertionLevelOn                    = 255        // kIOPMAssertionLevelOn
	cfStringEncodingUTF8                = 0x08000100 // kCFStringEncodingUTF8
)

var (
	initOnce sync.Once
	initErr  error

	cfStringCreateWithCString   func(alloc uintptr, cStr string, encoding uint32) uintptr
	cfRelease                   func(cf uintptr)
	iopmAssertionCreateWithName func(assertionType uintptr, level uint32, name uintptr, id *uint32) int32
	iopmAssertionRelease        func(id uint32) int32
)

type handle uint32 // IOPMAssertionID

func ensureInit() error {
	initOnce.Do(func() {
		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = fmt.Errorf("power: load CoreFoundation: %w", err)
			return
		}
		iokit, err := purego.Dlopen("/System/Library/Frameworks/IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = fmt.Errorf("power: load IOKit: %w", err)
			return
		}
		purego.RegisterLibFunc(&cfStringCreateWithCString, cf, "CFStringCreateWithCString")
		purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
		purego.RegisterLibFunc(&iopmAssertionCreateWithName, iokit, "IOPMAssertionCreateWithName")
		purego.RegisterLibFunc(&iopmAssertionRelease, iokit, "IOPMAssertionRelease")
	})
	return initErr
}

func preventSleep(reason string) (handle, error) {
	err := ensureInit()
	if err != nil {
		return 0, err
	}
	if reason == "" {
		reason = "native/power"
	}

	cfType := cfStringCreateWithCString(0, assertionTypePreventIdleSystemSleep, cfStringEncodingUTF8)
	if cfType == 0 {
		return 0, errors.New("power: CFStringCreateWithCString (assertion type) failed")
	}
	defer cfRelease(cfType)

	cfReason := cfStringCreateWithCString(0, reason, cfStringEncodingUTF8)
	if cfReason == 0 {
		return 0, errors.New("power: CFStringCreateWithCString (reason) failed")
	}
	defer cfRelease(cfReason)

	var id uint32
	ret := iopmAssertionCreateWithName(cfType, assertionLevelOn, cfReason, &id)
	if ret != 0 {
		return 0, fmt.Errorf("power: IOPMAssertionCreateWithName failed: %#x", ret)
	}
	return handle(id), nil
}

func release(h handle) error {
	err := ensureInit()
	if err != nil {
		return err
	}
	ret := iopmAssertionRelease(uint32(h))
	if ret != 0 {
		return fmt.Errorf("power: IOPMAssertionRelease failed: %#x", ret)
	}
	return nil
}
