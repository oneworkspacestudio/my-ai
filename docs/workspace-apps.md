# Use your Workspace apps with My AI

Ask My AI about the apps in your Workspace, then ask it to do something with
them. It receives the installed app list automatically in Chat and Code.
References to installed app names load their supported actions automatically;
other apps remain discoverable as the task develops. Requires My AI12 and Workspace93.

For example:

- “Find my unfinished tasks for this week.”
- “Save these notes in My Drive.”
- “Show my project's Git status.”
- “Deploy this saved project with My Hosting and give me its working link.”

Open **Tools & skills** to see your apps and their available connections. If an
app needs connecting, follow its **Open app** link and sign in normally. My AI
uses your account and the app's existing permissions. It cannot grant itself
access to another person's files or bypass an app's locked account.

My AI shows the proposed app change for review before submitting it. Declining
or stopping that review submits nothing. A Code conversation set to Preview
only cannot change another app. Read operations do not need a review.

## Git, storage and deployment

Projects remain in your private Projects folder in My Drive and their linked
private Git repositories. My AI can inspect history and status and save changes
using that Git app. It can also list accessible Drive sources, read text files
and save text documents through Drive's own permissions.

Deployment sends the exact saved and pushed Git revision to an existing My
Hosting site. My AI can create a static site when your Hosting role permits it.
It checks the real deployment job for its final result and URL. An accepted or
queued upload does not mean the site has finished deploying. If an upload's
response is lost, the connector checks the saved receipt and Hosting history
before starting another job.

This source handoff accepts regular files up to 4 MiB each and 32 MiB in total,
with at most 2,000 files. It excludes credentials, Git internals, linked files
and submodules. Executable source approval remains in My Hosting.

## Other apps

The installed app catalogue lists the exact operations supported by each app.
Connectors include Tasks, Notes, Calendar, Contacts, Mail reads, Photos, Music,
Tube, Cinema, Maps, Meet, Create, 3D, PDF, Social, Clips, Slicer reads, Monitor
for the owner and Search preferences. The native apps enforce their normal
sharing, roles, record versions and account limits.

An app with no registered AI operations can still be discovered and opened.
That does not grant access to all of its features. Unlocking Vault, account
administration, infrastructure changes and printer operation remain in their
own app interfaces. Mail's connector reads messages; it does not send mail.
Automatic calls require a model that supports tools.

Private app results, project files and conversation reads stay out of public web
tools, including on later turns. Separate public research uses the filtered
public question and its public evidence. The public stdio MCP server grants no
access to private Workspace apps. This feature uses My AI's native Chat and
Code tools; it does not certify the separate full Codex sandbox integration.

## Implementation and verification

The manager exposes a loopback-only registered operation gateway. Each request
requires both My AI's private service capability and the still-valid browser
session for the same issuer and subject. It forwards only the destination app's
native cookies and native CSRF proof. Neither credentials nor a generic HTTP
proxy are supplied to the model. App credentials are never serialized into a
conversation, tool result or review.

App calls use structured JSON objects and server-side validation. OpenAI uses
explicit non-strict function calling for this operation-specific native body;
fixed public tool schemas retain strict mode. This follows the
[official function calling schema contract](https://developers.openai.com/api/docs/guides/function-calling).

Current development verification is recorded in
`agents/history/2026-10-26-workspace-apps-compacted.md`. Publication and installed
availability must be confirmed there; this document alone is not a release or
acceptance claim.
