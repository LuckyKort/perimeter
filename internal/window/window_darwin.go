package window

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// A regular app while the panel is open: Dock icon, ⌘Tab, and an Edit menu,
// without which ⌘C / ⌘V / ⌘A don't work in the page's text fields.
static void perimeter_setup_app(const char *closeTitle, const char *quitTitle) {
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	NSMenu *bar = [NSMenu new];

	NSMenuItem *appItem = [NSMenuItem new];
	NSMenu *appMenu = [NSMenu new];
	[appMenu addItemWithTitle:[NSString stringWithUTF8String:closeTitle] action:@selector(performClose:) keyEquivalent:@"w"];
	[appMenu addItemWithTitle:[NSString stringWithUTF8String:quitTitle] action:@selector(terminate:) keyEquivalent:@"q"];
	appItem.submenu = appMenu;
	[bar addItem:appItem];

	NSMenuItem *editItem = [NSMenuItem new];
	NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Edit"];
	[edit addItemWithTitle:@"Undo" action:@selector(undo:) keyEquivalent:@"z"];
	[edit addItemWithTitle:@"Redo" action:@selector(redo:) keyEquivalent:@"Z"];
	[edit addItem:[NSMenuItem separatorItem]];
	[edit addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
	[edit addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
	[edit addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
	[edit addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];
	editItem.submenu = edit;
	[bar addItem:editItem];

	[NSApp setMainMenu:bar];
	[NSApp activateIgnoringOtherApps:YES];
}
*/
import "C"

import (
	"unsafe"

	webview "github.com/webview/webview_go"
)

// Open shows the panel and blocks until the window is closed.
func Open(url, title, closeTitle, quitTitle string) error {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle(title)
	w.SetSize(Width, Height, webview.HintNone)
	ct, qt := C.CString(closeTitle), C.CString(quitTitle)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(qt))
	w.Dispatch(func() { C.perimeter_setup_app(ct, qt) })
	w.Navigate(url)
	w.Run()
	return nil
}
