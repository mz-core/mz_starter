package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Asset struct {
	ID     int64  `json:"id"`
	Size   int64  `json:"size"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
	State  string `json:"state"`
}
type Release struct {
	Tag        string  `json:"tag"`
	Version    string  `json:"version"`
	LegacyTag  bool    `json:"legacy_tag"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
	Status     string  `json:"status"`
}
type QueryResult struct {
	Status           string    `json:"status"`
	Releases         []Release `json:"releases"`
	UnrecognizedTags []string  `json:"unrecognized_tags,omitempty"`
	RetryAfter       string    `json:"retry_after,omitempty"`
	RateLimitReset   string    `json:"rate_limit_reset_epoch,omitempty"`
}

// LatestResult is the outcome of the GitHub "latest release" endpoint. Version
// is the version encoded by the selected asset (which may differ from the tag,
// as happens with mz_vehicles), while Tag is the released tag itself.
type LatestResult struct {
	Status         string `json:"status"`
	Tag            string `json:"tag,omitempty"`
	Version        string `json:"version,omitempty"`
	LegacyTag      bool   `json:"legacy_tag,omitempty"`
	Asset          Asset  `json:"asset,omitempty"`
	RetryAfter     string `json:"retry_after,omitempty"`
	RateLimitReset string `json:"rate_limit_reset_epoch,omitempty"`
}
type GitHub struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

func ParseTag(id, tag string) (string, bool, bool) {
	legacy := false
	value := ""
	if strings.HasPrefix(tag, id+"-v") {
		legacy = true
		value = strings.TrimPrefix(tag, id+"-v")
	} else if strings.HasPrefix(tag, "v") {
		value = strings.TrimPrefix(tag, "v")
	}
	if !semverPattern.MatchString(value) {
		return "", false, false
	}
	withoutBuild := strings.SplitN(value, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	if len(parts) == 2 {
		for _, p := range strings.Split(parts[1], ".") {
			if len(p) > 1 && p[0] == '0' {
				numeric := true
				for _, ch := range p {
					if ch < '0' || ch > '9' {
						numeric = false
						break
					}
				}
				if numeric {
					return "", false, false
				}
			}
		}
	}
	return value, legacy, true
}

func (g GitHub) Releases(ctx context.Context, id, repository string) QueryResult {
	result := QueryResult{Status: "network_error", Releases: []Release{}}
	u, err := url.Parse(repository)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		result.Status = "invalid_repository"
		return result
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || !idPattern.MatchString(parts[0]) || parts[1] != id {
		result.Status = "invalid_repository"
		return result
	}
	base := g.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	if g.Token != "" && base != "https://api.github.com" {
		result.Status = "invalid_api_origin"
		return result
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	}
	for page := 1; page <= 10; page++ {
		requestURL := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100&page=%d", strings.TrimRight(base, "/"), parts[0], parts[1], page)
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if e != nil {
			return result
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "mz-starter-planner/1")
		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
		resp, e := client.Do(req)
		if e != nil {
			var n net.Error
			if errors.As(e, &n) && n.Timeout() {
				result.Status = "timeout"
			}
			return result
		}
		if resp.StatusCode != http.StatusOK {
			rateBody := false
			if resp.StatusCode == http.StatusForbidden {
				data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
				rateBody = strings.Contains(strings.ToLower(string(data)), "rate limit")
			}
			switch {
			case resp.StatusCode == 429 || (resp.StatusCode == 403 && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "" || rateBody)):
				result.Status = "rate_limited"
				result.RetryAfter = resp.Header.Get("Retry-After")
				result.RateLimitReset = resp.Header.Get("X-RateLimit-Reset")
			case resp.StatusCode == 401 || resp.StatusCode == 403:
				result.Status = "access_denied"
			case resp.StatusCode == 404:
				result.Status = "not_found_or_no_access"
			default:
				result.Status = "http_error"
			}
			resp.Body.Close()
			return result
		}
		var raw []struct {
			Tag        string  `json:"tag_name"`
			Draft      bool    `json:"draft"`
			Prerelease bool    `json:"prerelease"`
			Assets     []Asset `json:"assets"`
		}
		e = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&raw)
		link := resp.Header.Get("Link")
		resp.Body.Close()
		if e != nil {
			result.Status = "invalid_response"
			return result
		}
		for _, r := range raw {
			if r.Draft {
				continue
			}
			version, legacy, ok := ParseTag(id, r.Tag)
			if !ok {
				result.UnrecognizedTags = append(result.UnrecognizedTags, r.Tag)
				continue
			}
			entry := Release{Tag: r.Tag, Version: version, LegacyTag: legacy, Prerelease: r.Prerelease || strings.Contains(version, "-"), Assets: r.Assets, Status: "integrity_unverified"}
			zipCount := 0
			matching := 0
			digest := ""
			assetState := ""
			for _, a := range r.Assets {
				if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
					zipCount++
					if a.Name == id+"-v"+version+".zip" {
						matching++
						digest = a.Digest
						assetState = a.State
					}
				}
			}
			switch {
			case zipCount == 0:
				entry.Status = "asset_missing"
			case zipCount != 1 || matching != 1:
				entry.Status = "asset_ambiguous"
			case assetState != "" && assetState != "uploaded":
				entry.Status = "asset_unavailable"
			case digest == "":
				entry.Status = "integrity_missing"
			case !hashPattern.MatchString(digest):
				entry.Status = "integrity_invalid"
			}
			result.Releases = append(result.Releases, entry)
		}
		if !strings.Contains(link, `rel="next"`) {
			break
		}
		if page == 10 {
			result.Status = "pagination_incomplete"
			return result
		}
	}
	sort.Slice(result.Releases, func(i, j int) bool { return semverGreater(result.Releases[i].Version, result.Releases[j].Version) })
	result.Status = "ok"
	if len(result.Releases) == 0 {
		if len(result.UnrecognizedTags) > 0 {
			result.Status = "no_recognized_releases"
		} else {
			result.Status = "release_missing"
		}
	}
	return result
}

