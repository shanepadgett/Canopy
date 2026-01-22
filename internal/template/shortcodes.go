package template

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/shanepadgett/canopy/internal/core"
)

type shortcodeData struct {
	Name   string
	Params map[string]string
	Inner  any
	Page   *core.Page
}

// RenderShortcode executes a shortcode template with context.
func (e *Engine) RenderShortcode(name string, params map[string]string, inner string, innerIsHTML bool, page *core.Page) (string, error) {
	tplName := "shortcodes/" + name + ".html"
	tpl := e.templates.Lookup(tplName)
	if tpl == nil {
		return "", fmt.Errorf("shortcode template %q not found", tplName)
	}

	if params == nil {
		params = map[string]string{}
	}

	var innerValue any = inner
	if innerIsHTML {
		innerValue = template.HTML(inner)
	}

	data := shortcodeData{
		Name:   name,
		Params: params,
		Inner:  innerValue,
		Page:   page,
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("executing shortcode %q: %w", name, err)
	}

	return out.String(), nil
}

func (e *Engine) loadDefaultShortcodes() error {
	for name, content := range defaultShortcodes {
		if e.templates.Lookup(name) != nil {
			continue
		}
		if _, err := e.templates.New(name).Parse(content); err != nil {
			return fmt.Errorf("parsing default shortcode %s: %w", name, err)
		}
	}

	return nil
}

var defaultShortcodes = map[string]string{
	"shortcodes/callout.html":       defaultShortcodeCallout,
	"shortcodes/figure.html":        defaultShortcodeFigure,
	"shortcodes/youtube.html":       defaultShortcodeYouTube,
	"shortcodes/toc.html":           defaultShortcodeTOC,
	"shortcodes/key-takeaways.html": defaultShortcodeKeyTakeaways,
	"shortcodes/prereqs.html":       defaultShortcodePrereqs,
	"shortcodes/code-tabs.html":     defaultShortcodeCodeTabs,
}

const defaultShortcodeCallout = `<div class="shortcode-callout{{with index .Params "type"}} shortcode-callout-{{.}}{{end}}">
  {{with index .Params "title"}}<strong class="shortcode-callout-title">{{.}}</strong>{{end}}
  <div class="shortcode-callout-body">{{.Inner}}</div>
</div>
`

const defaultShortcodeFigure = `<figure class="shortcode-figure">
  <img src="{{index .Params "src"}}" alt="{{index .Params "alt"}}">
  {{with index .Params "caption"}}<figcaption>{{.}}</figcaption>{{end}}
</figure>
`

const defaultShortcodeYouTube = `<div class="shortcode-youtube">
  <iframe src="https://www.youtube.com/embed/{{index .Params "id"}}" title="{{with index .Params "title"}}{{.}}{{else}}YouTube video{{end}}" loading="lazy" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
</div>
`

const defaultShortcodeTOC = `<nav class="shortcode-toc">
  {{if .Page}}
  <ol>
    {{range .Page.TOC}}
    <li class="toc-level-{{.Level}}"><a href="#{{.ID}}">{{.Title}}</a></li>
    {{end}}
  </ol>
  {{end}}
</nav>
`

const defaultShortcodeKeyTakeaways = `<section class="shortcode-key-takeaways">
  <h3>Key takeaways</h3>
  <div class="shortcode-key-takeaways-body">{{.Inner}}</div>
</section>
`

const defaultShortcodePrereqs = `<section class="shortcode-prereqs">
  <h3>Prerequisites</h3>
  <div class="shortcode-prereqs-body">{{.Inner}}</div>
</section>
`

const defaultShortcodeCodeTabs = `<div class="shortcode-code-tabs">
  {{safeHTML .Inner}}
</div>
`
