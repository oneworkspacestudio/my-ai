//go:build !windows

package app

import "os/exec"

func hideCommand(c *exec.Cmd) {}
func killCommand(c *exec.Cmd) { _ = c.Process.Kill() }
