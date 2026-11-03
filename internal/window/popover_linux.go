package window

import (
	"errors"
	"io"
)

// Popover: on Linux the tray keeps its menu; desktops differ too much for a custom window.
func Popover(string, io.Reader) error { return errors.New("no tray window on Linux") }
