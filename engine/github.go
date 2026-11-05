package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Asset struct {
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
