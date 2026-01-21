# Canopy — Detailed Feature Specification

This document describes **what Canopy should actually include** as a reusable static site generator. It is written as a practical, buildable target rather than a high-level vision.

---

## 1. Inputs and project structure

A Canopy site should follow a conventional layout like:

```
site/
  site.json
  content/
    blog/
    guides/
  pages/
  templates/
    layouts/
    partials/
    shortcodes/
  static/
  public/        # generated output
```

Canopy should understand and rely on this structure.

### Required inputs

Canopy must accept:

* Markdown files (`.md`) as the primary authoring format
* Structured front matter in each Markdown file for metadata
* A site configuration file (`site.json`) containing at minimum:

  * site name
  * base URL
  * description
  * default language
  * optional navigation configuration

---

## 2. Content processing

Canopy must be able to:

* Automatically discover all Markdown content under `content/`
* Read and validate front matter fields such as:

  * `title`
  * `date`
  * `slug`
  * `tags`
  * `draft`
  * `description` (for SEO)
* Convert Markdown to safe, valid HTML
* Support a well-defined subset of Markdown including at least:

  * headings
  * paragraphs
  * links
  * lists
  * fenced code blocks
  * inline code
  * emphasis and bold

### Shortcodes (custom elements)

Canopy must support shortcodes embedded in Markdown, for example:

```
{{< callout type="warning" >}}
Be careful here.
{{< /callout >}}
```

Shortcodes should:

* Map to template files in `templates/shortcodes/`
* Accept named attributes
* Optionally wrap inner Markdown content
* Be reusable across any page type

Built-in shortcodes should include at minimum:

* `callout`
* `figure`
* `youtube` or generic `embed`
* `toc` (table of contents)

---

## 3. Page model

Every generated page should have a consistent internal representation including:

* Title
* URL
* Section (`blog`, `guides`, `pages`, etc.)
* Rendered HTML body
* Plain-text summary or excerpt
* All front matter fields available to templates

This model must be shared by:

* Blog posts
* Guides
* Standalone pages
* Index pages

---

## 4. Templating and theming

Canopy must render all pages using Go’s `html/template` system.

Required template features:

* A base layout that wraps every page
* Layouts for at least:

  * blog posts
  * guides
  * generic pages
* Reusable partial templates such as:

  * header
  * footer
  * navigation
* The ability to swap an entire theme without changing content or generator code

Templates must be able to access:

* Page data
* Site-wide data
* Lists of recent posts
* Tag groupings
* Navigation structure

---

## 5. Routing and URLs

Canopy must generate clean, predictable URLs such as:

* `/blog/my-post/`
* `/guides/getting-started/`
* `/about/`

It should support at least two permalink styles:

* section-based: `/blog/<slug>/`
* date-based (optional): `/blog/2026/01/my-post/`

Slug precedence should be:

* `slug` in front matter wins
* otherwise derive from file path

---

## 6. Site-wide generated pages

From the content collection, Canopy must generate:

### Index pages

* Home page
* Blog listing page
* Guides overview page

These pages should support:

* Sorting by date
* Pagination when there are many items

### Tag pages

For any tag used in content, Canopy must generate:

* `/tags/<tag>/` listing all related pages

---

## 7. Search

Canopy should provide a built-in search UI and index for client-side search.

### Search index

Canopy must generate `search.json` (or `search-index.json`) containing:

* `title`
* `url`
* `summary`
* `section`
* `tags`
* `body` (plain text, stripped)

### Search UI

Canopy should ship a minimal search UI that:

* consumes the generated index
* supports basic fuzzy matching for typos
* is themeable via templates and styles

---

## 8. Machine-readable outputs

Every Canopy site must include:

### RSS feed

* `rss.xml` (or `feed.xml`) containing:

  * latest posts
  * titles, links, dates, descriptions

### Sitemap

* `sitemap.xml` including:

  * all public pages
  * last-modified dates where available

### Robots file

* `robots.txt` allowing search engines to crawl the site by default

These files must be generated automatically on every build.

---

## 9. Static assets

Canopy must:

* Copy everything from `static/` directly into `public/`
* Preserve directory structure
* Leave assets untouched (no required bundling or transforms)

---

## 10. Documentation support

To work well for documentation sites, Canopy must support:

* Sidebar navigation driven by either:

  * a `nav.json` file, or
  * folder structure order
* Automatic table of contents generation from headings
* Previous and next links within a documentation section

---

## 11. Developer experience

For the site author, Canopy must provide:

### CLI commands

At minimum:

```
canopy build
canopy serve
canopy new post <title>
canopy new guide <title>
```

### Local server

`canopy serve` should:

* Run a local HTTP server
* Serve the generated site
* Rebuild pages when files change

### Quality overlays

During `canopy serve`, Canopy should surface visual overlays for:

* broken internal links
* missing required front matter
* duplicate slugs or URL conflicts
* missing metadata like title or description

Overlays should be driven by the same validation rules used during build.

### Error handling

When something breaks, Canopy must:

* Report the file name
* Report the line number when possible
* Clearly explain what went wrong (missing field, bad template, etc.)

### Lattice integration

Canopy should treat Lattice as a generator dependency and invoke it during builds
to prevent CSS utility clashes, without requiring per-site installation.

---

## 12. Build guarantees

Canopy builds must be:

* Deterministic: same input always produces the same output
* Fast enough for local iteration
* Safe to deploy to any static host

---

## 13. What a successful build produces

After `canopy build`, the `public/` directory must contain:

* All rendered HTML pages
* Copied static assets
* `rss.xml`
* `sitemap.xml`
* `robots.txt`

The `public/` folder should be immediately deployable as-is.

---

## 14. Guiding principle

Canopy should make content easy to write in Markdown, easy to style with templates, and trivial to deploy as pure static files, while remaining flexible enough to power both a personal blog and a full documentation site.

---

## 15. Differentiators (product bets)

Canopy should commit to a few features that are built-in and opinionated:

* Client-side search with a generated index and clear template contract
* Stable URL history via aliases or redirects for moved content
* Content integrity gates that can fail a build when required fields are missing
* Content contracts: per-section front matter schemas with defaults and validation

### Future ideas (not in scope yet)

* Queryable content (list pages driven by a small query language)
* Build explainability (trace any page back to inputs and templates)
* Versioned documentation sections
