// Linux: a GtkDialog via purego -- a label, an optional GtkEntry and the
// buttons -- run modally by hand (present + "response" signal + main-loop
// iteration), because gtk_dialog_run is gone in GTK4. GtkDialog exists in GTK3
// and GTK4 alike.
//
// Stack selection follows filedialog: join the GTK the process already loaded
// (RTLD_NOLOAD), else load GTK3, else GTK4 -- two GTKs in one process corrupt
// the GObject type system.
//
// Behavior matches the macOS alert: the first button is the default (Return,
// also from the text field) and is laid out rightmost; Escape or closing the
// window chooses the Cancel button, or the first when there is none.

package alert

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	rtldNoload = 0x4 // glibc RTLD_NOLOAD; purego does not export it

	margin = 12
)

var (
	initOnce sync.Once
	initErr  error
	gtk4     bool

	gtkInitCheck3 func(argc, argv uintptr) bool
	gtkInitCheck4 func() bool

	gtkDialogNew                func() uintptr
	gtkDialogAddButton          func(dialog uintptr, text string, response int32) uintptr
	gtkDialogSetDefaultResp     func(dialog uintptr, response int32)
	gtkDialogGetContentArea     func(dialog uintptr) uintptr
	gtkWindowSetTitle           func(window uintptr, title string)
	gtkWindowSetModal           func(window uintptr, modal bool)
	gtkWindowSetResizable       func(window uintptr, resizable bool)
	gtkWindowPresent            func(window uintptr)
	gtkLabelNew                 func(text string) uintptr
	gtkLabelSetLineWrap         func(label uintptr, wrap bool)
	gtkEntryNew                 func() uintptr
	gtkEntrySetActivatesDefault func(entry uintptr, on bool)
	gtkWidgetGrabFocus          func(widget uintptr)
	gtkWidgetSetMargin          [4]func(widget uintptr, margin int32) // start, end, top, bottom

	// GTK3
	gtkContainerAdd  func(container, widget uintptr)
	gtkWidgetShowAll func(widget uintptr)
	gtkWidgetDestroy func(widget uintptr)
	gtkEntrySetText  func(entry uintptr, text string)
	gtkEntryGetText  func(entry uintptr) uintptr

	// GTK4
	gtkBoxAppend       func(box, widget uintptr)
	gtkWindowDestroy   func(window uintptr)
	gtkEditableSetText func(editable uintptr, text string)
	gtkEditableGetText func(editable uintptr) uintptr

	gSignalConnectData    func(instance uintptr, signal string, handler, data, destroy, flags uintptr) uint64
	gMainContextIteration func(ctx uintptr, mayBlock bool) bool

	responseFn uintptr

	gtkInitOnce sync.Once
	gtkInitOK   bool

	// One alert at a time: it is modal and runs on the main thread.
	respMu   sync.Mutex
	respID   int32
	respDone bool

	// testHook, when set, runs after the dialog is on screen, so a test can
	// answer it without a user.
	testHook func(dialog, entry uintptr)
)

func openGTK() (uintptr, error) {
	for _, try := range []struct {
		name string
		mode int
		is4  bool
	}{
		{"libgtk-4.so.1", purego.RTLD_LAZY | rtldNoload, true},
		{"libgtk-3.so.0", purego.RTLD_LAZY | rtldNoload, false},
		{"libgtk-3.so.0", purego.RTLD_LAZY | purego.RTLD_GLOBAL, false},
		{"libgtk-4.so.1", purego.RTLD_LAZY | purego.RTLD_GLOBAL, true},
	} {
		lib, err := purego.Dlopen(try.name, try.mode)
		if err == nil {
			gtk4 = try.is4
			return lib, nil
		}
	}
	return 0, fmt.Errorf("alert: no libgtk-3.so.0 or libgtk-4.so.1: %w", ErrUnsupported)
}

func ensureInit() error {
	initOnce.Do(func() { initErr = load() })
	return initErr
}

