# Upgrade Ollama from My AI

The server owner can open **Models → Ollama server upgrade → Upgrade Ollama**.
Confirm that local AI requests may be interrupted. The operation continues on
this server if you close the browser; reopen Models to see status.

My AI checks the latest stable official Ollama release, downloads its Windows
installer into private temporary storage, verifies its published SHA256 and
Windows publisher signature, and runs the normal installer. It does not move or
delete the model library or change OLLAMA_MODELS. It only reports success after
the local Ollama API reports the target version or newer. Finish active My AI
requests and queued model downloads first. New local requests/downloads wait
until the upgrade finishes; cloud requests remain available.

Insufficient storage, blocked signatures, failed downloads and installer failures
produce actionable status. The official Windows installer link remains available
for installations requiring administrator-managed service maintenance. Existing
custom/system services may need their operator to restart/update their own runtime.
Automatic installation is Windows-only and requires enough installation/download
space. My AI does not weaken execution policy, permissions or antivirus settings.

Official references:
- https://github.com/ollama/ollama/releases/tag/v0.40.13
- https://github.com/ollama/ollama/blob/v0.40.13/docs/windows.mdx

This implementation has local backend/frontend tests; a real installer upgrade
against an owner installation was not used as a test. Release and managed app
installation are separate delivery steps. Once Windows launches the installer, it
can finish independently if My AI closes; reopen Models to check the server version.
