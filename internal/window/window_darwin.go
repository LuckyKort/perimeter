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
}

// Take the focus. Since macOS 14 the old activateIgnoringOtherApps is ignored
// for an app in the background; activate works because the app that was
// active (the tray window, the menu bar icon) yielded it to us.
static void perimeter_activate(void) {
	if (@available(macOS 14.0, *)) {
		[NSApp activate];
	} else {
		[NSApp activateIgnoringOtherApps:YES];
	}
}

// The window appears once the page has drawn its first frame with data (or
// after a timeout): a short fade instead of a blank page, then the unstyled
// one, then the real content.
static void perimeter_reveal(void *ptr) {
	NSWindow *w = (__bridge NSWindow *)ptr;
	perimeter_activate();
	[w makeKeyAndOrderFront:nil];
	if (w.alphaValue >= 1) return;
	[NSAnimationContext runAnimationGroup:^(NSAnimationContext *c) {
		c.duration = 0.15;
		w.animator.alphaValue = 1;
	} completionHandler:nil];
}

// Brings the open panel back: from the Dock, from behind other windows.
static void perimeter_focus(void *ptr) {
	NSWindow *w = (__bridge NSWindow *)ptr;
	if (w.miniaturized) [w deminiaturize:nil];
	perimeter_activate();
	[w makeKeyAndOrderFront:nil];
}

// The web view sits on the page's own background colour, not white, so no
// flash shows before the page paints.
static void perimeter_quiet_background(NSView *v) {
	if ([v isKindOfClass:NSClassFromString(@"WKWebView")]) [v setValue:@NO forKey:@"drawsBackground"];
	for (NSView *s in v.subviews) perimeter_quiet_background(s);
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
	// Practically invisible until perimeter_reveal, but not 0: macOS counts an
	// alpha-0 window as hidden, and WebKit then neither loads nor draws the page.
	w.alphaValue = 0.004;
	w.backgroundColor = [NSColor colorWithName:nil dynamicProvider:^NSColor *(NSAppearance *a) {
		BOOL dark = [a bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]] == NSAppearanceNameDarkAqua;
		return dark ? [NSColor colorWithWhite:0x14 / 255.0 alpha:1] : [NSColor colorWithWhite:0xF4 / 255.0 alpha:1]; // --bg in app.css
	}];
	perimeter_quiet_background(w.contentView);

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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// windowRadius: the corner radius of macOS 26+ windows with a sidebar.
const windowRadius = 26

// Open shows the panel and blocks until the window is closed. nextPage tells
// which page to switch to when Perimeter asks this panel to come forward
// (SIGUSR1) instead of opening a second one.
func Open(url, title, closeTitle, quitTitle string, nextPage func() string) error {
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

	// Shown when the page has drawn real content, at the latest after 2.5 s.
	var once sync.Once
	reveal := func() { once.Do(func() { w.Dispatch(func() { C.perimeter_reveal(win) }) }) }
	_ = w.Bind("perimeterReady", reveal)
	// ~0.8 s normally; the first start of a new build is slower (macOS checks it).
	fallback := time.AfterFunc(2500*time.Millisecond, reveal)
	defer fallback.Stop()

	// Perimeter asks this window to come forward, maybe on another page.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	defer signal.Stop(sig)
	go func() {
		for range sig {
			page := nextPage()
			w.Dispatch(func() {
				C.perimeter_focus(win)
				if page != "" && !strings.ContainsAny(page, "?#&/\\\"'<>") {
					w.Eval("location.hash = '#" + page + "'")
				}
			})
		}
	}()

	w.Navigate(url)
	w.Run()
	return nil
}
