package ui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// Since macOS 14 a background process can't take the focus on its own: the
// active app has to let it. The panel is another process of this same app.
static void perimeter_yield_activation(void) {
	if (@available(macOS 14.0, *)) {
		[NSApp yieldActivationToApplicationWithBundleIdentifier:NSBundle.mainBundle.bundleIdentifier ?: @""];
	}
}

static int perimeter_right_click(void) {
	NSEvent *e = [NSApp currentEvent];
	return e.type == NSEventTypeRightMouseDown || e.type == NSEventTypeRightMouseUp ||
		(e.modifierFlags & NSEventModifierFlagControl) != 0;
}
*/
import "C"

// rightClick: the menu bar library reports every mouse-down on the icon as a
// left click, so the event itself is asked (a Control-click counts as right).
func rightClick() bool { return C.perimeter_right_click() != 0 }

// yieldActivation lets the panel process come to the front (see the C side).
func yieldActivation() { C.perimeter_yield_activation() }
