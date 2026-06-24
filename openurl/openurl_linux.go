// Linux backend: xdg-open, the cross-desktop launcher. No native library and no
// shell — the argument is passed as a single argv entry, so it can't be word-
// split or interpreted by a shell. xdg-open returns once the handler has been
// started, so the exit status reports whether the launch (not the handler)
// succeeded.
//
// The "proper" path would be the XDG desktop portal (org.freedesktop.portal
// .OpenURI) over D-Bus, but xdg-open is what actually works on every desktop and
// keeps this package dependency-free; the portal can come later if needed.

package openurl

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func openURL(rawurl string) error {
	return runXdgOpen(rawurl)
}

// revealFile opens the folder containing absPath. Highlighting the specific file
// is file-manager specific on Linux (nautilus --select, dolphin --select, ...)
// and not portable, so openurl opens the containing directory instead.
func revealFile(absPath string) error {
	return runXdgOpen(filepath.Dir(absPath))
}

func runXdgOpen(arg string) error {
	err := exec.Command("xdg-open", arg).Run()
	if err != nil {
		return fmt.Errorf("openurl: xdg-open %q: %w", arg, err)
	}
	return nil
}
