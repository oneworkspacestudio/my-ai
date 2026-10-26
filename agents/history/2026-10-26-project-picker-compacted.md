# Code project picker and General fallback

User requests a visible select/create project option in Code, automatic General
when none is selected, and push/release/install on both PCs. Applied repository
history and AUI. Source branch codex/my-ai-tools continues clean preview10
handoff at9b92627. Preserve all app/account/model state and normal protected
update behavior; never use live Accounts/Drive as functional fixtures.

## Implementation

Composer has a project button and New project at phone/desktop widths. Existing
name-only My Drive/private Git setup is reused. Choosing/creating keeps an unsent
draft; changing projects after a conversation starts opens a new conversation.
Server session save assigns General to unselected Code; old empty-project Code
sessions also recover on their next turn with revision/running rechecks.
General uses a stable48hex account-derived ID, broker authorization/storage/Git,
and serialized setup. Existing ready General is read without syncing unfinished
edits from another conversation. Failed setup preserves the draft and sends no
model request. Chat and explicit projects are unchanged; write permissions and
project turn exclusion remain enforced. General is private per account.

## Verification and delivery state

Go all-package tests including real bundled SearXNG/public search and vet pass.
New tests cover concurrent retries, stable IDs/restart, distinct accounts,
no resync on reuse, setup failure, explicit selection and Chat. AUI/tsc/Vite pass.
Scans:0 reachable/imported Go findings,1unused required-module advisory;
production npm0. No dependencies/search runtime changed.

Fresh synthetic Workspace92 + real Accounts/Drive/Git + real qwen3.5:4b Code
request sent with no project selected created General, wrote15,098byte Snake
HTML, completed and pushed. Remote bare Git HEAD and file bytes equal the
registry/Drive. Named project creation and switching retained the unsent draft
at effective434px phone viewport without horizontal overflow. Final UI polish
also updates the picker immediately after session save and fixes focus target;
final candidate reload checks passed. Final real qwen3:0.6b calculation turn
reused the same General after restart, retained Preview only, and returned42;
picker updated immediately after server assignment. Phone screenshot timed out;
desktop capture works. No live owner functional turn/cloud/full Codex shell,
clean-host/service/reboot certification.

Artifacts I:/Projects/tooling/my-ai-project-picker-20261026. Synthetic fixture
root C:/Projects/.cache/my-ai-20261024/p11; program payloads under artifacts;
recorded manager PID in fixture-process.json. Clean these only after final
checks. Test roots are private NTFS, do not delete global model libraries.

Preview11 publication/install pending. Authorized coordination peer confirms
no active apply/promotion, primary manager92/MyAI10/27healthy; secondary last
92/MyAI10/27healthy, Photos6. Suite checkout workspace-my-ai fast-forwarded
to8199f65. Reconcile current dynamic installed versions before guarded normal
MyAI-only applies; use released92 updater, no protection bypass. Promote public
channel after app publication and retain every unrelated release/withholding.
Do not repeat active applies during tunnel downtime. Need clean source freeze,
hash-matched immutable11/public asset verification, channel/pin promotion,
both-PC installs/audit, fixture cleanup and final handoff/docs push.
