//go:build !windows

// Fallback for every platform without a working backend. macOS lands here on
// purpose: NSWindowSharingNone stopped working in macOS 15.4, Apple calls it
// a legacy constant and DTS says no public capture-prevention API exists —
// and on macOS 26 setting the legacy value can stop the window from rendering
// at all. Linux has no portable compositor API. Saying no beats pretending.

package nocapture

import "unsafe"

func protect(_ unsafe.Pointer) error { return ErrUnsupported }
