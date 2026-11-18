package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const officialReleaseCatalog = "https://raw.githubusercontent.com/mz-core/mz_starter/main/catalog/releases.json"

// The remote document only supplies release metadata; local installation policy remains authoritative.
func fetchReleaseCatalog(ctx context.Context, catalog Catalog, address string, client *http.Client) (map[string]Resource, error) {
	if address == "" {
		address = officialReleaseCatalog
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "mz-starter")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha de conexao")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var doc struct {
		Schema    int                                                `json:"schema_version"`
		Resources []struct{ ID, Version, Tag, Asset, SHA256 string } `json:"resources"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil || doc.Schema != 1 {
		return nil, fmt.Errorf("formato invalido")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("conteudo adicional")
	}
	allowed := make(map[string]Resource)
	for _, r := range catalog.Resources {
		if r.Type == "fivem_resource" && r.Source != "protected" {
			allowed[r.ID] = r
		}
	}
	result := make(map[string]Resource)
	for _, entry := range doc.Resources {
		r, ok := allowed[entry.ID]
		if !ok {
			continue
		}
		tagVersion, _, validTag := ParseTag(entry.ID, entry.Tag)
		_, _, validVersion := ParseTag(entry.ID, "v"+entry.Version)
		digest, validDigest := catalogDigest(entry.SHA256)
		if _, duplicate := result[entry.ID]; duplicate || !validTag || strings.Contains(tagVersion, "-") || !validVersion || strings.Contains(entry.Version, "-") || entry.Asset != entry.ID+"-v"+entry.Version+".zip" || !validDigest {
			return nil, fmt.Errorf("release invalida: %s", entry.ID)
		}
		r.Repository = repositoryForMZ(r)
		owner, _, validRepo := githubRepository(r.ID, r.Repository)
		if !validRepo || owner != "mz-core" {
			return nil, fmt.Errorf("repositorio invalido: %s", entry.ID)
		}
		r.Version, r.Tag, r.Asset, r.SHA256 = entry.Version, entry.Tag, entry.Asset, digest
		result[entry.ID] = r
	}
	return result, nil
}
