package markdown

import (
	"fmt"
	"os"
	"strings"
)

type shortcodeReplacement struct {
	html  string
	block bool
}

type shortcodeTag struct {
	name      string
	params    map[string]string
	delimiter byte
	isClose   bool
	start     int
	end       int
	raw       string
}

func (r *renderer) processShortcodes(input string) string {
	if r.options.ShortcodeRenderer == nil {
		return input
	}

	if r.shortcodes == nil {
		r.shortcodes = make(map[string]shortcodeReplacement)
	}

	var out strings.Builder
	var segment strings.Builder
	lines := strings.Split(input, "\n")
	inCode := false

	flushSegment := func() {
		if segment.Len() == 0 {
			return
		}
		out.WriteString(r.processShortcodesSegment(segment.String()))
		segment.Reset()
	}

	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inCode {
				flushSegment()
				inCode = true
			} else {
				inCode = false
			}
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		if inCode {
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		segment.WriteString(line)
		if i < len(lines)-1 {
			segment.WriteByte('\n')
		}
	}

	flushSegment()
	return out.String()
}

func (r *renderer) processShortcodesSegment(input string) string {
	var out strings.Builder
	idx := 0

	for idx < len(input) {
		next := strings.Index(input[idx:], "{{")
		if next == -1 {
			out.WriteString(input[idx:])
			break
		}
		next += idx
		out.WriteString(input[idx:next])

		tag, ok := parseShortcodeTag(input, next)
		if !ok {
			out.WriteString(input[next : next+2])
			idx = next + 2
			continue
		}

		if tag.isClose {
			r.warnShortcode("mismatched closing shortcode %q", tag.name)
			out.WriteString(tag.raw)
			idx = tag.end
			continue
		}

		standalone := isTagStandalone(input, tag.start, tag.end)
		if standalone {
			inner, end, closed := r.extractShortcodeInner(input, tag)
			if closed {
				renderedInner, innerIsHTML := r.renderShortcodeInner(tag, inner)
				html, ok := r.renderShortcode(tag, renderedInner, innerIsHTML)
				if !ok {
					out.WriteString(input[tag.start:end])
				} else {
					token := r.addShortcodePlaceholder(html, true)
					out.WriteString(token)
				}
				idx = end
				continue
			}
		}

		html, ok := r.renderShortcode(tag, "", false)
		if !ok {
			out.WriteString(tag.raw)
		} else {
			token := r.addShortcodePlaceholder(html, standalone)
			out.WriteString(token)
		}
		idx = tag.end
	}

	return out.String()
}

func (r *renderer) extractShortcodeInner(input string, tag shortcodeTag) (string, int, bool) {
	type frame struct {
		name      string
		delimiter byte
	}

	stack := []frame{{name: tag.name, delimiter: tag.delimiter}}
	idx := tag.end
	var mismatched []shortcodeTag

	for idx < len(input) {
		next := strings.Index(input[idx:], "{{")
		if next == -1 {
			return "", 0, false
		}
		next += idx

		nested, ok := parseShortcodeTag(input, next)
		if !ok {
			idx = next + 2
			continue
		}

		if nested.isClose {
			if len(stack) == 0 {
				mismatched = append(mismatched, nested)
				idx = nested.end
				continue
			}

			current := stack[len(stack)-1]
			if current.name != nested.name || current.delimiter != nested.delimiter {
				mismatched = append(mismatched, nested)
				idx = nested.end
				continue
			}

			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				inner := input[tag.end:next]
				for _, mismatch := range mismatched {
					r.warnShortcode("mismatched closing shortcode %q", mismatch.name)
				}
				return inner, nested.end, true
			}
		} else if isTagStandalone(input, nested.start, nested.end) {
			stack = append(stack, frame{name: nested.name, delimiter: nested.delimiter})
		}

		idx = nested.end
	}

	return "", 0, false
}

