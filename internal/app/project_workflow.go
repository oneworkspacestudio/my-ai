package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

type projectTask struct{ Write, Publish, PublishOnly bool }

var projectDirective = regexp.MustCompile(`(?i)^(?:(?:please|can you|could you|would you|i want you to|i need you to)\s+)*(?:create|build|make|develop|code|implement|edit|fix|update|change|deploy|publish|host)\b`)
var codeSubject = regexp.MustCompile(`(?i)\b(?:html|css|javascript|typescript|python|code|app|application|website|webpage|web page|game|project|repository|script|component|api|program|bug)\b|\.(?:html?|css|[cm]?js|jsx|tsx?|py|go|rs|java|cpp|cs|vue|svelte|json)\b`)
var publishVerb = regexp.MustCompile(`(?i)\b(?:deploy|publish|host)\b`)
var changeVerb = regexp.MustCompile(`(?i)\b(?:create|build|make|develop|code|implement|edit|fix|update|change)\b`)
var negativeDirective = regexp.MustCompile(`(?i)\b(?:don't|do not|never|without|not yet|rather than|instead of)\s+(?:deploy|publish|host)\b`)
var deployDirective = regexp.MustCompile(`(?i)\bdeploy\b`)
var referencedArtifact = regexp.MustCompile(`(?i)\b(?:it|this|that|these|the site|the app|the game|the project|the file|the code|attached (?:file|page|code))\b`)
var pushedCommit = regexp.MustCompile(`^[a-f0-9]{40,64}$`)

// Only the current person's directive grants authority. History, source text and
// quoted code can supply artifacts, never permission to mutate apps.
func requestedProjectTask(text string, source bool) projectTask {
	text = strings.TrimSpace(strings.SplitN(text, "```", 2)[0])
	text = strings.SplitN(text, "\n", 2)[0]
	if !projectDirective.MatchString(text) {
		return projectTask{}
	}
	pub := publishVerb.MatchString(text) && !negativeDirective.MatchString(text)
	change := changeVerb.MatchString(text)
	write := change && (source && referencedArtifact.MatchString(text) || codeSubject.MatchString(text))
	// Deploy is an unambiguous project action; host/publish need a code subject or
	// an artifact/project, so "host a party" and "publish a note" stay ordinary Chat.
	if !codeSubject.MatchString(text) && !deployDirective.MatchString(text) && !(source && referencedArtifact.MatchString(text)) {
		pub = false
	}
	return projectTask{Write: write, Publish: pub, PublishOnly: pub && !change}
}

type conversationFile struct{ Content, Language string }

func hasConversationFiles(s *Session) bool {
	for _, m := range s.Messages {
		if len(m.Attachments) > 0 || len(conversationFiles(m.Content)) > 0 || conversationDocument(m.Content) != "" {
			return true
		}
	}
	return false
}
func projectFilesChangedThisTurn(s *Session) bool {
	turn := currentTurn(s)
	for _, m := range s.Messages {
		if m.Role != "tool" || m.TurnID != turn {
			continue
		}
		switch m.ToolName {
		case "write_project_file", "edit_project_file", "copy_attachment_to_project":
		default:
			continue
		}
		var v map[string]any
		if json.Unmarshal([]byte(m.Content), &v) == nil && v["error"] == nil {
			return true
		}
	}
	return false
}
func conversationFiles(text string) []conversationFile {
	out := []conversationFile{}
	lines := strings.SplitAfter(text, "\n")
	for i := 0; i < len(lines); i++ {
		fence := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(fence, "```") {
			continue
		}
		lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(fence, "```")))
		start := i + 1
		i = start
		for i < len(lines) && strings.TrimSpace(lines[i]) != "```" {
			i++
		}
		if i == len(lines) {
			break
		} // Never materialize an incomplete generated response.
		body := strings.Join(lines[start:i], "")
		if len(body) > 0 && len(body) <= 1<<20 && utf8.ValidString(body) && !strings.ContainsRune(body, 0) {
			out = append(out, conversationFile{body, lang})
		}
	}
	return out
}
func completeHTML(text string) bool {
	s := strings.ToLower(strings.TrimSpace(text))
	return (strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html")) && strings.HasSuffix(s, "</html>")
}
func latestHTMLArtifact(s *Session, currentOnly bool) (string, error) {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		m := s.Messages[i]
		if currentOnly && m.Role == "user" {
			break
		}
		if (m.Role != "assistant" && m.Role != "user") || m.Status == "imported" || len(m.Calls) > 0 {
			continue
		}
		if body := conversationDocument(m.Content); body != "" {
			return body, nil
		}
		files := conversationFiles(m.Content)
		if len(files) == 0 {
			continue
		}
		if len(files) == 1 && completeHTML(files[0].Content) {
			return files[0].Content, nil
		}
		return "", errors.New("The conversation has several files or a partial code fragment. My AI must assemble these in the selected project before deployment")
	}
	return "", nil
}

