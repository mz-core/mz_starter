package engine

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
)

type luaToken struct {
	kind  byte
	value string
}

// shortcut: a conservative Lua subset only; upgrade to a static parser before trusting other syntax.
func manifestTokens(src string) ([]luaToken, error) {
	var out []luaToken
	for i := 0; i < len(src); {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		if strings.HasPrefix(src[i:], "--[[") {
			end := strings.Index(src[i+4:], "]]")
			if end < 0 {
				return out, fmt.Errorf("unclosed comment")
			}
			i += end + 6
			continue
		}
		if strings.HasPrefix(src[i:], "--") {
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				break
			}
			i += end
			continue
		}
		if strings.HasPrefix(src[i:], "[[") {
			end := strings.Index(src[i+2:], "]]")
			if end < 0 {
				return out, fmt.Errorf("unclosed string")
			}
			out = append(out, luaToken{'s', src[i+2 : i+2+end]})
			i += end + 4
			continue
		}
		if c == '\'' || c == '"' {
			q := c
			i++
			var b strings.Builder
			closed := false
			for i < len(src) {
				if src[i] == '\\' {
					return out, fmt.Errorf("unsupported string escape")
				}
				if src[i] == q {
					i++
					closed = true
					break
				}
				b.WriteByte(src[i])
				i++
			}
			if !closed {
				return out, fmt.Errorf("unclosed string")
			}
			out = append(out, luaToken{'s', b.String()})
			continue
		}
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			start := i
			i++
			for i < len(src) {
				x := rune(src[i])
				if x != '_' && !unicode.IsLetter(x) && !unicode.IsDigit(x) {
					break
				}
				i++
			}
			out = append(out, luaToken{'i', src[start:i]})
			continue
		}
		out = append(out, luaToken{'p', string(c)})
		i++
	}
	return out, nil
}

func matchResourcePattern(pattern, name string) (bool, bool) {
	re, ok := compileResourcePattern(pattern)
	if !ok {
		return false, false
	}
	return re.MatchString(name), true
}

func compileResourcePattern(pattern string) (*regexp.Regexp, bool) {
	if len(pattern) > 1024 || strings.ContainsAny(pattern, `\:[]?`) || strings.HasPrefix(pattern, "/") || path.Clean(pattern) != pattern || strings.HasPrefix(pattern, "../") {
		return nil, false
	}
	var b strings.Builder
	b.WriteByte('^')
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					b.WriteString("(?:.*/)?")
					i++
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		} else {
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
	}
	b.WriteByte('$')
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, false
	}
	return re, true
}

func inspectReferences(source string, files map[string]bool) (missing, unknown []string) {
	tokens, err := manifestTokens(source)
	if err != nil {
		return nil, []string{"manifest_parse_inconclusive"}
	}
	fields := map[string]bool{"client_script": true, "client_scripts": true, "server_script": true, "server_scripts": true, "shared_script": true, "shared_scripts": true, "files": true, "file": true, "ui_page": true, "data_file": true}
	count := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.kind != 'i' || !fields[tok.value] {
			continue
		}
		j := i + 1
		for j < len(tokens) && tokens[j].value == "(" {
			j++
		}
		var refs []string
		dynamic := false
		if j >= len(tokens) {
			dynamic = true
		} else if tokens[j].value == "{" {
			depth := 1
			j++
			for j < len(tokens) && depth > 0 {
				v := tokens[j]
				switch {
				case v.value == "{":
					depth++
				case v.value == "}":
					depth--
				case depth > 0 && v.kind == 's':
					refs = append(refs, v.value)
				case depth > 0 && v.kind == 'i':
					dynamic = true
				case depth > 0 && v.kind == 'p' && v.value != "," && v.value != ";":
					dynamic = true
				}
				j++
			}
			if depth != 0 {
				dynamic = true
			}
		} else if tok.value == "data_file" {
			if j < len(tokens) && tokens[j].kind == 's' {
				j++
				if j < len(tokens) && tokens[j].value == "," {
					j++
				}
				if j < len(tokens) && tokens[j].kind == 's' {
					refs = append(refs, tokens[j].value)
					if j+1 < len(tokens) && tokens[j+1].value == "." {
						dynamic = true
					}
				} else {
					dynamic = true
				}
			} else {
				dynamic = true
			}
		} else if tokens[j].kind == 's' {
			refs = append(refs, tokens[j].value)
			if j+1 < len(tokens) && tokens[j+1].value == "." {
				dynamic = true
			}
		} else {
			dynamic = true
		}
		if dynamic {
			unknown = append(unknown, "manifest_dynamic_reference:"+tok.value)
		}
		for _, ref := range refs {
			count++
			if count > 256 {
				return missing, append(unknown, "manifest_reference_limit")
			}
			if strings.HasPrefix(ref, "@") || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
				unknown = append(unknown, "external_reference_unverified:"+ref)
				continue
			}
			re, valid := compileResourcePattern(ref)
			matched := false
			if valid {
				if !strings.Contains(ref, "*") {
					matched = files[ref]
				} else {
					for name := range files {
						if re.MatchString(name) {
							matched = true
							break
						}
					}
				}
			}
			if !valid {
				unknown = append(unknown, "reference_pattern_unsupported:"+ref)
			} else if !matched {
				if strings.Contains(ref, "*") {
					unknown = append(unknown, "reference_glob_unmatched:"+ref)
				} else {
					missing = append(missing, "reference_missing:"+ref)
				}
			}
		}
	}
	return missing, unknown
}