func (r *renderer) renderShortcodeInner(tag shortcodeTag, inner string) (string, bool) {
	if tag.delimiter == '<' {
		innerOptions := r.options
		innerOptions.SkipPageTOC = true
		result := RenderWithOptions(inner, innerOptions)
		return result.HTML, true
	}

	return r.renderRawShortcodes(inner), false
}

func (r *renderer) renderRawShortcodes(inner string) string {
	if r.options.ShortcodeRenderer == nil {
		return inner
	}

	nested := &renderer{
		input:   inner,
		options: r.options,
	}

	nested.input = nested.processShortcodes(inner)
	return nested.replaceShortcodes(nested.input)
}

func (r *renderer) renderShortcode(tag shortcodeTag, inner string, innerIsHTML bool) (string, bool) {
	if r.options.ShortcodeRenderer == nil {
		return "", false
	}

	html, err := r.options.ShortcodeRenderer.RenderShortcode(tag.name, tag.params, inner, innerIsHTML, r.options.Page)
	if err != nil {
		r.warnShortcode("rendering shortcode %q failed: %v", tag.name, err)
		return "", false
	}

	return html, true
}

func (r *renderer) addShortcodePlaceholder(html string, block bool) string {
	r.shortcodeCounter++
	token := fmt.Sprintf("::canopy-shortcode-%d::", r.shortcodeCounter)
	r.shortcodes[token] = shortcodeReplacement{html: html, block: block}
	return token
}

func (r *renderer) replaceShortcodes(html string) string {
	if len(r.shortcodes) == 0 {
		return html
	}

	for token, replacement := range r.shortcodes {
		html = strings.ReplaceAll(html, token, replacement.html)
	}
	return html
}

func (r *renderer) blockShortcodeToken(line string) (string, bool) {
	if len(r.shortcodes) == 0 {
		return "", false
	}

	token := strings.TrimSpace(line)
	if token == "" {
		return "", false
	}

	replacement, ok := r.shortcodes[token]
	if !ok || !replacement.block {
		return "", false
	}

	return token, true
}

func (r *renderer) warnShortcode(format string, args ...any) {
	prefix := "shortcode"
	if r.options.Page != nil && r.options.Page.SourcePath != "" {
		prefix = r.options.Page.SourcePath
	}
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "warning: %s: %s\n", prefix, message)
}

func isTagStandalone(input string, start, end int) bool {
	lineStart := strings.LastIndex(input[:start], "\n") + 1
	lineEnd := len(input)
	if next := strings.Index(input[end:], "\n"); next != -1 {
		lineEnd = end + next
	}

	before := strings.TrimSpace(input[lineStart:start])
	after := strings.TrimSpace(input[end:lineEnd])
	return before == "" && after == ""
}

