# Task 02 - Shortcodes

## Goal

Support template-backed shortcodes in Markdown with attributes, optional inner content, and Hugo-style delimiter behavior.

## Scope

### Syntax

- Support two shortcode syntaxes:
  - `{{< name attr="value" >}} ... {{< /name >}}` for Markdown-processed inner content.
  - `{{% name attr="value" %}} ... {{% /name %}}` for raw (literal) inner content.
- Support inline shortcodes without closing tag (no inner content).
- Shortcode names: `[a-zA-Z][a-zA-Z0-9_-]*`.
- Attributes: quoted values only (`key="value"` or `key='value'`).
- Allow nesting; mismatched close leaves raw + warning.

### Template Mapping

- Map shortcode name to `templates/shortcodes/<name>.html`.
- Provide template context:
  - `.Name` (string)
  - `.Params` (map[string]string)
  - `.Inner` (HTML or raw string based on delimiter)
  - `.Page` (current page)

### Built-ins

Ship a full starter kit:

- `callout` (block) for admonitions
- `figure` (inline) for image + caption
- `youtube` (inline) for embeds
- `toc` (inline) to render Page.TOC
- `key-takeaways` (block)
- `prereqs` (block)
- `code-tabs` (block, raw inner content)

## Notes

- Shortcodes should render after Markdown parse or during Markdown render.
- TOC shortcode should access `Page.TOC`.
- Keep parsing strict and predictable; avoid unquoted attributes or boolean flags.
