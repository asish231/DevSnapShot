package start

import (
	"archive/tar"
	"compress/gzip"
	"devsnap/pkg/metadata"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Unpack opens the snapshot and extracts it to the destDir.
// Returns the meta and any error.
func Unpack(snapshotPath, destDir string) (metadata.SnapshotMetadata, error) {
	var meta metadata.SnapshotMetadata

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return meta, fmt.Errorf("failed to create dest dir: %w", err)
	}

	file, err := os.Open(snapshotPath)
	if err != nil {
		return meta, fmt.Errorf("failed to open snapshot: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return meta, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// Meta file name to look for
	metaFileName := "metadata.json"
	metaFound := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return meta, fmt.Errorf("tar reading error: %w", err)
		}

		// Sanitize header name to prevent ZipSlip
		// On Windows, extracting a file named "F:/..." is bad.
		// We force all paths to be relative to result dir.
		cleanName := filepath.Clean(header.Name)
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, "..") {
			fmt.Printf("⚠️ Warning: Skipping unsafe file path: %s\n", header.Name)
			continue
		}

		target := filepath.Join(destDir, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return meta, err
			}
		case tar.TypeReg:
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return meta, err
			}

			// Extract file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return meta, err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return meta, err
			}
			outFile.Close()

			// If this is the metadata file, read it immediately
			if filepath.Base(header.Name) == metaFileName && !metaFound {
				bytes, err := ioutil.ReadFile(target)
				if err == nil {
					if err := json.Unmarshal(bytes, &meta); err == nil {
						metaFound = true
					}
				}
			}
		}
	}

	if !metaFound {
		return meta, fmt.Errorf("invalid snapshot: metadata.json not found")
	}

	return meta, nil
}
