package window

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

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

// A transparent strip over the top of the page that moves the window: over a
// web view macOS doesn't reliably treat the transparent title bar as one, as
// the page takes the mouse first. A double click zooms, as on a title bar.
@interface PerimeterDragStrip : NSView
@end
@implementation PerimeterDragStrip
- (BOOL)mouseDownCanMoveWindow { return YES; }
- (void)mouseDown:(NSEvent *)event {
	if (event.clickCount == 2) {
		[self.window performZoom:nil];
		return;
	}
	[self.window performWindowDragWithEvent:event];
}
@end

// Modern window corners. On macOS 26+ apps with a sidebar (Finder, Telegram)
// have 26 pt corners, while AppKit gives this window the generic 16 pt. The
// window frame (NSThemeFrame) asks itself for the radius it draws, masks and
// shadows with; for ordinary windows of this process - the panel - it answers
// `radius`. [window class] also matches the KVO subclass AppKit swaps in for a
// window with a toolbar. Installed before the window exists, so the frame
// never caches the old value. Returns the radius in effect, 0 if untouched.
static double perimeter_modern_corners(double radius) {
	if (![[NSProcessInfo processInfo] isOperatingSystemAtLeastVersion:(NSOperatingSystemVersion){26, 0, 0}]) return 0;
	Class frame = NSClassFromString(@"NSThemeFrame");
	if (!frame) return 0;
	BOOL done = NO;
	for (NSString *name in @[@"_cornerRadius", @"_getCachedDefaultWindowCornerRadius", @"cornerRadius"]) {
		SEL sel = NSSelectorFromString(name);
		Method m = class_getInstanceMethod(frame, sel);
		if (!m || strcmp(method_getTypeEncoding(m), "d16@0:8") != 0) continue; // not the getter we know
		IMP orig = method_getImplementation(m);
		IMP mine = imp_implementationWithBlock(^double(NSView *view) {
			if ([view.window class] == [NSWindow class]) return radius;
			return ((double (*)(id, SEL))orig)(view, sel);
		});
		// on NSThemeFrame itself, so other views keep NSView's cornerRadius
		if (!class_addMethod(frame, sel, mine, method_getTypeEncoding(m))) method_setImplementation(m, mine);
		done = YES;
	}
	return done ? radius : 0;
}

// No title bar: the page fills the window, like Telegram. The traffic lights
// stay, and an empty unified toolbar makes the top strip taller, so they sit
// lower, inside the page's floating sidebar. The strip is still the window's
// title bar: dragging it moves the window, a double click zooms.
static void perimeter_frameless(void *ptr) {
	NSWindow *w = (__bridge NSWindow *)ptr;
	w.styleMask |= NSWindowStyleMaskFullSizeContentView;
	w.titlebarAppearsTransparent = YES;
	w.titleVisibility = NSWindowTitleHidden;
	NSToolbar *bar = [[NSToolbar alloc] initWithIdentifier:@"perimeter"];
	w.toolbar = bar;
	w.toolbarStyle = NSWindowToolbarStyleUnified;
	w.titlebarSeparatorStyle = NSTitlebarSeparatorStyleNone;
	w.minSize = NSMakeSize(760, 520);

	// The page keeps this strip free of controls (see html.mac in app.css).
	NSView *content = w.contentView;
	const CGFloat h = 44;
	NSRect r = content.bounds;
	BOOL flipped = content.isFlipped;
	PerimeterDragStrip *strip = [[PerimeterDragStrip alloc]
		initWithFrame:NSMakeRect(0, flipped ? 0 : NSHeight(r) - h, NSWidth(r), h)];
	strip.autoresizingMask = NSViewWidthSizable | (flipped ? NSViewMaxYMargin : NSViewMinYMargin);
	[content addSubview:strip positioned:NSWindowAbove relativeTo:nil];
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// windowRadius: the corner radius of macOS 26+ windows with a sidebar.
const windowRadius = 26

// Open shows the panel and blocks until the window is closed.
func Open(url, title, closeTitle, quitTitle string) error {
	r := float64(C.perimeter_modern_corners(windowRadius)) // before the window is created
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle(title)
	w.SetSize(Width, Height, webview.HintNone)
	ct, qt := C.CString(closeTitle), C.CString(quitTitle)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(qt))
	win := w.Window()
	// Inner shapes are concentric with the window (radius = window radius - inset).
	if r > 0 {
		w.Init(fmt.Sprintf("document.documentElement.style.setProperty('--win-r','%gpx')", r))
	}
	w.Dispatch(func() {
		C.perimeter_setup_app(ct, qt)
		C.perimeter_frameless(win)
	})
	w.Navigate(url)
	w.Run()
	return nil
}
