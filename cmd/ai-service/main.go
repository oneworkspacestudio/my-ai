package main

import (
	"github.com/oneworkspacestudio/my-ai/internal/aiservice"
	"github.com/oneworkspacestudio/my-ai/internal/webui"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"os"
)

var version = "0.1.0-preview.26"

func main() {
	if e := apphost.Main(apphost.Config{ID: "aiservice", Name: "AI Service", Version: version, Listen: "127.0.0.1:9042"}, webui.Files(), func(h *apphost.Host) (apphost.Application, error) { return aiservice.New(h) }); e != nil {
		apphost.Report(e)
		os.Exit(1)
	}
}
