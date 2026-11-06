package app

import (
	"context"
	"net/http"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

// A queued/streaming turn must stop when its exact browser session is revoked.
func (a *App) liveTurn(w http.ResponseWriter, r *http.Request) {
	cookie, e := r.Cookie(a.H.Config.ID + "_session")
	if e != nil {
		apphost.Problem(w, 401, "Reconnect to My AI")
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	uid := apphost.Current(r).User.ID
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				var expiry, role string
				e := a.H.DB.QueryRow(`SELECT s.expires,u.role FROM host_sessions s JOIN host_users u ON u.id=s.user_id WHERE s.id=? AND u.id=?`, apphost.Hash(cookie.Value), uid).Scan(&expiry, &role)
				at, err := time.Parse(time.RFC3339Nano, expiry)
				if e != nil || err != nil || role == "disabled" || !time.Now().Before(at) {
					cancel()
					return
				}
			}
		}
	}()
	a.turn(w, r.WithContext(ctx))
}
