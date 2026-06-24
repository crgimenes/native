package openurl

import "testing"

// Smoke test for the macOS backend that exercises the Objective-C marshaling
// (framework load, class lookup, NSString/NSURL/NSArray construction) without
// calling openURL:/activateFileViewerSelectingURLs:, which would actually launch
// the browser/Finder. Catches a broken framework path or selector name on CI.
func TestDarwinObjcMarshaling(t *testing.T) {
	if err := ensureInit(); err != nil {
		t.Fatalf("ensureInit: %v", err)
	}
	if _, err := class("NSWorkspace"); err != nil {
		t.Fatalf("class NSWorkspace: %v", err)
	}
	urlCls, err := class("NSURL")
	if err != nil {
		t.Fatalf("class NSURL: %v", err)
	}
	arrCls, err := class("NSArray")
	if err != nil {
		t.Fatalf("class NSArray: %v", err)
	}
	autorelease(func() {
		if u := urlCls.Send(sel("URLWithString:"), nsstr("https://example.com")); u == 0 {
			t.Error("URLWithString: returned nil")
		}
		fileURL := urlCls.Send(sel("fileURLWithPath:"), nsstr("/tmp"))
		if fileURL == 0 {
			t.Error("fileURLWithPath: returned nil")
		}
		if arr := arrCls.Send(sel("arrayWithObject:"), fileURL); arr == 0 {
			t.Error("arrayWithObject: returned nil")
		}
	})
}
