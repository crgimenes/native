//go:build !darwin && !windows

// Every platform without a tray backend (Linux included). A Linux tray means a
// StatusNotifierItem plus a com.canonical.dbusmenu export over D-Bus — a D-Bus
// dependency this module avoids, fragmented across desktops (GNOME needs an
// extension). Out of scope here, so Run reports ErrUnsupported.

package tray

func run(_ Config) error { return ErrUnsupported }

func stop() {}
