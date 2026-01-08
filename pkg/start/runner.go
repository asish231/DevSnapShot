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
	fmt.Printf("🚀 Starting sandbox for '%s'...\n", meta.Name)
	if manualMode {
		fmt.Println("🎮 Manual Control Mode Active: You will be prompted before each step.")
	}

	// 0. Env Guard (Check Secrets)
	ensureEnvTemplate(dir, meta.RequiredVars)
	loadEnvFile(dir)
	// Validate secrets
	if len(meta.RequiredVars) > 0 {
		fmt.Printf("🔐 Checking %d required environment variables...\n", len(meta.RequiredVars))
		for _, v := range meta.RequiredVars {
			if os.Getenv(v) == "" {
				fmt.Printf("   ⚠️  Missing Secret: '%s'\n", v)
				fmt.Printf("      Enter value for %s: ", v)
				var val string
				fmt.Scanln(&val)
				if val != "" {
					os.Setenv(v, strings.TrimSpace(val))
					fmt.Printf("      ✅ Set %s for this session.\n", v)
				} else {
					fmt.Printf("      ⚠️  Skipped %s (App might fail)\n", v)
				}
			}
		}
	}

	// 1. Iterate Environments
	for _, env := range meta.Environments {
		fmt.Printf("\n🌍 Setting up environment: %s (%s)\n", env.Type, env.Version)

		// A. Pre-flight Check (Runtime Availability)
		if !checkRuntime(env.Type) {
			fmt.Printf("   ❌ Compiler/Runtime not found: '%s'. Skipping setup & run.\n", env.Type)
			continue
		}

		// B. Setup
		if len(env.Setup) > 0 {
			if manualMode && !promptUser(fmt.Sprintf("Install dependencies for %s?", env.Type)) {
				fmt.Println("   ⏭️  Skipping setup...")
			} else {
				fmt.Println("   📦 Installing dependencies...")
				for _, cmdStr := range env.Setup {
					// Check for devpack marker with filename support
					// Format: #DEVPACK:filename or legacy #DEVPACK_INSTALL
					if strings.HasPrefix(cmdStr, "#DEVPACK") {
						filename := "dependencies.devpack" // Default legacy
						if strings.HasPrefix(cmdStr, "#DEVPACK:") {
							filename = strings.TrimPrefix(cmdStr, "#DEVPACK:")
						}

						if err := installFromDevpack(dir, filename, manualMode); err != nil {
							fmt.Printf("      ⚠️  Devpack install failed: %v\n", err)
						}
						continue
					}

					if err := execute(dir, cmdStr); err != nil {
						fmt.Printf("      ⚠️  Setup command failed: %v\n", err)
					}
				}
			}
		}

		// C. Run
		if env.Run != "" {
			// In standard mode, we *ask* before running to avoid blocking the shell forever on the first service
			// But user wants "One-Click".
			// Compromise: In manual mode we ask. In auto mode, we warn "Starting X..."
			// BUT: If we have multiple run commands (Node + Python), the first one creates a blocking process?
			// devsnap is designed for single-process snapshots usually.
			// For Polyglot, maybe we should run them in background?
			// For now, let's keep it sequential / blocking.

			if promptUser(fmt.Sprintf("Run start command for %s?\n    CMD: %s", env.Type, env.Run)) {
				fmt.Printf("▶️  Running: %s\n", env.Run)
				if err := execute(dir, env.Run); err != nil {
					return fmt.Errorf("run failed: %w", err)
				}
			} else {
				fmt.Println("   ⏭️  Skipping run command.")
			}
		}
	}

	return nil
}

func checkRuntime(envType string) bool {
	var cmd *exec.Cmd
	switch envType {
	case "go":
		cmd = exec.Command("go", "version")
	case "node", "angular":
		cmd = exec.Command("node", "-v")
	case "python":
		cmd = exec.Command("python", "--version")
	case "rust":
		cmd = exec.Command("cargo", "--version")
	case "java":
		// Check for Maven as it's the primary tool we use
		cmd = exec.Command("mvn", "-version")
	case "php":
		cmd = exec.Command("php", "-v")
	default:
		return true // Unknown types assumed present or generic
	}
	return cmd.Run() == nil
}

