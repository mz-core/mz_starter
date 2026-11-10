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

	"github.com/mz-core/mz_starter/engine"
)

type job struct {
	Report engine.PackageReport `json:"report"`
	Done   bool                 `json:"done"`
	cancel context.CancelFunc
}
type verificationJobs struct {
	mu          sync.Mutex
	jobs        map[string]*job
	active      int
	catalog     engine.Catalog
	maxDownload int64
	serverRoots []string
}

func (m *verificationJobs) addServer(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(path) == "" {
		return true
	}
	for _, v := range m.serverRoots {
		if v == path {
			return true
		}
	}
	if m.active > 0 {
		return false
	}
	m.serverRoots = append(m.serverRoots, path)
	return true
}

func (m *verificationJobs) handler(w http.ResponseWriter, r *http.Request) {
	respond := func(status int, v any) { w.WriteHeader(status); _ = json.NewEncoder(w).Encode(v) }
	id := strings.TrimPrefix(r.URL.Path, "/api/verifications")
	id = strings.TrimPrefix(id, "/")
	if id == "" && r.Method == "POST" {
		var q struct {
			Resource string `json:"resource"`
			Tag      string `json:"tag"`
			Server   string `json:"server"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		d.DisallowUnknownFields()
		if d.Decode(&q) != nil || len(q.Tag) > 128 {
			respond(400, apiError{"invalid_request", "invalid request"})
			return
		}
		var res *engine.Resource
		for i := range m.catalog.Resources {
			if m.catalog.Resources[i].ID == q.Resource {
				res = &m.catalog.Resources[i]
				break
			}
		}
		if res == nil {
			respond(404, apiError{"resource_not_found", "resource not found"})
			return
		}
		if !m.addServer(q.Server) {
			respond(409, apiError{"verification_in_progress", "wait for active verification"})
			return
		}
		var b [16]byte
		if _, e := rand.Read(b[:]); e != nil {
			respond(500, apiError{"random_error", "cannot start"})
			return
		}
		id = hex.EncodeToString(b[:])
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		m.mu.Lock()
		if len(m.jobs) >= 50 {
			for key, old := range m.jobs {
				if old.Done {
					delete(m.jobs, key)
					break
				}
			}
		}
		if m.active >= 2 || len(m.jobs) >= 50 {
			m.mu.Unlock()
			cancel()
			respond(429, apiError{"verification_limit", "verification limit reached"})
			return
		}
		m.active++
		roots := append([]string{}, m.serverRoots...)
		j := &job{Report: engine.PackageReport{Resource: res.ID, State: "queued"}, cancel: cancel}
		m.jobs[id] = j
		m.mu.Unlock()
		resource := *res
		go func() {
			defer cancel()
			result := engine.VerifyPackage(ctx, resource, q.Tag, engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}, engine.VerifyOptions{MaxDownload: m.maxDownload, ForbiddenRoots: roots, Progress: func(p engine.PackageReport) { m.mu.Lock(); j.Report = p; m.mu.Unlock() }})
			m.mu.Lock()
			j.Report = result
			j.Done = true
			m.active--
			m.mu.Unlock()
		}()
		respond(202, map[string]string{"id": id})
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[id]
	if j == nil {
		respond(404, apiError{"not_found", "verification not found"})
		return
	}
	switch r.Method {
	case "GET":
		respond(200, j)
	case "DELETE":
		if j.Done {
			delete(m.jobs, id)
		} else {
			j.cancel()
		}
		respond(200, map[string]string{"status": "cancel_requested"})
	default:
		respond(405, apiError{"method", "method not allowed"})
	}
}
