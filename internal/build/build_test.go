package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildShortcodes(t *testing.T) {
	configPath := testdataPath(t, "testdata", "site", "site.json")
	outputDir := t.TempDir()

	stats, err := Build(Options{
		ConfigPath: configPath,
		OutputDir:  outputDir,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	outputFile := filepath.Join(stats.Output, "guides", "shortcodes", "index.html")
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	html := string(data)
	assertContains(t, html, `class="shortcode-callout`)        // callout
	assertContains(t, html, `class="shortcode-callout-title"`) // callout title
	assertContains(t, html, `class="shortcode-figure"`)        // figure
	assertContains(t, html, `youtube.com/embed/dQw4w9WgXcQ`)   // youtube
	assertContains(t, html, `class="shortcode-toc"`)           // toc
	assertContains(t, html, `toc-level-2`)                     // toc entries
	assertContains(t, html, `class="shortcode-key-takeaways"`) // key takeaways
	assertContains(t, html, `class="shortcode-prereqs"`)       // prereqs
	assertContains(t, html, `class="shortcode-code-tabs"`)     // code tabs
	assertContains(t, html, `*Not Markdown*`)                  // raw inner content
	if strings.Contains(html, "<em>Not Markdown</em>") {
		t.Fatalf("expected raw code-tabs inner content")
	}
}

func testdataPath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to locate test file")
	}

	dir := filepath.Dir(file)
	root := filepath.Dir(filepath.Dir(dir))
	return filepath.Join(append([]string{root}, parts...)...)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q", needle)
	}
}
