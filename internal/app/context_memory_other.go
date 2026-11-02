//go:build !windows

package app

import (
	"context"
	"os"
	"strconv"
	"strings"
)

func readHostMemory(ctx context.Context) hostMemory {
	m := hostMemory{}
	raw, _ := os.ReadFile("/proc/meminfo")
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		n, _ := strconv.ParseUint(f[1], 10, 64)
		if f[0] == "MemTotal:" {
			m.RAMTotal = n * 1024
		}
		if f[0] == "MemAvailable:" {
			m.RAMAvailable = n * 1024
		}
	}
	readNvidiaMemory(ctx, &m)
	return m
}