// Migrate an unsaved, complete HTML artifact only when the selected project has
// no entry file. Existing project files always remain the source of truth.
func (a *App) materializeHTML(ctx context.Context, scope toolScope, s *Session, send writer, currentOnly bool) (bool, error) {
	p, err := a.managedProject(ctx, scope.UID, scope.Project)
	if err != nil {
		return false, err
	}
	body := ""
	if !currentOnly {
		for i := len(s.Messages) - 1; i >= 0; i-- {
			m := s.Messages[i]
			if m.Role != "user" {
				continue
			}
			body = conversationDocument(m.Content)
			if body == "" && len(m.Attachments) == 1 {
				_, raw, e := loadAttachment(scope.Root, m.Attachments[0].ID)
				if e != nil {
					return false, e
				}
				if completeHTML(string(raw)) {
					body = string(raw)
				}
			}
			if len(m.Attachments) > 1 && requestedProjectTask(s.ToolInput, true).PublishOnly {
				return false, errors.New("Choose the entry page before deploying multiple attached files")
			}
			break
		}
	}
	entry, err := projectRelative(p.Path, "index.html")
	if err != nil {
		return false, err
	}
	exists := false
	if st, e := os.Stat(entry); e == nil {
		if st.IsDir() {
			return false, errors.New("index.html is a folder; choose the project's real output folder")
		}
		exists = true
	} else if !os.IsNotExist(e) {
		return false, e
	}
	if exists && !currentOnly && body == "" {
		return true, nil
	}
	// Never deploy raw framework sources as if they were an already-built site.
	if _, e := os.Stat(filepath.Join(p.Path, "package.json")); e == nil {
		return false, nil
	}
	if body == "" {
		body, err = latestHTMLArtifact(s, currentOnly)
		if err != nil {
			return false, err
		}
	}
	if body == "" && !currentOnly {
		for i := len(s.Messages) - 1; i >= 0; i-- {
			m := s.Messages[i]
			if m.Role != "user" || len(m.Attachments) == 0 {
				continue
			}
			if len(m.Attachments) != 1 {
				return false, errors.New("Choose which attached page is the entry file before deploying this multi-file project")
			}
			_, b, e := loadAttachment(scope.Root, m.Attachments[0].ID)
			if e != nil {
				return false, e
			}
			if completeHTML(string(b)) {
				body = string(b)
			}
			break
		}
	}
	if body == "" {
		return false, nil
	}
	value, err := a.editProjectFile(ctx, scope, "copy_conversation_file_to_project", map[string]string{"path": "index.html", "content": body})
	if err != nil {
		return false, err
	}
	data, _ := json.Marshal(map[string]any{"message": "Saved the complete HTML artifact in your selected Git project", "file": value})
	s.Events = append(s.Events, Event{At: now(), Method: "project/artifact-saved", Data: data})
	send("event", map[string]any{"method": "project/artifact-saved", "params": json.RawMessage(data)})
	a.checkpointResponse(scope, s, send)
	return true, nil
}

