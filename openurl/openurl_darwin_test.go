package openurl

import "testing"

// Smoke test for the macOS backend that exercises the Objective-C marshaling
// (framework load, class lookup, NSString/NSURL/NSArray construction) without
// calling openURL:/activateFileViewerSelectingURLs:, which would actually launch
// the browser/Finder. Catches a broken framework path or selector name on CI.
func TestDarwinObjcMarshaling(t *testing.T) {
	err := ensureInit()
	if err != nil {
		t.Fatalf("ensureInit: %v", err)
	}
	_, err = class("NSWorkspace")
	if err != nil {
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
		u := urlCls.Send(sel("URLWithString:"), nsstr("https://example.com"))
		if u == 0 {
			t.Error("URLWithString: returned nil")
		}
		fileURL := urlCls.Send(sel("fileURLWithPath:"), nsstr("/tmp"))
		if fileURL == 0 {
			t.Error("fileURLWithPath: returned nil")
		}
		arr := arrCls.Send(sel("arrayWithObject:"), fileURL)
		if arr == 0 {
			t.Error("arrayWithObject: returned nil")
		}
	})
}
