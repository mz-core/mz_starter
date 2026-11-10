package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Resource struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Group         string     `json:"group"`
	Repository    string     `json:"repository"`
	Visibility    string     `json:"visibility"`
	Source        string     `json:"source"`
	Version       string     `json:"version"`
	Tag           string     `json:"tag"`
	Asset         string     `json:"asset"`
	SHA256        string     `json:"sha256"`
	Dependencies  []string   `json:"dependencies"`
	RequiredFiles []string   `json:"required_files"`
	FileRules     []FileRule `json:"file_rules,omitempty"`
	MutablePolicy string     `json:"mutable_policy"`
	Channel       string     `json:"channel"`
	Availability  string     `json:"availability"`
}

// FileRule is maintained by the Starter, never by the inspected archive.
type FileRule struct {
	Version        string `json:"version"`
	SHA256         string `json:"sha256"`
	Path           string `json:"path"`
	Classification string `json:"classification"`
	Condition      string `json:"condition,omitempty"`
	Evidence       string `json:"evidence"`
	Origin         string `json:"origin"`
}

type Catalog struct {
	SchemaVersion int        `json:"schema_version"`
	Profile       string     `json:"profile"`
	GeneratedFrom string     `json:"generated_from"`
	ObservedAtUTC string     `json:"observed_at_utc"`
	Resources     []Resource `json:"resources"`
}

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var hashPattern = regexp.MustCompile(`^(sha256:)?[a-fA-F0-9]{64}$`)
var groupPattern = regexp.MustCompile(`^\[[a-zA-Z0-9_-]+\]$`)
var versionPattern = regexp.MustCompile(`(?m)^\s*version\s*\(?\s*['"]([^'"]+)['"]`)
var depsBlock = regexp.MustCompile(`(?s)\bdependencies\s*\{([^}]*)\}`)
var depSingle = regexp.MustCompile(`(?m)^\s*dependency\s*\(?\s*['"]([^'"]+)['"]`)
var quoted = regexp.MustCompile(`['"]([^'"]+)['"]`)
var dependencyDeclaration = regexp.MustCompile(`(?m)^\s*dependencies\b`)
var dependencySingleLine = regexp.MustCompile(`(?m)^\s*dependency\b[^\r\n]*`)
var literalDependencyLine = regexp.MustCompile(`^\s*dependency\s*\(?\s*['"][^'"]+['"]\s*\)?\s*(--.*)?$`)
var luaLineComment = regexp.MustCompile(`--[^\r\n]*`)

func LoadCatalog(path string) (Catalog, error) {
	var c Catalog
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	return ReadCatalog(f)
}

