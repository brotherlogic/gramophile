//go:build windows

package main

import (
	"os"
	"os/exec"
)

func restartBinary(targetPath string) error {
	cmd := exec.Command(targetPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
