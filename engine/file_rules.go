package engine

import (
	"archive/zip"
	"strings"
)

type FileFinding struct {
	Path           string `json:"path"`
	Classification string `json:"classification"`
	Condition      string `json:"condition,omitempty"`
	Evidence       string `json:"evidence"`
	Origin         string `json:"origin"`
	Present        bool   `json:"present"`
}

// The curated rule is bound to the complete ZIP digest, not to a claim inside it.
func inspectFileRules(filename string, r Resource, version, digest string) []FileFinding {
	var matching []FileRule
	for _, rule := range r.FileRules {
		if rule.Version == version && strings.EqualFold(strings.TrimPrefix(rule.SHA256, "sha256:"), digest) {
			matching = append(matching, rule)
		}
	}
	if len(matching) == 0 {
		return nil
	}
	z, err := zip.OpenReader(filename)
	if err != nil {
		return nil
	}
	defer z.Close()
	var findings []FileFinding
	for _, rule := range matching {
		re, ok := compileResourcePattern(rule.Path)
		if !ok {
			continue
		}
		present := false
		for _, file := range z.File {
			if !file.FileInfo().IsDir() && strings.HasPrefix(file.Name, r.ID+"/") && re.MatchString(strings.TrimPrefix(file.Name, r.ID+"/")) {
				present = true
				break
			}
		}
		findings = append(findings, FileFinding{rule.Path, rule.Classification, rule.Condition, rule.Evidence, rule.Origin, present})
	}
	return findings
}
