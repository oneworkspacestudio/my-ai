package app

import (
	"context"
	"golang.org/x/sys/windows"
	"unsafe"
)

func readHostMemory(ctx context.Context) hostMemory {
	var state struct {
		Length, Load                                                                         uint32
		Total, Available, PageTotal, PageAvailable, VirtualTotal, VirtualAvailable, Extended uint64
	}
	state.Length = uint32(unsafe.Sizeof(state))
	ok, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&state)))
	m := hostMemory{}
	if ok != 0 {
		m.RAMTotal, m.RAMAvailable = state.Total, state.Available
	}
	readNvidiaMemory(ctx, &m)
	return m
}
