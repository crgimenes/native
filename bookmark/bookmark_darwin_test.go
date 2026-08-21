package bookmark

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScopedRoundTrip exercises the real NSURL machinery. macOS allows
// creating security-scoped bookmark data outside the App Sandbox on
// some versions and refuses on others; when it refuses, the refusal is
// itself proof the objc call ran, and the test skips honestly. The full
// in-sandbox behavior can only run inside a sandboxed, signed app.
func TestScopedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "keep"), []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	token, err := scopedToken(dir)
	if err != nil {
		t.Skipf("scoped bookmark creation refused outside the sandbox: %v", err)
	}
	if token[0] != kindScoped {
		t.Fatalf("token kind = %q, want scoped", token[0])
	}

	path, release, err := resolveScoped(token[2:])
	if err != nil {
		t.Fatalf("resolveScoped: %v", err)
	}
	// TempDir on macOS lives under /var, a symlink to /private/var; the
	// resolved URL states the real path.
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	got, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(resolved): %v", err)
	}
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
	release()
	release()
}

func TestSandboxDetection(t *testing.T) {
	t.Setenv("APP_SANDBOX_CONTAINER_ID", "")
	if sandboxed() {
		t.Error("sandboxed() = true with the container id unset")
	}
	t.Setenv("APP_SANDBOX_CONTAINER_ID", "com.example.test")
	if !sandboxed() {
		t.Error("sandboxed() = false with the container id set")
	}
}
