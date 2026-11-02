package window

import "errors"

// Open has no window of its own on Linux; the caller falls back to the browser.
func Open(_, _, _, _ string) error { return errors.New("no native window on Linux") }