type projectPublication struct {
	Schema  int    `json:"schema"`
	Project string `json:"project"`
	Site    string `json:"site,omitempty"`
	Slug    string `json:"slug"`
	Pending string `json:"pending,omitempty"`
	Job     string `json:"job,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

var nativeObjectID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,160}$`)

func publicationPath(scope toolScope) (string, error) {
	if scope.Root == "" || !idRE.MatchString(scope.Project) {
		return "", errors.New("Select an authenticated managed project before deploying")
	}
	return safePath(scope.Root, "Publications", scope.Project+".json")
}
func loadPublication(scope toolScope) (projectPublication, error) {
	p, e := publicationPath(scope)
	if e != nil {
		return projectPublication{}, e
	}
	b := projectPublication{Schema: 1, Project: scope.Project, Slug: "project-" + scope.Project[:12]}
	raw, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return b, nil
	}
	if e != nil || len(raw) > 4096 || json.Unmarshal(raw, &b) != nil || b.Schema != 1 || b.Project != scope.Project || !nativeObjectID.MatchString(b.Slug) || (b.Site != "" && !nativeObjectID.MatchString(b.Site)) || (b.Job != "" && !nativeObjectID.MatchString(b.Job)) {
		return b, errors.New("The project's saved Hosting connection needs attention; no deployment was submitted")
	}
	return b, nil
}
func savePublication(scope toolScope, b projectPublication) error {
	p, e := publicationPath(scope)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	return atomic(p, b)
}

// Older releases already stored canonical native call/receipt pairs. Recover
// their exact project/site/job association rather than creating a second site.
func publicationFromTranscript(s *Session, project string) (projectPublication, bool) {
	calls := map[string]functionCall{}
	for _, m := range s.Messages {
		if m.Status == "imported" {
			continue
		}
		for _, c := range m.Calls {
			calls[c.ID] = c
		}
	}
	for i := len(s.Messages) - 1; i >= 0; i-- {
		m := s.Messages[i]
		if m.Role != "tool" || m.Status == "imported" {
			continue
		}
		c, ok := calls[m.CallID]
		if !ok {
			continue
		}
		var args map[string]string
		if json.Unmarshal([]byte(c.Arguments), &args) != nil {
			continue
		}
		site, commit, selected := args["site"], args["commit"], args["project"]
		if c.Name == "use_workspace_app" {
			if args["app"] != "hosting" || args["operation"] != "deploy_project" {
				continue
			}
			var input workspaceArgs
			if json.Unmarshal([]byte(args["arguments"]), &input) != nil {
				continue
			}
			site, selected = input.Parameters["site"], input.Parameters["project"]
			var body struct {
				Commit string `json:"commit"`
			}
			json.Unmarshal(input.Body, &body)
			commit = body.Commit
		} else if c.Name != "workspace_hosting_deploy_project" {
			continue
		}
		if selected != project || !nativeObjectID.MatchString(site) || !pushedCommit.MatchString(commit) {
			continue
		}
		var receipt map[string]any
		if json.Unmarshal([]byte(m.Content), &receipt) != nil {
			continue
		}
		job := nativeJob(receipt)
		id, _ := job["id"].(string)
		if !nativeObjectID.MatchString(id) || job["projectId"] != site {
			return projectPublication{}, false // A later unconfirmed call must not be hidden by an older receipt.
		}
		return projectPublication{Schema: 1, Project: project, Site: site, Job: id, Commit: commit, Slug: "project-" + project[:12]}, true
	}
	return projectPublication{}, false
}

