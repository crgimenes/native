package openurl

import (
	"errors"
	"testing"
)

// The scheme allow-list is the package's security boundary, so it is tested
// directly. The tests never call Open/Reveal with a launchable target, so
// nothing actually opens a browser or file manager on CI.

func TestValidateSchemeAllows(t *testing.T) {
	allowed := []string{
		"http://example.com",
		"https://example.com/a?b=c#d",
		"HTTPS://EXAMPLE.COM", // scheme is matched case-insensitively
		"mailto:someone@example.com",
		"file:///tmp/report.pdf",
	}
	for _, u := range allowed {
		if err := validateScheme(u); err != nil {
			t.Errorf("validateScheme(%q) = %v, want nil", u, err)
		}
	}
}

func TestValidateSchemeRejects(t *testing.T) {
	rejected := []string{
		"",                     // no scheme
		"example.com",          // bare host, no scheme
		"/etc/passwd",          // bare path
		"ftp://example.com",    // not in the allow-list
		"javascript:alert(1)",  // would run script in some handlers
		"vbscript:msgbox(1)",   //
		"data:text/html,<h1>x", //
		"smb://host/share",     //
	}
	for _, u := range rejected {
		if err := validateScheme(u); !errors.Is(err, ErrScheme) {
			t.Errorf("validateScheme(%q) = %v, want ErrScheme", u, err)
		}
	}
}

// TestOpenRejectsBadScheme checks the public entry point refuses a disallowed
// scheme before it reaches the platform backend (so nothing is launched).
func TestOpenRejectsBadScheme(t *testing.T) {
	if err := Open("javascript:alert(1)"); !errors.Is(err, ErrScheme) {
		t.Fatalf("Open(javascript:) = %v, want ErrScheme", err)
	}
}
