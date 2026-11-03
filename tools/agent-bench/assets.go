package benchassets

import "embed"

// Files are app-owned synthetic test code, never model-generated executables.
//
//go:embed bench.py summarize.py resources.py safe_bench.py
var Files embed.FS
