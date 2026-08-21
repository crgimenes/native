// Package bookmark persists access to user-selected files and folders
// across application launches, cgo-free.
//
// Under the macOS App Store sandbox, access granted through an open
// panel (the powerbox) dies with the process; a security-scoped
// bookmark is the supported way to carry that grant to the next
// launch. On Windows and Linux, and on macOS outside the sandbox,
// filesystem access already persists, so the token simply records the
// path. The API is uniform: callers store the opaque token and never
// branch by platform.
//
// The usual flow: after the user picks a file or folder, Create a
// token and store it; on the next launch, Resolve the token before
// touching the item, and call the returned release function when done
// with it (a sandboxed process holds a limited number of concurrent
// grants). A resolved bookmark can go stale when the item moves;
// recreate the token after a successful Resolve when convenient.
package bookmark

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnsupported reports a token that cannot be resolved on this
// platform (a macOS security-scoped token on Windows or Linux).
var ErrUnsupported = errors.New("bookmark: token not resolvable on this platform")

// Token kinds. A token is the kind byte, a zero byte, then the
// payload: the literal path for kindPath, the security-scoped bookmark
// data for kindScoped.
const (
	kindPath   = 'p'
	kindScoped = 's'
)

// Create returns an opaque token that later restores access to path.
// The item must exist: macOS cannot bookmark a missing path, and the
// other platforms match that behavior so callers see one contract.
func Create(path string) ([]byte, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("bookmark: %w", err)
	}
	return create(path)
}

// Resolve turns a token back into a usable path and starts the access
// grant. The returned release function ends the grant and is safe to
// call more than once. The path may differ from the one given to
// Create if the user moved the item (macOS resolves the bookmark, not
// the path). Resolve does not check that the item still exists.
func Resolve(token []byte) (path string, release func(), err error) {
	kind, payload, err := splitToken(token)
	if err != nil {
		return "", nil, err
	}
	if kind == kindPath {
		return string(payload), func() {}, nil
	}
	return resolveScoped(payload)
}

func plainToken(path string) []byte {
	return append([]byte{kindPath, 0}, path...)
}

func splitToken(token []byte) (kind byte, payload []byte, err error) {
	if len(token) < 2 || token[1] != 0 {
		return 0, nil, errors.New("bookmark: malformed token")
	}
	switch token[0] {
	case kindPath, kindScoped:
		return token[0], token[2:], nil
	default:
		return 0, nil, fmt.Errorf("bookmark: unknown token kind %q", token[0])
	}
}
