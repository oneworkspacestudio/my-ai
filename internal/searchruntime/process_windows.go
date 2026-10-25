package searchruntime

import (
	"os/exec"
	"syscall"
)

func hide(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
func stop(c *exec.Cmd) {
	if c.Process != nil {
		_ = c.Process.Kill()
	}
}
func linked(path string) bool {
	p, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return true
	}
	a, e := syscall.GetFileAttributes(p)
	return e != nil || a&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
