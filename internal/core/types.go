// Package core defines the central data types for Canopy.
package core

import (
	"time"
)

// Site represents the entire site being generated.
type Site struct {
	Config   Config
	Sections map[string]*Section
	Pages    []*Page
	Tags     map[string][]*Page
}

// NewSite creates a new site with initialized maps.
func NewSite(cfg Config) *Site {
	return &Site{
		Config:   cfg,
		Sections: make(map[string]*Section),
		Tags:     make(map[string][]*Page),
	}
}

// Section represents a content section (blog, guides, etc.).
type Section struct {
	Name  string
	Pages []*Page
}

// Page represents a single page in the site.
type Page struct {
	// Identity
	SourcePath string // relative path to source file
	URL        string // final URL path
	Slug       string

	// Content
	Title       string
	Description string
	Body        string // rendered HTML
	RawContent  string // original markdown (without front matter)
	Summary     string // plain text excerpt
	TOC         []TOCEntry

	// Classification
	Section string
	Tags    []string
	Draft   bool

	// Timestamps
	Date    time.Time
	LastMod time.Time
	Aliases []string // redirect URLs

	// Navigation (for docs)
	Weight   int
	PrevPage *Page
	NextPage *Page

	// Arbitrary front matter fields for templates
	Params map[string]any
}

// TOCEntry represents a table of contents item.
type TOCEntry struct {
	Level int
	ID    string
	Title string
}

// Config holds site-wide configuration from site.json.
type Config struct {
	// Required
	Name    string `json:"name"`
	BaseURL string `json:"baseURL"`

	// Optional with defaults
	Title       string `json:"title"`
	Description string `json:"description"`
	Language    string `json:"language"`

	// Directories (relative to site root)
	ContentDir  string `json:"contentDir"`
	TemplateDir string `json:"templateDir"`
	StaticDir   string `json:"staticDir"`
	OutputDir   string `json:"outputDir"`

	// Build options
	BuildDrafts bool `json:"buildDrafts"`

	// Search options
	Search SearchConfig `json:"search"`

	// Permalink styles per section
	Permalinks map[string]string `json:"permalinks"`

	// Navigation structure
	Nav []NavItem `json:"nav"`

	// Section-specific front matter schemas
	Sections map[string]SectionConfig `json:"sections"`

	// Arbitrary config for templates
	Params map[string]any `json:"params"`
}

// NavItem represents a navigation entry.
type NavItem struct {
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Weight   int       `json:"weight"`
	Children []NavItem `json:"children,omitempty"`
}

// SectionConfig defines per-section settings.
type SectionConfig struct {
	// Default front matter values
	Defaults map[string]any `json:"defaults"`

	// Required fields (build fails if missing)
	Required []string `json:"required"`

	// Permalink pattern override
	Permalink string `json:"permalink"`
}

// SearchConfig defines search behavior.
type SearchConfig struct {
	Enabled bool `json:"enabled"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Language:    "en",
		ContentDir:  "content",
		TemplateDir: "templates",
		StaticDir:   "static",
		OutputDir:   "public",
		Search: SearchConfig{
			Enabled: true,
		},
		Permalinks: make(map[string]string),
		Sections:   make(map[string]SectionConfig),
		Params:     make(map[string]any),
	}
}
