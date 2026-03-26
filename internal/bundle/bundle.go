package bundle

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Find locates the bundle path. If path is empty, uses cwd.
// Returns the resolved path and whether it's a directory.
func Find(path string) (string, bool, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("could not determine working directory: %w", err)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", false, fmt.Errorf("path not found: %s", path)
	}

	if info.IsDir() {
		manifest := filepath.Join(path, "plugin.json")
		if _, err := os.Stat(manifest); err != nil {
			return "", false, fmt.Errorf("no plugin.json found in %s", path)
		}
		return path, true, nil
	}

	// It's a file — accept .sweater or .zip
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".sweater" && ext != ".zip" {
		return "", false, fmt.Errorf("expected a .sweater file or directory containing plugin.json, got: %s", path)
	}
	return path, false, nil
}

// Zip creates a zip archive of the directory in memory and returns the bytes.
func Zip(dir string) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and common ignore patterns
		base := filepath.Base(path)
		if base != "." && strings.HasPrefix(base, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if base == "node_modules" || base == "__pycache__" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(writer, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	return buf, nil
}

// Validate runs local pre-flight checks on a plugin directory.
// Returns a slice of errors found (empty means valid).
func Validate(dir string) []error {
	var errs []error

	iconPath := filepath.Join(dir, "images", "icon.png")
	f, err := os.Open(iconPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("missing required file: images/icon.png"))
		return errs
	}
	defer f.Close()

	cfg, err := png.DecodeConfig(f)
	if err != nil {
		errs = append(errs, fmt.Errorf("images/icon.png is not a valid PNG: %w", err))
		return errs
	}
	if cfg.Width != 512 || cfg.Height != 512 {
		errs = append(errs, fmt.Errorf("images/icon.png must be 512x512, got %dx%d", cfg.Width, cfg.Height))
	}

	return errs
}

// Open returns a reader for the bundle. If it's a directory, zips it first.
// Returns reader, size in bytes, and filename.
func Open(path string) (io.Reader, int64, string, error) {
	resolved, isDir, err := Find(path)
	if err != nil {
		return nil, 0, "", err
	}

	if isDir {
		if errs := Validate(resolved); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			return nil, 0, "", fmt.Errorf("bundle validation failed:\n  %s", strings.Join(msgs, "\n  "))
		}

		buf, err := Zip(resolved)
		if err != nil {
			return nil, 0, "", err
		}
		name := filepath.Base(resolved) + ".sweater"
		return buf, int64(buf.Len()), name, nil
	}

	f, err := os.Open(resolved)
	if err != nil {
		return nil, 0, "", err
	}
	info, _ := f.Stat()
	return f, info.Size(), filepath.Base(resolved), nil
}
