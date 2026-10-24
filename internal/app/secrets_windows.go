package app

import (
	"errors"
	"golang.org/x/sys/windows"
	"unsafe"
)

func protect(b []byte) ([]byte, error) {
	var out windows.DataBlob
	in := windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
	if e := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); e != nil {
		return nil, e
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
func unprotect(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("empty credential store")
	}
	var out windows.DataBlob
	in := windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
	if e := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); e != nil {
		return nil, e
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func rejectLink(p string) bool {
	u, e := windows.UTF16PtrFromString(p)
	if e != nil {
		return true
	}
	v, e := windows.GetFileAttributes(u)
	return e == nil && v&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
