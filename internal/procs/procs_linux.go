package procs

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

type osController struct{}

func New() Controller { return &osController{} }

func (c *osController) List() ([]Proc, error) {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	out := make([]Proc, 0, len(ents))
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		dir := "/proc/" + e.Name()
		path, _ := os.Readlink(dir + "/exe")
		path = strings.TrimSuffix(path, " (deleted)")
		name := filepath.Base(path)
		if path == "" {
			comm, _ := os.ReadFile(dir + "/comm")
			name = strings.TrimSpace(string(comm))
		}
		ppid, start := statFields(dir)
		p := Proc{PID: pid, PPID: ppid, Start: start, Name: name, Path: path}
		if isInterpreter(name) {
			raw, _ := os.ReadFile(dir + "/cmdline")
			p.Args = strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
		}
		out = append(out, p)
	}
	return out, nil
}

// statFields reads the parent PID (4th field) and the start time in clock ticks
// (22nd) from /proc/<pid>/stat; the 2nd (comm) may contain spaces and ")".
func statFields(dir string) (ppid int, start int64) {
	b, err := os.ReadFile(dir + "/stat")
	if err != nil {
		return 0, 0
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0, 0
	}
	f := strings.Fields(s[i+1:]) // f[0] is field 3
	if len(f) < 20 {
		return 0, 0
	}
	ppid, _ = strconv.Atoi(f[1])
	start, _ = strconv.ParseInt(f[19], 10, 64)
	return ppid, start
}

// send signals p unless its PID now belongs to another process. A pidfd pins
// the process it was opened for: if the start time read after opening it
// matches, the signal reaches exactly that process or nobody.
func send(p Proc, sig syscall.Signal) error {
	same := func() bool {
		if p.Start == 0 {
			return true
		}
		_, start := statFields("/proc/" + strconv.Itoa(p.PID))
		return start == p.Start
	}
	fd, err := unix.PidfdOpen(p.PID, 0)
	if err != nil {
		if errors.Is(err, unix.ESRCH) {
			return nil
		}
		// Kernels before 5.3 have no pidfd: check just before the signal.
		if !same() {
			return nil
		}
		return signal(p.PID, sig)
	}
	defer unix.Close(fd)
	if !same() {
		return nil
	}
	if err := unix.PidfdSendSignal(fd, sig, nil, 0); err != nil && !errors.Is(err, unix.ESRCH) {
		return err
	}
	return nil
}
