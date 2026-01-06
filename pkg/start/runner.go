package start

import (
	"devsnap/pkg/metadata"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run executes the lifecycle commands in the given directory
func Run(dir string, meta metadata.SnapshotMetadata) error {
	fmt.Printf("🚀 Starting sandbox for '%s' (%s)...\n", meta.Name, meta.Environment.Type)

	// 1. Setup
	if len(meta.Commands.Setup) > 0 {
		fmt.Println("📦 Setting up environment...")
		for _, cmdStr := range meta.Commands.Setup {
			// Check for devpack marker
			if cmdStr == "#DEVPACK_INSTALL" {
				if err := installFromDevpack(dir); err != nil {
					return fmt.Errorf("devpack install failed: %w", err)
				}
				continue
			}

			if err := execute(dir, cmdStr); err != nil {
				return fmt.Errorf("setup failed: %w", err)
			}
		}
	}

	// 2. Run
	if meta.Commands.Run != "" {
		fmt.Printf("▶️  Running: %s\n", meta.Commands.Run)
		// Assuming generic runner for now (using system shell logic implicitly via execute)
		if err := execute(dir, meta.Commands.Run); err != nil {
			return fmt.Errorf("run failed: %w", err)
		}
	}

	return nil
}

func execute(dir, cmdStr string) error {
	// Simple execution: split by space (naive, but works for simple commands)
	// For better support, we might want to use "sh -c" or "cmd /C"
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil
	}

	// Windows fallback for npm/python
	// On Windows, 'npm' is often a .cmd file, so we need to be careful.
	// exec.Command usually handles PATH lookups well.

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Interactive support

	fmt.Printf("   [$] %s\n", cmdStr)
	return cmd.Run()
}

func installFromDevpack(dir string) error {
	devpackPath := filepath.Join(dir, "dependencies.devpack")
	// Using generic unmarshal to map since we know the structure
	content, err := ioutil.ReadFile(devpackPath)
	if err != nil {
		fmt.Println("   ⚠️ Could not find dependencies.devpack")
		return nil
	}

	var pack struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(content, &pack); err != nil {
		return fmt.Errorf("failed to parse devpack: %w", err)
	}

	if len(pack.Dependencies) == 0 {
		return nil
	}

	// Construct installation command
	args := []string{"install"}
	for pkg, ver := range pack.Dependencies {
		if ver == "latest" {
			args = append(args, pkg)
		} else {
			args = append(args, fmt.Sprintf("%s@%s", pkg, ver))
		}
	}

	fmt.Printf("   📦 Installing imports from devpack: %v\n", args[1:])

	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
