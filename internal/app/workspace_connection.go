package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

// Verify through a registered read operation so the existing Workspace broker
// validates the native app's issuer/subject and permissions. Discard all data.
func (a *App) verifyWorkspaceConnection(ctx context.Context, scope toolScope, id string) (bool, error) {
	catalog, err := a.workspaceApps(ctx, scope)
	if err != nil {
		return false, err
	}
	for _, app := range catalog.Apps {
		if app.ID != id {
			continue
		}
		if !app.NativeAccess {
			return false, errors.New("This app has no registered native connection")
		}
		for _, op := range app.Operations {
			if op.Review {
				continue
			}
			call := workspaceCall{App: app.ID, Operation: op.ID}
			var result map[string]any
			if err = a.workspaceBridge(ctx, scope, "POST", call, &result); err != nil {
				return false, err
			}
			if needed, _ := result["connectionRequired"].(bool); needed {
				return false, nil
			}
			if _, ok := result["result"]; !ok {
				return false, errors.New("The app connection could not be verified")
			}
			return true, nil
		}
		return false, errors.New("This app has no registered read operation for sign-in verification")
	}
	return false, errors.New("Choose an installed Workspace app")
}
func (a *App) workspaceConnection(w http.ResponseWriter, r *http.Request) {
	var input struct{ App string }
	if !apphost.Decode(w, r, &input) {
		return
	}
	scope := toolScope{UID: apphost.Current(r).User.ID}
	if cookie, err := r.Cookie("workspace_session"); err == nil {
		scope.WorkspaceSession = cookie.Value
	}
	connected, err := a.verifyWorkspaceConnection(r.Context(), scope, input.App)
	if err != nil {
		apphost.Problem(w, 503, "Workspace could not verify this app's sign-in or permissions. No app changes were submitted.")
		return
	}
	apphost.JSON(w, 200, map[string]bool{"connected": connected, "connectionRequired": !connected})
}
