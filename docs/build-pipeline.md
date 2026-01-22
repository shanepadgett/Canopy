# Build Pipeline Specification

This document specifies the core build pipeline for Canopy. It defines the phases, data flow, and contracts that drive `canopy build`.

---

## Pipeline Phases

```text
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  1. Config  │───▶│  2. Collect │───▶│  3. Render  │───▶│ 4. Template │───▶│  5. Output  │
│    Load     │    │   Content   │    │  Markdown   │    │   Execute   │    │   Write     │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

### Phase 1: Config Load

**Input:** `site.json` path (or auto-discovered from cwd).

**Output:** `core.Config` struct.

**Behavior:**

- Parse JSON into Config.
- Validate required fields: `name`, `baseURL`.
- Apply defaults for missing optional fields.
- Resolve directory paths relative to site root.

**Package:** `internal/config`

---

### Phase 2: Collect Content

**Input:** Config with resolved paths.

**Output:** `[]*core.Page` (unsorted, unrendered).

**Behavior:**

1. Walk `contentDir` recursively for `.md` files.
2. For each file:
   - Read file contents.
   - Parse front matter using `core.ParseFrontMatter`.
   - Apply section defaults from `Config.Sections[section].Defaults`.
   - Validate required fields from `Config.Sections[section].Required`.
   - Derive section from first path segment under contentDir.
   - Derive slug: front matter `slug` > filename without extension.
   - Compute URL from permalink pattern.
   - Build `core.Page` with RawContent (body without front matter).
3. Skip files where `draft=true` unless `buildDrafts=true`.
4. Collect validation errors; fail build if any required fields missing.
5. Index pages into `Site.Sections` and `Site.Tags`.

**Package:** `internal/content`

**Slug Derivation Rules:**

```text
content/blog/hello-world.md  →  slug: "hello-world"
content/blog/2024/intro.md   →  slug: "intro"
front matter slug: "custom"  →  slug: "custom" (wins)
```

**URL Computation:**

```text
permalink pattern: "/blog/:slug/"
slug: "hello-world"
URL: "/blog/hello-world/"
```

Supported permalink tokens:

- `:slug` - page slug
- `:section` - section name
- `:year`, `:month`, `:day` - from date field

---

### Phase 3: Render Markdown

**Input:** `[]*core.Page` with RawContent.

**Output:** `[]*core.Page` with Body (HTML) and Summary populated.

**Behavior:**

1. For each page:
   - Convert RawContent (Markdown) to HTML.
   - Generate TOC entries from headings.
   - Extract summary: first paragraph, max 200 chars, plain text.
2. Store rendered HTML in `Page.Body`.
3. Store TOC in `Page.TOC`.

**Package:** `internal/markdown`

**Markdown Features (MVP):**

- Headings (h1-h6)
- Paragraphs
- Links (inline and reference)
- Lists (ordered and unordered)
- Fenced code blocks with language hint
- Inline code
- Emphasis (*italic*) and strong (**bold**)
- Horizontal rules
- Blockquotes

**Not in MVP:**

- Shortcodes (Phase 2)
- Tables (Phase 2)
- Footnotes (Phase 3)

---

### Phase 4: Template Execute

**Input:** `*core.Site` with all pages rendered.

**Output:** Map of URL → rendered HTML string.

**Behavior:**

1. Load templates from `templateDir`:
   - `layouts/base.html` - base wrapper
   - `layouts/<section>.html` - section-specific layouts
   - `layouts/page.html` - fallback for standalone pages
   - `layouts/list.html` - section index pages
   - `partials/*.html` - reusable fragments
2. For each page:
   - Select layout: `layouts/<section>.html` or `layouts/page.html`.
   - Execute template with page + site data.
   - Wrap in base layout.
3. Generate section index pages (`/blog/`, `/guides/`).
4. Generate home page.

**Package:** `internal/template`

**Template Data Contract:**

```go
// Available in all templates
type TemplateData struct {
    Page    *core.Page    // current page (nil for list pages)
    Site    *core.Site    // full site data
    Section *core.Section // current section (for list pages)
    Pages   []*core.Page  // pages to list (for list pages)
}
```

**Template Functions (MVP):**

- `safeHTML` - mark string as safe HTML
- `now` - current time
- `dateFormat` - format time
- `lower`, `upper`, `title` - string transforms
- `slice` - create slice from args
- `first`, `last` - slice helpers

---

### Phase 5: Output Write

**Input:** Map of URL → HTML, static dir path.

**Output:** Files written to `outputDir`.

**Behavior:**

1. Clean or create `outputDir`.
2. For each URL → HTML:
   - Convert URL to file path: `/blog/hello/` → `blog/hello/index.html`
   - Write HTML file.
3. Copy `staticDir` contents to `outputDir` preserving structure.
4. Return build stats.

**Package:** `internal/build`

**Output Mapping:**

```text
URL: /blog/hello-world/  →  public/blog/hello-world/index.html
URL: /about/             →  public/about/index.html
URL: /                   →  public/index.html
```

---

## Build Options

From CLI flags:

- `--drafts` / `-d`: Include draft content
- `--output` / `-o`: Override output directory

From config:

- `buildDrafts`: Default draft behavior
- `outputDir`: Default output directory

CLI flags override config.

---

## Error Handling

Build should fail with clear errors for:

1. Missing `site.json`
2. Invalid JSON in config
3. Missing required config fields (`name`, `baseURL`)
4. Invalid front matter JSON/syntax
5. Missing required front matter fields (per section config)
6. Template parse errors
7. Template execution errors
8. File write errors

Error format:

```text
error: <file>:<line>: <message>
```

---

## Build Stats

After successful build, print:

```text
Built site:
  Pages:    12
  Sections: 3
  Tags:     8
  Output:   public/
  Time:     142ms
```

---

## Package Structure

```text
internal/
  build/
    build.go       # orchestrates pipeline
    writer.go      # writes output files
  content/
    loader.go      # discovers and loads content
    url.go         # URL computation
  markdown/
    render.go      # Markdown to HTML
    toc.go         # TOC extraction
  template/
    engine.go      # template loading and execution
    funcs.go       # template functions
```

---

## Test Strategy

1. Unit tests per package with table-driven cases.
2. Integration test using `testdata/site`:
   - Run full build.
   - Assert expected files exist in output.
   - Assert content matches snapshots.
3. Error case tests:
   - Missing config.
   - Invalid front matter.
   - Missing required fields.

---

## Future Phases

### Phase 2: Extended Content

- Shortcodes
- Tables
- Image processing

### Phase 3: Generated Outputs

- RSS feed
- Sitemap
- robots.txt
- Search index

### Phase 4: Dev Server

- `canopy serve`
- File watching
- Live reload
- Quality overlays
