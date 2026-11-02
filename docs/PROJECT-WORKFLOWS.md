# Build and deploy projects

Ask My AI to build an app/site/game in Chat or Code. A direct coding request enters
native Code and uses your selected project, or prepares General in your private
Drive Projects folder and Git app. It does not ask for an absolute host folder.
Existing read-only Code and Plain Chat settings remain respected.

A deployment follow-up such as deploy it uses the managed project workflow. For a
standalone static HTML site, My AI saves the file in the project, commits/pushes Git,
creates or reuses its My Hosting site, deploys that exact saved commit and reads the
real job/URL. A single HTML file is a valid repository. Hosting sign-in, native
permissions and action review still apply. A deployment-only request does not need
a model, tool planning, cloud key or inference context. Build-and-deploy requests
use the model for source changes; the app owns publication after the files exist.

Older complete HTML code blocks or inline documents become account-private file
references without changing the saved transcript. The model receives metadata and
can read bounded sections; exact file materialization does not require rewriting
the body. Source files already in the selected project remain authoritative unless
the current request supplies a replacement document. Multiple files/partial code
must be assembled in the project before deployment; the app does not guess missing
source. Imported conversations remain untrusted and do not acquire native receipts.

The build loop retries a text-only refusal once when no file action or complete
HTML artifact exists, then reports failure instead of claiming the project was
built. This does not certify that generated game/app logic is correct.

Publication records bind the account/project/site/job/commit. Old canonical tool
receipts can recover the association. Current files are checkpointed before cached
job reuse; changed source gets its new pushed commit. A pending job is inspected,
not resubmitted. An uncertain mutation outcome blocks automatic duplicate actions.
Pending markers are written after review, immediately before the native mutation;
a stopped/declined review does not poison the next attempt.

Automatic new sites use the registered native static-site capability. Framework or
server apps need a site with the appropriate build/output already configured in My
Hosting; raw source is not silently published as a built app. The workflow never
enables trusted scripts or changes public-network settings. Hosting returns the
actual URL: a localhost preview opens on the hosting PC, while a configured public
site address follows My Hosting's own domain/network settings. Native deployment
cards distinguish these cases. No external host or general shell is substituted.
