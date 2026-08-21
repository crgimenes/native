// Linux open/save panels: GtkFileChooserNative via purego, ported from glaze's
// proven dialog_linux.go. The modal is driven manually (set_modal + show +
// "response" signal + main-loop iteration) because gtk_native_dialog_run was
// removed in GTK4; the manual sequence is exactly what it did internally and
// works on both GTK3 and GTK4.
//
// Stack selection: loading GTK3 and GTK4 into one process corrupts the GObject
// type system (the glaze real-hardware crash), so the package first probes with
// RTLD_NOLOAD for a GTK the host process ALREADY loaded (a glaze/GTK app) and
// joins it. Only when neither is present does it load one fresh: GTK3 first
// (the stack most desktops ship), then GTK4.
//
// Threading: everything here runs on the calling thread, which the package
// contract requires to be the main thread; gtk_init and every later GTK call
// then agree on the thread. A process with no display fails gtk_init_check and
// every panel returns "".

package filedialog

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	// RTLD_NOLOAD (glibc and musl): return the handle only if the library is
	// already mapped, never load it. purego does not export it.
	rtldNoload = 0x4

	gtkFileChooserActionOpen         = 0
	gtkFileChooserActionSave         = 1
	gtkFileChooserActionSelectFolder = 2

	gtkResponseAccept = -3
)

var (
	initOnce sync.Once
	initErr  error
	gtk4     bool

	gtkInitCheck3 func(argc, argv uintptr) bool
	gtkInitCheck4 func() bool

	gtkFileChooserNativeNew      func(title string, parent uintptr, action int, accept, cancel string) uintptr
	gtkNativeDialogShow          func(dialog uintptr)
	gtkNativeDialogHide          func(dialog uintptr)
	gtkNativeDialogSetModal      func(dialog uintptr, modal bool)
	gtkFileChooserSetCurrentName func(chooser uintptr, name string)
	gtkFileFilterNew             func() uintptr
	gtkFileFilterSetName         func(filter uintptr, name string)
	gtkFileFilterAddPattern      func(filter uintptr, pattern string)
	gtkFileChooserAddFilter      func(chooser, filter uintptr)

	// GTK3 path-based result + folder selection.
	gtkFileChooserGetFilename      func(chooser uintptr) uintptr // char*
	gtkFileChooserSetCurrentFolder func(chooser uintptr, path string) bool

	// GTK4 GFile-based result + folder selection (need GIO).
	gtkFileChooserGetFile           func(chooser uintptr) uintptr // GFile*
	gtkFileChooserSetCurrentFolder4 func(chooser, file, err uintptr) bool
	gFileNewForPath                 func(path string) uintptr
	gFileGetPath                    func(file uintptr) uintptr // char*

	gFree                 func(p uintptr)
	gObjectUnref          func(obj uintptr)
	gSignalConnectData    func(instance uintptr, signal string, handler, data uintptr, destroy, flags uintptr) uint64
	gMainContextIteration func(ctx uintptr, mayBlock bool) bool

	dialogResponseFn uintptr // GtkNativeDialog "response" callback

	gtkInitOnce sync.Once
	gtkInitOK   bool
)

// dialogResp captures a single modal dialog's response. Dialogs run one at a
// time on the UI thread (modal), keyed by an integer token passed as the
// signal's user_data so only integers cross into C.
type dialogResp struct {
	response int
	done     bool
}

var (
	dialogRespMu     sync.Mutex
	dialogRespStates = map[uintptr]*dialogResp{}
	dialogRespSeq    uintptr
)

