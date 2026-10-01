package mmap_test

import (
	"errors"
	"testing"

	"github.com/crgimenes/native/mmap"
)

// The sentinel must wrap the standard errors.ErrUnsupported.
//
// Without it a caller using several of these packages has to import every one
// of them just to name its sentinel, and a platform behaving exactly as
// documented reads as a failure to anyone who reasonably wrote the one check
// the standard library defines for the purpose. Adding a package to this module
// would silently break such a caller again, which is what this test prevents.
func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(mmap.ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("mmap.ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", mmap.ErrUnsupported)
	}
}
