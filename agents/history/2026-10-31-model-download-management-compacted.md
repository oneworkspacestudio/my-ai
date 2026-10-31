# Model download management and model-screen layout

## Scope and implementation
User requested a reusable solution for every installation: Cancel on queued/running downloads and Remove on terminal entries, plus a less crowded, professional model screen. Implemented in My AI source on codex/my-ai-tools, based on preview16; no release/version bump, commit, push or install authorized in this task.

DELETE /api/models/downloads/{id} removes only completed/failed/cancelled queue history. Existing owner-only shared-queue permissions, apphost authentication/CSRF/account checks and audit apply. Active worker ownership rejects removal until stopping finishes. No Ollama deletion endpoint or model/partial-file manipulation. Changes persist atomically with in-memory rollback on storage errors. Cancellation now persists before signaling the worker and rolls back on failed storage; errors do not disclose private paths.

AUI remains the UI authority. Models lead the screen; Add a local model, Download history and Server & maintenance are native keyboard-accessible disclosures. Active transfers stay visible; busy server upgrade status stays visible outside maintenance. Terminal entries show status and Retry/Remove without indeterminate progress. Owner controls are hidden from members. Operation feedback is visible inside the models dialog. Generated Workspace.tsx updated by the normal production build.

## Verification and resume
Production generation/TypeScript/Vite build passes (existing >500kB bundle warning remains). Synthetic production-asset browser checks cover owner cancel/remove request IDs and CSRF, collapsed sections, active-only progress, member hiding, desktop/mobile overflow and upgrade confirmation/progress/completion. Screenshot review used synthetic APIs only; screenshots are outside Git in tooling/my-ai-download-management-20261031. Go internal/app suite passes (6.687s final run) and includes persisted terminal removal, active rejection, model sentinel preservation, save-failure rollback, cancellation ordering, late worker callbacks and anonymous mutation rejection.

No owner data/library or second PC used as a fixture. No existing failed entries were locally purged. Shipping these changes requires an explicitly requested new immutable My AI release and supported Workspace public channel/update flow. Do not overwrite preview16 or claim installed PCs from these source tests. Prior delivery procedure is in 2026-10-31-ollama-upgrade-compacted.md.

## Authorized delivery
User requested release and installation. Preparing immutable preview17 and normal public channel promotion; primary PC only, second PC remains pending.
