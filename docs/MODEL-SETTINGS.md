# Local model settings

Use the sliders button beside a local model in Models or the composer.
Context size, maximum reply and temperature are shown first. Advanced settings
include prompt batch size, Top P, Top K, repetition penalty, seed and keep-alive.
Profiles are saved per account and exact model tag, across browsers and restarts.
Changes affect the next request, including Code; running responses keep their snapshot.

Leave a field blank for automatic values. A custom context with automatic reply
length reserves up to a quarter of the window, capped at 1,024 reply tokens.
A custom reply limit must be smaller than the context window.

Low-memory preset fills 2,048 context tokens, 512 reply tokens, temperature 0.2,
batch size 128, Plain chat and unload after the reply. Review and Save to apply.
Plain chat disables tools, web research and app actions in Chat. Code still uses
its required project tools. Choose Automatic tools for Chat app workflows.
Use defaults clears draft overrides; Save persists that reset for this model only.

Context must fit the model and your hardware. Longer conversations, attachments
and app workflows may require more context. No preset guarantees that every model
fits a particular computer. Research controllers keep deterministic sampling and
stricter reply caps; context and memory controls still apply.

Profiles use authenticated API requests and atomic private storage. Numeric limits
are validated, zero sampling values are preserved, at most 128 profiles are stored
per account, and unreadable data is not silently replaced. These are per-request
Ollama preferences; cloud providers and legacy Codex transport are unchanged.
See https://docs.ollama.com/api/chat and https://docs.ollama.com/modelfile.
