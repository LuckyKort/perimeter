package ui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static int kordon_right_click(void) {
	NSEvent *e = [NSApp currentEvent];
	return e.type == NSEventTypeRightMouseDown || e.type == NSEventTypeRightMouseUp ||
		(e.modifierFlags & NSEventModifierFlagControl) != 0;
}
*/
import "C"

// rightClick: the menu bar library reports every mouse-down on the icon as a
// left click, so the event itself is asked (a Control-click counts as right).
func rightClick() bool { return C.kordon_right_click() != 0 }
