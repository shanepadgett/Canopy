// Package build orchestrates the site build pipeline.
package build

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
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

	// Render tag index pages
	if len(site.Tags) > 0 {
		var tags []string
		for tag := range site.Tags {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		tagPages := make([]*core.Page, 0, len(tags))

		for _, tag := range tags {
			pages := site.Tags[tag]
			section := &core.Section{Name: tag, Pages: pages}
			url := "/tags/" + tag + "/"
			html, err := engine.RenderList(section, site)
			if err != nil {
				return nil, fmt.Errorf("rendering tag %s: %w", tag, err)
			}
			outputs[url] = html

			tagPages = append(tagPages, &core.Page{Title: tag, URL: url})
		}

		tagIndex := &core.Section{Name: "tags", Pages: tagPages}
		tagIndexHTML, err := engine.RenderList(tagIndex, site)
		if err != nil {
			return nil, fmt.Errorf("rendering tags index: %w", err)
		}
		outputs["/tags/"] = tagIndexHTML
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

	if err := writer.WriteFile("robots.txt", renderRobots(cfg)); err != nil {
		return nil, fmt.Errorf("writing robots.txt: %w", err)
	}

	if err := writer.WriteFile("sitemap.xml", renderSitemap(cfg, outputs, site.Pages)); err != nil {
		return nil, fmt.Errorf("writing sitemap.xml: %w", err)
	}

	if rss, err := renderRSS(cfg, site.Pages); err != nil {
		return nil, fmt.Errorf("writing rss.xml: %w", err)
	} else if err := writer.WriteFile("rss.xml", rss); err != nil {
		return nil, fmt.Errorf("writing rss.xml: %w", err)
	}

	if cfg.Search.Enabled {
		if err := writer.WriteFile("search.json", renderSearchIndex(site.Pages)); err != nil {
			return nil, fmt.Errorf("writing search.json: %w", err)
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

func renderRobots(cfg core.Config) string {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	return fmt.Sprintf("User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", baseURL)
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func renderSitemap(cfg core.Config, outputs map[string]string, pages []*core.Page) string {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	lastMods := make(map[string]string)
	for _, page := range pages {
		if !page.Date.IsZero() {
			lastMods[page.URL] = page.Date.Format("2006-01-02")
		}
	}

	urls := make([]sitemapURL, 0, len(outputs))
	for url := range outputs {
		entry := sitemapURL{
			Loc: baseURL + url,
		}
		if lastMod, ok := lastMods[url]; ok {
			entry.LastMod = lastMod
		}
		urls = append(urls, entry)
	}

	sort.Slice(urls, func(i, j int) bool {
		return urls[i].Loc < urls[j].Loc
	})

	set := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	return xmlHeader() + marshalXML(set)
}

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel rssChannel
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language,omitempty"`
	PubDate     string    `xml:"pubDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Guid        string `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
}

func renderRSS(cfg core.Config, pages []*core.Page) (string, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	var blogPages []*core.Page
	for _, page := range pages {
		if page.Section == "blog" {
			blogPages = append(blogPages, page)
		}
	}

	sort.Slice(blogPages, func(i, j int) bool {
		return blogPages[i].Date.After(blogPages[j].Date)
	})
	if len(blogPages) > 20 {
		blogPages = blogPages[:20]
	}

	items := make([]rssItem, 0, len(blogPages))
	for _, page := range blogPages {
		link := baseURL + page.URL
		item := rssItem{
			Title:       page.Title,
			Link:        link,
			Guid:        link,
			Description: page.Description,
		}
		if item.Description == "" {
			item.Description = page.Summary
		}
		if !page.Date.IsZero() {
			item.PubDate = page.Date.Format(time.RFC1123Z)
		}
		items = append(items, item)
	}

	pubDate := ""
	if len(blogPages) > 0 && !blogPages[0].Date.IsZero() {
		pubDate = blogPages[0].Date.Format(time.RFC1123Z)
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       cfg.Title,
			Link:        baseURL,
			Description: cfg.Description,
			Language:    cfg.Language,
			PubDate:     pubDate,
			Items:       items,
		},
	}

	return xmlHeader() + marshalXML(feed), nil
}

type searchEntry struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	Section string   `json:"section"`
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
}

func renderSearchIndex(pages []*core.Page) string {
	entries := make([]searchEntry, 0, len(pages))
	for _, page := range pages {
		summary := strings.TrimSpace(page.Summary)
		if summary == "" {
			summary = strings.TrimSpace(page.Description)
		}
		entries = append(entries, searchEntry{
			URL:     page.URL,
			Title:   page.Title,
			Section: page.Section,
			Tags:    page.Tags,
			Summary: summary,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "[]\n"
	}
	return string(data) + "\n"
}

func xmlHeader() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
}

func marshalXML(v any) string {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return ""
	}
	return buf.String() + "\n"
}
