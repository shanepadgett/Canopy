package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shanepadgett/canopy/internal/core"
)

type stubShortcodeRenderer struct{}

func (stubShortcodeRenderer) RenderShortcode(name string, params map[string]string, inner string, innerIsHTML bool, page *core.Page) (string, error) {
	if innerIsHTML {
		return fmt.Sprintf("<sc name=%s html=%t>%s</sc>", name, innerIsHTML, inner), nil
	}
	return fmt.Sprintf("<sc name=%s html=%t>%s</sc>", name, innerIsHTML, inner), nil
}

func TestRenderInlineShortcode(t *testing.T) {
	input := "Hello {{< youtube id=\"abc\" >}} world"
	result := RenderWithOptions(input, RenderOptions{ShortcodeRenderer: stubShortcodeRenderer{}})

	if !strings.Contains(result.HTML, "<sc name=youtube") {
		t.Errorf("expected inline shortcode, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "Hello ") || !strings.Contains(result.HTML, " world") {
		t.Errorf("expected inline shortcode in paragraph, got %q", result.HTML)
	}
}

func TestRenderBlockShortcodeMarkdownInner(t *testing.T) {
	input := "{{< callout >}}\nInner **bold**\n{{< /callout >}}"
	result := RenderWithOptions(input, RenderOptions{ShortcodeRenderer: stubShortcodeRenderer{}})

	if strings.Contains(result.HTML, "<p><sc") {
		t.Errorf("expected block shortcode without paragraph wrapper, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "<strong>bold</strong>") {
		t.Errorf("expected markdown inner rendering, got %q", result.HTML)
	}
}

func TestRenderBlockShortcodeRawInner(t *testing.T) {
	input := "{{% code-tabs %}}\n*not markdown*\n{{% /code-tabs %}}"
	result := RenderWithOptions(input, RenderOptions{ShortcodeRenderer: stubShortcodeRenderer{}})

	if strings.Contains(result.HTML, "<em>") {
		t.Errorf("expected raw inner content, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "*not markdown*") {
		t.Errorf("expected raw inner text, got %q", result.HTML)
	}
}