const latestCacheTTL = 10 * time.Minute

type latestCacheEntry struct {
	result  LatestResult
	expires time.Time
}

var latestCache = struct {
	sync.Mutex
	entries map[string]latestCacheEntry
}{entries: map[string]latestCacheEntry{}}

func latestCacheKey(base, token, id, repository string) string {
	sum := sha256.Sum256([]byte(token))
	return base + "\x00" + hex.EncodeToString(sum[:8]) + "\x00" + id + "\x00" + repository
}

func cachedLatest(key string) (LatestResult, bool) {
	latestCache.Lock()
	defer latestCache.Unlock()
	entry, ok := latestCache.entries[key]
	if !ok || time.Now().After(entry.expires) {
		return LatestResult{}, false
	}
	return entry.result, true
}

func storeLatest(key string, result LatestResult) {
	// Only successful metadata is cached. Transient or hard failures must be
	// retried on the next resolution so a recovered GitHub or a corrected
	// release is picked up immediately.
	if result.Status != "ok" {
		return
	}
	latestCache.Lock()
	defer latestCache.Unlock()
	now := time.Now()
	if len(latestCache.entries) > 128 {
		for k, entry := range latestCache.entries {
			if now.After(entry.expires) {
				delete(latestCache.entries, k)
			}
		}
	}
	latestCache.entries[key] = latestCacheEntry{result: result, expires: now.Add(latestCacheTTL)}
}

