# Local model loading — 25 October

User reports HTTP500 after installing eriktonon/bonsai-2-uncensored:4b; authorizes fix, push, release and installation on both PCs. Delivered. Preserve model libraries, owner identity/settings and other apps; no Actions/subagents or live owner conversation/account/Drive fixtures.

## Diagnosis and change

Secondary API initially0.33.3. Its previous server log confirms the exact15:56 failure: model metadata loader reports `tensor "token_embd.weight" size overflow`, followed by HTTP500/api/chat. This is a local loader failure; the old MyAI credential/quota message was misleading. Model publisher requests0.40.11. Real isolated CPU/native MyAI requests with the same Bonsai4b passed on official0.40.11. They also passed on primary's previous0.40.7: do not claim all older versions fail. No live owner chat was used as a fixture.

Preview6 classifies bounded JSON HTTP/stream failures into fixed safe local runtime/format, memory, missing model, context and busy guidance. Unknown local failures advise update/memory checks and retain the downloaded model, including this tensor-overflow wording. Upstream bodies, prompts and credentials never reach the browser. Cloud errors remain provider-specific. Existing Chat tool fallback/project tool enforcement retained. Models adds strictly validated Ollama version with independent2-second timeout; failure does not hide models. No UI tree, access scope, model-selection or managed-project change. Minimum manager88 retained.

## Release and checks

Immutable v0.1.0-preview.6 source34660842fe85b4fdeeaeff65e3c06842b9fd6c92. Executable SHA25689e3d61cb4b80342f65b92592789d612024e7a9ce3c07efb19ee3783fd92eca0; portable ZIPb9fbb762e0c2edcb436610a5a0f0defe7813bb45d6f74c6c93ea11b1f9ad5c94. All published asset sizes/digests verified; source branch/main/tag pushed. Suite pin/default6 and update index promoted in7ee6290; anonymous channel includes exact6 executable. Concurrent later f85c419 withholds Social9/manager89; retained, never reinstall those candidates. Later history commits do not alter release binaries.

Full Go tests/vet, frontend build/audit0, govulncheck0 reachable/imported (one unused required-module advisory) passed. Regression coverage: HTTP/stream classification/privacy, Chat tool fallback/project refusal, version sanitization. Two real isolated Bonsai native-protocol tests passed on0.40.7 and0.40.11. Final binary version smoke passed. No live cloud, owner chat, clean-host/service/reboot test claimed.

## Installed and verified

Both PCs now run MyAI0.1.0-preview.6 and official Ollama0.40.11; Workspace88 retained. Official Ollama installer and ZIP matched GitHub digests; installer signature valid Ollama Inc. Updated existing vendor installations in place, without moving/redownloading model stores. Primary actual qwen3:0.6b inference passed after upgrade; all18 preexisting model name/digest/size tuples match. Secondary Models shows0.40.11, four installed local models including Bonsai4b and completed original download; existing selected-model preference retained. No independent remote model digest audit claimed.

Primary normal SDK MyAI-only public update completed. Exact6 binary hash installed; normalized config, identity/credentials, all28 apps, every other app contract/package/executable byte retained; manager hash4018bbe82916c2c844056afdbb0455b351f7bba7afe666db66a62767681b46fa unchanged. All27 services healthy. New verified recovery update-20261025T185122-f29aeb821f56 retained; normal cleanup retired one older automatic copy and retained preexisting legacy items with cleanup warning.

Secondary normal selected MyAI-only update completed22m48s, all apps healthy, all9 stages complete. Fresh release check confirms MyAI6 up to date, Workspace88 and all25 other displayed installed-version rows retained (26 total). New verified recovery update-20261025T184525-766dc85bfeeb retained; normal cleanup retired one older automatic copy. No independent remote executable/credential disk audit claimed.

Artifacts outside repos: tooling/my-ai-model-fix-20261025 contains publication verification, primary/secondary install receipts, runtime/library receipts and installed-model-runtime.jpg / secondary-complete.jpg. Both Git worktrees clean after final handoff commits. No required delivery work remains.

## Operational continuity

Old idle updater helper from before88 rejected optional MyAI docs. Close updater, reload and Check updates started the installed88 helper and accepted6; no release mutation or bypass. During secondary downtime the public page retained stale disconnected progress. Read-only non-secret startup-progress.json through existing RDP confirmed ready; a normal GET reload reconnected to active cleanup. Never repeat the install POST. After reload displayed checkbox defaults are not evidence of the originally selected request; installed-version audit confirms only MyAI changed.

Primary C reserve was satisfied with lossless NTFS compression: newest usable recovery SDK-verified before/after and every public Ollama library hash checked before/after. No reserve bypass, user-data deletion, pagefile/security change or removal of newest usable rollback. Normal successful cleanup retained the new recovery copy. Temporary fixture test source and tracked servers removed/stopped; dedicated private NTFS profile removed with path/reparse/process guards. Public model/test artifacts remain onI. Original C checkout unrelated changes untouched. No terminal UI, Windows Run, password manager or RDP PIN automation.
