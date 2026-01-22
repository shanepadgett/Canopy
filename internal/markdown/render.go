// Package markdown converts Markdown to HTML.
package markdown

import (
	"html"
	"regexp"
	"strings"

	"github.com/shanepadgett/canopy/internal/core"
)

// RenderResult contains the rendered HTML and extracted metadata.
type RenderResult struct {
	HTML    string
	TOC     []core.TOCEntry
	Summary string
}

// Render converts Markdown to HTML and extracts TOC and summary.
func Render(markdown string) RenderResult {
	r := &renderer{
		input: markdown,
	}
	return r.render()
}

type renderer struct {
	input   string
	toc     []core.TOCEntry
	summary string
}

func (r *renderer) render() RenderResult {
	lines := strings.Split(r.input, "\n")
	var out strings.Builder
	var i int

	for i < len(lines) {
		line := lines[i]

		// Fenced code block
		if strings.HasPrefix(line, "```") {
			html, consumed := r.renderCodeBlock(lines[i:])
			out.WriteString(html)
			i += consumed
			continue
		}

		// Heading
		if strings.HasPrefix(line, "#") {
			html, toc := r.renderHeading(line)
			out.WriteString(html)
			if toc != nil {
				r.toc = append(r.toc, *toc)
			}
			i++
			continue
		}

		// Horizontal rule
		if isHorizontalRule(line) {
			out.WriteString("<hr>\n")
			i++
			continue
		}

		// Blockquote
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			html, consumed := r.renderBlockquote(lines[i:])
			out.WriteString(html)
			i += consumed
			continue
		}

		// Unordered list
		if isUnorderedListItem(line) {
			html, consumed := r.renderUnorderedList(lines[i:])
			out.WriteString(html)
			i += consumed
			continue
		}

		// Ordered list
		if isOrderedListItem(line) {
			html, consumed := r.renderOrderedList(lines[i:])
			out.WriteString(html)
			i += consumed
			continue
		}

		// Empty line
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Paragraph
		html, consumed := r.renderParagraph(lines[i:])
		out.WriteString(html)

		// Extract first paragraph as summary
		if r.summary == "" {
			r.summary = extractPlainText(html)
			if len(r.summary) > 200 {
				r.summary = r.summary[:200] + "..."
			}
		}

		i += consumed
	}

	return RenderResult{
		HTML:    out.String(),
		TOC:     r.toc,
		Summary: r.summary,
	}
}

func (r *renderer) renderHeading(line string) (string, *core.TOCEntry) {
	level := 0
	for _, c := range line {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	if level > 6 {
		level = 6
	}

	text := strings.TrimSpace(line[level:])
	id := slugify(text)

	// Apply inline formatting to heading text
	formattedText := renderInline(text)

	toc := &core.TOCEntry{
		Level: level,
		ID:    id,
		Title: text,
	}

	return "<h" + itoa(level) + " id=\"" + id + "\">" + formattedText + "</h" + itoa(level) + ">\n", toc
}

func (r *renderer) renderCodeBlock(lines []string) (string, int) {
	if len(lines) == 0 {
		return "", 0
	}

	// Extract language hint
	opener := lines[0]
	lang := strings.TrimPrefix(opener, "```")
	lang = strings.TrimSpace(lang)

	var code strings.Builder
	consumed := 1

	for i := 1; i < len(lines); i++ {
		consumed++
		if strings.HasPrefix(lines[i], "```") {
			break
		}
		if code.Len() > 0 {
			code.WriteString("\n")
		}
		code.WriteString(lines[i])
	}

	escapedCode := html.EscapeString(code.String())

	if lang != "" {
		return "<pre><code class=\"language-" + lang + "\">" + escapedCode + "</code></pre>\n", consumed
	}
	return "<pre><code>" + escapedCode + "</code></pre>\n", consumed
}

func (r *renderer) renderBlockquote(lines []string) (string, int) {
	var content strings.Builder
	consumed := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ">") && trimmed != "" {
			break
		}
		consumed++

		if trimmed == "" {
			continue
		}

		// Strip the > prefix
		text := strings.TrimPrefix(trimmed, ">")
		text = strings.TrimPrefix(text, " ")
		content.WriteString(text)
		content.WriteString("\n")
	}

	inner := strings.TrimSpace(content.String())
	return "<blockquote><p>" + renderInline(inner) + "</p></blockquote>\n", consumed
}

func (r *renderer) renderUnorderedList(lines []string) (string, int) {
	var out strings.Builder
	out.WriteString("<ul>\n")

	consumed := 0
	for _, line := range lines {
		if !isUnorderedListItem(line) {
			break
		}
		consumed++

		// Strip list marker
		text := strings.TrimSpace(line)
		text = strings.TrimPrefix(text, "-")
		text = strings.TrimPrefix(text, "*")
		text = strings.TrimPrefix(text, "+")
		text = strings.TrimSpace(text)

		out.WriteString("<li>" + renderInline(text) + "</li>\n")
	}

	out.WriteString("</ul>\n")
	return out.String(), consumed
}

func (r *renderer) renderOrderedList(lines []string) (string, int) {
	var out strings.Builder
	out.WriteString("<ol>\n")

	consumed := 0
	for _, line := range lines {
		if !isOrderedListItem(line) {
			break
		}
		consumed++

		// Strip number and period
		text := strings.TrimSpace(line)
		if idx := strings.Index(text, "."); idx > 0 {
			text = strings.TrimSpace(text[idx+1:])
		}

		out.WriteString("<li>" + renderInline(text) + "</li>\n")
	}

	out.WriteString("</ol>\n")
	return out.String(), consumed
}

func (r *renderer) renderParagraph(lines []string) (string, int) {
	var content strings.Builder
	consumed := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			consumed++ // consume the blank line
			break
		}

		// Stop at block-level elements
		if strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "```") ||
			strings.HasPrefix(trimmed, ">") ||
			isUnorderedListItem(line) ||
			isOrderedListItem(line) ||
			isHorizontalRule(line) {
			break
		}

		consumed++
		if content.Len() > 0 {
			content.WriteString(" ")
		}
		content.WriteString(trimmed)
	}

	text := content.String()
	if text == "" {
		return "", consumed
	}

	return "<p>" + renderInline(text) + "</p>\n", consumed
}

// renderInline handles inline formatting: bold, italic, code, links.
func renderInline(text string) string {
	// Escape HTML entities first
	text = html.EscapeString(text)

	// Inline code (must come before bold/italic to avoid conflicts)
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")

	// Links: [text](url)
	text = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)

	// Bold: **text** or __text__
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")
	text = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(text, "<strong>$1</strong>")

	// Italic: *text* or _text_
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "<em>$1</em>")
	text = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(text, "<em>$1</em>")

	return text
}

func isUnorderedListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ")
}

func isOrderedListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	for i, c := range trimmed {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 {
			return true
		}
		break
	}
	return false
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}

	// --- or *** or ___
	allSame := true
	char := trimmed[0]
	if char != '-' && char != '*' && char != '_' {
		return false
	}

	for _, c := range trimmed {
		if c != rune(char) && c != ' ' {
			allSame = false
			break
		}
	}

	return allSame
}

func slugify(text string) string {
	// Lowercase
	s := strings.ToLower(text)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove non-alphanumeric except hyphens
	var result strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}

	return result.String()
}

func extractPlainText(html string) string {
	// Strip HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, "")

	// Decode common entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	return strings.TrimSpace(text)
}

func itoa(i int) string {
	return string(rune('0' + i))
}
