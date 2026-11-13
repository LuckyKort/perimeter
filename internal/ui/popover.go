package ui

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"perimeter/internal/app"
)

// popover drives `perimeter popover`, the tray window. The process starts with the
// tray and keeps the window loaded but hidden, so a click shows it at once.
type popover struct {
	mu    sync.Mutex
	in    io.WriteCloser
	pid   int
	fails []time.Time
}

// popoverSupported: Linux desktops keep the plain menu.
func popoverSupported() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "windows" }

func (p *popover) start() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "popover")
	cmd.Env = append(os.Environ(), "PERIMETER_CHILD=1")
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.in, p.pid = in, cmd.Process.Pid
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.in == in {
			p.in = nil
			p.fails = append(p.fails, time.Now())
			app.Logf("окно в трее закрылось: %v", err)
		}
	}()
	return nil
}

// broken: the window keeps dying, so the icon falls back to the menu.
func (p *popover) broken() bool {
	n := 0
	for _, t := range p.fails {
		if time.Since(t) < 10*time.Minute {
			n++
		}
	}
	return n >= 3
}

// send passes a command ("toggle", "show", "hide"), starting the window if needed.
// false means there is no window to show, and the caller should show the menu.
func (p *popover) send(cmd string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.in == nil {
		if p.broken() || p.start() != nil {
			return false
		}
	}
	letForeground(p.pid)
	if _, err := io.WriteString(p.in, cmd+"\n"); err != nil {
		p.in = nil
		return false
	}
	return true
}

func (p *popover) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.in != nil {
		p.in.Close() // the window exits when its stdin closes
		p.in = nil
	}
}
