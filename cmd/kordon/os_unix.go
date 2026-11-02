//go:build !windows

package main

import "syscall"

func attachConsole() {}

func processAlive(pid int) bool { return pid > 0 && syscall.Kill(pid, 0) == nil }
