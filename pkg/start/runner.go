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
func Run(dir string, meta metadata.SnapshotMetadata, manualMode bool) error {
	fmt.Printf("🚀 Starting sandbox for '%s' (%s)...\n", meta.Name, meta.Environment.Type)
	if manualMode {
		fmt.Println("🎮 Manual Control Mode Active: You will be prompted before each step.")
	}

	// 1. Setup
	if len(meta.Commands.Setup) > 0 {
		fmt.Println("📦 Setting up environment...")
		for _, cmdStr := range meta.Commands.Setup {
			// Check for devpack marker
			if cmdStr == "#DEVPACK_INSTALL" {
				if err := installFromDevpack(dir, manualMode); err != nil {
					return fmt.Errorf("devpack install failed: %w", err)
				}
				continue
			}

			if manualMode {
				if !promptUser(fmt.Sprintf("Run setup command: '%s'?", cmdStr)) {
					fmt.Println("   ⏭️  Skipping...")
					continue
				}
			}

			if err := execute(dir, cmdStr); err != nil {
				return fmt.Errorf("setup failed: %w", err)
			}
		}
	}

	// 2. Run
	if meta.Commands.Run != "" {
		if manualMode {
			if !promptUser(fmt.Sprintf("Start application: '%s'?", meta.Commands.Run)) {
				fmt.Println("   ⏭️  Skipping run command.")
				return nil
			}
		}

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

func installFromDevpack(dir string, manualMode bool) error {
	devpackPath := filepath.Join(dir, "dependencies.devpack")
	// Using generic unmarshal to map since we know the structure
	content, err := ioutil.ReadFile(devpackPath)
	if err != nil {
		fmt.Println("   ⚠️ Could not find dependencies.devpack")
		return nil
	}

	var pack struct {
		Type         string            `json:"type"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(content, &pack); err != nil {
		return fmt.Errorf("failed to parse devpack: %w", err)
	}

	if len(pack.Dependencies) == 0 {
		return nil
	}

	// Branch based on type
	if pack.Type == "go" {
		return installGoDeps(dir, pack.Dependencies, manualMode)
	} else if pack.Type == "node" || pack.Type == "angular" {
		return installNodeDeps(dir, pack.Dependencies, manualMode)
	}

	return nil
}

func installNodeDeps(dir string, deps map[string]string, manualMode bool) error {
	// Construct installation command
	args := []string{"install"}
	for pkg, ver := range deps {
		if ver == "latest" {
			args = append(args, pkg)
		} else {
			args = append(args, fmt.Sprintf("%s@%s", pkg, ver))
		}
	}

	cmdStr := fmt.Sprintf("npm %s", strings.Join(args, " "))
	if manualMode {
		if !promptUser(fmt.Sprintf("Install Node dependencies (%d packages)?\n   '%s'", len(deps), cmdStr)) {
			fmt.Println("   ⏭️  Skipping dependency installation...")
			return nil
		}
	}

	fmt.Printf("   📦 Installing Node imports from devpack: %v\n", args[1:])

	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installGoDeps(dir string, deps map[string]string, manualMode bool) error {
	fmt.Println("   📦 Installing Go dependencies...")

	for pkg, ver := range deps {
		target := pkg
		if ver != "latest" && ver != "" {
			target = fmt.Sprintf("%s@%s", pkg, ver)
		} else {
			target = pkg + "@latest"
		}

		if manualMode {
			if !promptUser(fmt.Sprintf("Install Go package: '%s'?", target)) {
				fmt.Printf("   ⏭️  Skipping %s...\n", pkg)
				continue
			}
		}

		fmt.Printf("      -> go get %s\n", target)
		cmd := exec.Command("go", "get", target)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg, err)
		}
	}
	return nil
}

// promptUser asks for confirmation (Y/n)
func promptUser(question string) bool {
	fmt.Printf("\n[?] %s (Y/n): ", question)
	var response string
	fmt.Scanln(&response) // Wait for enter
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "" || response == "y" || response == "yes"
}
