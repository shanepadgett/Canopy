package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FrontMatter holds parsed front matter from a content file.
type FrontMatter struct {
	Title       string    `json:"title"`
	Date        time.Time `json:"date"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Draft       bool      `json:"draft"`
	Aliases     []string  `json:"aliases"`
	Weight      int       `json:"weight"`

	// Extra holds any additional fields not in the struct
	Extra map[string]any `json:"-"`
}

// ParseFrontMatter extracts front matter from content.
// Supports JSON front matter delimited by ---.
// Returns the front matter and the remaining content.
func ParseFrontMatter(content []byte) (FrontMatter, []byte, error) {
	var fm FrontMatter
	fm.Extra = make(map[string]any)

	content = bytes.TrimSpace(content)

	// Check for front matter delimiter
	if !bytes.HasPrefix(content, []byte("---")) {
		return fm, content, nil
	}

	// Find closing delimiter
	rest := content[3:]
	rest = bytes.TrimPrefix(rest, []byte("\n"))

	endIdx := bytes.Index(rest, []byte("\n---"))
	if endIdx == -1 {
		return fm, content, errors.New("unclosed front matter: missing closing ---")
	}

	fmData := rest[:endIdx]
	body := rest[endIdx+4:]
	body = bytes.TrimPrefix(body, []byte("\n"))

	// Try JSON first
	if err := parseJSONFrontMatter(fmData, &fm); err != nil {
		// Fall back to simple key: value parsing
		if err := parseSimpleFrontMatter(fmData, &fm); err != nil {
			return fm, body, fmt.Errorf("parsing front matter: %w", err)
		}
	}

	return fm, body, nil
}

func parseJSONFrontMatter(data []byte, fm *FrontMatter) error {
	// First unmarshal into struct fields
	if err := json.Unmarshal(data, fm); err != nil {
		return err
	}

	// Then unmarshal again to capture extra fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Remove known fields
	known := []string{"title", "date", "slug", "description", "tags", "draft", "aliases", "weight"}
	for _, k := range known {
		delete(raw, k)
	}

	fm.Extra = raw
	return nil
}

func parseSimpleFrontMatter(data []byte, fm *FrontMatter) error {
	lines := bytes.Split(data, []byte("\n"))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		idx := bytes.Index(line, []byte(":"))
		if idx == -1 {
			continue
		}

		key := strings.ToLower(string(bytes.TrimSpace(line[:idx])))
		val := string(bytes.TrimSpace(line[idx+1:]))

		switch key {
		case "title":
			fm.Title = unquote(val)
		case "description":
			fm.Description = unquote(val)
		case "slug":
			fm.Slug = unquote(val)
		case "draft":
			fm.Draft = val == "true" || val == "yes"
		case "date":
			t, err := parseDate(val)
			if err == nil {
				fm.Date = t
			}
		case "tags":
			fm.Tags = parseList(val)
		case "weight":
			fmt.Sscanf(val, "%d", &fm.Weight)
		default:
			fm.Extra[key] = unquote(val)
		}
	}

	return nil
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func parseDate(s string) (time.Time, error) {
	s = unquote(s)
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

func parseList(s string) []string {
	s = strings.TrimSpace(s)
	// Handle JSON array syntax
	if strings.HasPrefix(s, "[") {
		var list []string
		if json.Unmarshal([]byte(s), &list) == nil {
			return list
		}
	}
	// Handle comma-separated
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = unquote(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ValidationError represents a front matter validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks front matter against section requirements.
func (fm *FrontMatter) Validate(required []string) []ValidationError {
	var errs []ValidationError

	for _, field := range required {
		switch field {
		case "title":
			if fm.Title == "" {
				errs = append(errs, ValidationError{Field: "title", Message: "required"})
			}
		case "date":
			if fm.Date.IsZero() {
				errs = append(errs, ValidationError{Field: "date", Message: "required"})
			}
		case "description":
			if fm.Description == "" {
				errs = append(errs, ValidationError{Field: "description", Message: "required"})
			}
		case "slug":
			if fm.Slug == "" {
				errs = append(errs, ValidationError{Field: "slug", Message: "required"})
			}
		default:
			// Check extra fields
			if _, ok := fm.Extra[field]; !ok {
				errs = append(errs, ValidationError{Field: field, Message: "required"})
			}
		}
	}

	return errs
}

// ApplyDefaults fills in missing fields from defaults.
func (fm *FrontMatter) ApplyDefaults(defaults map[string]any) {
	for k, v := range defaults {
		switch k {
		case "draft":
			if b, ok := v.(bool); ok && !fm.Draft {
				fm.Draft = b
			}
		default:
			if _, exists := fm.Extra[k]; !exists {
				fm.Extra[k] = v
			}
		}
	}
}
