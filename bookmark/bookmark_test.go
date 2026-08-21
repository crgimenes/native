package bookmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateResolveRoundTrip(t *testing.T) {
	// Outside a sandbox every platform uses the plain-path token; clear
	// the container id so the test is deterministic even in odd CI envs.
	t.Setenv("APP_SANDBOX_CONTAINER_ID", "")

	dir := t.TempDir()
	file := filepath.Join(dir, "data.txt")
	err := os.WriteFile(file, []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, target := range []string{dir, file} {
		token, err := Create(target)
		if err != nil {
			t.Fatalf("Create(%q): %v", target, err)
		}
		if token[0] != kindPath {
			t.Fatalf("token kind = %q, want plain path outside the sandbox", token[0])
		}

		path, release, err := Resolve(token)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if path != target {
			t.Errorf("Resolve = %q, want %q", path, target)
		}
		release()
		release() // must be safe to call more than once
	}
}

func TestCreateMissingPath(t *testing.T) {
	_, err := Create(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for a missing path")
	}
}

func TestResolveMalformedToken(t *testing.T) {
	for _, token := range [][]byte{nil, {}, {'p'}, {'x', 0, 'a'}, []byte("plain text")} {
		_, _, err := Resolve(token)
		if err == nil {
			t.Errorf("Resolve(%q) = nil error, want malformed-token error", token)
		}
	}
}

func TestResolveScopedTokenErrors(t *testing.T) {
	// A scoped token holds macOS bookmark data. Garbage data must fail
	// on macOS, and any scoped token must fail on the other platforms.
	_, _, err := Resolve([]byte{kindScoped, 0, 1, 2, 3})
	if err == nil {
		t.Fatal("expected error resolving a garbage scoped token")
	}
	if !strings.Contains(err.Error(), "bookmark") {
		t.Errorf("unexpected error: %v", err)
	}
}
