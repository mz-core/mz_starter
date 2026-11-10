package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	starter "github.com/mz-core/mz_starter"
	"github.com/mz-core/mz_starter/engine"
)

type planRequest struct {
	Server           string `json:"server"`
	Workspace        string `json:"workspace"`
	Online           bool   `json:"online"`
	InstallationMode string `json:"installation_mode"`
	Profile          string `json:"profile"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := flags.Int("port", 0, "porta loopback (0 automatica)")
	noOpen := flags.Bool("no-open", false, "nao abrir navegador")
	maxDownload := flags.Int("max-download-mib", 512, "limite de download em MiB (1-2048)")
	flags.Parse(args)
	if flags.NArg() != 0 || *port < 0 || *port > 65535 || *maxDownload < 1 || *maxDownload > 2048 {
		fail("argumentos invalidos")
	}
	f, e := starter.Assets.Open("catalog/resources.json")
	if e != nil {
		fail(e.Error())
	}
	defer f.Close()
	catalog, e := engine.ReadCatalog(f)
	if e != nil {
		fail(e.Error())
	}
	if e = engine.ValidateBootstrap(catalog, starter.Assets); e != nil {
		fail(e.Error())
	}
	listener, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if e != nil {
		fail("Nao foi possivel iniciar o servidor local: " + e.Error())
	}
	defer listener.Close()
	b := make([]byte, 32)
	if _, e = rand.Read(b); e != nil {
		fail(e.Error())
	}
	token := hex.EncodeToString(b)
	host := listener.Addr().String()
	origin := "http://" + host
	dist, e := fs.Sub(starter.Assets, "web/dist")
	if e != nil {
		fail(e.Error())
	}
	mux := http.NewServeMux()
	jobs := &verificationJobs{jobs: map[string]*job{}, catalog: catalog, maxDownload: int64(*maxDownload) << 20}
	mux.HandleFunc("/api/verifications", jobs.handler)
	mux.HandleFunc("/api/verifications/", jobs.handler)
	installJobs := &installationJobs{jobs: map[string]*installationJob{}, catalog: catalog, maxDownload: int64(*maxDownload) << 20}
	mux.HandleFunc("/api/installations", installJobs.handler)
	mux.HandleFunc("/api/installations/", installJobs.handler)
	writeError := func(w http.ResponseWriter, status int, code, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(apiError{code, message})
	}
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writeError(w, 405, "method", "method not allowed")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"catalog": catalog, "os": runtime.GOOS, "bootstrap_files": engine.BootstrapFiles, "required_configuration": engine.BootstrapConfigurations, "phases": engine.PlanningPhases})
	})
	mux.HandleFunc("/api/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeError(w, 405, "method", "method not allowed")
			return
		}
		var q planRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		dec.DisallowUnknownFields()
		if dec.Decode(&q) != nil || strings.TrimSpace(q.Server) == "" {
			writeError(w, 400, "invalid_request", "invalid server directory")
			return
		}
		if !jobs.addServer(q.Server) {
			writeError(w, 409, "verification_in_progress", "wait for active verification")
			return
		}
		var boot []string
		if q.InstallationMode == "" {
			var err error
			boot, err = engine.ReadServerBoot(q.Server)
			if err != nil {
				writeError(w, 400, "boot_unavailable", "cannot read cfg/resources.cfg")
				return
			}
			if len(boot) == 0 {
				writeError(w, 400, "boot_empty", "no resources found in cfg/resources.cfg")
				return
			}
		} else if _, err := engine.PlanInstallation(catalog, starter.Assets, q.Server, q.Workspace, q.InstallationMode, q.Profile, nil); err != nil {
			writeError(w, 400, err.Error(), "invalid installation target")
			return
		}
		remote := map[string]engine.QueryResult{}
		if q.Online {
			g := engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}
			limited := false
			for _, res := range catalog.Resources {
				if r.Context().Err() != nil {
					return
				}
				if res.Source == "protected" || res.Type != "fivem_resource" {
					continue
				}
				repo := res.Repository
				if repo == "" {
					repo = "https://github.com/mz-core/" + res.ID
				}
				if limited {
					remote[res.ID] = engine.QueryResult{Status: "not_queried_rate_limit", Releases: []engine.Release{}}
					continue
				}
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				remote[res.ID] = g.Releases(ctx, res.ID, repo)
				cancel()
				limited = remote[res.ID].Status == "rate_limited"
			}
		}
		if q.InstallationMode == "" {
			_ = json.NewEncoder(w).Encode(engine.BuildPlan(catalog, boot, q.Server, q.Workspace, remote))
			return
		}
		p, err := engine.PlanInstallation(catalog, starter.Assets, q.Server, q.Workspace, q.InstallationMode, q.Profile, remote)
		if err != nil {
			writeError(w, 400, err.Error(), "invalid installation target")
			return
		}
		_ = json.NewEncoder(w).Encode(p)
	})
	mux.Handle("/", http.FileServer(http.FS(dist)))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remote, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(remote) == nil || !net.ParseIP(remote).IsLoopback() || r.Host != host {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			o := r.Header.Get("Origin")
			if (o != "" && o != origin) || (r.Method != "GET" && o != origin) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-MZ-Session")), []byte(token)) != 1 {
				writeError(w, 403, "forbidden", "invalid session or origin")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
	url := origin + "/#" + token
	fmt.Println("MZ Starter:", url)
	if !*noOpen {
		var c *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "linux":
			c = exec.Command("xdg-open", url)
		}
		if c != nil {
			if err := c.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "Nao foi possivel abrir o navegador. Abra o endereco exibido acima manualmente.")
			} else {
				go func() {
					if c.Wait() != nil {
						fmt.Fprintln(os.Stderr, "Nao foi possivel abrir o navegador. Abra o endereco exibido acima manualmente.")
					}
				}()
			}
		} else {
			fmt.Fprintln(os.Stderr, "Abra o endereco exibido acima manualmente.")
		}
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		fail(err.Error())
	}
}
