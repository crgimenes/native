//go:build !darwin

package bookmark

import "fmt"

// Outside macOS there is no powerbox: access to a path the user chose
// persists across launches on its own, so the token is just the path.
func create(path string) ([]byte, error) {
	return plainToken(path), nil
}

func resolveScoped(_ []byte) (string, func(), error) {
	return "", nil, fmt.Errorf("%w: security-scoped token created on macOS", ErrUnsupported)
}
