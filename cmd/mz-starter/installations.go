package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	starter "github.com/mz-core/mz_starter"
	"github.com/mz-core/mz_starter/engine"
)

type installationJob struct {
	Report engine.InstallProgress `json:"report"`
	Done   bool                   `json:"done"`
	cancel context.CancelFunc
}
type installationJobs struct {
	mu          sync.Mutex
	jobs        map[string]*installationJob
	active      bool
	catalog     engine.Catalog
	maxDownload int64
}

func (m *installationJobs) handler(w http.ResponseWriter, r *http.Request) {
	respond := func(code int, v any) { w.WriteHeader(code); _ = json.NewEncoder(w).Encode(v) }
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/installations"), "/")
	if id == "" && r.Method == "POST" {
		var q struct {
			Server           string   `json:"server"`
			Resources        []string `json:"resources"`
			PartialConfirmed bool     `json:"partial_confirmed"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		d.DisallowUnknownFields()
		if d.Decode(&q) != nil || strings.TrimSpace(q.Server) == "" || !q.PartialConfirmed || len(q.Resources) == 0 || len(q.Resources) > 51 {
			respond(400, apiError{"invalid_request", "partial confirmation and resources required"})
			return
		}
		if _, e := engine.PlanInstallation(m.catalog, starter.Assets, q.Server, "", "new", "", nil); e != nil {
			respond(400, apiError{e.Error(), "invalid destination"})
			return
		}
		var b [16]byte
		if _, e := rand.Read(b[:]); e != nil {
			respond(500, apiError{"random_error", "cannot start"})
			return
		}
		id = hex.EncodeToString(b[:])
		ctx, cancel := context.WithCancel(context.Background())
		m.mu.Lock()
		if m.active {
			m.mu.Unlock()
			cancel()
			respond(409, apiError{"installation_in_progress", "installation already active"})
			return
		}
		for key, old := range m.jobs {
			if old.Done {
				delete(m.jobs, key)
			}
		}
		j := &installationJob{Report: engine.InstallProgress{State: "preflight"}, cancel: cancel}
		m.jobs[id] = j
		m.active = true
		m.mu.Unlock()
		go func() {
			defer cancel()
			result, _ := engine.InstallPartial(ctx, m.catalog, starter.Assets, q.Server, q.Resources, engine.InstallOptions{GitHub: engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}, Verify: engine.VerifyOptions{MaxDownload: m.maxDownload}, Progress: func(p engine.InstallProgress) {
				p.Installed = append([]string{}, p.Installed...)
				p.Blocked = append([]string{}, p.Blocked...)
				p.Files = append([]string{}, p.Files...)
				m.mu.Lock()
				j.Report = p
				m.mu.Unlock()
			}})
			m.mu.Lock()
			j.Report = result
			j.Done = true
			m.active = false
			m.mu.Unlock()
		}()
		respond(202, map[string]string{"id": id})
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[id]
	if j == nil {
		respond(404, apiError{"not_found", "installation not found"})
		return
	}
	switch r.Method {
	case "GET":
		respond(200, j)
	case "DELETE":
		if j.Done {
			delete(m.jobs, id)
		} else if j.Report.State == "publishing" {
			respond(409, apiError{"publication_in_progress", "publication cannot be cancelled"})
			return
		} else {
			j.cancel()
		}
		respond(200, map[string]string{"status": "cancel_requested"})
	default:
		respond(405, apiError{"method", "method not allowed"})
	}
}
