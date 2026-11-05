# notify

Post a desktop notification, cgo-free.

```go
import "github.com/crgimenes/native/notify"

err := notify.Send(notify.Message{Title: "Build done", Body: "all green"})
if err != nil {
	// errors.Is(err, notify.ErrUnsupported) where there is no backend
	log.Print(err)
}
```

`Send` returns once the OS has accepted the notification. It does not wait for
the user to see or dismiss it. It is safe to call from any goroutine, and it
needs no event loop or main thread.

## API

| Symbol | Description |
| --- | --- |
| `Send(m Message) error` | Post one notification. A message with neither title nor body is rejected. |
| `Message` | `Title`, `Body`. Plain text. |
| `ErrUnsupported` | Returned on every platform but macOS. Wraps `errors.ErrUnsupported`. |

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS, app bundle | `UNUserNotificationCenter` | ✅ seen on macOS 27 |
| macOS, bare binary | `osascript` `display notification` | ✅ seen on macOS 27 |
| Linux | — (`org.freedesktop.Notifications` over D-Bus would be the route) | ⬜ `ErrUnsupported` |
| Windows | — (a WinRT toast would be the route) | ⬜ `ErrUnsupported` |

Only macOS has a real consumer today, so Linux and Windows return
`ErrUnsupported`. Check it with `errors.Is(err, notify.ErrUnsupported)`.

## macOS: bundle or bare binary

macOS ties notifications to an app identity, so there is no single path that
works for every Go program. `Send` picks one from the process:

- **The process has a bundle identifier** (an `.app` such as a glaze app):
  `UNUserNotificationCenter`. The notification carries the app's name and icon,
  and the user controls it under System Settings > Notifications like any other
  app. The first `Send` shows the permission prompt and waits up to a minute for
  the answer. After that the stored answer comes back at once, and a refusal
  turns into an error from `Send`. usernoted has to be able to find and validate
  the bundle through LaunchServices: an ad-hoc-signed `.app` under `/private/tmp`
  was rejected ("Notifications are not allowed for this application"), while
  the same bundle in an ordinary folder worked, whether started with `open` or
  by running `Contents/MacOS/<exe>` directly.
- **A bare binary** (a CLI, a launchd job): `/usr/bin/osascript -e 'display
  notification …'`, with the title and body passed as `argv` so nothing in them
  is parsed as AppleScript. The banner is attributed to **Script Editor** (its
  icon, its entry in Notification settings), and macOS asks once to allow Script
  Editor. It costs a process spawn and plays no sound.

Why not the deprecated `NSUserNotification` for bare binaries, the classic CLI
route? `NSUserNotificationCenter` gives a process without a bundle identifier a
nil center, so it only works by giving the main bundle a borrowed identifier.
On macOS 27, usernoted refuses that borrow when the real owner is a modern
(UserNotifications) client ("You can't mix modern clients with legacy clients"),
which rules out the natural choice, the launching app's `__CFBundleIdentifier`.
Borrowing `com.apple.Terminal` still gets through, but the notifications then
need Terminal's permission, a deprecated API, and a process-wide class swap on
`NSBundle`, and they would break as soon as Terminal moved to
UserNotifications. `osascript` costs one process spawn and avoids all of that.

## Example

```bash
go run ./examples/notify "Build done" "all green"
```

## Testing

A real notification lands on the screen of whoever runs the tests, so `TestSend`
runs only with `NATIVE_NOTIFY_TEST=1`, which CI sets on the macOS runner (the
test binary has no bundle, so that is the osascript path).

## Conventions

Part of [native](../README.md). Public API in the tag-free `notify.go`, the
backend in `notify_darwin.go`, and `notify_other.go` (`!darwin`) returning
`ErrUnsupported` so every `GOOS` builds.