func ReadCatalog(r io.Reader) (Catalog, error) {
	var c Catalog
	d := json.NewDecoder(io.LimitReader(r, 4<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("conteudo adicional no catalogo")
	}
	return c, ValidateCatalog(c)
}

func ValidateCatalog(c Catalog) error {
	if c.SchemaVersion != 1 || c.Profile == "" || len(c.Resources) == 0 {
		return fmt.Errorf("catalogo invalido ou vazio")
	}
	seen := map[string]bool{}
	for _, r := range c.Resources {
		if !idPattern.MatchString(r.ID) || !groupPattern.MatchString(r.Group) || r.Name == "" || r.Type == "" || r.Source == "" || r.Visibility == "" || r.Availability == "" || r.MutablePolicy == "" || (r.Channel != "stable" && r.Channel != "prerelease") {
			return fmt.Errorf("entrada invalida: %s", r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("recurso duplicado: %s", r.ID)
		}
		seen[r.ID] = true
		if !map[string]bool{"fivem_resource": true, "cfx": true, "artifact": true, "external": true}[r.Type] || !map[string]bool{"public": true, "private": true, "unknown": true, "artifact": true}[r.Visibility] || !map[string]bool{"github_release": true, "private_release_requires_authorization": true, "unavailable": true, "protected": true, "system_artifact": true, "external": true}[r.Source] {
			return fmt.Errorf("tipo, visibilidade ou fonte invalida: %s", r.ID)
		}
		if r.Repository != "" {
			u, e := url.Parse(r.Repository)
			if e != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(strings.Split(strings.Trim(u.Path, "/"), "/")) != 2 {
				return fmt.Errorf("repositorio invalido: %s", r.ID)
			}
		}
		if r.Asset != "" && (strings.ContainsAny(r.Asset, `/\:`) || !strings.HasSuffix(strings.ToLower(r.Asset), ".zip")) {
			return fmt.Errorf("asset invalido: %s", r.ID)
		}
		if r.Version != "" {
			if _, _, ok := ParseTag(r.ID, "v"+r.Version); !ok {
				return fmt.Errorf("versao invalida: %s", r.ID)
			}
		}
		if r.SHA256 != "" && !hashPattern.MatchString(r.SHA256) {
			return fmt.Errorf("sha256 invalido: %s", r.ID)
		}
		if r.ID == "mz_banguard" && r.Source != "protected" {
			return fmt.Errorf("mz_banguard requer fonte protegida")
		}
		deps := map[string]bool{}
		for _, d := range r.Dependencies {
			if !idPattern.MatchString(d) {
				return fmt.Errorf("dependencia invalida: %s", d)
			}
			if deps[d] {
				return fmt.Errorf("dependencia duplicada: %s", d)
			}
			deps[d] = true
		}
		for _, file := range r.RequiredFiles {
			if file == "" || filepath.IsAbs(file) || strings.ContainsAny(file, `\\:`) || strings.Contains("/"+file+"/", "/../") {
				return fmt.Errorf("arquivo obrigatorio invalido: %s", file)
			}
		}
		for _, rule := range r.FileRules {
			_, validPath := compileResourcePattern(rule.Path)
			u, err := url.Parse(rule.Origin)
			if rule.Version == "" || !hashPattern.MatchString(rule.SHA256) || !validPath || rule.Evidence == "" || err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || !strings.HasPrefix(rule.Origin, r.Repository+"/") || !map[string]bool{"required": true, "optional": true, "conditional": true, "external": true, "inconclusive": true}[rule.Classification] || (rule.Classification == "conditional" && rule.Condition == "") {
				return fmt.Errorf("regra de arquivo invalida: %s", r.ID)
			}
		}
	}
	return nil
}

func ReadBoot(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseBoot(f)
}

func ParseBoot(r io.Reader) ([]string, error) {
	var ids []string
	scan := bufio.NewScanner(io.LimitReader(r, 1<<20))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && (strings.EqualFold(fields[0], "ensure") || strings.EqualFold(fields[0], "start")) && idPattern.MatchString(fields[1]) {
			ids = append(ids, fields[1])
		}
	}
	return ids, scan.Err()
}

func ReadServerBoot(root string) ([]string, error) {
	p, err := noSymlinkPath(root, "cfg", "resources.cfg")
	if err != nil {
		return nil, err
	}
	return ReadBoot(p)
}

// shortcut: somente declaracoes literais; usar parser Lua antes de aceitar manifestos dinamicos.
func ParseManifest(data []byte) (string, []string) {
	s := string(data)
	version := ""
	if m := versionPattern.FindStringSubmatch(s); m != nil {
		version = m[1]
	}
	deps := []string{}
	seen := map[string]bool{}
	add := func(v string) {
		if idPattern.MatchString(v) && !seen[v] {
			seen[v] = true
			deps = append(deps, v)
		}
	}
	for _, b := range depsBlock.FindAllStringSubmatch(s, -1) {
		for _, q := range quoted.FindAllStringSubmatch(b[1], -1) {
			add(q[1])
		}
	}
	for _, d := range depSingle.FindAllStringSubmatch(s, -1) {
		add(d[1])
	}
	return version, deps
}

// HasDynamicDependencies is deliberately conservative; no Lua is executed.
func HasDynamicDependencies(data []byte) bool {
	s := string(data)
	blocks := depsBlock.FindAllStringSubmatch(s, -1)
	if len(dependencyDeclaration.FindAllString(s, -1)) != len(blocks) {
		return true
	}
	for _, block := range blocks {
		content := quoted.ReplaceAllString(block[1], "")
		content = luaLineComment.ReplaceAllString(content, "")
		if strings.Trim(content, " \t\r\n,;") != "" {
			return true
		}
	}
	for _, line := range dependencySingleLine.FindAllString(s, -1) {
		if !literalDependencyLine.MatchString(line) {
			return true
		}
	}
	return false
}
