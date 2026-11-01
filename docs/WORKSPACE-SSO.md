# Automatic Workspace app sign-in

My AI starts an installed app's normal MyAccounts SSO flow when a native action
needs a session. A cold My Hosting session should not require opening the app
manually or telling the model to retry. My AI verifies the native connection
through an existing registered read operation, then resumes the exact request.
The model receives automatic-authentication capability metadata, rather than
interpreting a missing cached browser session as missing app capability.

The browser flow is restricted to the current origin and the installed app's
exact Workspace opening URL. The existing manager checks issuer, subject,
browser session, native identity, permissions and CSRF. No recovery/admin
credentials are used and no sessions are minted by My AI.

Signing in never approves an app mutation. Create-site and deployment review
remain separate and require the person's decision. Cancellation removes the
temporary authentication frame, cancels requests and prevents stale approval
resumption. An expired or mismatched identity fails closed. If MyAccounts requires
interactive authentication or browser policy prevents silent sign-in, My AI
offers sign-in recovery rather than claiming an action succeeded.

The regression fixture reproduced the old cold-session/manual-connect failure
against production frontend assets on /myai/. The fix passed native redirect/
callback/cookie sign-in and exact resume, separate mutation review, cancellation
cleanup and mismatched-identity denial. Backend tests verify read-only probing,
bound issuer/subject/browser identity, disabled-account denial and authenticated
routes. Fixtures use synthetic identities only. This does not claim a live owner
deployment, arbitrary external identity-provider policy or physical-phone test.
