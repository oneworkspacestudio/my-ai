# Release My AI

Follow the central One Workspace app lifecycle and use synthetic private NTFS
fixtures. Preserve owner installations, identities, files, credentials and model
preferences. Build and publish locally with GitHub Actions disabled.

Run the Python benchmark regressions, full Go tests with the bundled search
runtime gate, Go vet/vulnerability checks, production dependency audit and the
AUI/TypeScript/Vite build. Verify native background-job permissions, progress,
stop/unload, restart and evidence using an isolated local model transport. Real
model benchmarking on a requested installation follows managed delivery.

Prepare the portable ZIP and run `scripts/package-managed.py`. Validate the actual
candidate manifest against Workspace's independent update planner **before
publishing**. Use a verified public baseline catalog and the matching Workspace
source, then run there:

```powershell
$env:MYAI_RELEASE_CANDIDATE = 'C:/build/my-ai/candidate'
$env:WORKSPACE_RELEASE_BASELINE = 'C:/build/public-workspace/catalog.json'
go test ./internal/distribution -run '^TestMyAIPrepublicationManifestContract$' -count=1 -v
```

This checks both minimal and full documentation layouts, exact package ownership
and retention of unrelated apps. Existing managers recognize four named My AI
guides plus its compatibility document. Put additional guidance in SUPPORT.md;
new managed paths require an explicitly compatible controller contract. Full
catalog validation and independent app-update validation are separate gates.

Commit complete source, record the exact accepted binary and clean provenance,
then publish a new immutable tag with checked package metadata, licenses and
corresponding SearXNG source. Never replace public assets. Promote the normal
public channel and verify anonymous selection/download. Verify a supported fresh
installation and protected app-only upgrade with identity, unrelated packages,
preferences and recovery retained. Update the ordinary managed-seed builder pin
and verify its complete catalog; legacy pin fields alone do not update a seed.
Any public suite release additionally uses the mandatory signed-Android entrypoint.

Record source, release/channel, synthetic acceptance and each actual installed
target separately. Report only measured target-specific model rankings, with
tested context, resource reserves and guard limits. Keep pending targets explicit.
