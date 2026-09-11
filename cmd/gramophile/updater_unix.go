//go:build !windows

package main

import (
	"os"
	"syscall"
)

func restartBinary(targetPath string) error {
	return syscall.Exec(targetPath, os.Args, os.Environ())
}
