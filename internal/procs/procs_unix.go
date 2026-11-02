//go:build darwin || linux

package procs

import (
	"errors"
	"syscall"
)

func signal(pid int, sig syscall.Signal) error {
	err := syscall.Kill(pid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil // already gone
	}
	return err
}

func (c *osController) Freeze(p Proc) error { return signal(p.PID, syscall.SIGSTOP) }
func (c *osController) Resume(p Proc) error { return signal(p.PID, syscall.SIGCONT) }
func (c *osController) Kill(p Proc) error   { return signal(p.PID, syscall.SIGKILL) }

// Close sends SIGTERM and then SIGCONT: a stopped process only handles
// the pending SIGTERM once it runs again, and quits instead of carrying on.
func (c *osController) Close(p Proc) error {
	if err := signal(p.PID, syscall.SIGTERM); err != nil {
		return err
	}
	return signal(p.PID, syscall.SIGCONT)
}