func load() error {
	gtk, err := openGTK()
	if err != nil {
		return err
	}
	glib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	gobject, err := purego.Dlopen("libgobject-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&gtkDialogNew, gtk, "gtk_dialog_new")
	purego.RegisterLibFunc(&gtkDialogAddButton, gtk, "gtk_dialog_add_button")
	purego.RegisterLibFunc(&gtkDialogSetDefaultResp, gtk, "gtk_dialog_set_default_response")
	purego.RegisterLibFunc(&gtkDialogGetContentArea, gtk, "gtk_dialog_get_content_area")
	purego.RegisterLibFunc(&gtkWindowSetTitle, gtk, "gtk_window_set_title")
	purego.RegisterLibFunc(&gtkWindowSetModal, gtk, "gtk_window_set_modal")
	purego.RegisterLibFunc(&gtkWindowSetResizable, gtk, "gtk_window_set_resizable")
	purego.RegisterLibFunc(&gtkWindowPresent, gtk, "gtk_window_present")
	purego.RegisterLibFunc(&gtkLabelNew, gtk, "gtk_label_new")
	purego.RegisterLibFunc(&gtkEntryNew, gtk, "gtk_entry_new")
	purego.RegisterLibFunc(&gtkEntrySetActivatesDefault, gtk, "gtk_entry_set_activates_default")
	purego.RegisterLibFunc(&gtkWidgetGrabFocus, gtk, "gtk_widget_grab_focus")
	for i, side := range []string{"start", "end", "top", "bottom"} {
		purego.RegisterLibFunc(&gtkWidgetSetMargin[i], gtk, "gtk_widget_set_margin_"+side)
	}
	purego.RegisterLibFunc(&gSignalConnectData, gobject, "g_signal_connect_data")
	purego.RegisterLibFunc(&gMainContextIteration, glib, "g_main_context_iteration")

	// "response" delivers (GtkDialog*, gint response_id, gpointer). gint is
	// 32-bit: narrow before use so -4 survives the widening into a register.
	responseFn = purego.NewCallback(func(dialog, id, data uintptr) uintptr {
		respMu.Lock()
		respID = int32(uint32(id)) // #nosec G115 -- deliberate gint narrowing
		respDone = true
		respMu.Unlock()
		return 0
	})

	if gtk4 {
		purego.RegisterLibFunc(&gtkInitCheck4, gtk, "gtk_init_check")
		purego.RegisterLibFunc(&gtkBoxAppend, gtk, "gtk_box_append")
		purego.RegisterLibFunc(&gtkWindowDestroy, gtk, "gtk_window_destroy")
		purego.RegisterLibFunc(&gtkEditableSetText, gtk, "gtk_editable_set_text")
		purego.RegisterLibFunc(&gtkEditableGetText, gtk, "gtk_editable_get_text")
		purego.RegisterLibFunc(&gtkLabelSetLineWrap, gtk, "gtk_label_set_wrap")
		return nil
	}
	purego.RegisterLibFunc(&gtkInitCheck3, gtk, "gtk_init_check")
	purego.RegisterLibFunc(&gtkContainerAdd, gtk, "gtk_container_add")
	purego.RegisterLibFunc(&gtkWidgetShowAll, gtk, "gtk_widget_show_all")
	purego.RegisterLibFunc(&gtkWidgetDestroy, gtk, "gtk_widget_destroy")
	purego.RegisterLibFunc(&gtkEntrySetText, gtk, "gtk_entry_set_text")
	purego.RegisterLibFunc(&gtkEntryGetText, gtk, "gtk_entry_get_text")
	purego.RegisterLibFunc(&gtkLabelSetLineWrap, gtk, "gtk_label_set_line_wrap")
	return nil
}

// gtkReady loads and initializes GTK once, on the calling (main) thread. A
// process that already initialized GTK (a glaze window) passes straight
// through gtk_init_check.
func gtkReady() error {
	err := ensureInit()
	if err != nil {
		return err
	}
	gtkInitOnce.Do(func() {
		if gtk4 {
			gtkInitOK = gtkInitCheck4()
			return
		}
		gtkInitOK = gtkInitCheck3(0, 0)
	})
	if !gtkInitOK {
		return errors.New("alert: gtk_init_check failed (no display?)")
	}
	return nil
}

func show(opts Options) (Result, error) {
	err := gtkReady()
	if err != nil {
		return Result{}, err
	}
	dlg := gtkDialogNew()
	gtkWindowSetTitle(dlg, opts.Title)
	gtkWindowSetModal(dlg, true)
	gtkWindowSetResizable(dlg, false)

	area := gtkDialogGetContentArea(dlg)
	add := func(w uintptr) {
		for _, set := range gtkWidgetSetMargin {
			set(w, margin)
		}
		if gtk4 {
			gtkBoxAppend(area, w)
			return
		}
		gtkContainerAdd(area, w)
	}
	for _, text := range []string{opts.Title, opts.Message} {
		if text == "" {
			continue
		}
		label := gtkLabelNew(text)
		gtkLabelSetLineWrap(label, true)
		add(label)
	}
	var entry uintptr
	if opts.Input {
		entry = gtkEntryNew()
		setEntryText(entry, opts.Text)
		gtkEntrySetActivatesDefault(entry, true)
		add(entry)
	}

	// Added last-first, so the first button, the default, ends up rightmost.
	cancel := 0
	for i := len(opts.Buttons) - 1; i >= 0; i-- {
		gtkDialogAddButton(dlg, opts.Buttons[i].Title, int32(i)) // #nosec G115 -- a handful of buttons
		if opts.Buttons[i].Cancel {
			cancel = i
		}
	}
	gtkDialogSetDefaultResp(dlg, 0)

	respMu.Lock()
	respDone = false
	respMu.Unlock()
	gSignalConnectData(dlg, "response", responseFn, 0, 0, 0)

	if gtk4 {
		gtkWindowPresent(dlg)
	} else {
		gtkWidgetShowAll(dlg)
	}
	if entry != 0 {
		gtkWidgetGrabFocus(entry)
	}
	if testHook != nil {
		testHook(dlg, entry)
	}
	for !answered() {
		gMainContextIteration(0, true)
	}

	var res Result
	respMu.Lock()
	res.Button = int(respID)
	respMu.Unlock()
	if res.Button < 0 { // Escape, or the window closed
		res.Button = cancel
	}
	if entry != 0 {
		res.Text = entryText(entry)
	}
	if gtk4 {
		gtkWindowDestroy(dlg)
	} else {
		gtkWidgetDestroy(dlg)
	}
	// Let the dialog actually leave the screen before the caller goes on.
	for gMainContextIteration(0, false) {
	}
	return res, nil
}

func answered() bool {
	respMu.Lock()
	defer respMu.Unlock()
	return respDone
}

func setEntryText(entry uintptr, text string) {
	if gtk4 {
		gtkEditableSetText(entry, text)
		return
	}
	gtkEntrySetText(entry, text)
}

// entryText copies the entry's text: GTK owns the returned string.
func entryText(entry uintptr) string {
	if gtk4 {
		return cstr(gtkEditableGetText(entry))
	}
	return cstr(gtkEntryGetText(entry))
}

func cstr(p uintptr) string {
	if p == 0 {
		return ""
	}
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&p)) // #nosec G103 -- C string owned by GTK
	n := 0
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n)) // #nosec G103 -- copies the C string
}
