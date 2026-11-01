# Hosting automatic SSO and combined model controls

## Scope, source and continuity
User requests reproduce/fix Hosting connection failures, then release/install.
Standing authorization in this chat is to release and install app changes on this
PC by default; second PC remains pending. Public baseline is My AI17 /3033f3d.
I: filesystem access hangs, including pending earlier version18 source operations.
PowerShell startup also hangs while touching drive providers. Do not reissue the
old queued I: commit/build or overwrite that working tree.

A fresh public clone was created at C:/Projects/.cache/my-ai-20261024/hosting-fix-src,
branch codex/hosting-auto-connect, for isolated work. Because I: is unavailable,
the requested pending model controls were reconstructed from the already-reviewed
feature contract, alongside the Hosting fix. Offline I: work remains untouched.
This checkout is authoritative for the combined release; reconcile I: later by
preserving its work before merging public main. Temporary source, dependency and
compiler artifacts remain under this named C: cache; no owner fixture was used.

## Cause and implementation
Workspace broker requires captured native app cookies; My AI's old API-only warm
probes cannot trigger document-only seamless OIDC. Reproduced production UI cold
session: manual Open My Hosting/Continue, zero SSO starts. New browser helper starts
only the exact same-origin installed app opening flow, verifies readiness through
a registered read operation, then acknowledges only workspace/connect. Actual app
changes retain separate human review. No new session minting, credentials, manager
API, role bypass or arbitrary proxy. Cold connected:false metadata is withheld
from model discovery and instructions discourage conceptual completion/manual
internal-tool instructions. Stop/abort removes frame and blocks stale resumption.
Genuine authentication recovery remains available.

Combined local model controls preserve per-account/exact-tag JSON profiles, context,
output, sampling, seed, batch and memory controls; low-memory preset, Plain Chat,
defaults reset and sticky mobile Save. Plain Chat skips research/native tools and
orphan tool replay, while Code retains project tools. Defaults unchanged; internal
research keeps deterministic sampling and stricter reply caps.

## Verification and delivery
Backend suite passes with mock actual wire requests, profile persistence/isolation/
validation/reset/corruption, Plain Chat isolation, read-only SSO verification and
authentication denial. Production UI SSO fixture passes cold native OAuth-style
redirect/callback/cookie resume, exact single continuation, separate mutation
review, stop cleanup and cross-account denial. Model-settings UI passes save/reset,
zero values, validation/storage errors, role access and mobile/desktop visible Save.
Download and Ollama-upgrade UI regressions pass. No owner account/model used as a
functional fixture; no real low-memory hardware, live owner deployment or arbitrary
IdP frame-policy acceptance is claimed. Pending publication/install outcome follows.
