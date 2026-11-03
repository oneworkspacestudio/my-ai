# Chat formatting and small-model workflows

Messages support headings, lists, task lists, quotes, tables, links, inline code,
syntax-highlighted code blocks and LaTeX maths (`$...$`, `$$...$$`, `\(...\)`,
`\[...\]`). Each saved message has Copy text, Copy Markdown and Download.
Code blocks have their own Copy, Download and Wrap controls. Downloads retain
UTF-8 source bytes; the operating system may normalize clipboard line endings.
Untrusted HTML is displayed as text, never executed. Images are explicit links
so merely opening a conversation does not fetch a remote tracking image.

Edit in new branch restores the selected prompt and its attachments for editing.
Branch here copies the conversation through that response and paired tool results.
The original is preserved. Branching does not roll back project files or app
actions and does not rerun a request automatically. Imported transcripts cannot
become trusted tool receipts. Desktop Enter sends; Shift+Enter adds a line.
Scrolling up pauses automatic following; Jump to latest resumes it.

## Small local models

Bundled workflows cover research, project editing, attachments, conversations,
Workspace app discovery and verified deployment. Local models receive a compact
catalog, selected-project guidance and concise tool descriptions. Additional
readers appear after `load_skill`; attachment readers are available immediately
when the conversation has files. Native app schemas load through discovery.
Skills never grant permissions, access arbitrary skill paths or execute scripts.

Attachments stay stored files. Large tool responses become bounded references
with exact private receipt IDs; `read_tool_result` retrieves Unicode character
pages. Bounded file reads and native action schemas stay exact. Complete earlier
tool exchanges can be condensed when a local context fills, omitting repeated
source-code arguments while retaining real results. The original transcript is
unchanged. User instructions, system instructions, tool schemas and unresolved
calls are not silently dropped. This is deterministic receipt compaction, not a
model-generated summary of the whole conversation. A conversation containing too
much user text still reports its context limit explicitly.

Automatic reply space may shrink to fit the selected context, with a 512-token
floor. An explicit reply setting is preserved. Existing model/hardware estimates
and separate Chat/Code context settings remain in force. Memory estimates and
shorter schemas do not guarantee a model's reasoning or successful execution.

The managed Git/Hosting workflow uses My AI's provider engine. The separate
official Codex app-server adapter remains available in the source; this change
does not replace the local model with Codex or claim full ChatGPT feature parity.
The design follows documented [skill disclosure](https://developers.openai.com/plugins/concepts/skills)
and [Codex app-server](https://learn.chatgpt.com/docs/app-server) concepts while
retaining Workspace identity, app reviews and private-storage boundaries.

## Validation

Run `npm ci`, `npm run build` and `node tests/chat-ui.mjs` from `web` (Chrome
required for the isolated browser test). Set `MYAI_UI_EVIDENCE` to a temporary
output folder to save screenshots. The fixture contains only synthetic content.
Run the Go suite and vet from the repository root. The tests check real provider
request construction at an 8K local context, exact paged receipt retrieval,
private-account boundaries, paired calls, branches, and preserved manual limits.
They do not certify real-model task accuracy or cloud-provider acceptance.
