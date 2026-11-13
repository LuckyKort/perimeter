package window

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit -framework QuartzCore
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <QuartzCore/QuartzCore.h>

static const CGFloat popWidth = 340;

// A borderless panel that can take the keyboard (Escape, text selection)
// without activating the app, like the system's own menu bar popovers.
@interface PerimeterPanel : NSPanel
@end
@implementation PerimeterPanel
- (BOOL)canBecomeKeyWindow { return YES; }
@end

@interface PerimeterPop : NSObject <NSWindowDelegate, WKScriptMessageHandler, WKNavigationDelegate>
@property (strong) PerimeterPanel *panel;
@property (strong) WKWebView *web;
@property (assign) CFTimeInterval hiddenAt;
@property (assign) CGFloat height;
@end

@implementation PerimeterPop

- (void)setupWithURL:(NSString *)url {
	self.height = 420;
	self.panel = [[PerimeterPanel alloc] initWithContentRect:NSMakeRect(0, 0, popWidth, self.height)
		styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		backing:NSBackingStoreBuffered defer:NO];
	self.panel.level = NSPopUpMenuWindowLevel;
	self.panel.opaque = NO;
	self.panel.backgroundColor = NSColor.clearColor;
	self.panel.hasShadow = YES;
	self.panel.hidesOnDeactivate = NO;
	self.panel.releasedWhenClosed = NO;
	self.panel.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
		NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorTransient;
	self.panel.delegate = self;

	// Real system blur behind the page; the page only adds a light tint.
	NSVisualEffectView *blur = [[NSVisualEffectView alloc] initWithFrame:NSMakeRect(0, 0, popWidth, self.height)];
	blur.material = NSVisualEffectMaterialPopover;
	blur.blendingMode = NSVisualEffectBlendingModeBehindWindow;
	blur.state = NSVisualEffectStateActive;
	blur.wantsLayer = YES;
	blur.layer.cornerRadius = 18;
	blur.layer.masksToBounds = YES;
	blur.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	self.panel.contentView = blur;

	WKWebViewConfiguration *cfg = [WKWebViewConfiguration new];
	[cfg.userContentController addScriptMessageHandler:self name:@"perimeter"];
	self.web = [[WKWebView alloc] initWithFrame:blur.bounds configuration:cfg];
	[self.web setValue:@NO forKey:@"drawsBackground"];
	self.web.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	self.web.navigationDelegate = self;
	[blur addSubview:self.web];
	// Backup for the focus: a click in any other app closes the window too.
	[NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskLeftMouseDown | NSEventMaskRightMouseDown
		handler:^(NSEvent *e) { [self hide]; }];
	[self.web loadRequest:[NSURLRequest requestWithURL:[NSURL URLWithString:url]]];
}

// Pinned under the menu bar, centred on the click (the status item is under the mouse).
- (void)place {
	NSPoint m = [NSEvent mouseLocation];
	NSScreen *screen = NSScreen.mainScreen;
	for (NSScreen *s in NSScreen.screens) {
		if (NSPointInRect(m, s.frame)) { screen = s; break; }
	}
	NSRect vis = screen.visibleFrame;
	CGFloat x = m.x - popWidth / 2;
	// Opened by Perimeter itself (first start), not by a click: the right corner.
	if (m.y < NSMaxY(screen.frame) - 40) x = NSMaxX(vis) - popWidth - 12;
	x = MAX(NSMinX(vis) + 8, MIN(x, NSMaxX(vis) - popWidth - 8));
	CGFloat top = NSMaxY(vis) - 6;
	CGFloat h = MIN(self.height, NSHeight(vis) - 16); // a long card scrolls on a small screen
	[self.panel setFrame:NSMakeRect(x, top - h, popWidth, h) display:YES];
}

- (void)show {
	[self place];
	// Take the focus like the system's own menu bar popovers do: Escape works,
	// and a click anywhere else takes it away, which closes the window.
	[NSApp activateIgnoringOtherApps:YES];
	[self.panel makeKeyAndOrderFront:nil];
	[self.web evaluateJavaScript:@"window.perimeterShow && perimeterShow()" completionHandler:nil];
}

- (void)hide {
	if (!self.panel.visible) return;
	[self.panel orderOut:nil];
	self.hiddenAt = CACurrentMediaTime();
	if (NSApp.active) [NSApp hide:nil]; // the app you were in gets the focus back
	[self.web evaluateJavaScript:@"window.perimeterHide && perimeterHide()" completionHandler:nil];
}

- (void)toggle {
	if (self.panel.visible) { [self hide]; return; }
	// A click on the icon first takes the focus away (which hides the window)
	// and only then arrives here: that click means "close", not "open again".
	if (CACurrentMediaTime() - self.hiddenAt < 0.35) return;
	[self show];
}

- (void)windowDidResignKey:(NSNotification *)n { [self hide]; }

- (void)userContentController:(WKUserContentController *)c didReceiveScriptMessage:(WKScriptMessage *)msg {
	if (![msg.body isKindOfClass:NSDictionary.class]) return;
	NSString *op = msg.body[@"op"];
	if ([op isEqualToString:@"hide"]) {
		[self hide];
	} else if ([op isEqualToString:@"resize"]) {
		CGFloat h = [msg.body[@"height"] doubleValue];
		if (h < 120 || h > 1100 || fabs(h - self.height) < 1) return;
		self.height = h;
		NSScreen *screen = self.panel.screen ?: NSScreen.mainScreen;
		h = MIN(h, NSHeight(screen.visibleFrame) - 16);
		NSRect f = self.panel.frame;
		CGFloat top = NSMaxY(f);
		[self.panel setFrame:NSMakeRect(f.origin.x, top - h, popWidth, h) display:YES animate:self.panel.visible];
	}
}

// Links never open inside the popover.
- (void)webView:(WKWebView *)w decidePolicyForNavigationAction:(WKNavigationAction *)a decisionHandler:(void (^)(WKNavigationActionPolicy))h {
	NSString *host = a.request.URL.host;
	h([host isEqualToString:@"127.0.0.1"] ? WKNavigationActionPolicyAllow : WKNavigationActionPolicyCancel);
}
@end

static PerimeterPop *pop;

static void perimeter_pop_run(const char *url) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
		pop = [PerimeterPop new];
		[pop setupWithURL:[NSString stringWithUTF8String:url]];
		[NSApp run];
	}
}

static void perimeter_pop_cmd(int cmd) {
	dispatch_async(dispatch_get_main_queue(), ^{
		switch (cmd) {
		case 0: [pop toggle]; break;
		case 1: [pop show]; break;
		case 2: [pop hide]; break;
		}
	});
}
*/
import "C"

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"unsafe"
)

func init() { runtime.LockOSThread() } // Cocoa wants the main thread

// Popover runs the tray window: a hidden, ready window that shows on "toggle",
// "show" and "hide" lines from cmds, until cmds closes (Perimeter has exited).
func Popover(url string, cmds io.Reader) error {
	go func() {
		sc := bufio.NewScanner(cmds)
		for sc.Scan() {
			switch sc.Text() {
			case "toggle":
				C.perimeter_pop_cmd(0)
			case "show":
				C.perimeter_pop_cmd(1)
			case "hide":
				C.perimeter_pop_cmd(2)
			}
		}
		os.Exit(0)
	}()
	cu := C.CString(url)
	defer C.free(unsafe.Pointer(cu))
	C.perimeter_pop_run(cu)
	return nil
}
