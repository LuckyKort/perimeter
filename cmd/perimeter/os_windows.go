package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// The .exe is built as a GUI program (no console window flashing at login), so
// when it's run from a terminal with a command, it borrows that terminal for output.
func attachConsole() {
	if len(os.Args) < 2 {
		return
	}
	k := windows.NewLazySystemDLL("kernel32.dll")
	const attachParent = ^uintptr(0) // ATTACH_PARENT_PROCESS
	if r, _, _ := k.NewProc("AttachConsole").Call(attachParent); r == 0 {
		return
	}
	if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = f, f
	}
}

func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == 259 // STILL_ACTIVE
}

// stopProcess: there is no SIGTERM on Windows, and a forced kill would leave
// apps frozen, so the user restarts Perimeter from the menu.
func stopProcess(int) bool { return false }
