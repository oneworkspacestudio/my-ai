package app

import (
	"os/exec"
	"strconv"
	"syscall"
)

func hideCommand(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
func killCommand(c *exec.Cmd) {
	k := exec.Command("taskkill.exe", "/PID", strconv.Itoa(c.Process.Pid), "/T", "/F")
	hideCommand(k)
	_ = k.Run()
	_ = c.Process.Kill()
}
