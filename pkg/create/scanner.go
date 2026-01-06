package create

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnores is a list of patterns/directories to always ignore
var DefaultIgnores = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
	".devsnap":     true,
	".env":         true, // Security: don't snapshot secrets by default
	"dist":         true,
	"build":        true,
}

// ScanDirectory walks the given path and returns a list of files to include
func ScanDirectory(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for checking against ignores
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Normalize path separators for consistent checking
		parts := strings.Split(filepath.ToSlash(relPath), "/")

		// Check if any part of the path is in the ignore list
		for _, part := range parts {
			if DefaultIgnores[part] {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// Special check for .devsnap files to avoid recursion
			if strings.HasSuffix(part, ".devsnap") {
				return nil
			}
		}

		// Don't add directories to the file list, only files
		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
