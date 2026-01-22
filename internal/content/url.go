package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/shanepadgett/canopy/internal/core"
)

// computeURL generates the URL for a page based on permalink patterns.
func computeURL(cfg core.Config, section, slug string, date time.Time) string {
	// Look for section-specific permalink pattern
	pattern := ""
	if sectionCfg, ok := cfg.Sections[section]; ok && sectionCfg.Permalink != "" {
		pattern = sectionCfg.Permalink
	} else if p, ok := cfg.Permalinks[section]; ok {
		pattern = p
	}

	// Default pattern if none specified
	if pattern == "" {
		if section != "" {
			pattern = "/" + section + "/:slug/"
		} else {
			pattern = "/:slug/"
		}
	}

	// Replace tokens
	url := pattern
	url = strings.ReplaceAll(url, ":slug", slug)
	url = strings.ReplaceAll(url, ":section", section)

	// Date tokens
	if !date.IsZero() {
		url = strings.ReplaceAll(url, ":year", fmt.Sprintf("%04d", date.Year()))
		url = strings.ReplaceAll(url, ":month", fmt.Sprintf("%02d", date.Month()))
		url = strings.ReplaceAll(url, ":day", fmt.Sprintf("%02d", date.Day()))
	}

	// Ensure leading slash
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	// Ensure trailing slash
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	return url
}