func execute(dir, cmdStr string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Printf("   [$] %s\n", cmdStr)
	return cmd.Run()
}

func installFromDevpack(dir, filename string, manualMode bool) error {
	devpackPath := filepath.Join(dir, filename)
	// Using generic unmarshal to map since we know the structure
	content, err := ioutil.ReadFile(devpackPath)
	if err != nil {
		fmt.Printf("   ⚠️ Could not find %s\n", filename)
		return nil
	}

	var pack struct {
		Type         string            `json:"type"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(content, &pack); err != nil {
		return fmt.Errorf("failed to parse devpack: %w", err)
	}

	// Branch based on type
	if pack.Type == "go" {
		return installGoDeps(dir, pack.Dependencies, manualMode)
	} else if strings.HasPrefix(pack.Type, "node") || pack.Type == "angular" { // Handle "node (TypeScript)"
		return installNodeDeps(dir, pack.Dependencies, manualMode)
	} else if pack.Type == "python" {
		return installPythonDeps(dir, pack.Dependencies, manualMode)
	} else if pack.Type == "rust" || pack.Type == "java" || pack.Type == "php" {
		fmt.Printf("   📦 %s dependencies are handled by the build tool (cargo/mvn/composer).\n", pack.Type)
		return nil
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

func installPythonDeps(dir string, deps map[string]string, manualMode bool) error {
	// pip install pkg==ver pkg2==ver2
	args := []string{"install"}

	for pkg, ver := range deps {
		if ver == "latest" || ver == "" {
			args = append(args, pkg)
		} else {
			args = append(args, fmt.Sprintf("%s==%s", pkg, ver))
		}
	}

	cmdStr := fmt.Sprintf("pip %s", strings.Join(args, " "))
	if manualMode {
		if !promptUser(fmt.Sprintf("Install Python dependencies (%d packages)?\n   '%s'", len(deps), cmdStr)) {
			fmt.Println("   ⏭️  Skipping dependency installation...")
			return nil
		}
	}

	fmt.Printf("   📦 Installing Python imports from devpack: %v\n", args[1:])
	cmd := exec.Command("pip", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// promptUser asks for confirmation (Y/n)
func promptUser(question string) bool {
	fmt.Printf("\n[?] %s (Y/n): ", question)
	var response string
	fmt.Scanln(&response) // Wait for enter
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "" || response == "y" || response == "yes"
}

func loadEnvFile(dir string) {
	envPath := filepath.Join(dir, ".env")
	content, err := ioutil.ReadFile(envPath)
	if err != nil {
		return // No .env file, ignore
	}

	fmt.Println("📄 Found .env file, loading variables...")
	lines := strings.Split(string(content), "\n")
	loaded := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Basic cleanup of quotes
			val = strings.Trim(val, `"'`)

			// Only set if not already set (optional, but standard behavior usually overrides)
			os.Setenv(key, val)
			loaded++
		}
	}
	if loaded > 0 {
		fmt.Printf("   ✅ Loaded %d variables from .env\n", loaded)
	}
}

func ensureEnvTemplate(dir string, required []string) {
	if len(required) == 0 {
		return
	}
	envPath := filepath.Join(dir, ".env")

	// Read existing content
	existing := ""
	if content, err := ioutil.ReadFile(envPath); err == nil {
		existing = string(content)
	}

	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	added := 0
	for _, v := range required {
		// Simple check if key exists in file
		// Note: robust parsing is better, but this suffices for "adding missing keys"
		if !strings.Contains(existing, v+"=") {
			if _, err := f.WriteString(fmt.Sprintf("\n%s=", v)); err == nil {
				added++
			}
		}
	}

	if added > 0 {
		fmt.Printf("   📄 Added %d missing keys to .env (values are empty)\n", added)
	}
}
