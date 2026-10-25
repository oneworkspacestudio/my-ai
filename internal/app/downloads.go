package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

type Download struct {
	ID        string `json:"id"`
	Model     string `json:"model"`
	State     string `json:"state"`
	Status    string `json:"status"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
	Error     string `json:"error,omitempty"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

func normalizeModelTag(raw string) (string, error) {
	tag := strings.TrimSpace(raw)
	for _, prefix := range []string{"ollama run ", "ollama pull "} {
		tag = strings.TrimPrefix(tag, prefix)
	}
	if strings.HasPrefix(tag, "https://ollama.com/") {
		u, e := url.Parse(tag)
		if e != nil || u.Host != "ollama.com" || u.User != nil {
			return "", errors.New("use an exact model tag or an official Ollama model link")
		}
		tag = strings.TrimPrefix(strings.TrimPrefix(u.Path, "/"), "library/")
	}
	if !modelRE.MatchString(tag) || strings.HasPrefix(tag, "/") || strings.Contains(tag, "//") || strings.Contains(tag, "..") || strings.Contains(tag, "://") {
		return "", errors.New("use a model tag such as qwen3:0.6b, an official Ollama model URL, or its ollama run command")
	}
	if !strings.Contains(tag[strings.LastIndex(tag, "/")+1:], ":") {
		tag += ":latest"
	}
	return tag, nil
}

type downloadQueue struct {
	mu       sync.Mutex
	path     string
	jobs     []Download
	wake     chan struct{}
	cancel   context.CancelFunc
	active   string
	provider Provider
}

func newDownloadQueue(data string) (*downloadQueue, error) {
	q := &downloadQueue{path: filepath.Join(data, "model-downloads", "jobs.json"), wake: make(chan struct{}, 1), provider: defaults()[2], jobs: []Download{}}
	b, e := os.ReadFile(q.path)
	if e == nil {
		if e = json.Unmarshal(b, &q.jobs); e != nil {
			return nil, errors.New("model download history is unreadable; preserve it before repair")
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	for i := range q.jobs {
		if q.jobs[i].State == "downloading" || q.jobs[i].State == "queued" {
			q.jobs[i].State = "queued"
			q.jobs[i].Status = "Waiting to resume after server restart"
		}
	}
	return q, nil
}
func (q *downloadQueue) saveLocked() error { return atomic(q.path, q.jobs) }
func (q *downloadQueue) list() []Download {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := append([]Download{}, q.jobs...)
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out
}
func (q *downloadQueue) add(model string) (Download, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, j := range q.jobs {
		if strings.EqualFold(j.Model, model) && (j.State == "queued" || j.State == "downloading") {
			return j, nil
		}
	}
	j := Download{ID: apphost.ID(), Model: model, State: "queued", Status: "Waiting for the server download worker", Created: now(), Updated: now()}
	previous := append([]Download{}, q.jobs...)
	q.jobs = append(q.jobs, j)
	if len(q.jobs) > 100 {
		for i, v := range q.jobs {
			if v.State != "queued" && v.State != "downloading" {
				q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
				break
			}
		}
	}
	if e := q.saveLocked(); e != nil {
		q.jobs = previous
		return Download{}, e
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return j, nil
}
func (q *downloadQueue) stop(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		j := &q.jobs[i]
		if j.ID != id {
			continue
		}
		if j.State != "queued" && j.State != "downloading" {
			return nil
		}
		j.State = "cancelled"
		j.Status = "Cancelled by the owner"
		j.Updated = now()
		if q.active == id && q.cancel != nil {
			q.cancel()
		}
		return q.saveLocked()
	}
	return errors.New("download not found")
}
func (q *downloadQueue) busy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, j := range q.jobs {
		if j.State == "queued" || j.State == "downloading" {
			return true
		}
	}
	return false
}
func (q *downloadQueue) update(id string, change func(*Download), persist bool) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].ID == id {
			if q.jobs[i].State == "cancelled" {
				return nil
			}
			change(&q.jobs[i])
			q.jobs[i].Updated = now()
			if persist {
				if e := q.saveLocked(); e != nil {
					q.jobs[i].State = "failed"
					q.jobs[i].Error = "The server could not save download progress. Check protected storage and free disk space."
					q.jobs[i].Status = q.jobs[i].Error
					return e
				}
				return nil
			}
			return nil
		}
	}
	return errors.New("download disappeared")
}
func (q *downloadQueue) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		var job Download
		q.mu.Lock()
		for i := range q.jobs {
			if q.jobs[i].State == "queued" {
				job = q.jobs[i]
				break
			}
		}
		q.mu.Unlock()
		if job.ID == "" {
			select {
			case <-ctx.Done():
				return
			case <-q.wake:
				continue
			}
		}
		if job.ID != "" {
			q.download(ctx, job)
		}
	}
}
func (q *downloadQueue) download(parent context.Context, j Download) {
	ctx, cancel := context.WithTimeout(parent, 24*time.Hour)
	defer cancel()
	q.mu.Lock()
	canStart := false
	for _, job := range q.jobs {
		if job.ID == j.ID && job.State == "queued" {
			canStart = true
		}
	}
	if !canStart {
		q.mu.Unlock()
		return
	}
	q.active = j.ID
	q.cancel = cancel
	q.mu.Unlock()
	defer func() { q.mu.Lock(); q.active = ""; q.cancel = nil; q.mu.Unlock() }()
	fail := func(message string) {
		_ = q.update(j.ID, func(v *Download) {
			if parent.Err() != nil {
				v.State = "queued"
				v.Error = ""
				v.Status = "Waiting to resume after server restart"
			} else {
				v.State = "failed"
				v.Error = message
				v.Status = message
			}
		}, true)
	}
	if e := q.update(j.ID, func(v *Download) {
		v.State = "downloading"
		v.Status = "Connecting to the shared Ollama server"
		v.Error = ""
	}, true); e != nil {
		return
	}
	resp, e := request(ctx, q.provider, "POST", "/api/pull", map[string]any{"model": j.Model, "stream": true})
	if e != nil {
		fail("Ollama could not start this download. Check that the server is running and the model tag exists.")
		return
	}
	defer resp.Body.Close()
	scan := bufio.NewScanner(resp.Body)
	scan.Buffer(make([]byte, 64<<10), 2<<20)
	success := false
	last := time.Time{}
	layers := map[string][2]int64{}
	for scan.Scan() {
		var p struct {
			Status    string `json:"status"`
			Digest    string `json:"digest"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
			Error     string `json:"error"`
		}
		if json.Unmarshal(scan.Bytes(), &p) != nil {
			fail("Ollama returned invalid download progress")
			return
		}
		if p.Error != "" {
			fail(first(p.Error, 500))
			return
		}
		if p.Total > 0 && p.Digest != "" {
			layers[p.Digest] = [2]int64{p.Completed, p.Total}
		}
		var completed, total int64
		for _, v := range layers {
			completed += v[0]
			total += v[1]
		}
		persist := time.Since(last) > 500*time.Millisecond || p.Status == "success"
		if persist {
			last = time.Now()
		}
		if e = q.update(j.ID, func(v *Download) {
			v.Status = p.Status
			v.Completed = completed
			v.Total = total
			if p.Status == "success" {
				v.State = "completed"
				v.Completed = v.Total
			}
		}, persist); e != nil {
			cancel()
			return
		}
		if p.Status == "success" {
			success = true
			return
		}
	}
	if parent.Err() != nil {
		_ = q.update(j.ID, func(v *Download) { v.State = "queued"; v.Status = "Waiting to resume after server restart" }, true)
		return
	}
	if ctx.Err() != nil {
		fail("Download timed out or was stopped. Retry to reuse Ollama's partial data.")
		return
	}
	if scan.Err() != nil || !success {
		fail("Ollama disconnected before confirming success. Retry to resume the download.")
	}
}
func (a *App) downloadList(w http.ResponseWriter, r *http.Request) {
	apphost.JSON(w, 200, a.downloads.list())
}
func (a *App) downloadCancel(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Only the owner can cancel shared downloads")
		return
	}
	if e := a.downloads.stop(r.PathValue("id")); e != nil {
		apphost.Problem(w, 404, e.Error())
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "model-download-cancel", r.PathValue("id"))
	apphost.JSON(w, 200, map[string]bool{"cancelled": true})
}
