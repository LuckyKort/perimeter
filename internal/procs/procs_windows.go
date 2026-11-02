package procs

import (
	"errors"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntdll            = windows.NewLazySystemDLL("ntdll.dll")
	ntSuspendProcess = ntdll.NewProc("NtSuspendProcess")
	ntResumeProcess  = ntdll.NewProc("NtResumeProcess")
	user32           = windows.NewLazySystemDLL("user32.dll")
	enumWindows      = user32.NewProc("EnumWindows")
	getWindowPID     = user32.NewProc("GetWindowThreadProcessId")
	postMessage      = user32.NewProc("PostMessageW")
)

const (
	processSuspendResume = 0x0800
	wmClose              = 0x0010
)

type cached struct {
	created int64
	path    string
}

type osController struct {
	mu    sync.Mutex
	cache map[int]cached
}

func New() Controller { return &osController{cache: map[int]cached{}} }

func (c *osController) List() ([]Proc, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	c.mu.Lock()
	defer c.mu.Unlock()
	seen := map[int]bool{}
	var out []Proc
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		pid := int(e.ProcessID)
		if pid == 0 {
			continue
		}
		seen[pid] = true
		name := windows.UTF16ToString(e.ExeFile[:])
		created := creationTime(pid)
		ce, ok := c.cache[pid]
		if !ok || ce.created != created {
			ce = cached{created: created, path: imagePath(pid)}
			c.cache[pid] = ce
		}
		out = append(out, Proc{PID: pid, Name: name, Path: ce.path})
	}
	for pid := range c.cache {
		if !seen[pid] {
			delete(c.cache, pid)
		}
	}
	return out, nil
}

func creationTime(pid int) int64 {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(h)
	var c, x, k, u windows.Filetime
	if windows.GetProcessTimes(h, &c, &x, &k, &u) != nil {
		return 0
	}
	return c.Nanoseconds()
}

func imagePath(pid int) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func withHandle(pid int, access uint32, fn func(windows.Handle) error) error {
	h, err := windows.OpenProcess(access, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil // already gone
		}
		return err
	}
	defer windows.CloseHandle(h)
	return fn(h)
}

func ntCall(proc *windows.LazyProc, pid int) error {
	return withHandle(pid, processSuspendResume, func(h windows.Handle) error {
		if st, _, _ := proc.Call(uintptr(h)); st != 0 {
			return windows.NTStatus(st)
		}
		return nil
	})
}

func (c *osController) Freeze(p Proc) error { return ntCall(ntSuspendProcess, p.PID) }
func (c *osController) Resume(p Proc) error { return ntCall(ntResumeProcess, p.PID) }

func (c *osController) Kill(p Proc) error {
	return withHandle(p.PID, windows.PROCESS_TERMINATE, func(h windows.Handle) error {
		return windows.TerminateProcess(h, 1)
	})
}

// EnumWindows callbacks can't be freed (Windows allows ~1000 per process),
// so there is one, and the target pid is passed through a guarded variable.
var (
	closeMu     sync.Mutex
	closeTarget uint32
	closeCB     = windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		getWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == closeTarget {
			postMessage.Call(hwnd, wmClose, 0, 0)
		}
		return 1
	})
)

// Close resumes the process and posts WM_CLOSE to its windows. Apps without
// windows (or that hide to the tray) are force-closed after close_timeout.
func (c *osController) Close(p Proc) error {
	_ = c.Resume(p)
	closeMu.Lock()
	defer closeMu.Unlock()
	closeTarget = uint32(p.PID)
	enumWindows.Call(closeCB, 0)
	return nil
}
