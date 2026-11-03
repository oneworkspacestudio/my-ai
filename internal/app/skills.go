package app

import (
	"embed"
	"errors"
	"strings"
)

// These audited workflows ship with the app. They cannot grant permissions or
// load arbitrary executable instructions from a project or a public page.
//
//go:embed skills/*.md
var skillFiles embed.FS

type workflowSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func workflowSkills(scope toolScope) []workflowSkill {
	out := []workflowSkill{}
	if scope.Root != "" {
		out = append(out, workflowSkill{"attachments", "Attachments", "Read/search stored files or copy them exactly into a project."}, workflowSkill{"conversations", "Conversations", "Find and read earlier private conversations in bounded pages."})
	}
	if !scope.NoWeb && (scope.PrivateAppRead == nil || !*scope.PrivateAppRead) {
		out = append(out, workflowSkill{"research", "Research", "Find current facts, resolve dates, verify sources and cite evidence."})
	}
	if scope.Project != "" {
		out = append(out, workflowSkill{"project", "Project editing", "Inspect, copy, edit and save files in the selected Git project."})
	}
	if scope.WorkspaceSession != "" {
		out = append(out, workflowSkill{"workspace", "Workspace apps", "Discover and use installed apps with their native permissions."})
	}
	if scope.Project != "" && scope.Writable && scope.WorkspaceSession != "" {
		out = append(out, workflowSkill{"deployment", "Deployment", "Save the exact Git revision, publish with My Hosting and verify the job."})
	}
	return out
}
func loadWorkflowSkill(scope toolScope, id string) (any, error) {
	for _, s := range workflowSkills(scope) {
		if s.ID == id {
			if scope.LoadedSkills != nil {
				scope.LoadedSkills[id] = true
			}
			b, err := skillFiles.ReadFile("skills/" + id + ".md")
			return map[string]any{"id": id, "instructions": string(b), "notice": "Workflow guidance only. Existing tools, permissions and reviews still apply."}, err
		}
	}
	return nil, errors.New("skill is unavailable in this scope; choose an ID from the supplied skill catalog")
}
func skillInstructions(scope toolScope) string {
	var b strings.Builder
	b.WriteString(" Available workflow skills (use load_skill with the ID when needed):")
	for _, s := range workflowSkills(scope) {
		b.WriteString(" " + s.ID + ": " + s.Description)
	}
	// Core project/app steps are small enough to include when their scope is
	// active, so a small model need not discover how to discover its tools.
	for _, id := range []string{"project", "workspace"} {
		if (id == "project" && scope.Project != "") || (id == "workspace" && scope.WorkspaceSession != "" && (!scope.LazyTools || len(scope.WorkspaceNative) > 0 || scope.LoadedSkills["workspace"])) {
			v, _ := skillFiles.ReadFile("skills/" + id + ".md")
			b.WriteString("\n" + string(v))
		}
	}
	return b.String()
}