// Every step uses the existing authenticated native connector and review. This
// is app-owned orchestration, not a model's claim that a deployment happened.
func (a *App) projectNativeStep(ctx context.Context, scope toolScope, s *Session, send writer, operation string, parameters map[string]string, body any) (map[string]any, error) {
	rawBody, _ := json.Marshal(body)
	argsRaw, _ := json.Marshal(workspaceArgs{Parameters: parameters, Body: rawBody})
	args := map[string]string{"app": "hosting", "operation": operation, "arguments": string(argsRaw)}
	callRaw, _ := json.Marshal(args)
	scope.CallID = "workflow_" + apphost.ID()
	scope.InvokedTool = "use_workspace_app"
	a.recordAssistant(scope, s, send, "", []functionCall{{ID: scope.CallID, Name: "use_workspace_app", Arguments: string(callRaw)}}, "completed")
	result := a.runTool(ctx, scope, s, send, "use_workspace_app", args)
	a.recordToolResult(scope, s, send, result)
	var value map[string]any
	if json.Unmarshal([]byte(result), &value) != nil {
		return nil, errors.New("My Hosting returned an unreadable action result")
	}
	if failure, ok := value["error"].(string); ok && failure != "" {
		return value, errors.New(failure)
	}
	if needed, _ := value["connectionRequired"].(bool); needed {
		return value, errors.New("My Hosting sign-in must complete before deployment can continue")
	}
	return value, nil
}
func resultObject(v map[string]any) map[string]any {
	result, _ := v["result"].(map[string]any)
	return result
}
func nativeJob(v map[string]any) map[string]any {
	result := resultObject(v)
	if job, ok := result["deployment"].(map[string]any); ok {
		return job
	}
	return result
}
func deploymentSummary(job map[string]any) (string, error) {
	state, _ := job["state"].(string)
	link, _ := job["url"].(string)
	switch state {
	case "published":
		if link == "" {
			return "", errors.New("My Hosting reports published but has not returned a URL; open its deployment details")
		}
		u, e := url.Parse(link)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || strings.ContainsAny(link, "\r\n<>\\") {
			return "", errors.New("My Hosting returned an invalid site URL; open its deployment details")
		}
		return "Saved in your Git project and deployed through My Hosting: [Open your site](<" + u.String() + ">).", nil
	case "queued", "building", "running", "deploying", "pending":
		return "Your Git project was submitted to My Hosting and is still running. The deployment card shows its latest confirmed status.", nil
	case "failed", "cancelled", "interrupted":
		return "", fmt.Errorf("My Hosting deployment %s. Your source is saved in Git; inspect the deployment details before retrying", state)
	}
	return "", errors.New("My Hosting has not confirmed a deployment outcome; check its recorded job before submitting again")
}
func (a *App) publishManagedProject(ctx context.Context, scope toolScope, s *Session, send writer) (string, error) {
	if !scope.Writable {
		return "", errors.New("This project is in Preview. Allow changes before deploying")
	}
	scope.OwnPublication = false
	scope.WorkspaceSession = s.WorkspaceSession
	scope.WorkspaceKnown = map[string]bool{}
	scope.WorkspaceNative = map[string]workspaceNativeTool{}
	declined := false
	scope.WorkspaceDeclined = &declined
	if _, e := a.workspaceDiscovery(ctx, scope, "describe_workspace_app", "hosting"); e != nil {
		return "", e
	}
	b, e := loadPublication(scope)
	if e != nil {
		return "", e
	}
	if b.Site == "" || b.Pending != "" && b.Job == "" {
		if old, ok := publicationFromTranscript(s, scope.Project); ok && (b.Site == "" || b.Site == old.Site) {
			b = old
			if e = savePublication(scope, b); e != nil {
				return "", e
			}
		}
	}
	project, e := a.managedProject(ctx, scope.UID, scope.Project)
	if e != nil {
		return "", e
	}
	if b.Pending != "" && b.Job == "" {
		return "", errors.New("An earlier Hosting action has an unconfirmed outcome. Inspect its history before retrying; no action was repeated")
	}
	saved, e := a.saveProjectCheckpoint(ctx, scope, s, send)
	if e != nil {
		return "", e
	}
	if saved.ID != scope.Project || !pushedCommit.MatchString(saved.Commit) {
		return "", errors.New("Git has not confirmed the selected project's exact pushed commit; nothing was deployed")
	}
	scope.BeforeWorkspaceMutation = func(app, operation string) error {
		if app != "hosting" {
			return errors.New("Unexpected app in project publication")
		}
		switch operation {
		case "create_site":
			b.Pending = "site"
		case "deploy_project":
			b.Pending = "deployment"
			b.Job = ""
			b.Commit = saved.Commit
		default:
			return errors.New("Unexpected mutation in project publication")
		}
		return savePublication(scope, b)
	}
	if b.Site == "" {
		// Automatic new sites are static. Framework/server projects need a configured
		// native Hosting site; never publish their raw source under a false success.
		if _, e = os.Stat(filepath.Join(project.Path, "package.json")); e == nil {
			return "", errors.New("This project needs its build/output configured in My Hosting before it can be published; raw framework source was not deployed")
		}
		entry, e := projectRelative(project.Path, "index.html")
		if e != nil {
			return "", e
		}
		if st, e := os.Stat(entry); e != nil || st.IsDir() {
			return "", errors.New("The selected Git project has no index.html yet. Build and save the site in this project before deployment")
		}
		if b.Pending != "" {
			return "", errors.New("An earlier site-creation outcome is unconfirmed. Check My Hosting before creating another site; no action was repeated")
		}
		created, e := a.projectNativeStep(ctx, scope, s, send, "create_site", nil, map[string]any{"name": project.Name, "slug": b.Slug, "adapter": "static", "output": ".", "previews": true})
		if e != nil {
			if declined {
				b.Pending = ""
				if saveErr := savePublication(scope, b); saveErr != nil {
					return "", saveErr
				}
			}
			return "", e
		}
		site, _ := resultObject(created)["id"].(string)
		if !nativeObjectID.MatchString(site) {
			return "", errors.New("My Hosting created-site receipt has no valid site ID; check it before retrying")
		}
		b.Site = site
		b.Pending = ""
		if e = savePublication(scope, b); e != nil {
			return "", e
		}
	}
	if b.Job != "" {
		checked, e := a.projectNativeStep(ctx, scope, s, send, "deployment", map[string]string{"id": b.Site, "deployment": b.Job}, nil)
		if e != nil {
			return "", e
		}
		job := nativeJob(checked)
		if job["id"] != b.Job || job["projectId"] != b.Site {
			return "", errors.New("My Hosting returned a different deployment identity; nothing was resubmitted")
		}
		state, _ := job["state"].(string)
		if state != "published" && state != "failed" && state != "cancelled" && state != "interrupted" {
			return deploymentSummary(job)
		}
		if state == "published" && b.Commit == saved.Commit {
			return deploymentSummary(job)
		}
		b.Pending = ""
	}
	if b.Pending != "" {
		return "", errors.New("The prior deployment submission is unconfirmed. Inspect My Hosting's deployment history before retrying; no duplicate was submitted")
	}
	receipt, e := a.projectNativeStep(ctx, scope, s, send, "deploy_project", map[string]string{"site": b.Site, "project": scope.Project}, map[string]string{"commit": saved.Commit, "environment": "preview"})
	if e != nil {
		if declined {
			b.Pending = ""
			if saveErr := savePublication(scope, b); saveErr != nil {
				return "", saveErr
			}
		}
		return "", e
	}
	if submitted, ok := receipt["deploymentSubmitted"].(bool); ok && !submitted {
		b.Pending = ""
		if e = savePublication(scope, b); e != nil {
			return "", e
		}
		return "", errors.New("Source changed while deployment was being prepared. Git saved it, but nothing was submitted; retry with the new saved revision")
	}
	job := nativeJob(receipt)
	id, _ := job["id"].(string)
	if !nativeObjectID.MatchString(id) || job["projectId"] != b.Site {
		return "", errors.New("My Hosting has not returned a matching deployment receipt; do not submit again until its history is checked")
	}
	b.Job = id
	b.Pending = ""
	if e = savePublication(scope, b); e != nil {
		return "", e
	}
	return deploymentSummary(job)
}
