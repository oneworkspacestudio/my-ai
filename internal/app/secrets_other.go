//go:build !windows

package app

import "errors"

func protect([]byte) ([]byte, error) {
	return nil, errors.New("secure provider credentials require Windows DPAPI in this build")
}
func unprotect([]byte) ([]byte, error) {
	return nil, errors.New("secure provider credentials require Windows DPAPI in this build")
}

func rejectLink(string) bool { return false }
