package app

import (
	"errors"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
)

func branchSession(s Session, message string, before bool) (Session, error) {
	end := -1
	for i, m := range s.Messages {
		if m.ID == message && (m.Role == "user" || m.Role == "assistant") {
			end = i + 1
			if before {
				if m.Role != "user" {
					return Session{}, errors.New("Edit from a user message")
				}
				end = i
			} else {
				for end < len(s.Messages) && s.Messages[end].Role == "tool" {
					end++
				}
			}
			break
		}
	}
	if end < 0 {
		return Session{}, errors.New("Message not found")
	}
	s.Messages = append([]Message{}, s.Messages[:end]...)
	for i := range s.Messages {
		s.Messages[i].ResponseItems = nil
		s.Messages[i].ResponseGroup = ""
		s.Messages[i].ReplayReasoning = ""
	}
	s.ID = apphost.ID()
	s.Title = first(s.Title, 140) + " · branch"
	s.Revision = 0
	s.Thread = ""
	s.Events = []Event{}
	s.Pinned = false
	s.Archived = false
	s.ResearchContext = nil
	return s, nil
}
func (a *App) branch(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	var req struct {
		Message  string `json:"message"`
		Before   bool   `json:"before"`
		Revision int    `json:"revision"`
	}
	if !apphost.Decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, err := a.read(root, r.PathValue("id"))
	if err != nil {
		apphost.Problem(w, 404, "Conversation not found")
		return
	}
	if a.running[apphost.Current(r).User.ID+":"+s.ID] != nil || s.Revision != req.Revision {
		apphost.Problem(w, 409, "Finish the active turn and reload the conversation before branching")
		return
	}
	out, err := branchSession(s, req.Message, req.Before)
	if err != nil {
		apphost.Problem(w, 400, err.Error())
		return
	}
	if err = a.write(root, &out); err != nil {
		apphost.Problem(w, 503, "Could not save this branch")
		return
	}
	apphost.JSON(w, 200, sessionView(out))
}
