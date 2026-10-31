//go:build !windows

package app

import (
	"context"
	"errors"
)

func ollamaInstallSpace(string, int64) error {
	return errors.New("Automatic Ollama upgrade is available on Windows; update Ollama on its host")
}
func installOllama(context.Context, string) error {
	return errors.New("Automatic Ollama upgrade is available on Windows")
}
