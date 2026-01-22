package build

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Writer handles writing output files.
type Writer struct {
	outputDir string
}

// NewWriter creates a new output writer.
func NewWriter(outputDir string) *Writer {
	return &Writer{outputDir: outputDir}
}

// Clean removes and recreates the output directory.
func (w *Writer) Clean() error {
	// Remove existing output
	if err := os.RemoveAll(w.outputDir); err != nil {
		return fmt.Errorf("removing output dir: %w", err)
	}

	// Create fresh output directory
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	return nil
}

// WritePage writes an HTML page for the given URL.
// URL /blog/hello/ -> outputDir/blog/hello/index.html
// URL / -> outputDir/index.html
func (w *Writer) WritePage(url, html string) error {
	// Convert URL to file path
	filePath := w.urlToPath(url)

	// Create parent directories
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}

	return nil
}

func (w *Writer) urlToPath(url string) string {
	// Remove leading slash
	url = strings.TrimPrefix(url, "/")

	// Handle root URL
	if url == "" || url == "/" {
		return filepath.Join(w.outputDir, "index.html")
	}

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Create clean URL structure: /blog/post/ -> blog/post/index.html
	return filepath.Join(w.outputDir, url, "index.html")
}

// CopyStatic copies the static directory to the output directory.
func (w *Writer) CopyStatic(staticDir string) error {
	// Check if static directory exists
	info, err := os.Stat(staticDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("static directory does not exist")
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("static path is not a directory")
	}

	return filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path
		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(w.outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		return copyFile(path, destPath)
	})
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