// openGTK picks the GTK stack: join the one already mapped into the process if
// any (RTLD_NOLOAD probe — never loads), else load GTK3, else GTK4.
func openGTK() (uintptr, error) {
	lib, err := purego.Dlopen("libgtk-4.so.1", purego.RTLD_LAZY|rtldNoload)
	if err == nil {
		gtk4 = true
		return lib, nil
	}
	lib, err = purego.Dlopen("libgtk-3.so.0", purego.RTLD_LAZY|rtldNoload)
	if err == nil {
		return lib, nil
	}
	lib, err = purego.Dlopen("libgtk-3.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err == nil {
		return lib, nil
	}
	lib, err = purego.Dlopen("libgtk-4.so.1", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err == nil {
		gtk4 = true
		return lib, nil
	}
	return 0, errors.New("filedialog: neither libgtk-3.so.0 nor libgtk-4.so.1 could be loaded")
}

func ensureInit() error {
	initOnce.Do(func() {
		gtkLib, err := openGTK()
		if err != nil {
			initErr = err
			return
		}
		glib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = err
			return
		}
		gobject, err := purego.Dlopen("libgobject-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = err
			return
		}

		purego.RegisterLibFunc(&gtkFileChooserNativeNew, gtkLib, "gtk_file_chooser_native_new")
		purego.RegisterLibFunc(&gtkNativeDialogShow, gtkLib, "gtk_native_dialog_show")
		purego.RegisterLibFunc(&gtkNativeDialogHide, gtkLib, "gtk_native_dialog_hide")
		purego.RegisterLibFunc(&gtkNativeDialogSetModal, gtkLib, "gtk_native_dialog_set_modal")
		purego.RegisterLibFunc(&gtkFileChooserSetCurrentName, gtkLib, "gtk_file_chooser_set_current_name")
		purego.RegisterLibFunc(&gtkFileFilterNew, gtkLib, "gtk_file_filter_new")
		purego.RegisterLibFunc(&gtkFileFilterSetName, gtkLib, "gtk_file_filter_set_name")
		purego.RegisterLibFunc(&gtkFileFilterAddPattern, gtkLib, "gtk_file_filter_add_pattern")
		purego.RegisterLibFunc(&gtkFileChooserAddFilter, gtkLib, "gtk_file_chooser_add_filter")

		purego.RegisterLibFunc(&gFree, glib, "g_free")
		purego.RegisterLibFunc(&gMainContextIteration, glib, "g_main_context_iteration")
		purego.RegisterLibFunc(&gObjectUnref, gobject, "g_object_unref")
		purego.RegisterLibFunc(&gSignalConnectData, gobject, "g_signal_connect_data")

		// "response" delivers (GtkNativeDialog*, gint response_id, gpointer
		// token). gint is 32-bit; mask before interpreting so a negative id
		// (ACCEPT = -3) survives the widening into a uintptr register.
		dialogResponseFn = purego.NewCallback(func(dialog, responseID, token uintptr) uintptr {
			dialogRespMu.Lock()
			st := dialogRespStates[token]
			if st != nil {
				st.response = int(int32(uint32(responseID))) // #nosec G115 -- deliberate gint narrowing
				st.done = true
			}
			dialogRespMu.Unlock()
			return 0
		})

		if gtk4 {
			purego.RegisterLibFunc(&gtkInitCheck4, gtkLib, "gtk_init_check")
			purego.RegisterLibFunc(&gtkFileChooserGetFile, gtkLib, "gtk_file_chooser_get_file")
			purego.RegisterLibFunc(&gtkFileChooserSetCurrentFolder4, gtkLib, "gtk_file_chooser_set_current_folder")
			gio, e := purego.Dlopen("libgio-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if e != nil {
				initErr = e
				return
			}
			purego.RegisterLibFunc(&gFileNewForPath, gio, "g_file_new_for_path")
			purego.RegisterLibFunc(&gFileGetPath, gio, "g_file_get_path")
			return
		}
		purego.RegisterLibFunc(&gtkInitCheck3, gtkLib, "gtk_init_check")
		purego.RegisterLibFunc(&gtkFileChooserGetFilename, gtkLib, "gtk_file_chooser_get_filename")
		purego.RegisterLibFunc(&gtkFileChooserSetCurrentFolder, gtkLib, "gtk_file_chooser_set_current_folder")
	})
	return initErr
}

// gtkReady loads GTK and initializes it once, on the calling (main) thread.
// false means no usable display (or no GTK), in which case every panel
// degrades to "".
func gtkReady() bool {
	err := ensureInit()
	if err != nil {
		return false
	}
	gtkInitOnce.Do(func() {
		// gtk_init_check's arity differs by major version: GTK3 takes
		// (int *argc, char ***argv), GTK4 takes no arguments.
		if gtk4 {
			gtkInitOK = gtkInitCheck4()
			return
		}
		gtkInitOK = gtkInitCheck3(0, 0)
	})
	return gtkInitOK
}

// --- the panels ------------------------------------------------------------

func open(opts Options) string {
	return runFileChooser(gtkFileChooserActionOpen, opts)
}

func save(opts Options) string {
	return runFileChooser(gtkFileChooserActionSave, opts)
}

