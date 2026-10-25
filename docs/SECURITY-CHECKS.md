# Runtime and dependency checks

Preview2 uses Go1.27.3 or newer to fix GO-2026-6809 (CVE-2026-111943), which was detected by the final reachable-symbol scan in the Go1.27.2 preview1 build. Preview1 and Workspace81 remain immutable historical releases and are excluded from the promoted update channel. No owner installation used those builds in this task.

Use govulncheck against ./... from the repository root and npm audit --omit=dev from web before publication. Record reachable findings separately from unused module findings. Runtime keys and user data must never enter scan inputs. Current source/receipts describe the checks actually performed. Full security certification is not claimed.

Reference: https://pkg.go.dev/vuln/GO-2026-6809 .

Go1.27.3 scan: zero reachable vulnerabilities and zero advisories in imported packages; three findings occur only in unused required-module symbols. Production npm audit: zero advisories. Codex is started in ephemeral credential mode, with an isolated home and a child-only provider key; no existing login is adopted.

Preview3 adds bounded read-only tool execution and a bundled public-only MCP server. URL tests reject loopback/private/link-local/metadata/CGNAT destinations, custom ports, credential URLs and non-web schemes; the actual resolved address is checked and pinned at connection time and redirects are revalidated. Identity-scoped conversation reads and cross-user denial tests pass. New Go scan: zero reachable/imported-package advisories; one unused required-module symbol advisory remains. Production npm audit: zero.
