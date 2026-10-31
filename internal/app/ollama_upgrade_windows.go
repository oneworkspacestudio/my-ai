package app

import (
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func ollamaInstallSpace(data string, size int64) error {
	home, e := os.UserHomeDir()
	if e != nil {
		return errors.New("Cannot locate the server user's installation folder")
	}
	for _, p := range []struct {
		path string
		need uint64
	}{{data, uint64(size) + (4 << 30)}, {home, 4 << 30}} {
		s, e := windows.UTF16PtrFromString(p.path)
		if e != nil {
			return errors.New("Cannot check installer storage")
		}
		var free uint64
		if windows.GetDiskFreeSpaceEx(s, &free, nil, nil) != nil || free < p.need {
			return errors.New("Ollama needs at least 4 GB free for installation, plus space for the installer download")
		}
	}
	return nil
}
func installOllama(ctx context.Context, file string) error {
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, shell, "-NoProfile", "-NonInteractive", "-Command", `$ErrorActionPreference='Stop'; $s=Get-AuthenticodeSignature -LiteralPath $env:MYAI_OLLAMA_INSTALLER; @{Status=$s.Status.ToString();Signer=$s.SignerCertificate.GetNameInfo([Security.Cryptography.X509Certificates.X509NameType]::SimpleName,$false)}|ConvertTo-Json -Compress`)
	cmd.Env = append(os.Environ(), "MYAI_OLLAMA_INSTALLER="+file)
	hideCommand(cmd)
	raw, e := cmd.Output()
	var s struct{ Status, Signer string }
	if e != nil || json.Unmarshal(raw, &s) != nil || s.Status != "Valid" || (s.Signer != "Ollama" && s.Signer != "Ollama, Inc." && s.Signer != "Ollama Inc.") {
		return errors.New("Windows could not verify the official Ollama publisher")
	}
	// The desktop broker owns the installer, not My AI's kill-on-close job.
	launchCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd = exec.CommandContext(launchCtx, shell, "-NoProfile", "-NonInteractive", "-Command", `$ErrorActionPreference='Stop';$loc=0;$root=0;$handle=0;$sh=New-Object -ComObject Shell.Application;$desktop=$sh.Windows().FindWindowSW([ref]$loc,[ref]$root,8,[ref]$handle,1);if(!$desktop){throw 'No interactive desktop'};$desktop.Document.Application.ShellExecute($env:MYAI_OLLAMA_INSTALLER,'/SILENT /NORESTART','','open',0)`)
	cmd.Env = append(os.Environ(), "MYAI_OLLAMA_INSTALLER="+file)
	hideCommand(cmd)
	if cmd.Run() != nil {
		return errors.New("The Windows desktop could not launch the Ollama installer; sign in on the server and retry")
	}
	return nil
}
