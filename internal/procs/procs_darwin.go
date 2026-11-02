package procs

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

type cached struct {
	start      int64
	path, args string
}

type osController struct {
	mu    sync.Mutex
	cache map[int]cached // pid -> exec path; keyed with start time so a reused pid is re-read
}

func New() Controller { return &osController{cache: map[int]cached{}} }

func (c *osController) List() ([]Proc, error) {
	kps, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := make(map[int]bool, len(kps))
	out := make([]Proc, 0, len(kps))
	for i := range kps {
		kp := &kps[i]
		pid := int(kp.Proc.P_pid)
		if pid <= 0 {
			continue
		}
		seen[pid] = true
		start := kp.Proc.P_starttime.Sec*1_000_000 + int64(kp.Proc.P_starttime.Usec)
		ce, ok := c.cache[pid]
		if !ok || ce.start != start {
			// procargs2 is readable for our own user's processes, which is what we protect.
			path, args := procArgs(pid)
			ce = cached{start: start, path: path, args: args}
			c.cache[pid] = ce
		}
		name := filepath.Base(ce.path)
		if ce.path == "" {
			name = unix.ByteSliceToString(kp.Proc.P_comm[:])
		}
		p := Proc{PID: pid, Name: name, Path: ce.path}
		if isInterpreter(name) {
			p.Args = ce.args
		}
		out = append(out, p)
	}
	for pid := range c.cache {
		if !seen[pid] {
			delete(c.cache, pid)
		}
	}
	return out, nil
}

// procArgs reads kern.procargs2: argc (int32), exec path, NUL padding, argv..., env...
func procArgs(pid int) (path, args string) {
	buf, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(buf) < 4 {
		return "", ""
	}
	argc := int(binary.LittleEndian.Uint32(buf[:4]))
	rest := buf[4:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return "", ""
	}
	path = string(rest[:i])
	rest = rest[i:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	var argv []string
	for n := 0; n < argc && len(rest) > 0; n++ {
		j := bytes.IndexByte(rest, 0)
		if j < 0 {
			j = len(rest)
		}
		argv = append(argv, string(rest[:j]))
		if j == len(rest) {
			break
		}
		rest = rest[j+1:]
	}
	return path, joinArgs(argv)
}

func joinArgs(argv []string) string {
	var b bytes.Buffer
	for i, a := range argv {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(a)
	}
	return b.String()
}
