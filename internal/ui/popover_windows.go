package ui

import "golang.org/x/sys/windows"

var allowSetForeground = windows.NewLazySystemDLL("user32.dll").NewProc("AllowSetForegroundWindow")

// letForeground: Windows only lets the process that got the click bring a
// window forward, so the tray hands that right to the tray window's process.
func letForeground(pid int) { allowSetForeground.Call(uintptr(pid)) }
