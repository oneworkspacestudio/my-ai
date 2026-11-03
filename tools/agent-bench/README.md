# My AI task benchmark

Python 3 standard library only for inference and grading. Uses a local Ollama
HTTP API. No cloud keys, real app mutations, shell execution or owner files.
Every sample gets fresh synthetic in-memory files and a simulated Git/Hosting
state machine. App-owned native deployment in My AI20 is intentionally a
separate integration layer and is not scored by this model benchmark.

```powershell
python -m unittest -v test_bench
python pull.py qwen3.5:9b ministral-3:3b granite4:3b
python bench.py --out results --models qwen3.5:4b qwen3.5:9b ministral-3:3b granite4:3b
python summarize.py results
python verify_results.py results
```

Resume with the same command: samples are keyed by model/case/seed. Use a new
output directory if model bytes, benchmark code, budgets, prompt or settings
change. Never merge unlike configurations. `run.json` records exact digests,
size/quantization, inference version, parameters and system-independent hardware.
`samples.jsonl` preserves full synthetic messages and inference metrics. Keep
unavailable models explicit. `--cases`/`--seeds` allow focused diagnostics, not
comparison with the full run unless marked separately.

Default: 12 cases, seeds17/29, temperature0.2, thinkingfalse, context8192,
768 output tokens/round,16 rounds,180s total/sample. Normalized direct tool
schemas and discovered generic native-shaped app actions are the same for all
candidates. This is a small workflow evaluation, not a universal leaderboard.

Success is deterministic resulting state plus observed receipts. The failure
case also checks the final explanation for a failure indication. Safety counts
attempted unauthorized mutation even when the simulator blocks it. No partial
credit for statements alone. The JSON/heading edits are small code-adjacent
tasks; arbitrary game correctness/visual design/general shell are not evaluated.

The AUI report uses pinned My AI web dependencies (`@codedia/parser0.2.1`,
React19.3.1,Vite8.3.4). Link or install those into `report/node_modules`, then:

```powershell
python package_report.py results published
```

Only synthetic data, measured metadata and benchmark source enter the published
directory. No accounts, cookies, API keys, native app private paths or owner
conversations are used. The final directory can be committed to My Git and
deployed as a native static site through My Hosting. This tool is not a new
managed Workspace app and does not change My AI preferences or executable.

The benchmark measures the model plus its installed Ollama parser. The time
tool is a fixed run-start UTC snapshot for replay, and all other tools are
simulated. `verify_results.py` replays every tool receipt and checks prompts,
grades, unique sample identities and complete per-model runs against source.
The packager requires this verification before producing publication files.
