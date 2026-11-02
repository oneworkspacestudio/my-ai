package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

// Complete documents are artifacts, including older chats that returned code
// instead of creating files. Quoted contents never grant action authority.
func conversationDocument(text string) string {
	files := conversationFiles(text)
	if len(files) == 1 && completeHTML(files[0].Content) {
		return files[0].Content
	}
	if len(files) > 0 {
		return ""
	}
	lower := strings.ToLower(text)
	start := strings.Index(lower, "<!doctype html")
	if start < 0 {
		start = strings.Index(lower, "<html")
	}
	end := strings.LastIndex(lower, "</html>")
	if start < 0 || end < start {
		return ""
	}
	body := text[start : end+len("</html>")]
	if strings.Count(strings.ToLower(body), "<!doctype html") > 1 || !completeHTML(body) {
		return ""
	}
	return body
}
func documentIsReference(body string, ref AttachmentRef) bool {
	sum := sha256.Sum256([]byte(body))
	return ref.SHA256 == hex.EncodeToString(sum[:]) && ref.Size == len(body)
}
func (a *App) captureConversationDocuments(scope toolScope, s *Session) error {
	for i := range s.Messages {
		m := &s.Messages[i]
		if (m.Role != "user" && m.Role != "assistant") || m.Status == "imported" || len(m.Calls) > 0 || len(m.Attachments) > 0 {
			continue
		}
		body := conversationDocument(m.Content)
		if body == "" {
			continue
		}
		if len(body) > 1<<20 {
			return errors.New("The conversation document is too large; attach a UTF-8 file up to1MiB instead")
		}
		id := apphost.ID()
		p, e := safePath(scope.Root, "Attachments", id+".txt")
		if e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		_, e = f.WriteString(body)
		if e == nil {
			e = f.Sync()
		}
		closed := f.Close()
		if e != nil {
			return e
		}
		if closed != nil {
			return closed
		}
		ref := attachmentRef(id, "index.html", []byte(body))
		if e = atomic(filepath.Join(filepath.Dir(p), id+".json"), ref); e != nil {
			return e
		}
		m.Attachments = []AttachmentRef{ref}
	}
	return nil
}
