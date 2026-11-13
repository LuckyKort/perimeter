package procs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as the "protected app": with PERIMETER_HELPER set it
// appends a line to a file every 20 ms until killed.
func TestMain(m *testing.M) {
	if f := os.Getenv("PERIMETER_HELPER"); f != "" {
		for i := 0; ; i++ {
			fh, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				fmt.Fprintln(fh, i)
				fh.Close()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	os.Exit(m.Run())
}

func lines(t *testing.T, f string) int {
	b, _ := os.ReadFile(f)
	return strings.Count(string(b), "\n")
}

// TestRealProcess freezes, resumes and closes a real process on the current OS.
func TestRealProcess(t *testing.T) {
	out := filepath.Join(t.TempDir(), "ticks")
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "PERIMETER_HELPER="+out)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	defer cmd.Process.Kill()
	time.Sleep(300 * time.Millisecond)

	c := New()
	all, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	var p Proc
	for _, x := range all {
		if x.PID == cmd.Process.Pid {
			p = x
		}
	}
	if p.PID == 0 {
		t.Fatalf("helper pid %d not listed among %d processes", cmd.Process.Pid, len(all))
	}
	self, _ := os.Executable()
	if !strings.EqualFold(p.Name, filepath.Base(self)) {
		t.Errorf("name = %q, want %q", p.Name, filepath.Base(self))
	}
	if p.Path == "" {
		t.Error("path is empty")
	}
	if p.PPID != os.Getpid() {
		t.Errorf("ppid = %d, want %d", p.PPID, os.Getpid())
	}
	if !(Rule{Names: []string{p.Name}}).Match(p) {
		t.Error("rule by name doesn't match the listed process")
	}

	if p.Start == 0 {
		t.Error("start time is unknown")
	}
	// Same PID, other start time: as if the PID went to another process. Hands off.
	stranger := p
	stranger.Start++
	if err := c.Freeze(stranger); err != nil {
		t.Fatalf("freeze stranger: %v", err)
	}
	mark := lines(t, out)
	time.Sleep(300 * time.Millisecond)
	if lines(t, out) == mark {
		c.Resume(p)
		t.Fatal("a process with another start time was frozen")
	}

	if err := c.Freeze(p); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	before := lines(t, out)
	time.Sleep(400 * time.Millisecond)
	if after := lines(t, out); after != before {
		t.Fatalf("frozen process kept running: %d -> %d lines", before, after)
	}

	if err := c.Resume(p); err != nil {
		t.Fatalf("resume: %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	if after := lines(t, out); after <= before {
		t.Fatal("resumed process doesn't run")
	}

	// Freeze again, then close: a frozen process must still go away.
	c.Freeze(p)
	c.Close(p)
	select {
	case <-done:
		return
	case <-time.After(2 * time.Second):
	}
	// Console processes have no window to close on Windows; that's what Kill is for.
	if err := c.Kill(p); err != nil {
		t.Fatalf("kill: %v", err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("process survived Kill")
	}
}
