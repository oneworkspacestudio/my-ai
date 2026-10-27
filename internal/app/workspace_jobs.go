package app

import (
	"context"
	"encoding/json"
	"time"
)

// Deployment is a durable asynchronous operation. Record its real receipt before
// waiting, then follow the registered read operation with identifiers from that
// receipt. A local model need not guess how a native site's ID relates to a job.
func (a *App) trackWorkspaceDeployment(ctx context.Context, scope toolScope, s *Session, send writer, app workspaceApp, submitted workspaceCall, receipt map[string]any) map[string]any {
	result, ok := receipt["result"].(map[string]any)
	if !ok {
		return receipt
	}
	job, ok := result["deployment"].(map[string]any)
	if !ok {
		return receipt
	}
	site, _ := result["site"].(string)
	id, _ := job["id"].(string)
	parent, _ := job["projectId"].(string)
	if site == "" || id == "" || site != submitted.Parameters["site"] || parent != site {
		return receipt
	}
	result["statusTool"] = map[string]any{"name": "workspace_hosting_deployment", "arguments": map[string]string{"site": site, "deployment": id}}
	raw, _ := json.Marshal(receipt)
	a.recordToolResult(scope, s, send, string(raw))
	canRead := false
	for _, op := range app.Operations {
		if op.ID == "deployment" && !op.Review {
			canRead = true
		}
	}
	if !canRead {
		return receipt
	}
	pollCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	last := ""
	for {
		state, _ := job["state"].(string)
		if state != last {
			progress := map[string]any{"site": site, "deployment": id, "state": state, "url": job["url"], "checkedAt": now()}
			send("event", map[string]any{"method": "deployment/progress", "params": progress})
			b, _ := json.Marshal(progress)
			s.Events = append(s.Events, Event{At: now(), Method: "deployment/progress", Data: b})
			last = state
		}
		switch state {
		case "published", "failed", "cancelled", "interrupted":
			result["notice"] = "My Hosting returned this final deployment state and actual URL"
			return receipt
		}
		var response map[string]any
		call := workspaceCall{App: app.ID, Operation: "deployment", Parameters: map[string]string{"id": site, "deployment": id}}
		if err := a.workspaceBridge(pollCtx, scope, "POST", call, &response); err != nil {
			result["trackingNotice"] = "Source was submitted; the last recorded status is shown. Recheck this job before claiming success or submitting again."
			return receipt
		}
		if needed, _ := response["connectionRequired"].(bool); needed {
			result["trackingNotice"] = "Source was submitted, but native sign-in is needed to check its current status. Reconnect My Hosting and read this job; do not resubmit."
			return receipt
		}
		value, ok := response["result"].(map[string]any)
		if !ok {
			return receipt
		}
		// The native status endpoint may wrap the job in a deployment field.
		if nested, ok := value["deployment"].(map[string]any); ok {
			value = nested
		}
		if value["id"] != id || value["projectId"] != site {
			result["trackingNotice"] = "The returned job identity did not match the submitted deployment; outcome is unconfirmed"
			return receipt
		}
		job = value
		result["deployment"] = job
		raw, _ = json.Marshal(receipt)
		a.recordToolResult(scope, s, send, string(raw))
		state, _ = job["state"].(string)
		switch state {
		case "published", "failed", "cancelled", "interrupted":
			continue
		}
		select {
		case <-pollCtx.Done():
			result["trackingNotice"] = "The deployment continues in My Hosting. Check this job's real status before reporting completion."
			return receipt
		case <-time.After(2 * time.Second):
		}
	}
}
