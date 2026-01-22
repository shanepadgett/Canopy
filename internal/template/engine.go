// Package template handles template loading and execution.
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shanepadgett/canopy/internal/core"
)

// Engine loads and executes templates.
type Engine struct {
	templateDir string
	templates   *template.Template
}

// Data is passed to templates during execution.
type Data struct {
	Page    *core.Page
	Site    *core.Site
	Section *core.Section
	Pages   []*core.Page
}

// NewEngine creates a template engine with templates from the given directory.
func NewEngine(templateDir string) (*Engine, error) {
	e := &Engine{
		templateDir: templateDir,
	}

	if err := e.load(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Engine) load() error {
	e.templates = template.New("").Funcs(templateFuncs())

	// Walk template directory and parse all .html files
	err := filepath.WalkDir(e.templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Read template content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Compute template name relative to template dir
		relPath, err := filepath.Rel(e.templateDir, path)
		if err != nil {
			return err
		}

		// Normalize path separators for template names
		name := filepath.ToSlash(relPath)

		// Parse template
		_, err = e.templates.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		// If template directory doesn't exist, use embedded defaults
		if os.IsNotExist(err) {
			return e.loadDefaults()
		}
		return err
	}

	// Ensure we have at least a base template
	if e.templates.Lookup("layouts/base.html") == nil {
		if err := e.loadDefaults(); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) loadDefaults() error {
	// Default base layout
	_, err := e.templates.New("layouts/base.html").Parse(defaultBaseLayout)
	if err != nil {
		return err
	}

	// Default page layout
	_, err = e.templates.New("layouts/page.html").Parse(defaultPageLayout)
	if err != nil {
		return err
	}

	// Default list layout
	_, err = e.templates.New("layouts/list.html").Parse(defaultListLayout)
	if err != nil {
		return err
	}

	// Default home layout
	_, err = e.templates.New("layouts/home.html").Parse(defaultHomeLayout)
	if err != nil {
		return err
	}

	return nil
}

// RenderPage renders a single page.
func (e *Engine) RenderPage(page *core.Page, site *core.Site) (string, error) {
	// Find section-specific layout or fall back to page layout
	layoutName := "layouts/" + page.Section + ".html"
	layout := e.templates.Lookup(layoutName)
	if layout == nil {
		layout = e.templates.Lookup("layouts/page.html")
	}
	if layout == nil {
		return "", fmt.Errorf("no layout found for section %q", page.Section)
	}

	data := Data{
		Page: page,
		Site: site,
	}

	// Execute content layout
	var content bytes.Buffer
	if err := layout.Execute(&content, data); err != nil {
		return "", fmt.Errorf("executing layout: %w", err)
	}

	// Wrap in base layout
	return e.wrapInBase(content.String(), page.Title, site)
}

// RenderList renders a section index page.
func (e *Engine) RenderList(section *core.Section, site *core.Site) (string, error) {
	layout := e.templates.Lookup("layouts/list.html")
	if layout == nil {
		return "", fmt.Errorf("no list layout found")
	}

	data := Data{
		Site:    site,
		Section: section,
		Pages:   section.Pages,
	}

	var content bytes.Buffer
	if err := layout.Execute(&content, data); err != nil {
		return "", fmt.Errorf("executing list layout: %w", err)
	}

	title := strings.Title(section.Name)
	return e.wrapInBase(content.String(), title, site)
}

// RenderHome renders the home page.
func (e *Engine) RenderHome(site *core.Site) (string, error) {
	layout := e.templates.Lookup("layouts/home.html")
	if layout == nil {
		layout = e.templates.Lookup("layouts/list.html")
	}
	if layout == nil {
		return "", fmt.Errorf("no home layout found")
	}

	data := Data{
		Site:  site,
		Pages: site.Pages,
	}

	var content bytes.Buffer
	if err := layout.Execute(&content, data); err != nil {
		return "", fmt.Errorf("executing home layout: %w", err)
	}

	return e.wrapInBase(content.String(), site.Config.Title, site)
}

func (e *Engine) wrapInBase(content, title string, site *core.Site) (string, error) {
	base := e.templates.Lookup("layouts/base.html")
	if base == nil {
		// No base layout, return content as-is
		return content, nil
	}

	baseData := struct {
		Title   string
		Content template.HTML
		Site    *core.Site
	}{
		Title:   title,
		Content: template.HTML(content),
		Site:    site,
	}

	var out bytes.Buffer
	if err := base.Execute(&out, baseData); err != nil {
		return "", fmt.Errorf("executing base layout: %w", err)
	}

	return out.String(), nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"now": func() time.Time {
			return time.Now()
		},
		"dateFormat": func(layout string, t time.Time) string {
			return t.Format(layout)
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
		"slice": func(args ...any) []any {
			return args
		},
		"first": func(n int, items []*core.Page) []*core.Page {
			if n > len(items) {
				n = len(items)
			}
			return items[:n]
		},
		"last": func(n int, items []*core.Page) []*core.Page {
			if n > len(items) {
				n = len(items)
			}
			return items[len(items)-n:]
		},
	}
}

// Default templates
const defaultBaseLayout = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}} - {{.Site.Config.Name}}</title>
  <meta name="description" content="{{.Site.Config.Description}}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{.Site.Config.Name}}</a>
      {{range .Site.Config.Nav}}
      <a href="{{.URL}}">{{.Title}}</a>
      {{end}}
    </nav>
  </header>
  <main>
    {{.Content}}
  </main>
  <footer>
    <p>&copy; {{now.Year}} {{.Site.Config.Name}}</p>
  </footer>
</body>
</html>`

const defaultPageLayout = `<article>
  <h1>{{.Page.Title}}</h1>
  {{if not .Page.Date.IsZero}}
  <time datetime="{{dateFormat "2006-01-02" .Page.Date}}">{{dateFormat "January 2, 2006" .Page.Date}}</time>
  {{end}}
  <div class="content">
    {{safeHTML .Page.Body}}
  </div>
  {{if .Page.Tags}}
  <div class="tags">
    {{range .Page.Tags}}
    <a href="/tags/{{.}}/">{{.}}</a>
    {{end}}
  </div>
  {{end}}
</article>`

const defaultListLayout = `<h1>{{.Section.Name}}</h1>
<ul>
{{range .Pages}}
  <li>
    <a href="{{.URL}}">{{.Title}}</a>
    {{if not .Date.IsZero}}
    <time datetime="{{dateFormat "2006-01-02" .Date}}">{{dateFormat "Jan 2, 2006" .Date}}</time>
    {{end}}
  </li>
{{end}}
</ul>`

const defaultHomeLayout = `<h1>{{.Site.Config.Title}}</h1>
<p>{{.Site.Config.Description}}</p>
{{if .Pages}}
<h2>Recent</h2>
<ul>
{{range first 5 .Pages}}
  <li>
    <a href="{{.URL}}">{{.Title}}</a>
  </li>
{{end}}
</ul>
{{end}}`
