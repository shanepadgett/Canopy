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
  {{if .Site.Config.Search.Enabled}}
  <style>
    .search-button {
      margin-left: 1rem;
      padding: 0.35rem 0.75rem;
      border-radius: 999px;
      border: 1px solid #2f3b52;
      background: linear-gradient(135deg, #fff4da, #f2e5c9);
      color: #1f2a44;
      font-size: 0.9rem;
      cursor: pointer;
    }
    .search-button:hover {
      background: linear-gradient(135deg, #fff9e6, #f1e0c4);
    }
    .search-overlay {
      position: fixed;
      inset: 0;
      background: rgba(18, 24, 34, 0.55);
      display: flex;
      align-items: flex-start;
      justify-content: center;
      padding: 12vh 1.5rem 2rem;
      z-index: 1000;
    }
    .search-overlay[hidden] {
      display: none;
    }
    .search-panel {
      width: min(720px, 100%);
      border-radius: 18px;
      background: #fdf6e7;
      color: #1c2434;
      box-shadow: 0 24px 60px rgba(17, 24, 39, 0.25);
      border: 1px solid #e6d6ba;
      overflow: hidden;
    }
    .search-header {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 0.9rem 1rem;
      border-bottom: 1px solid #e5d7bf;
    }
    .search-input {
      flex: 1;
      border: none;
      background: transparent;
      font-size: 1rem;
      outline: none;
      color: inherit;
    }
    .search-hint {
      font-size: 0.75rem;
      color: #6a758c;
      white-space: nowrap;
    }
    .search-results {
      list-style: none;
      margin: 0;
      padding: 0;
      max-height: 60vh;
      overflow-y: auto;
    }
    .search-result {
      border-bottom: 1px solid #f0e4cd;
    }
    .search-result-link {
      display: flex;
      flex-direction: column;
      gap: 0.3rem;
      padding: 0.85rem 1rem;
      color: inherit;
      text-decoration: none;
    }
    .search-result.is-active {
      background: #f4e8cf;
    }
    .search-result-title {
      font-weight: 600;
    }
    .search-result-summary {
      font-size: 0.9rem;
      color: #4a566b;
    }
    .search-result-meta {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      color: #7b8293;
    }
    .search-empty {
      padding: 1rem;
      color: #5b6475;
      font-size: 0.9rem;
    }
  </style>
  {{end}}
</head>
<body>
  <header>
    <nav>
      <a href="/">{{.Site.Config.Name}}</a>
      {{range .Site.Config.Nav}}
      <a href="{{.URL}}">{{.Title}}</a>
      {{end}}
      {{if .Site.Config.Search.Enabled}}
      <button class="search-button" type="button" data-search-open>Search</button>
      {{end}}
    </nav>
  </header>
  <main>
    {{.Content}}
  </main>
  <footer>
    <p>&copy; {{now.Year}} {{.Site.Config.Name}}</p>
  </footer>
  {{if .Site.Config.Search.Enabled}}
  <div id="search-overlay" class="search-overlay" aria-hidden="true" hidden>
    <div class="search-panel" role="dialog" aria-modal="true" aria-label="Search">
      <div class="search-header">
        <input id="search-input" class="search-input" type="search" placeholder="Search" autocomplete="off" />
        <div class="search-hint">Esc to close</div>
      </div>
      <ul id="search-results" class="search-results"></ul>
      <div id="search-empty" class="search-empty" hidden>No results.</div>
    </div>
  </div>
  <script>
    (function() {
      var openButton = document.querySelector('[data-search-open]');
      var overlay = document.getElementById('search-overlay');
      var input = document.getElementById('search-input');
      var resultsList = document.getElementById('search-results');
      var emptyState = document.getElementById('search-empty');
      if (!openButton || !overlay || !input || !resultsList || !emptyState) {
        return;
      }

      var searchData = null;
      var currentResults = [];
      var activeIndex = 0;
      var debounceTimer = null;

      function openSearch() {
        overlay.hidden = false;
        overlay.setAttribute('aria-hidden', 'false');
        input.focus();
        input.select();
        loadSearchData();
        updateResults();
      }

      function closeSearch() {
        overlay.hidden = true;
        overlay.setAttribute('aria-hidden', 'true');
      }

      function loadSearchData() {
        if (searchData) {
          return;
        }
        fetch('/search.json')
          .then(function(response) {
            if (!response.ok) {
              throw new Error('search index failed');
            }
            return response.json();
          })
          .then(function(data) {
            searchData = Array.isArray(data) ? data : [];
            updateResults();
          })
          .catch(function() {
            searchData = [];
            updateResults();
          });
      }

      function isOpen() {
        return overlay.hidden === false;
      }

      function isBoundary(char) {
        return char === '' || char === ' ' || char === '-' || char === '_' || char === '/' || char === '.' || char === ',' || char === ':' || char === ';';
      }

      function scoreText(query, text) {
        if (!query || !text) {
          return -1;
        }
        var lowerQuery = query.toLowerCase();
        var lowerText = text.toLowerCase();
        var score = 0;
        var lastIndex = -1;
        var consecutive = 0;

        for (var i = 0; i < lowerQuery.length; i += 1) {
          var char = lowerQuery[i];
          var index = lowerText.indexOf(char, lastIndex + 1);
          if (index === -1) {
            return -1;
          }
          if (index === lastIndex + 1) {
            consecutive += 1;
            score += 10;
          } else {
            consecutive = 0;
          }
          if (index === 0 || isBoundary(lowerText[index - 1])) {
            score += 5;
          }
          score -= index;
          lastIndex = index;
        }
        return score;
      }

      function scoreEntry(entry, query) {
        if (!query) {
          return 0;
        }
        var best = -1;
        var titleScore = scoreText(query, entry.title || '');
        if (titleScore >= 0) {
          best = Math.max(best, titleScore + 100);
        }
        var summaryScore = scoreText(query, entry.summary || '');
        if (summaryScore >= 0) {
          best = Math.max(best, summaryScore);
        }
        var tagScore = scoreText(query, (entry.tags || []).join(' '));
        if (tagScore >= 0) {
          best = Math.max(best, tagScore);
        }
        var sectionScore = scoreText(query, entry.section || '');
        if (sectionScore >= 0) {
          best = Math.max(best, sectionScore);
        }
        return best;
      }

      function updateResults() {
        if (!searchData) {
          return;
        }
        var query = input.value.trim();
        if (!query) {
          currentResults = searchData.slice(0, 10);
        } else {
          currentResults = searchData
            .map(function(entry) {
              return {
                entry: entry,
                score: scoreEntry(entry, query)
              };
            })
            .filter(function(result) {
              return result.score >= 0;
            })
            .sort(function(a, b) {
              return b.score - a.score;
            })
            .slice(0, 10)
            .map(function(result) {
              return result.entry;
            });
        }
        activeIndex = 0;
        renderResults();
      }

      function renderResults() {
        resultsList.innerHTML = '';
        if (!currentResults.length) {
          emptyState.hidden = false;
          return;
        }
        emptyState.hidden = true;
        currentResults.forEach(function(item, index) {
          var li = document.createElement('li');
          li.className = 'search-result' + (index === activeIndex ? ' is-active' : '');

          var link = document.createElement('a');
          link.className = 'search-result-link';
          link.href = item.url || '#';

          var title = document.createElement('div');
          title.className = 'search-result-title';
          title.textContent = item.title || item.url || 'Untitled';

          link.appendChild(title);

          if (item.summary) {
            var summary = document.createElement('div');
            summary.className = 'search-result-summary';
            summary.textContent = item.summary;
            link.appendChild(summary);
          }

          var metaText = [];
          if (item.section) {
            metaText.push(item.section);
          }
          if (item.tags && item.tags.length) {
            metaText.push(item.tags.join(', '));
          }
          if (metaText.length) {
            var meta = document.createElement('div');
            meta.className = 'search-result-meta';
            meta.textContent = metaText.join(' | ');
            link.appendChild(meta);
          }

          li.appendChild(link);
          li.addEventListener('mouseenter', function() {
            activeIndex = index;
            renderResults();
          });
          resultsList.appendChild(li);
        });
      }

      function moveSelection(delta) {
        if (!currentResults.length) {
          return;
        }
        activeIndex += delta;
        if (activeIndex < 0) {
          activeIndex = currentResults.length - 1;
        }
        if (activeIndex >= currentResults.length) {
          activeIndex = 0;
        }
        renderResults();
      }

      function goToSelection() {
        if (!currentResults.length) {
          return;
        }
        var item = currentResults[activeIndex];
        if (item && item.url) {
          window.location.href = item.url;
        }
      }

      openButton.addEventListener('click', function() {
        openSearch();
      });

      overlay.addEventListener('click', function(event) {
        if (event.target === overlay) {
          closeSearch();
        }
      });

      input.addEventListener('input', function() {
        if (debounceTimer) {
          window.clearTimeout(debounceTimer);
        }
        debounceTimer = window.setTimeout(updateResults, 150);
      });

      document.addEventListener('keydown', function(event) {
        var key = event.key;
        if ((event.metaKey || event.ctrlKey) && key.toLowerCase() === 'k') {
          event.preventDefault();
          if (!isOpen()) {
            openSearch();
          } else {
            closeSearch();
          }
          return;
        }

        if (!isOpen()) {
          return;
        }

        if (key === 'Escape') {
          closeSearch();
          return;
        }

        if (key === 'ArrowDown') {
          event.preventDefault();
          moveSelection(1);
          return;
        }

        if (key === 'ArrowUp') {
          event.preventDefault();
          moveSelection(-1);
          return;
        }

        if (key === 'Enter') {
          event.preventDefault();
          goToSelection();
        }
      });
    })();
  </script>
  {{end}}
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
