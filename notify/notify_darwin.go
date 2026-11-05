// macOS: UNUserNotificationCenter when the process has a bundle identifier,
// osascript's display notification otherwise. Without a bundle
// UNUserNotificationCenter raises an uncaught NSException and the process
// aborts, and the deprecated NSUserNotificationCenter needs a borrowed
// identifier that usernoted refuses when the owner is a modern client; see
// README.

package notify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	unAuthorizationOptionSound = 1 << 1
	unAuthorizationOptionAlert = 1 << 2

	// The first Send of a bundled app waits for the user to answer the
	// permission prompt; afterwards the stored answer comes back at once.
	authorizeTimeout = time.Minute
	addTimeout       = 10 * time.Second

	scriptTimeout = 10 * time.Second
)

var foundation = sync.OnceValues(func() (uintptr, error) {
	return purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_GLOBAL|purego.RTLD_NOW)
})

var loadUserNotifications = sync.OnceValue(func() error {
	_, err := purego.Dlopen("/System/Library/Frameworks/UserNotifications.framework/UserNotifications", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	return err
})

func sel(name string) objc.SEL { return objc.RegisterName(name) }

func class(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

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

func nsError(e objc.ID) error {
	if e == 0 {
		return nil
	}
	return errors.New("notify: " + cstr(e.Send(sel("localizedDescription")).Send(sel("UTF8String"))))
}

func send(m Message) error {
	_, err := foundation()
	if err != nil {
		return fmt.Errorf("notify: load Foundation: %w", err)
	}
	// The autorelease pool is thread-local; drain must run where it was made.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := class("NSAutoreleasePool").Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))

	bundle := class("NSBundle").Send(sel("mainBundle"))
	if bundle.Send(sel("bundleIdentifier")) != 0 {
		return sendUN(m)
	}
	return sendScript(m)
}

func sendUN(m Message) error {
	err := loadUserNotifications()
	if err != nil {
		return fmt.Errorf("notify: load UserNotifications: %w", err)
	}
	center := class("UNUserNotificationCenter").Send(sel("currentNotificationCenter"))

	authorized := make(chan error, 1)
	authBlock := objc.NewBlock(func(_ objc.Block, granted bool, e objc.ID) {
		err := nsError(e)
		if err == nil && !granted {
			err = errors.New("notify: notifications are not allowed for this app (System Settings > Notifications)")
		}
		authorized <- err
	})
	defer authBlock.Release()
	center.Send(sel("requestAuthorizationWithOptions:completionHandler:"),
		uint(unAuthorizationOptionAlert|unAuthorizationOptionSound), authBlock)
	err = wait(authorized, authorizeTimeout, "authorization")
	if err != nil {
		return err
	}

	content := class("UNMutableNotificationContent").Send(sel("alloc")).Send(sel("init"))
	defer content.Send(sel("release"))
	content.Send(sel("setTitle:"), nsstr(m.Title))
	content.Send(sel("setBody:"), nsstr(m.Body))
	content.Send(sel("setSound:"), class("UNNotificationSound").Send(sel("defaultSound")))
	id := class("NSUUID").Send(sel("UUID")).Send(sel("UUIDString"))
	req := class("UNNotificationRequest").Send(sel("requestWithIdentifier:content:trigger:"), id, content, objc.ID(0))

	added := make(chan error, 1)
	addBlock := objc.NewBlock(func(_ objc.Block, e objc.ID) { added <- nsError(e) })
	defer addBlock.Release()
	center.Send(sel("addNotificationRequest:withCompletionHandler:"), req, addBlock)
	return wait(added, addTimeout, "delivery")
}

func wait(ch <-chan error, d time.Duration, what string) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(d):
		return fmt.Errorf("notify: %s did not complete within %s", what, d)
	}
}

// argv carries the text, so nothing in it is parsed as AppleScript.
func sendScript(m Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/osascript", // #nosec G204 -- fixed binary; the text is argv data, never script
		"-e", "on run argv",
		"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
		"-e", "end run",
		m.Title, m.Body).CombinedOutput()
	if err != nil {
		return fmt.Errorf("notify: osascript: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
