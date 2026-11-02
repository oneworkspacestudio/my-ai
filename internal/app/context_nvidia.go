package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Optional read-only vendor utility. Never download tools or execute model text.
func readNvidiaMemory(ctx context.Context, m *hostMemory) {
	paths := []string{"/usr/bin/nvidia-smi", "/usr/local/bin/nvidia-smi"}
	if runtime.GOOS == "windows" {
		paths = []string{filepath.Join(os.Getenv("SystemRoot"), "System32", "nvidia-smi.exe"), filepath.Join(os.Getenv("ProgramFiles"), "NVIDIA Corporation", "NVSMI", "nvidia-smi.exe")}
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			continue
		}
		if _, e := os.Stat(path); e != nil {
			continue
		}
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		cmd := exec.CommandContext(probe, path, "--query-gpu=memory.total,memory.free", "--format=csv,noheader,nounits")
		hideCommand(cmd)
		out, e := cmd.Output()
		cancel()
		if e != nil || len(out) > 8192 {
			return
		}
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Split(line, ",")
			if len(f) != 2 {
				continue
			}
			total, e1 := strconv.ParseUint(strings.TrimSpace(f[0]), 10, 64)
			free, e2 := strconv.ParseUint(strings.TrimSpace(f[1]), 10, 64)
			if e1 == nil && e2 == nil && free <= total && total < 1<<30 && (m.GPUTotal == 0 || free*1048576 > m.GPUFree) {
				m.GPUTotal, m.GPUFree = total*1048576, free*1048576
			}
		}
		return
	}
}
