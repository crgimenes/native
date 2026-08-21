# bookmark

Persistent access to user-selected files and folders across launches,
cgo-free.

```go
// After the user picks a folder (e.g. filedialog.PickDirectory):
token, err := bookmark.Create(dir)
// store token wherever the app keeps per-machine state

// On the next launch, before touching the folder:
path, release, err := bookmark.Resolve(token)
defer release()
```

Under the macOS App Sandbox, access granted through an open panel (the
powerbox) dies with the process; the token captures it as a
security-scoped bookmark so the next launch can restore it. On Windows
and Linux, and on macOS outside the sandbox, filesystem access already
persists, so the token simply records the path. Callers store the
opaque token and never branch by platform.

Details of the contract:

- `Create` requires the item to exist (macOS cannot bookmark a missing
  path; the other platforms match, so there is one behavior).
- `Resolve` starts the access grant and returns a release function,
  safe to call more than once. A sandboxed process holds a limited
  number of concurrent grants: release when done.
- The resolved path can differ from the created one when the user moved
  the item - macOS resolves the bookmark, not the path.
- A bookmark can go stale after the item moves; recreate the token
  after a successful `Resolve` when convenient.
- A security-scoped token resolved on another platform reports
  `ErrUnsupported` (tokens can travel in synced app state).

No main-thread requirement: unlike the panels in `filedialog`, the
bookmark calls have no UI.
