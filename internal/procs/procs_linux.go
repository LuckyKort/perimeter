package procs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		p := Proc{PID: pid, Name: name, Path: path}
		if isInterpreter(name) {
			raw, _ := os.ReadFile(dir + "/cmdline")
			p.Args = strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
		}
		out = append(out, p)
	}
	return out, nil
}
