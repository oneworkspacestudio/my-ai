# My AI architecture decision — 24 October 2026

Choose an original Go backend and React interface compiled from AUI, embedded in MyAI.exe. This is a new application, not an Open WebUI fork. No Open WebUI code/assets are redistributed. Open WebUI v0.12.3, audited main91ea4e3421d1d8069f13d4a463fcb5986d94ea60, uses SvelteKit/Python FastAPI and supports OIDC/provider connections. Its post0.6.6 branding conditions make a broadly redistributed white-label uncertain; the BSD0.6.5 commit07d8460126a686de9a99e2662d06106e22c3f6b6 has maintenance/security tradeoffs. An enterprise licence is not present. The original implementation avoids both the white-label assumption and a bundled Python service.

Sources: https://github.com/open-webui/open-webui ; https://docs.openwebui.com/license/ ; https://docs.openwebui.com/features/authentication-access/auth/sso/ .

Reuse Workspace apphost for MyAccounts PKCE, nonce, issuer/audience/signature checks, stable issuer+subject identity, CSRF, protected NTFS credentials and backups. Extend the existing Quantum private-source pattern used by My Social: a default-deny My AI source outside broad shared Files, with per-identity physical scopes. My AI writes authoritative atomic versioned session files there after a loopback capability guard binds the same OIDC subject in Quantum. No root Drive token reaches the app/browser. Standalone uses its dedicated protected folder. No Google Drive sync. Provider keys use Windows DPAPI, never session files.

Use official Codex app-server JSON-RPC, isolated CODEX_HOME per user and approved project. Never adopt global credentials or configuration. Local Codex0.157.1 detected; runtime is separate, optionally administrator-configured, not silently installed. OpenAI API credentials must be supplied explicitly. DeepSeek chat does not imply Responses/harness compatibility; Code mode is disabled for that provider. Ollama chat uses host-side /api/chat; Code mode remains disabled until that particular model passes a harness compatibility check. Existing Ollama service is reused, not started/reconfigured. Model installs/deletion require owner action and confirmation. Downloads are real /api/pull streams; no catalog benchmark claims.

Sources: https://learn.chatgpt.com/docs/app-server ; https://github.com/openai/codex (Apache2.0); https://docs.ollama.com/api ; https://api-docs.deepseek.com/api/list-models/ .

Default execution is read-only, registered projects only, owner only. Workspace-write requires explicit selection and Codex sandbox support; full access is unavailable. Unknown server requests are denied. No generic remote terminal endpoint. Approval events are tied to an authenticated session. No copied proprietary icons or live desktop inspection: supplied screenshots establish spatial layout. One main EXE embeds web assets; Codex and Ollama remain separate processes. No Docker or manual Node/Python at runtime. Release is an evaluation preview until real-provider and clean-host acceptance gates are demonstrated. Every limitation is tracked in SUPPORT.md.

## October25: managed projects and native file tools

This supersedes the original owner-folder/Codex-only project design above.
Project creation is name-only and creates a private per-identity Projects Drive
source plus a native private Git repository. A loopback capability broker in
Workspace owns Git identity mapping, protected metadata, commits and verified
pushes. Existing native Gitea is unchanged. Exact MyAccounts subject/source
mapping is required; no email account linking or new administrator rights.
Browser/model receive no Git/Drive administrative credential.

Basic project editing uses the existing provider tool loop, scoped to one selected
managed project, with bounded UTF-8 writes and exact edits. Members can work only
on their own managed projects. Git internals/credentials stay outside editable
files; no shell is exposed. Native Ollama editing and actual Git pushes passed.
Completed changes sync automatically; failures preserve files, remote changes
fast-forward only when safe, and conflicting history is never force-pushed.
This removes the command sandbox prerequisite for basic editing; the optional
legacy Codex transport and its separate acceptance limits remain.