func parseShortcodeTag(input string, start int) (shortcodeTag, bool) {
	if start+3 >= len(input) {
		return shortcodeTag{}, false
	}
	if !strings.HasPrefix(input[start:], "{{") {
		return shortcodeTag{}, false
	}

	delimiter := input[start+2]
	if delimiter != '<' && delimiter != '%' {
		return shortcodeTag{}, false
	}

	idx := start + 3
	idx = skipSpaces(input, idx)
	if idx >= len(input) {
		return shortcodeTag{}, false
	}

	isClose := false
	if input[idx] == '/' {
		isClose = true
		idx++
		idx = skipSpaces(input, idx)
	}

	nameStart := idx
	if idx >= len(input) || !isNameStart(input[idx]) {
		return shortcodeTag{}, false
	}
	idx++
	for idx < len(input) && isNameChar(input[idx]) {
		idx++
	}
	name := input[nameStart:idx]

	if isClose {
		idx = skipSpaces(input, idx)
		end := consumeClosing(input, idx, delimiter)
		if end == -1 {
			return shortcodeTag{}, false
		}
		raw := input[start:end]
		return shortcodeTag{name: name, delimiter: delimiter, isClose: true, start: start, end: end, raw: raw}, true
	}

	var params map[string]string
	for {
		idx = skipSpaces(input, idx)
		if idx >= len(input) {
			return shortcodeTag{}, false
		}
		if end := consumeClosing(input, idx, delimiter); end != -1 {
			if params == nil {
				params = map[string]string{}
			}
			raw := input[start:end]
			return shortcodeTag{name: name, params: params, delimiter: delimiter, start: start, end: end, raw: raw}, true
		}

		if !isNameStart(input[idx]) {
			return shortcodeTag{}, false
		}
		keyStart := idx
		idx++
		for idx < len(input) && isNameChar(input[idx]) {
			idx++
		}
		key := input[keyStart:idx]
		idx = skipSpaces(input, idx)
		if idx >= len(input) || input[idx] != '=' {
			return shortcodeTag{}, false
		}
		idx++
		idx = skipSpaces(input, idx)
		if idx >= len(input) {
			return shortcodeTag{}, false
		}
		quote := input[idx]
		if quote != '"' && quote != '\'' {
			return shortcodeTag{}, false
		}
		idx++
		valueStart := idx
		for idx < len(input) && input[idx] != quote {
			idx++
		}
		if idx >= len(input) {
			return shortcodeTag{}, false
		}
		value := input[valueStart:idx]
		idx++

		if params == nil {
			params = make(map[string]string)
		}
		params[key] = value
	}
}

func stripShortcodes(input string) string {
	var out strings.Builder
	idx := 0

	for idx < len(input) {
		next := strings.Index(input[idx:], "{{")
		if next == -1 {
			out.WriteString(input[idx:])
			break
		}
		next += idx
		out.WriteString(input[idx:next])

		tag, ok := parseShortcodeTag(input, next)
		if !ok {
			out.WriteString(input[next : next+2])
			idx = next + 2
			continue
		}

		if tag.isClose {
			out.WriteString(tag.raw)
			idx = tag.end
			continue
		}

		if isTagStandalone(input, tag.start, tag.end) {
			if end, ok := findShortcodeEnd(input, tag); ok {
				idx = end
				continue
			}
		}

		idx = tag.end
	}

	return out.String()
}

func findShortcodeEnd(input string, tag shortcodeTag) (int, bool) {
	type frame struct {
		name      string
		delimiter byte
	}

	stack := []frame{{name: tag.name, delimiter: tag.delimiter}}
	idx := tag.end

	for idx < len(input) {
		next := strings.Index(input[idx:], "{{")
		if next == -1 {
			return 0, false
		}
		next += idx

		nested, ok := parseShortcodeTag(input, next)
		if !ok {
			idx = next + 2
			continue
		}

		if nested.isClose {
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				if current.name == nested.name && current.delimiter == nested.delimiter {
					stack = stack[:len(stack)-1]
					if len(stack) == 0 {
						return nested.end, true
					}
				}
			}
		} else if isTagStandalone(input, nested.start, nested.end) {
			stack = append(stack, frame{name: nested.name, delimiter: nested.delimiter})
		}
		idx = nested.end
	}

	return 0, false
}

func skipSpaces(input string, idx int) int {
	for idx < len(input) {
		if input[idx] != ' ' && input[idx] != '\t' && input[idx] != '\n' && input[idx] != '\r' {
			return idx
		}
		idx++
	}
	return idx
}

func consumeClosing(input string, idx int, delimiter byte) int {
	if delimiter == '<' {
		if strings.HasPrefix(input[idx:], ">}}") {
			return idx + 3
		}
		return -1
	}

	if strings.HasPrefix(input[idx:], "%}}") {
		return idx + 3
	}
	return -1
}

func isNameStart(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isNameChar(char byte) bool {
	return isNameStart(char) || (char >= '0' && char <= '9') || char == '_' || char == '-'
}
