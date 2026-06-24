package openurl

import (
	"os"
	"path/filepath"
	"testing"
)

// On Linux the backend shells out to xdg-open. A fake xdg-open on PATH records
// its argument, so the real exec path (and the URL/folder it is given) is
// exercised on CI without launching a browser or file manager.
func TestLinuxInvokesXdgOpen(t *testing.T) {
	dir := t.TempDir()
	argFile := filepath.Join(dir, "arg")
	fake := filepath.Join(dir, "xdg-open")
	script := "#!/bin/sh\nprintf '%s' \"$1\" > " + argFile + "\n"
	err := os.WriteFile(fake, []byte(script), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	readArg := func() string {
		b, err := os.ReadFile(argFile)
		if err != nil {
			t.Fatalf("read recorded arg: %v", err)
		}
		return string(b)
	}

	// Open hands the URL to xdg-open unchanged.
	err = Open("https://example.com/x")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := readArg()
	if got != "https://example.com/x" {
		t.Fatalf("xdg-open arg = %q, want the URL", got)
	}

	// Reveal opens the file's containing folder.
	sub := filepath.Join(dir, "sub")
	err = os.MkdirAll(sub, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "file.txt")
	err = os.WriteFile(file, []byte("x"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = Reveal(file)
	if err != nil {
		t.Fatalf("Reveal: %v", err)
	}
	got = readArg()
	if got != sub {
		t.Fatalf("xdg-open arg = %q, want the folder %q", got, sub)
	}
}
