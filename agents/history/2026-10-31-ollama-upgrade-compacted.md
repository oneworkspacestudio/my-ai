# Owner-driven Ollama upgrade

User reports model requires Ollama0.40.13 while current server is0.40.11 and asks
for My AI to let users upgrade. Started clean at3d53055 (MyAI15) in I:/Projects/my-ai.
Per-command safe.directory is needed on this exFAT source checkout; no global Git
safety changes were made. Current source/public app is15; do not publish changed
15 bytes. Next publication must use a new version and normal app/channel delivery.

Added owner-only GET/POST /api/ollama/upgrade using existing apphost Owner/CSRF/
expected-account checks and explicit confirm:true. Models AUI panel shows runtime
version, background state/download progress and Upgrade Ollama. Members are told
to ask the owner. UI uses existing confirmation and fixed official-installer link.

Upgrade explicitly fetches official stable GitHub metadata, validates exact asset
URL/name/size/SHA256, validates redirect hosts, downloads bounded bytes to private
storage, verifies hash and valid Windows signer (Ollama Inc. observed on existing
signed binary), runs official /SILENT /NORESTART installer, and confirms local
/api/version reaches target or newer. No automatic startup downloads. It refuses
insufficient disk space and concurrent jobs. New local turns/model downloads are
blocked during upgrade; unrelated cloud turns are unaffected. No model paths,
registry environment, security policy or library files are modified by My AI.
Installer temp files are removed on completion/failure. App shutdown cancels work.

Tests pass: complete internal/app suite, exact version-error classification with
prompt redaction, semantic ordering (including CPU suffix), bad metadata/digest/
redirect rejection, explicit confirmation, anonymous route rejection, and mocked
successful/corrupt download pipelines preserving a synthetic model sentinel.
Production frontend generation/TypeScript/Vite build passes. Native Windows candidate build passes. No live Ollama installer, user models or owner account used as
functional fixtures. No release, commit, push or installation performed
by this task. docs/OLLAMA-UPGRADE.md explains scope and limitations.

Before release: update version, rebuild on committed source,
package/publish immutable app release and promote Workspace channel using established
MyAI delivery workflow. Test actual upgrade only when authorized with sufficient
space. Preserve all models/providers/accounts, and never claim both-PC install or
actual installer end-to-end verification from mocked tests. Artifact directory:
I:/Projects/tooling/my-ai-ollama-upgrade-20261031. Temporary Go caches remain on I:.

Release/install explicitly requested. Preparing preview16. Review found that
direct installer execution would couple the shared Ollama process to My AI's
kill-on-close job. Windows now uses the Explorer desktop's ShellDispatch broker
after checksum/publisher verification; no admin or machine-policy changes. A
harmless signed loopback ping probe confirmed explorer.exe, rather than the
observer, owns the brokered process. No Ollama installer was run. Runtime API
version remains the final success criterion. An ordinary Shell.Application
ShellExecute probe stayed parented to the caller and was rejected as a solution.

Preview16 backend suite passes after desktop-broker change. Production-asset UI
acceptance uses synthetic APIs only: owner confirmation, pending button disabled,
progress/completion, member hiding,1440/390 widths/no overflow/runtime errors.
No real installer was run; source is ready for immutable16 publication.
