package create

import (
	"devsnap/pkg/metadata"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectProject inspects the files and determins the environment configuration
func DetectProject(root string) (metadata.EnvironmentConfig, metadata.LifecycleCommands, string) {
	// Defaults
	env := metadata.EnvironmentConfig{Type: "generic"}
	cmds := metadata.LifecycleCommands{}
	name := filepath.Base(root)

	// Check for Node.js
	if exists(filepath.Join(root, "package.json")) {
		env.Type = "node"
		env.Version = ">=18.0.0"
		cmds.Setup = []string{"npm install"}
		cmds.Run = "npm start"
		cmds.Test = "npm test"
		return env, cmds, name
	}

	// Advanced Node Detection (Missing package.json)
	// Scan recursively for .js, .ts, .jsx, .tsx files
	var codeFiles []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" || info.Name() == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" {
			codeFiles = append(codeFiles, path)
		}
		return nil
	})

	if len(codeFiles) > 0 {
		fmt.Printf("   ℹ️  Found %d script files. Checking for imports (missing package.json)...\n", len(codeFiles))
		dependencies := scanForNodeImports(codeFiles)

		if len(dependencies) > 0 {
			env.Type = "node"
			env.Version = ">=18.0.0"

			// Resolve versions into a map
			depMap := make(map[string]string)
			for _, dep := range dependencies {
				version := resolveNodeVersion(root, dep)
				if version != "" {
					depMap[dep] = version
				} else {
					depMap[dep] = "latest"
				}
			}

			// Generate .devpack file content
			devpackContent := map[string]interface{}{
				"type":         "node",
				"dependencies": depMap,
			}

			devpackPath := filepath.Join(root, "dependencies.devpack")
			bytes, _ := json.MarshalIndent(devpackContent, "", "  ")
			ioutil.WriteFile(devpackPath, bytes, 0644)
			fmt.Println("   📝 Generated dependencies.devpack")

			// Setup command now uses the devpack
			cmds.Setup = []string{"#DEVPACK_INSTALL"}

			// Guess run command
			if exists(filepath.Join(root, "vite.config.ts")) || exists(filepath.Join(root, "vite.config.js")) {
				cmds.Run = "npx vite"
			} else if exists(filepath.Join(root, "index.js")) {
				cmds.Run = "node index.js"
			} else {
				cmds.Run = "node " + filepath.Base(codeFiles[0])
			}

			return env, cmds, name
		}
	}

	// Check for Python
	if exists(filepath.Join(root, "requirements.txt")) {
		env.Type = "python"
		env.Version = ">=3.9"
		cmds.Setup = []string{"pip install -r requirements.txt"}

		if exists(filepath.Join(root, "manage.py")) {
			cmds.Run = "python manage.py runserver"
			cmds.Test = "python manage.py test"
		} else {
			cmds.Run = "python main.py"
		}
		return env, cmds, name
	}

	// Check for Go
	if exists(filepath.Join(root, "go.mod")) {
		env.Type = "go"
		cmds.Setup = []string{"go mod download"}
		cmds.Run = "go run ."
		cmds.Test = "go test ./..."
		return env, cmds, name
	}

	return env, cmds, name
}

// Detects valid package names from imports, ignoring local paths
func scanForNodeImports(files []string) []string {
	deps := make(map[string]bool)

	// Regex for require('x') and import ... from 'x'
	requireRegex := regexp.MustCompile(`require\(['"]([^'"]+)['"]\)`)
	importRegex := regexp.MustCompile(`from ['"]([^'"]+)['"]`)
	// Dynamic import regex: import('x')
	dynamicImportRegex := regexp.MustCompile(`import\(['"]([^'"]+)['"]\)`)

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		strContent := string(content)

		// Helper to process matches
		processMatch := func(match []string) {
			if len(match) > 1 {
				name := match[1]
				// Filter out local imports and non-packages
				if isLocalImport(name) || isBuiltinModule(name) {
					return
				}
				// Handle scoped packages (e.g. @types/node -> keep full name)
				// Handle subpaths (e.g. lodash/fp -> lodash)
				rootPkg := getRootPackageName(name)
				deps[rootPkg] = true
			}
		}

		for _, m := range requireRegex.FindAllStringSubmatch(strContent, -1) {
			processMatch(m)
		}
		for _, m := range importRegex.FindAllStringSubmatch(strContent, -1) {
			processMatch(m)
		}
		for _, m := range dynamicImportRegex.FindAllStringSubmatch(strContent, -1) {
			processMatch(m)
		}
	}

	var result []string
	for dep := range deps {
		result = append(result, dep)
	}
	return result
}

func isLocalImport(path string) bool {
	return strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "~/") ||
		strings.HasPrefix(path, "@/")
}

func getRootPackageName(path string) string {
	if strings.HasPrefix(path, "@") {
		// Scoped package: @scope/pkg/subpath -> @scope/pkg
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	} else {
		// Regular package: pkg/subpath -> pkg
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	return path
}

// resolveNodeVersion looks in node_modules for the package.json, or tries CLI
func resolveNodeVersion(root, packageName string) string {
	// 1. Try node_modules (Best source of truth for project)
	pkgPath := filepath.Join(root, "node_modules", packageName, "package.json")
	content, err := ioutil.ReadFile(pkgPath)
	if err == nil {
		var pkg struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(content, &pkg); err == nil {
			return pkg.Version
		}
	}

	// 2. Try CLI (Fallback for tools installed globally/in-path)
	// Only makes sense for bin-like packages, but we can try blindly for now.
	// We suppress stderr to avoid noise.
	cmd := exec.Command(packageName, "--version")
	out, err := cmd.Output()
	if err == nil {
		// Extract version using regex (find first x.y.z)
		re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
		match := re.FindString(string(out))
		if match != "" {
			return match
		}
	}

	return "" // Not found, caller will use "latest"
}

func isBuiltinModule(name string) bool {
	// Simplified list of node built-ins to ignore
	builtins := map[string]bool{
		"fs": true, "path": true, "os": true, "http": true, "https": true,
		"crypto": true, "util": true, "events": true, "child_process": true,
	}
	return builtins[name]
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
