package openurl

import "testing"

// Smoke test for the Windows backend: it resolves shell32!ShellExecuteW so a
// broken library/symbol name is caught on CI. ShellExecuteW itself is not called
// because it would actually launch the browser/Explorer.
func TestWindowsBinding(t *testing.T) {
	err := ensureInit()
	if err != nil {
		t.Fatalf("ensureInit: %v", err)
	}
	if shellExecuteW == nil {
		t.Fatal("ShellExecuteW was not resolved")
	}
}
