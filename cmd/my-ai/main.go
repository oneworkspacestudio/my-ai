package main

import (
	"github.com/oneworkspacestudio/my-ai/internal/app"
	"github.com/oneworkspacestudio/my-ai/internal/webui"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"os"
)

var version = "0.1.0-preview.11"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp-serve" {
		if e := app.RunMCP(os.Stdin, os.Stdout); e != nil {
			os.Exit(1)
		}
		return
	}
	if e := apphost.Main(apphost.Config{ID: "myai", Name: "My AI", Version: version, Listen: "127.0.0.1:9019"}, webui.Files(), func(h *apphost.Host) (apphost.Application, error) { return app.New(h) }); e != nil {
		apphost.Report(e)
		os.Exit(1)
	}
}
