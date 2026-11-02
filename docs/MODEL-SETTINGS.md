# Local model settings and files

Use Model settings beside a local model in Models or the composer. Profiles are
private to the signed-in account and exact model tag, across browsers and restarts.
Changes affect the next turn; running turns retain their settings snapshot.

## Automatic and custom context

Leave fields blank for automatic sizing. Chat targets32,768 tokens; Code65,536.
Effective values are limited by the native model window and a conservative memory
estimate. The dialog shows model maximum, automatic and saved effective budgets.
The host probes Ollama metadata, weights, loaded allocations, system RAM and,
when an existing NVIDIA vendor utility is available, one GPU's VRAM. It reserves
host memory and estimates fp16 KV-cache costs. Unknown architectures use a
conservative allowance. Missing model limits fall back to8,192 tokens; missing
memory data also limits automatic sizing conservatively. AMD/Intel VRAM is not
probed. Concurrent activity, offloading and architecture details affect memory;
these estimates do not guarantee a model will fit.

Context size overrides Chat and Code for older-profile compatibility. Code context
size overrides Code separately. Valid saved sizes are512..1,048,576 tokens;
execution caps them to native model limits without rewriting the saved profile.
Manual values can exceed the memory estimate, with a notice. Replies occupy part
of the window: automatic Chat replies up to4,096 tokens, Code up to8,192, research
has tighter caps. Custom reply limits must be below both explicit context sizes.

A conservative byte-based upper bound checks local prompt, schemas and reply
reserve before sending. Oversize prompts fail explicitly instead of relying on
silent provider truncation. Some tokenizer-compatible prompts may be rejected;
this is not a tokenizer-exact count. Conversation compaction is not added here.

Advanced controls include batch, sampling, seed and keep-alive. Low-memory preset
selects2,048 context,512 reply, temperature0.2, batch128, Plain Chat and unload
after reply; Review and Save applies it. Plain Chat disables Chat tools/research;
Code retains project tools. Use defaults clears draft overrides; Save persists
that reset. Cloud context and legacy official Codex transport are unchanged.
These controls apply to local Ollama Chat and native Code.

## Attachments stay private files

Uploads remain UTF-8 text up to1MiB. The original bytes and SHA-256 metadata stay
in account-private storage. Model messages contain filename, size and file ID,
not the whole body. File names/content are untrusted reference. Cards download
the original with its filename.

The model can read bounded Unicode ranges (2,048 characters default,4,096 max),
search literal text for bounded excerpts, or copy the file exactly into a selected
writable project. Exact copying requires no file body generation. Project reads
also paginate; targeted edits support copied files up to1MiB. Whole-file generated
writes retain256KiB limits. Account, path, digest and project permissions remain
enforced; native Hosting review and deployment permissions still apply.

An explicit attached build/edit/host request in Chat prepares a managed General
project and enters native Code. Existing read-only Code is respected. Plain Chat
requires automatic tools for these actions. Private attachment conversations
suppress public web research. Imports strip attachment references. Legacy inline
dumps remain saved but are omitted from model replay; reattach the original for
file tools. This is literal search, not semantic/vector retrieval. Binary, PDF and
image parsing are not added by this change.

## Documented approaches elsewhere

Direct file inputs can consume context. OpenAI documents both
[direct inputs](https://developers.openai.com/api/docs/guides/file-inputs) and
[File Search](https://developers.openai.com/api/docs/guides/tools-file-search),
which retrieves relevant material through tools. This does not establish a single
universal internal ChatGPT upload pipeline. Anthropic's
[Files API](https://platform.claude.com/docs/en/build-with-claude/files) separates
storage/reference from input-token/context accounting. Google's
[File Search](https://ai.google.dev/gemini-api/docs/file-search) indexes and
retrieves relevant data. A stored file ID does not make reading context-free.

Ollama documents [context/memory tradeoffs](https://docs.ollama.com/context-length)
and recommends64K for coding/agents when hardware permits. Probes use documented
[show](https://docs.ollama.com/api-reference/show-model-details) and
[ps](https://docs.ollama.com/api/ps) endpoints.
