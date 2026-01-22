// Package build orchestrates the site build pipeline.
package build

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shanepadgett/canopy/internal/config"
	"github.com/shanepadgett/canopy/internal/content"
	"github.com/shanepadgett/canopy/internal/core"
	"github.com/shanepadgett/canopy/internal/markdown"
	"github.com/shanepadgett/canopy/internal/template"
)

// Options configures the build.
type Options struct {
	ConfigPath  string
	OutputDir   string // overrides config if set
	BuildDrafts bool
}

// Stats contains build statistics.
type Stats struct {
	Pages    int
	Sections int
	Tags     int
	Output   string
	Duration time.Duration
}

// Build runs the complete build pipeline.
func Build(opts Options) (*Stats, error) {
	start := time.Now()

	// Phase 1: Load config
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	rootDir := "."
	if opts.ConfigPath != "" {
		rootDir = config.RootDir(opts.ConfigPath)
	} else {
		// Find config to get root dir
		foundPath, err := config.Find()
		if err != nil {
			return nil, err
		}
		rootDir = config.RootDir(foundPath)
	}

	// Apply CLI overrides
	if opts.OutputDir != "" {
		cfg.OutputDir = opts.OutputDir
	}
	buildDrafts := cfg.BuildDrafts || opts.BuildDrafts

	// Phase 2: Collect content
	loader := content.NewLoader(rootDir, cfg, buildDrafts)
	result, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}

	// Check for content errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Printf("error: %s\n", e.Error())
		}
		return nil, fmt.Errorf("%d content errors", len(result.Errors))
	}

	// Build site model
	site := core.NewSite(cfg)
	site.Pages = result.Pages

	// Index pages by section and tags
	for _, page := range site.Pages {
		// Add to section
		section, ok := site.Sections[page.Section]
		if !ok {
			section = &core.Section{Name: page.Section}
			site.Sections[page.Section] = section
		}
		section.Pages = append(section.Pages, page)

		// Add to tags
		for _, tag := range page.Tags {
			site.Tags[tag] = append(site.Tags[tag], page)
		}
	}

	// Phase 3: Render Markdown
	for _, page := range site.Pages {
		result := markdown.Render(page.RawContent)
		page.Body = result.HTML
		page.TOC = result.TOC
		if page.Summary == "" {
			page.Summary = result.Summary
		}
	}

	// Phase 4: Template execute
	templateDir := filepath.Join(rootDir, cfg.TemplateDir)
	engine, err := template.NewEngine(templateDir)
	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	// Collect rendered pages: URL -> HTML
	outputs := make(map[string]string)

	// Render individual pages
	for _, page := range site.Pages {
		html, err := engine.RenderPage(page, site)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", page.SourcePath, err)
		}
		outputs[page.URL] = html
	}

	// Render section index pages
	for _, section := range site.Sections {
		url := "/" + section.Name + "/"
		html, err := engine.RenderList(section, site)
		if err != nil {
			return nil, fmt.Errorf("rendering section %s: %w", section.Name, err)
		}
		outputs[url] = html
	}

	// Render home page
	homeHTML, err := engine.RenderHome(site)
	if err != nil {
		return nil, fmt.Errorf("rendering home: %w", err)
	}
	outputs["/"] = homeHTML

	// Phase 5: Write output
	outputDir := filepath.Join(rootDir, cfg.OutputDir)
	staticDir := filepath.Join(rootDir, cfg.StaticDir)

	writer := NewWriter(outputDir)
	if err := writer.Clean(); err != nil {
		return nil, fmt.Errorf("cleaning output: %w", err)
	}

	for url, html := range outputs {
		if err := writer.WritePage(url, html); err != nil {
			return nil, fmt.Errorf("writing %s: %w", url, err)
		}
	}

	if err := writer.CopyStatic(staticDir); err != nil {
		// Static dir may not exist, that's ok
		if !isNotExist(err) {
			return nil, fmt.Errorf("copying static: %w", err)
		}
	}

	return &Stats{
		Pages:    len(site.Pages),
		Sections: len(site.Sections),
		Tags:     len(site.Tags),
		Output:   outputDir,
		Duration: time.Since(start),
	}, nil
}

func isNotExist(err error) bool {
	return err != nil && err.Error() == "static directory does not exist"
}
