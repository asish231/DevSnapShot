package create

import (
	"archive/tar"
	"compress/gzip"
	"devsnap/pkg/metadata"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateArchive packs the given files and metadata into a .devsnap tar.gz file
func CreateArchive(rootDir string, files []string, meta metadata.SnapshotMetadata, outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	// Create tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Write metadata.json as the first file
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	header := &tar.Header{
		Name: "metadata.json",
		Mode: 0644,
		Size: int64(len(metaJSON)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write metadata header: %w", err)
	}
	if _, err := tw.Write(metaJSON); err != nil {
		return fmt.Errorf("failed to write metadata body: %w", err)
	}

	// 2. Write project files
	for _, file := range files {
		if err := addFileToTar(tw, rootDir, file); err != nil {
			return fmt.Errorf("failed to archive file %s: %w", file, err)
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, rootDir, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(stat, stat.Name())
	if err != nil {
		return err
	}

	// Calculate relative path
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		// If we can't make it relative, fallback to base name to be safe
		relPath = filepath.Base(filePath)
	}

	// Ensure forward slashes for archive compatibility
	header.Name = filepath.ToSlash(relPath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}
