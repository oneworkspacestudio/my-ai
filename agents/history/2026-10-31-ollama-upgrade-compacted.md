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

## Published and installed outcome

Immutable v0.1.0-preview.16 source6f2d80b3d07c2030fd226cb3b6814e1f5a8e5422 is public.
All9 asset sizes/digests and anonymous checksum download were verified. Clean
source Windows build SHA256 is2c608227937d8bfd098b4e9a8774a8ff83965c900e6338b25e8aec539ba3b1ad.
Production UI uses synthetic APIs; no real installer/model inference fixture.
Normal Workspace builder pin/public channel76487b9 promotes16 and is anonymously
verified. No suite release, Actions or unrelated app update was performed.

Primary protected MyAI15-to16 update completed on Workspace109. Latest verified
recovery remains; normal cleanup removed one older automatic copy.34 unrelated
app contracts, locations, identity, Workspace credential contents and per-user
provider credential contents passed comparison (decrypted only in memory). The
new native bootstrap reports16; served JS/CSS match the release build and the
owner upgrade status endpoint denies anonymous requests. All34 services healthy.

Initial final-verifier wrongly required every optional document from the release
metadata to exist in this older installed catalog. The normal updater intentionally
omits its5 untracked compatibility/guide entries. A separate read-only audit
verified all4 registered MyAI payloads against public hashes and compared the
retained app contracts with the verified recovery config. Do not repeat the update
or patch these docs into installed metadata. Evidence:docs/ollama-upgrade16-delivery.json.

The MyAI repo's local branch is codex/my-ai-tools; publication used HEAD:main, not
a nonexistent local main. Keep safe.directory scoped to each Git command on I:.
Native/public source archives and notices are in tooling/release16. Ollama itself
was not upgraded; owner must choose Models -> Upgrade Ollama with sufficient space.
The desktop broker launch remains independent of the MyAI job; no live installer
end-to-end, physical-phone or second-PC installation is claimed. No pending work
for primary app delivery. A secondary update needs a separate authenticated target.
