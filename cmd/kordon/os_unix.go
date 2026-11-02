//go:build !windows

package main

import "syscall"

func attachConsole() {}

func processAlive(pid int) bool { return pid > 0 && syscall.Kill(pid, 0) == nil }

func stopProcess(pid int) bool { return syscall.Kill(pid, syscall.SIGTERM) == nil }