func pickDirectory(opts Options) string {
	return runFileChooser(gtkFileChooserActionSelectFolder, opts)
}

func runFileChooser(action int, opts Options) string {
	if !gtkReady() {
		return ""
	}
	accept := "_Open"
	if action == gtkFileChooserActionSave {
		accept = "_Save"
	}
	dlg := gtkFileChooserNativeNew(opts.Title, 0, action, accept, "_Cancel")
	if dlg == 0 {
		return ""
	}
	defer gObjectUnref(dlg)

	if opts.Directory != "" {
		setChooserFolder(dlg, opts.Directory)
	}
	if action == gtkFileChooserActionSave && opts.Filename != "" {
		gtkFileChooserSetCurrentName(dlg, opts.Filename)
	}
	if action != gtkFileChooserActionSelectFolder {
		applyChooserFilter(dlg, opts.Extensions)
	}

	if runNativeDialog(dlg) != gtkResponseAccept {
		return ""
	}
	return chooserPath(dlg)
}

// runNativeDialog shows a GtkNativeDialog modally and pumps the main loop until
// the user responds, returning the response id.
func runNativeDialog(dlg uintptr) int {
	dialogRespMu.Lock()
	dialogRespSeq++
	token := dialogRespSeq
	st := &dialogResp{}
	dialogRespStates[token] = st
	dialogRespMu.Unlock()
	defer func() {
		dialogRespMu.Lock()
		delete(dialogRespStates, token)
		dialogRespMu.Unlock()
	}()

	gSignalConnectData(dlg, "response", dialogResponseFn, token, 0, 0)
	gtkNativeDialogSetModal(dlg, true)
	gtkNativeDialogShow(dlg)
	for {
		dialogRespMu.Lock()
		done := st.done
		dialogRespMu.Unlock()
		if done {
			break
		}
		gMainContextIteration(0, true)
	}
	gtkNativeDialogHide(dlg)
	return st.response
}

func setChooserFolder(chooser uintptr, dir string) {
	if gtk4 {
		file := gFileNewForPath(dir)
		if file == 0 {
			return
		}
		gtkFileChooserSetCurrentFolder4(chooser, file, 0)
		gObjectUnref(file)
		return
	}
	gtkFileChooserSetCurrentFolder(chooser, dir)
}

// applyChooserFilter restricts the chooser to Options.Extensions as one
// GtkFileFilter ("*.a, *.b"). No restriction adds no filter.
func applyChooserFilter(chooser uintptr, exts []string) {
	clean := cleanExtensions(exts)
	if clean == nil {
		return
	}
	filter := gtkFileFilterNew()
	name := ""
	for i, e := range clean {
		if i > 0 {
			name += ", "
		}
		name += "*." + e
	}
	gtkFileFilterSetName(filter, name)
	for _, e := range clean {
		gtkFileFilterAddPattern(filter, "*."+e)
	}
	gtkFileChooserAddFilter(chooser, filter) // transfers ownership to the chooser
}

// chooserPath reads the selected path out of a chooser after an accepted run,
// branching on the GTK version.
func chooserPath(chooser uintptr) string {
	if gtk4 {
		file := gtkFileChooserGetFile(chooser) // transfer full
		if file == 0 {
			return ""
		}
		defer gObjectUnref(file)
		return gfilePath(file)
	}
	cs := gtkFileChooserGetFilename(chooser) // char*, owned by caller
	if cs == 0 {
		return ""
	}
	p := cstr(cs)
	gFree(cs)
	return p
}

// gfilePath returns a GFile's local path ("" if it has none).
func gfilePath(file uintptr) string {
	cs := gFileGetPath(file) // char*, owned by caller
	if cs == 0 {
		return ""
	}
	p := cstr(cs)
	gFree(cs)
	return p
}

// --- C string helpers ------------------------------------------------------

// ptr reinterprets a uintptr's bits as an unsafe.Pointer without a direct
// uintptr->Pointer conversion. The values it is fed are C heap pointers the Go
// GC neither owns nor moves; the spelling only keeps go vet's unsafeptr check
// quiet.
func ptr(u uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&u)) } // #nosec G103 -- audited FFI reinterpret

// cstr reads a NUL-terminated C string.
func cstr(p uintptr) string {
	if p == 0 {
		return ""
	}
	base := ptr(p)
	var n int
	for *(*byte)(unsafe.Add(base, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(base), n)) // #nosec G103 -- slice over the C string buffer
}
