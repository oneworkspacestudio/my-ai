# Set up My AI

With One Workspace88 or newer, select Workspace and My AI together. MyAccounts,
My Drive and Git are required. Open My AI from Apps and sign in with MyAccounts.
The app uses your existing local Ollama server, or your own OpenAI/DeepSeek API
keys added through Settings. Cloud prompts leave this machine; the composer
identifies the selected provider.

## Create and work on a project

In Code, the project button above your message lets you select an existing
project. Choose New project to create one by name. Your unsent message stays in
the composer when you select or create a project. Changing the project after a
conversation has started opens a new conversation in that project.

You can also send straight away: when nothing is selected, My AI creates or
reuses your private General project in My Drive / Projects and connects it to
Git. This is one General project per account across conversations, tabs and
restarts. Chat does not create a project. If setup fails, your draft stays in
place and no message is sent to the model.

Open Projects, give the project a name, and choose Create project. My AI creates
its folder in the private Projects area in My Drive, creates a private repository
in your Git app, adds a README and confirms the first push before saying it is
ready. Open in My Drive and View in Git link to those same files and history.
Each person can create their own projects. No host folder or Git command is needed.

Ask My AI to create or change files in the selected project. Changes allowed is
the default for a new project conversation; switch to Preview only to discuss
changes without editing. Project conversations use the same provider/model
catalogue as Chat. The model must support function calls. My AI supplies bounded
file-list/read/write/exact-edit tools, saves the files, commits completed changes,
pushes them to Git and checks the remote commit before showing Saved to Git.
Basic project editing needs no Codex installation or Windows command sandbox.
No shell commands, package installation, test runner or deployment is implied.

If setup or a push is interrupted, the project stays in the list with Finish
setup or Save to Git. Retry uses the same project and repository. A failed push
keeps the files and local history. Changes made in Git fast-forward into the
project before the next turn when safe. Competing changes are refused without
force-pushing or overwriting either side; review them before continuing.
Automatic conflict resolution is unavailable in this preview. Older manually
registered host folders are retained unchanged; new project editing uses the
managed My Drive/Git flow.

Projects is a private source in the My Drive app, physically separate from broad
shared Files. Each MyAccounts issuer+subject receives its own directory and
read-only Drive scope. Edit through My AI; viewing and downloading through Drive
remain available. Working files are under Private/myai-projects/Projects/users/
<identity hash>. Protected Git metadata, project registry and temporary connection
recovery records are outside that source and outside AI file access. Existing
My AI conversation storage stays under its separate private source. Neither
browser nor model receives a Drive administrator token or Git credential.

The manager uses Git's existing protected recovery connection only on loopback.
It validates the native MyAccounts source and exact subject mapping; it never
links accounts by matching email. New Git accounts are ordinary, private users.
Pushes use temporary repository-scope tokens, revoked after use; an interrupted
connection is recovered from a credential-free lease record. Git hooks, global
configuration and credential helpers are disabled for these operations. Credential
files are excluded from automatic staging; no force push is used.

## Models and connections

Settings accepts per-user OpenAI/DeepSeek API keys. Windows DPAPI encrypts them in
protected app data, never Drive or exports. An API key and provider credits are
required for cloud replies. My AI does not reuse a Codex desktop login. Native
Ollama project editing has been demonstrated; cloud adapters have protocol checks,
with live cloud-provider acceptance still requiring configured test keys.

Models shows local models first, with independently expandable/filterable OpenAI
and DeepSeek lists. Install Ollama from its official Windows installer if needed.
My AI reuses http://127.0.0.1:11434 and never starts a second server or changes
OLLAMA_MODELS. Owners confirm a model's licence, disk and memory requirements
before queuing a shared download. Every authenticated user sees the same installed
library and server progress. Downloads persist independently of the browser and
resume after My AI restarts; explicit cancellation remains cancelled. Registry and
connection errors remain visible with Retry. Models are shared per hosting server,
not copied automatically between different machines.

## Chat tools and MCP

Tools & skills lists real clock/timezone, arithmetic, public internet search/page
reading, own conversation/attachment lookup and own project reads. Tool activity
and web sources are saved. /time Europe/London, /calculate (12+3)*4, /search your
query and /read https://example.com work even when a chat model lacks automatic
tool calls. A project model without tools is refused rather than pretending it
edited files. Ask normally: current public questions are searched before the
model answers, even when it cannot call tools. Short follow-ups can reuse the
most recent qualifying public question; private files and attached text are
excluded. Relative dates use your browser timezone. There is no search mode to
choose. An old saved search preference is cleared when the new interface opens.
Say “don't search the web” to disable web tools for this conversation, and ask
it to search again when needed. /search also remains available. A private
SearXNG service is included in MyAI.exe and starts automatically in the background;
there is no URL, Python, Docker or account to configure. Chat and managed Code
projects share this search service. Searches use the typed public question and,
for short follow-ups, the most recent qualifying public question. Attached file
text is never added to the automatic or explicit question.
Automatic routing skips multiline, long, code, credential and obvious private
file/project questions; it is a conservative question heuristic, not a privacy
classifier. Ask My AI not to search for conversations that must not use the web. Search engines can still
be blocked, rate limited or temporarily unavailable. Public
requests disclose their query/URL; private networks, custom ports, credentials
and non-web protocols are rejected at connection and redirect time.

MyAI.exe mcp-serve exposes only get_time, calculate, web_search and read_webpage
through JSON-RPC stdio. It opens no account store or project filesystem. The
app-owned Codex session uses this same SearXNG search tool, with its built-in
search disabled. The private server binds only to a random loopback port,
requires a process secret and ends when My AI exits. SearXNG (AGPL-3.0-or-later),
its patches, build inputs and source are offered through Tools & skills and
the corresponding-source ZIP on the My AI release page. Licences and provenance
are retained inside the runtime. Standalone MCP starts its own private server. The optional
legacy Codex transport uses an isolated per-user home and ephemeral credentials,
never global Codex configuration/login. Actual Codex command execution remains
unverified without explicitly supported official Windows sandbox setup and keys;
this is separate from the native project file-editing flow.

## Storage, recovery and portable use

Sessions have atomic current/previous versions and revision checks. Prompts are
saved before provider dispatch. Stop or connection loss retains an interrupted
result and any project edits; incomplete work is not falsely marked pushed.
JSON/Markdown export, JSON import, pin/rename/archive/restore and private trash
are available. Native Workspace backups retain working files, Git history,
accounts and protected settings together. Do not remove the latest verified
rollback to make room for an update.

The portable EXE embeds browser assets, SearXNG and its private patched Python
3.13.18 runtime; it needs no separately installed Node, Python or Docker.
Standalone loopback Chat has a first-owner setup page and protected NTFS storage;
managed projects require the Workspace/My Drive/Git integration. Do not expose the
standalone listener directly to a network. Workspace supplies its existing HTTPS
proxy and optional startup mechanism. No service or OS elevation is installed
silently. Reconfigure DPAPI-bound provider keys after a move to a different
Windows context.

Build locally with npm --prefix web run build, go test ./... and go vet ./....
The release scripts build locally with GitHub Actions disabled. After packaging,
python scripts/package-managed.py <release-folder> --version <version> verifies
all9 portable members and emits workspace-update.json. Managed preview13 requires
Workspace93 for its native app gateway. See [Workspace app actions](workspace-apps.md).
Publish and promote the raw public update channel before delivery.
Unsigned evaluation build; no clean-host, physical reboot or production
certification is claimed. See SUPPORT.md for the acceptance matrix.
