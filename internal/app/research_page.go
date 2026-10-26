package app

import (
	"context"
	"encoding/json"
	"time"
)

func (a *App) researchPage(ctx context.Context, scope toolScope, s *Session, send writer, address, question string) (webDocument, error) {
	researchProgress(s, send, "reading")
	start := map[string]any{"name": "read_webpage", "arguments": map[string]string{"url": address}}
	raw, _ := json.Marshal(start)
	s.Events = append(s.Events, Event{At: now(), Method: "tool/start", Data: raw})
	send("event", map[string]any{"method": "tool/start", "params": start})
	readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	doc, err := a.readPublicPage(readCtx, address, question)
	var output any = doc
	if err != nil {
		output = map[string]string{"error": err.Error()}
	}
	end := map[string]any{"name": "read_webpage", "output": output, "failed": err != nil}
	raw, _ = json.Marshal(end)
	s.Events = append(s.Events, Event{At: now(), Method: "tool/result", Data: raw})
	send("event", map[string]any{"method": "tool/result", "params": end})
	if a.H != nil {
		a.H.Audit(scope.UID, "tool-call", "read_webpage")
	}
	return doc, err
}
