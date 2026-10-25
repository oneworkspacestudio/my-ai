# Runtime and dependency checks

Preview2 uses Go1.27.3 or newer to fix GO-2026-6809 (CVE-2026-111943), which was detected by the final reachable-symbol scan in the Go1.27.2 preview1 build. Preview1 and Workspace81 remain immutable historical releases and are excluded from the promoted update channel. No owner installation used those builds in this task.

Use govulncheck against ./... from the repository root and npm audit --omit=dev from web before publication. Record reachable findings separately from unused module findings. Runtime keys and user data must never enter scan inputs. Current source/receipts describe the checks actually performed. Full security certification is not claimed.

Reference: https://pkg.go.dev/vuln/GO-2026-6809 .

Go1.27.3 scan: zero reachable vulnerabilities and zero advisories in imported packages; three findings occur only in unused required-module symbols. Production npm audit: zero advisories. Codex is started in ephemeral credential mode, with an isolated home and a child-only provider key; no existing login is adopted.

Preview3 adds bounded read-only tool execution and a bundled public-only MCP server. URL tests reject loopback/private/link-local/metadata/CGNAT destinations, custom ports, credential URLs and non-web schemes; the actual resolved address is checked and pinned at connection time and redirects are revalidated. Identity-scoped conversation reads and cross-user denial tests pass. New Go scan: zero reachable/imported-package advisories; one unused required-module symbol advisory remains. Production npm audit: zero.

Preview5 adds managed file editing and the Workspace Git broker. Unit/native
checks cover project-root and linked-path rejection, Windows absolute/device
paths, selected-project write permissions, hidden host metadata, member creation
and cross-account app/Git/physical Drive denial. Native tests verify private
repository ownership, no-force conflict preservation, credential staging
exclusions, temporary token revocation and interruption recovery that retains
unrelated user tokens. Provider tool failures never fall back to pretending
project edits occurred. Repeat dependency scans on final release sources.


Preview6 adds regression checks for model-format/memory/context/not-found errors and streamed failures without exposing upstream prompts, keys, paths or HTML. Tool compatibility fallback remains Chat-only; managed projects still refuse models lacking file-tool support. Runtime/version metadata is length/format bounded.

Preview7 replaces Bing RSS with a bundled private SearXNG process. Runtime extraction validates trusted embedded hashes, file paths, reparse points and unexpected files before execution. Only authenticated loopback GET search/health is exposed. Child environments exclude app credentials; parent exit ends the server. Search results are bounded, sanitized public URLs and treated as untrusted reference. Explicit searches use typed text before attachment expansion. The interpreter is patched CPython3.13.18, built locally from verified official source;32 pinned Python wheels pass pip-audit without known advisories.
