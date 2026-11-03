# Find a model for this machine

Open **My AI → Models → Test this PC**. The server owner starts or stops a test;
everyone signed in can see the last result. Download candidate models in Models
first. The test measures the machine hosting My AI, including its current
Workspace apps, rather than the phone or browser used to open it.

Leave Workspace and its normal apps running. Finish downloads, chats, backups
and updates first. The full test runs 12 synthetic file-editing, tool-use and
Git/Hosting workflows twice for each installed candidate, one model at a time.
The optional six-task scan is preliminary. Default selection includes up to eight
installed models, each at most 12 GiB; advanced users can name exact installed
tags. No default model or preference is changed automatically.

Testing continues on the server if you navigate away. Return to Models for live
progress and the result, or download its evidence ZIP. A server restart marks an
unfinished test interrupted; it does not automatically resume a load test.
Starting normal local My AI work or model maintenance stops the test first.

## Keeping room for Workspace

Before loading each model, the advisor measures current available memory and CPU,
optional NVIDIA VRAM and utilisation, and the local Workspace manager's health.
It reserves at least 4 GiB or 25% of system RAM, whichever is larger, plus at
least 1 GiB or 20% of each measured GPU's VRAM. An optional larger RAM reserve
can increase this margin. Existing app usage is already subtracted from the
available memory; the reserve is additional room for Windows and app activity.

Model bytes plus conservative overhead must fit beyond that reserve. A busy CPU,
unhealthy Workspace, or missing RAM/CPU telemetry prevents loading. Unknown GPU
telemetry or multiple GPUs uses a CPU-only test, subject to sufficient RAM. The
profile uses 8,192 context tokens, batch 32, at most four CPU threads (and at most
half the available logical CPUs), with short quiet intervals between rounds.

During inference, sampled monitoring stops testing if RAM/VRAM falls below the
reserve, telemetry disappears, Workspace fails or becomes slow, or CPU pressure
persists. The benchmark worker has lower process priority and its own time limit.
Only the test's inference is stopped and its model is asked to unload.

These are estimates and sampled protection, **not an operating-system memory or
GPU quota**. A brief resource spike can occur between samples. Run under a
representative normal workload and rerun after significant app, hardware, model
or context changes. Larger coding context windows and concurrent users need
additional memory and are not certified by an 8K test. The reported recommendation
applies to this tested profile, not every possible future server load.

## Understanding the result

A recommendation needs a complete run within the observed resource limits, at
least 90% task success and deployment success, and every guardrail check passed.
The UI reports minimum free RAM/VRAM, peak sampled CPU and slowest Workspace health
response. An incomplete, stopped or skipped model cannot qualify. No candidate
may qualify; that is a useful result rather than a reason to recommend an unsafe
or unreliable model.

Inference uses real installed model bytes. Tool actions use fresh synthetic
in-memory files and simulated Git/Hosting receipts. They do not deploy user apps,
edit personal files, execute arbitrary shell commands or use cloud credentials.
This small benchmark does not measure visual quality or arbitrary programming.
Model parser, quantization, prompt and hardware differences affect the ranking.

## Standalone runner

The benchmark source in `tools/agent-bench` also runs with Python 3.10+ and local
Ollama on Windows, Linux and macOS, without third-party Python dependencies:

```powershell
python safe_bench.py --out machine-test-001 --models qwen3.5:4b qwen3.5:9b
```

Use a fresh output directory each time. Optional `--health` accepts the local
Workspace `/healthz` URL, `--reserve-gb` increases the RAM reserve, and `--quick`
runs the preliminary scan. Endpoints must be local HTTP. GPU telemetry currently
supports NVIDIA; unsupported GPUs fall back to CPU-only. The integrated app
ships its verified Python runtime on supported Windows machines; users do not
install Python separately. Tests export only synthetic evidence and resource
metadata, kept in My AI's private app data.
