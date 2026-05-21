//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func setWindowsUserEnvironment(name, value string) error {
	cmd := exec.Command("setx.exe", name, value)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
