// Package content handles content discovery and loading.
package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shanepadgett/canopy/internal/core"
)

// Loader discovers and loads content files into pages.
type Loader struct {
	rootDir     string
	contentDir  string
	config      core.Config
	buildDrafts bool
}

// NewLoader creates a content loader.
func NewLoader(rootDir string, cfg core.Config, buildDrafts bool) *Loader {
	return &Loader{
		rootDir:     rootDir,
		contentDir:  filepath.Join(rootDir, cfg.ContentDir),
		config:      cfg,
		buildDrafts: buildDrafts,
	}
}

// LoadResult contains the loaded pages and any errors encountered.
type LoadResult struct {
	Pages  []*core.Page
	Errors []LoadError
}

// LoadError represents an error loading a specific file.
type LoadError struct {
	Path    string
	Message string
}

func (e LoadError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Load discovers all content and returns pages.
func (l *Loader) Load() (*LoadResult, error) {
	result := &LoadResult{}

	err := filepath.WalkDir(l.contentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		page, loadErr := l.loadPage(path)
		if loadErr != nil {
			result.Errors = append(result.Errors, *loadErr)
			return nil
		}

		// Skip drafts unless buildDrafts is true
		if page.Draft && !l.buildDrafts {
			return nil
		}

		result.Pages = append(result.Pages, page)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking content dir: %w", err)
	}

	// Sort pages by date (newest first), then by weight, then by title
	sort.Slice(result.Pages, func(i, j int) bool {
		pi, pj := result.Pages[i], result.Pages[j]

		// By date descending
		if !pi.Date.Equal(pj.Date) {
			return pi.Date.After(pj.Date)
		}

		// By weight ascending
		if pi.Weight != pj.Weight {
			return pi.Weight < pj.Weight
		}

		// By title ascending
		return pi.Title < pj.Title
	})

	return result, nil
}

func (l *Loader) loadPage(path string) (*core.Page, *LoadError) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &LoadError{Path: path, Message: fmt.Sprintf("reading file: %v", err)}
	}

	// Parse front matter
	fm, body, err := core.ParseFrontMatter(data)
	if err != nil {
		return nil, &LoadError{Path: path, Message: fmt.Sprintf("parsing front matter: %v", err)}
	}

	// Derive relative path from content dir
	relPath, err := filepath.Rel(l.contentDir, path)
	if err != nil {
		return nil, &LoadError{Path: path, Message: fmt.Sprintf("computing relative path: %v", err)}
	}

	// Derive section from first path segment
	section := deriveSection(relPath)

	// Apply section defaults
	if sectionCfg, ok := l.config.Sections[section]; ok {
		fm.ApplyDefaults(sectionCfg.Defaults)
	}

	// Validate required fields
	if sectionCfg, ok := l.config.Sections[section]; ok {
		if errs := fm.Validate(sectionCfg.Required); len(errs) > 0 {
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			return nil, &LoadError{
				Path:    path,
				Message: fmt.Sprintf("validation failed: %s", strings.Join(msgs, ", ")),
			}
		}
	}

	// Derive slug
	slug := deriveSlug(relPath, fm.Slug)

	// Compute URL
	url := computeURL(l.config, section, slug, fm.Date)

	// Build page
	page := &core.Page{
		SourcePath:  relPath,
		URL:         url,
		Slug:        slug,
		Title:       fm.Title,
		Description: fm.Description,
		RawContent:  string(body),
		Section:     section,
		Tags:        fm.Tags,
		Draft:       fm.Draft,
		Date:        fm.Date,
		Aliases:     fm.Aliases,
		Weight:      fm.Weight,
		Params:      fm.Extra,
	}

	return page, nil
}

// deriveSection extracts the section from the relative path.
// content/blog/post.md -> "blog"
// content/guides/intro/start.md -> "guides"
func deriveSection(relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}

// deriveSlug determines the page slug.
// Front matter slug takes precedence over filename.
func deriveSlug(relPath, fmSlug string) string {
	if fmSlug != "" {
		return fmSlug
	}

	// Use filename without extension
	base := filepath.Base(relPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
