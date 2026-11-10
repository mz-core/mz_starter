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
	"time"

	starter "github.com/mz-core/mz_starter"
	"github.com/mz-core/mz_starter/engine"
)

type installationJob struct {
	Kind   string `json:"kind"`
	Report any    `json:"report"`
	Done   bool   `json:"done"`
	cancel context.CancelFunc
}
type installationJobs struct {
	mu          sync.Mutex
	jobs        map[string]*installationJob
	active      bool
	catalog     engine.Catalog
	maxDownload int64
}

type installRequest struct {
	Kind             string   `json:"kind"`
	Server           string   `json:"server"`
	Resources        []string `json:"resources"`
	PartialConfirmed bool     `json:"partial_confirmed"`
	FullConfirmed    bool     `json:"full_confirmed"`
	AllowIncomplete  bool     `json:"allow_incomplete"`
	LocalRoot        string   `json:"local_root"`
	LocalAuthorized  bool     `json:"local_authorized"`
	ArtifactsMode    string   `json:"artifacts_mode"`
	ArtifactsPath    string   `json:"artifacts_path"`
}

type resolveRequest struct {
	Server          string `json:"server"`
	LocalRoot       string `json:"local_root"`
	LocalAuthorized bool   `json:"local_authorized"`
	ArtifactsMode   string `json:"artifacts_mode"`
	ArtifactsPath   string `json:"artifacts_path"`
}

func reportState(report any) string {
	switch r := report.(type) {
	case engine.InstallProgress:
		return r.State
	case *engine.FullProgress:
		return r.State
	case engine.FullProgress:
		return r.State
	}
	return ""
}

func fullOptions(req resolveRequest, maxDownload int64) engine.FullOptions {
	return engine.FullOptions{
		GitHub:          engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")},
		Target:          req.Server,
		LocalRoot:       req.LocalRoot,
		LocalAuthorized: req.LocalAuthorized,
		ArtifactsMode:   req.ArtifactsMode,
		ArtifactsPath:   req.ArtifactsPath,
		MaxDownload:     maxDownload,
	}
}

func (m *installationJobs) handler(w http.ResponseWriter, r *http.Request) {
	respond := func(code int, v any) { w.WriteHeader(code); _ = json.NewEncoder(w).Encode(v) }
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/installations"), "/")
	if id == "resolve" && r.Method == "POST" {
		var q resolveRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		d.DisallowUnknownFields()
		if d.Decode(&q) != nil || strings.TrimSpace(q.Server) == "" {
			respond(400, apiError{"invalid_request", "server directory required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		defer cancel()
		plan, e := engine.ResolveFull(ctx, m.catalog, fullOptions(q, m.maxDownload))
		if e != nil {
			respond(400, apiError{e.Error(), "cannot resolve components"})
			return
		}
		respond(200, plan)
		return
	}
	if id == "" && r.Method == "POST" {
		var q installRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32768))
		d.DisallowUnknownFields()
		if d.Decode(&q) != nil || strings.TrimSpace(q.Server) == "" {
			respond(400, apiError{"invalid_request", "server directory required"})
			return
		}
		if q.Kind == "" {
			q.Kind = "partial"
		}
		if q.Kind != "partial" && q.Kind != "full" {
			respond(400, apiError{"invalid_request", "unknown installation kind"})
			return
		}
		if q.Kind == "partial" {
			if !q.PartialConfirmed || len(q.Resources) == 0 || len(q.Resources) > 51 {
				respond(400, apiError{"invalid_request", "partial confirmation and resources required"})
				return
			}
		} else {
			if !q.FullConfirmed {
				respond(400, apiError{"invalid_request", "full installation confirmation required"})
				return
			}
			if q.ArtifactsMode != "" && q.ArtifactsMode != "none" && q.ArtifactsMode != "existing" && q.ArtifactsMode != "download" && q.ArtifactsMode != "auto" {
				respond(400, apiError{"invalid_request", "invalid artifacts mode"})
				return
			}
			if (q.ArtifactsMode == "existing" || q.ArtifactsMode == "download") && strings.TrimSpace(q.ArtifactsPath) == "" {
				respond(400, apiError{"invalid_request", "artifacts path required"})
				return
			}
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
		j := &installationJob{Kind: q.Kind, cancel: cancel}
		if q.Kind == "partial" {
			j.Report = engine.InstallProgress{State: "preflight"}
		} else {
			j.Report = engine.FullProgress{State: "preflight"}
		}
		m.jobs[id] = j
		m.active = true
		m.mu.Unlock()
		go func() {
			defer cancel()
			if q.Kind == "partial" {
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
				return
			}
			result, _ := engine.InstallFull(ctx, m.catalog, starter.Assets, q.Server, engine.FullOptions{
				GitHub: engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}, Target: q.Server,
				LocalRoot: q.LocalRoot, LocalAuthorized: q.LocalAuthorized,
				AllowIncomplete: q.AllowIncomplete,
				ArtifactsMode:   q.ArtifactsMode, ArtifactsPath: q.ArtifactsPath,
				MaxDownload: m.maxDownload,
				Progress: func(p engine.FullProgress) {
					p.InstalledResources = append([]string{}, p.InstalledResources...)
					p.Missing = append([]string{}, p.Missing...)
					if p.State != "failed" && p.State != "cancelled" && p.State != "completed" && p.State != "completed_files_only" {
						p.Plan = nil
					}
					m.mu.Lock()
					j.Report = p
					m.mu.Unlock()
				},
			})
			m.mu.Lock()
			j.Report = result
			j.Done = true
			m.active = false
			m.mu.Unlock()
		}()
		respond(202, map[string]string{"id": id, "kind": q.Kind})
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
		} else if reportState(j.Report) == "publishing" {
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