// Latest resolves the stable release GitHub currently marks as latest. Drafts
// and pre-releases are rejected. The distribution asset must be uniquely
// identifiable, uploaded and carry a published SHA-256 digest.
func (g GitHub) Latest(ctx context.Context, id, repository string) (result LatestResult) {
	result = LatestResult{Status: "network_error"}
	owner, name, ok := githubRepository(id, repository)
	if !ok {
		result.Status = "invalid_repository"
		return result
	}
	base := g.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	if g.Token != "" && base != "https://api.github.com" {
		result.Status = "invalid_api_origin"
		return result
	}
	cacheKey := latestCacheKey(base, g.Token, id, repository)
	if cached, ok := cachedLatest(cacheKey); ok {
		return cached
	}
	defer func() { storeLatest(cacheKey, result) }()
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	}
	requestURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", strings.TrimRight(base, "/"), owner, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		result.Status = "invalid_request"
		return result
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mz-starter-planner/1")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		var n net.Error
		if errors.As(err, &n) && n.Timeout() {
			result.Status = "timeout"
		}
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rateBody := false
		if resp.StatusCode == http.StatusForbidden {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			rateBody = strings.Contains(strings.ToLower(string(data)), "rate limit")
		}
		switch {
		case resp.StatusCode == 429 || (resp.StatusCode == 403 && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "" || rateBody)):
			result.Status = "rate_limited"
			result.RetryAfter = resp.Header.Get("Retry-After")
			result.RateLimitReset = resp.Header.Get("X-RateLimit-Reset")
		case resp.StatusCode == 401 || resp.StatusCode == 403:
			result.Status = "access_denied"
		case resp.StatusCode == 404:
			result.Status = "release_missing"
		default:
			result.Status = "http_error"
		}
		return result
	}
	var raw struct {
		Tag        string  `json:"tag_name"`
		Draft      bool    `json:"draft"`
		Prerelease bool    `json:"prerelease"`
		Assets     []Asset `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&raw); err != nil {
		result.Status = "invalid_response"
		return result
	}
	if raw.Draft {
		result.Status = "draft_rejected"
		return result
	}
	version, legacy, recognized := ParseTag(id, raw.Tag)
	if !recognized {
		result.Status = "unrecognized_release"
		return result
	}
	if raw.Prerelease || strings.Contains(version, "-") {
		result.Status = "prerelease_rejected"
		return result
	}
	asset, assetVersion, status := selectReleaseAsset(id, version, raw.Assets)
	if status != "integrity_unverified" {
		result.Status = status
		return result
	}
	result.Status = "ok"
	result.Tag = raw.Tag
	result.Version = assetVersion
	result.LegacyTag = legacy
	result.Asset = asset
	return result
}

func githubRepository(id, repository string) (string, string, bool) {
	u, err := url.Parse(repository)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || !idPattern.MatchString(parts[0]) || parts[1] != id {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// releaseDigest normalizes a GitHub asset digest to lowercase hex without the
// sha256: prefix. Only published SHA-256 values are accepted.
func releaseDigest(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if !hashPattern.MatchString(trimmed) {
		return "", false
	}
	return strings.ToLower(strings.TrimPrefix(strings.ToLower(trimmed), "sha256:")), true
}

type releaseAssetCandidate struct {
	asset   Asset
	version string
	rank    int
}

// selectReleaseAsset chooses the distribution ZIP that belongs to the resource.
// Supported names, in priority order: <id>-v<tag>.zip, <id>-v<version>.zip,
// <id>.zip and <id>-<version>.zip. A ZIP named after another resource is never
// considered, and multiple equally valid candidates are rejected.
func selectReleaseAsset(id, tagVersion string, assets []Asset) (Asset, string, string) {
	var candidates []releaseAssetCandidate
	for _, a := range assets {
		version, rank, ok := releaseAssetVersion(id, tagVersion, a.Name)
		if !ok {
			continue
		}
		candidates = append(candidates, releaseAssetCandidate{asset: a, version: version, rank: rank})
	}
	if len(candidates) == 0 {
		return Asset{}, "", "asset_missing"
	}
	best := candidates[0].rank
	for _, c := range candidates {
		if c.rank < best {
			best = c.rank
		}
	}
	var chosen []releaseAssetCandidate
	for _, c := range candidates {
		if c.rank == best {
			chosen = append(chosen, c)
		}
	}
	if len(chosen) != 1 {
		return Asset{}, "", "asset_ambiguous"
	}
	winner := chosen[0]
	if winner.asset.State != "" && winner.asset.State != "uploaded" {
		return Asset{}, "", "asset_unavailable"
	}
	if winner.asset.ID <= 0 || winner.asset.Size <= 0 {
		return Asset{}, "", "invalid_asset_metadata"
	}
	if strings.TrimSpace(winner.asset.Digest) == "" {
		return Asset{}, "", "integrity_missing"
	}
	if _, ok := releaseDigest(winner.asset.Digest); !ok {
		return Asset{}, "", "integrity_invalid"
	}
	return winner.asset, winner.version, "integrity_unverified"
}

func releaseAssetVersion(id, tagVersion, name string) (string, int, bool) {
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".zip") {
		return "", 0, false
	}
	base := strings.TrimSuffix(lower, ".zip")
	idLower := strings.ToLower(id)
	switch {
	case tagVersion != "" && base == idLower+"-v"+strings.ToLower(tagVersion):
		return tagVersion, 0, true
	case strings.HasPrefix(base, idLower+"-v"):
		version := name[len(id)+2 : len(name)-4]
		if semverPattern.MatchString(version) {
			return version, 1, true
		}
	case base == idLower:
		if tagVersion != "" {
			return tagVersion, 2, true
		}
	case strings.HasPrefix(base, idLower+"-"):
		version := name[len(id)+1 : len(name)-4]
		if semverPattern.MatchString(version) {
			return version, 3, true
		}
	}
	return "", 0, false
}

func semverGreater(a, b string) bool {
	aa := strings.SplitN(strings.SplitN(a, "+", 2)[0], "-", 2)
	bb := strings.SplitN(strings.SplitN(b, "+", 2)[0], "-", 2)
	av := strings.Split(aa[0], ".")
	bv := strings.Split(bb[0], ".")
	for i := 0; i < 3; i++ {
		if len(av[i]) != len(bv[i]) {
			return len(av[i]) > len(bv[i])
		}
		if av[i] != bv[i] {
			return av[i] > bv[i]
		}
	}
	if len(aa) != len(bb) {
		return len(aa) == 1
	}
	if len(aa) == 1 {
		return false
	}
	ap := strings.Split(aa[1], ".")
	bp := strings.Split(bb[1], ".")
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if ap[i] == bp[i] {
			continue
		}
		aNumeric := numericIdentifier(ap[i])
		bNumeric := numericIdentifier(bp[i])
		if aNumeric && !bNumeric {
			return false
		}
		if !aNumeric && bNumeric {
			return true
		}
		if aNumeric && bNumeric && len(ap[i]) != len(bp[i]) {
			return len(ap[i]) > len(bp[i])
		}
		return ap[i] > bp[i]
	}
	return len(ap) > len(bp)
}

func numericIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
