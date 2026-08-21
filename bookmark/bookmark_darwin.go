// macOS security-scoped bookmarks: NSURL bookmark data via purego's
// Objective-C runtime (no cgo). Unlike the panels in filedialog, the
// bookmark calls have no UI and no main-thread requirement.

package bookmark

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	nsURLBookmarkCreationWithSecurityScope   = 1 << 11
	nsURLBookmarkResolutionWithSecurityScope = 1 << 10
)

var (
	initOnce sync.Once
	initErr  error
	selCache sync.Map
)

// ensureInit loads Foundation, which vends NSURL/NSData. It is usually
// already mapped in a GUI process, but dlopen'ing it is cheap and makes
// the package self-sufficient in a bare CLI or test binary.
func ensureInit() error {
	initOnce.Do(func() {
		const fw = "/System/Library/Frameworks/Foundation.framework/Foundation"
		_, err := purego.Dlopen(fw, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = fmt.Errorf("bookmark: load %s: %w", fw, err)
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

func class(name string) objc.ID {
	c := objc.GetClass(name)
	if c == 0 {
		panic(fmt.Sprintf("bookmark: objc class %q not found", name))
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

// nsErrorText renders an NSError for a Go error message.
func nsErrorText(errID objc.ID) string {
	if errID == 0 {
		return "unknown error"
	}
	return cstr(errID.Send(sel("localizedDescription")).Send(sel("UTF8String")))
}

// nsdataBytes copies an NSData's contents into Go memory, so the bytes
// survive the autorelease pool that owns the NSData.
func nsdataBytes(data objc.ID) []byte {
	n := int(data.Send(sel("length"))) // #nosec G115 -- bookmark blobs are small
	if n == 0 {
		return nil
	}
	raw := data.Send(sel("bytes"))
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&raw))              // #nosec G103 -- C buffer, not a Go pointer
	return append([]byte(nil), unsafe.Slice((*byte)(ptr), n)...) // #nosec G103 -- copy out of the C buffer
}

// sandboxed reports whether this process runs under the App Sandbox;
// the launchd-set container id is the standard signal.
func sandboxed() bool {
	return os.Getenv("APP_SANDBOX_CONTAINER_ID") != ""
}

// create picks the token kind: outside the sandbox a plain path is
// enough (and asking for a security scope there fails), inside it the
// grant must be captured as a security-scoped bookmark.
func create(path string) ([]byte, error) {
	if !sandboxed() {
		return plainToken(path), nil
	}
	return scopedToken(path)
}

func scopedToken(path string) (token []byte, err error) {
	err = ensureInit()
	if err != nil {
		return nil, err
	}

	autorelease(func() {
		url := class("NSURL").Send(sel("fileURLWithPath:"), nsstr(path))
		var errID objc.ID
		data := url.Send(
			sel("bookmarkDataWithOptions:includingResourceValuesForKeys:relativeToURL:error:"),
			uint(nsURLBookmarkCreationWithSecurityScope),
			objc.ID(0),
			objc.ID(0),
			unsafe.Pointer(&errID), // #nosec G103 -- NSError** out-param, written only during this synchronous call
		)
		if data == 0 {
			err = fmt.Errorf("bookmark: create for %s: %s", path, nsErrorText(errID))
			return
		}
		token = append([]byte{kindScoped, 0}, nsdataBytes(data)...)
	})
	return token, err
}

func resolveScoped(payload []byte) (path string, release func(), err error) {
	if len(payload) == 0 {
		return "", nil, fmt.Errorf("bookmark: empty scoped token")
	}
	err = ensureInit()
	if err != nil {
		return "", nil, err
	}

	var url objc.ID
	autorelease(func() {
		nsdata := class("NSData").Send(
			sel("dataWithBytes:length:"),
			unsafe.Pointer(&payload[0]), // #nosec G103 -- NSData copies the buffer during this synchronous call
			uint(len(payload)),
		)
		var errID objc.ID
		u := class("NSURL").Send(
			sel("URLByResolvingBookmarkData:options:relativeToURL:bookmarkDataIsStale:error:"),
			nsdata,
			uint(nsURLBookmarkResolutionWithSecurityScope),
			objc.ID(0),
			objc.ID(0),
			unsafe.Pointer(&errID), // #nosec G103 -- NSError** out-param, written only during this synchronous call
		)
		if u == 0 {
			err = fmt.Errorf("bookmark: resolve: %s", nsErrorText(errID))
			return
		}

		path = cstr(u.Send(sel("path")).Send(sel("UTF8String")))
		// BOOL returns are one byte; ignore whatever the register
		// carries above it.
		granted := u.Send(sel("startAccessingSecurityScopedResource"))&0xff != 0
		if !granted {
			err = fmt.Errorf("bookmark: the system refused the access grant for %s", path)
			return
		}
		// Retain past this pool: stopAccessing must reach the same
		// object later, from whatever goroutine calls release.
		url = u.Send(sel("retain"))
	})
	if err != nil {
		return "", nil, err
	}

	var once sync.Once
	release = func() {
		once.Do(func() {
			autorelease(func() {
				url.Send(sel("stopAccessingSecurityScopedResource"))
				url.Send(sel("release"))
			})
		})
	}
	return path, release, nil
}
