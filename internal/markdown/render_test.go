package markdown

import (
	"strings"
	"testing"
)

func TestRenderHeadings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHTML string
		wantTOC  int
	}{
		{
			name:     "h1",
			input:    "# Hello World",
			wantHTML: `<h1 id="hello-world">Hello World</h1>`,
			wantTOC:  1,
		},
		{
			name:     "h2",
			input:    "## Features",
			wantHTML: `<h2 id="features">Features</h2>`,
			wantTOC:  1,
		},
		{
			name:     "multiple headings",
			input:    "# Title\n\n## Section 1\n\n## Section 2",
			wantHTML: `<h1 id="title">Title</h1>`,
			wantTOC:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Render(tt.input)
			if !strings.Contains(result.HTML, tt.wantHTML) {
				t.Errorf("HTML = %q, want to contain %q", result.HTML, tt.wantHTML)
			}
			if len(result.TOC) != tt.wantTOC {
				t.Errorf("TOC len = %d, want %d", len(result.TOC), tt.wantTOC)
			}
		})
	}
}

func TestRenderParagraphs(t *testing.T) {
	input := "This is a paragraph.\n\nThis is another paragraph."
	result := Render(input)

	if !strings.Contains(result.HTML, "<p>This is a paragraph.</p>") {
		t.Errorf("expected first paragraph, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "<p>This is another paragraph.</p>") {
		t.Errorf("expected second paragraph, got %q", result.HTML)
	}
}

func TestRenderInlineFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bold", "**bold text**", "<strong>bold text</strong>"},
		{"italic", "*italic text*", "<em>italic text</em>"},
		{"code", "`inline code`", "<code>inline code</code>"},
		{"link", "[link](https://example.com)", `<a href="https://example.com">link</a>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Render(tt.input)
			if !strings.Contains(result.HTML, tt.want) {
				t.Errorf("HTML = %q, want to contain %q", result.HTML, tt.want)
			}
		})
	}
}

func TestRenderCodeBlock(t *testing.T) {
	input := "```go\nfunc main() {}\n```"
	result := Render(input)

	if !strings.Contains(result.HTML, `<pre><code class="language-go">`) {
		t.Errorf("expected code block with language, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "func main() {}") {
		t.Errorf("expected code content, got %q", result.HTML)
	}
}

func TestRenderLists(t *testing.T) {
	t.Run("unordered", func(t *testing.T) {
		input := "- Item 1\n- Item 2\n- Item 3"
		result := Render(input)

		if !strings.Contains(result.HTML, "<ul>") {
			t.Errorf("expected ul tag, got %q", result.HTML)
		}
		if !strings.Contains(result.HTML, "<li>Item 1</li>") {
			t.Errorf("expected list items, got %q", result.HTML)
		}
	})

	t.Run("ordered", func(t *testing.T) {
		input := "1. First\n2. Second\n3. Third"
		result := Render(input)

		if !strings.Contains(result.HTML, "<ol>") {
			t.Errorf("expected ol tag, got %q", result.HTML)
		}
		if !strings.Contains(result.HTML, "<li>First</li>") {
			t.Errorf("expected list items, got %q", result.HTML)
		}
	})
}

func TestRenderBlockquote(t *testing.T) {
	input := "> This is a quote"
	result := Render(input)

	if !strings.Contains(result.HTML, "<blockquote>") {
		t.Errorf("expected blockquote, got %q", result.HTML)
	}
}

func TestRenderSummary(t *testing.T) {
	input := "This is the first paragraph that should become the summary.\n\n## Heading\n\nMore content here."
	result := Render(input)

	if !strings.Contains(result.Summary, "This is the first paragraph") {
		t.Errorf("expected summary from first paragraph, got %q", result.Summary)
	}
}
