//go:build !windows

package searchruntime

import "os/exec"

func hide(c *exec.Cmd) {}
func stop(c *exec.Cmd) {
	if c.Process != nil {
		_ = c.Process.Kill()
	}
}
func linked(path string) bool { return false }
