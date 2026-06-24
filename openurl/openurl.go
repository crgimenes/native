// Package openurl opens URLs in the user's default handler and reveals files in
// the platform file manager, cgo-free.
//
// Each platform uses what the OS already ships — NSWorkspace on macOS,
// ShellExecuteW on Windows, xdg-open on Linux — with no C toolchain and no
// bundled native libraries.
//
//	openurl.Open("https://example.com")
//	openurl.Reveal("/path/to/file")
//
// Security: Open only accepts http, https, mailto and file URLs. Any other
// scheme (or a bare string with no scheme) is rejected with ErrScheme, so a
// hostile value can't launch an arbitrary protocol handler. The string is never
// passed through a shell.
package openurl

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnsupported is returned by operations on a platform that has no backend
// wired up yet.
var ErrUnsupported = errors.New("openurl: not supported on this platform")

// ErrScheme is returned by Open when the URL's scheme is not in the allow-list.
var ErrScheme = errors.New("openurl: refused URL scheme")

// allowedSchemes is the set Open will hand to the OS. Keeping it small is the
// safety boundary: an attacker-controlled string can at worst open a web page,
// an email draft, or a local file — never a custom protocol handler.
var allowedSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"file":   true,
}

// Open opens rawurl with the user's default handler (browser, mail client, ...).
// Only http, https, mailto and file URLs are allowed; anything else — including
// a bare hostname or path with no scheme — returns ErrScheme. For a local file
// use a file:// URL, or Reveal to show it in the file manager.
func Open(rawurl string) error {
	if err := validateScheme(rawurl); err != nil {
		return err
	}
	return openURL(rawurl)
}

// Reveal opens the platform file manager with path's location shown: Finder
// selects the file on macOS, Explorer selects it on Windows, and on Linux the
// containing folder is opened (selecting the file itself is file-manager
// specific and not portable). The path must exist.
func Reveal(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("openurl: resolve %q: %w", path, err)
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("openurl: reveal %q: %w", path, err)
	}
	return revealFile(abs)
}

// validateScheme enforces the Open allow-list. url.Parse normalizes the scheme
// to lower case, so the comparison is already case-insensitive.
func validateScheme(rawurl string) error {
	u, err := url.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("openurl: parse %q: %w", rawurl, err)
	}
	if !allowedSchemes[strings.ToLower(u.Scheme)] {
		return fmt.Errorf("%w: %q (allowed: http, https, mailto, file)", ErrScheme, u.Scheme)
	}
	return nil
}
