package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#include <stdlib.h>

// Perimeter is a menu bar app, so macOS counts it as always in the foreground
// and would hide its banners: the delegate asks to show them anyway.
@interface PerimeterNotes : NSObject <UNUserNotificationCenterDelegate>
@end
@implementation PerimeterNotes
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))handler {
	handler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList);
}
@end

static PerimeterNotes *perimeterNotes;

// Only an app bundle has a notification identity; a bare binary has none.
static int perimeterCanNotify(void) {
	return [[NSBundle mainBundle] bundleIdentifier] != nil;
}

static void perimeterNotify(const char *title, const char *body) {
	NSString *t = [NSString stringWithUTF8String:title];
	NSString *b = [NSString stringWithUTF8String:body];
	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		perimeterNotes = [PerimeterNotes new];
		center.delegate = perimeterNotes;
	});
	// Asks once (the system remembers the answer), then sends if allowed.
	[center requestAuthorizationWithOptions:UNAuthorizationOptionAlert
		completionHandler:^(BOOL granted, NSError *error) {
			if (!granted) return;
			UNMutableNotificationContent *content = [UNMutableNotificationContent new];
			content.title = t;
			content.body = b;
			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:[[NSUUID UUID] UUIDString]
				content:content trigger:nil];
			[center addNotificationRequest:request withCompletionHandler:nil];
		}];
}
*/
import "C"

import "unsafe"

// nativeNotify shows a notification from Perimeter itself (its name and icon,
// its switch in System Settings → Notifications). false: not an app bundle.
func nativeNotify(title, body string) bool {
	if C.perimeterCanNotify() == 0 {
		return false
	}
	ct, cb := C.CString(title), C.CString(body)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	C.perimeterNotify(ct, cb)
	return true
}
