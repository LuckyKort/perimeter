package appicon

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// The icon Finder shows for the file, drawn into a px x px RGBA bitmap.
static void *perimeterIconPNG(const char *path, int px, int *outLen) {
	@autoreleasepool {
		NSImage *img = [[NSWorkspace sharedWorkspace] iconForFile:[NSString stringWithUTF8String:path]];
		if (!img) return NULL;
		NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:px pixelsHigh:px
			bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
		if (!rep) return NULL;
		[NSGraphicsContext saveGraphicsState];
		[NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];
		[img drawInRect:NSMakeRect(0, 0, px, px) fromRect:NSZeroRect operation:NSCompositingOperationCopy fraction:1.0];
		[NSGraphicsContext restoreGraphicsState];
		NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		if (!png) return NULL;
		void *buf = malloc(png.length);
		memcpy(buf, png.bytes, png.length);
		*outLen = (int)png.length;
		return buf;
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// PNG renders the bundle's icon at px x px.
func PNG(bundle string, px int) ([]byte, error) {
	cs := C.CString(bundle)
	defer C.free(unsafe.Pointer(cs))
	var n C.int
	p := C.perimeterIconPNG(cs, C.int(px), &n)
	if p == nil {
		return nil, errors.New("no icon for " + bundle)
	}
	defer C.free(p)
	return C.GoBytes(p, n), nil
}
