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

	// 1. Check for Angular
	if exists(filepath.Join(root, "angular.json")) {
		env.Type = "angular"
		env.Version = ">=14.0.0" // Default assumption, improved by Sherlock
		cmds.Setup = []string{"npm install"}
		cmds.Run = "npm start"
		cmds.Test = "npm test"

		// Attempt to refine version if package.json exists
		if v := resolveNodeVersion(root, "@angular/core"); v != "" {
			env.Version = v
		}
		return env, cmds, name
	}

	// 2. Check for Node.js (package.json)
	if exists(filepath.Join(root, "package.json")) {
		env.Type = "node"
		env.Version = ">=18.0.0"
		cmds.Setup = []string{"npm install"}
		cmds.Run = "npm start"
		cmds.Test = "npm test"
		return env, cmds, name
	}

	// 3. Check for Go (go.mod)
	if exists(filepath.Join(root, "go.mod")) {
		env.Type = "go"
		// Parse go.mod for version
		if v := resolveGoModVersion(root); v != "" {
			env.Version = v
		} else {
			env.Version = "1.21"
		}
		cmds.Setup = []string{"go mod download"}
		cmds.Run = "go run ."
		cmds.Test = "go test ./..."
		return env, cmds, name
	}

	// 4. Sherlock Mode: No manifest files found!
	// Scan recursively for code files to guess the environment
	var codeFiles []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common ignore dirs
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" || info.Name() == "build" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" || ext == ".go" {
			codeFiles = append(codeFiles, path)
		}
		return nil
	})

	if len(codeFiles) > 0 {
		fmt.Printf("   🕵️  Sherlock Mode: Found %d source files. Analyzing...\n", len(codeFiles))

		// Check for Go files first (stronger signal if go.mod is missing but .go files exist)
		hasGo := false
		for _, f := range codeFiles {
			if strings.HasSuffix(f, ".go") {
				hasGo = true
				break
			}
		}

		if hasGo {
			env.Type = "go"
			env.Version = "1.21" // Safe default

			// Scan imports
			dependencies := scanForGoImports(codeFiles)
			if len(dependencies) > 0 {
				fmt.Printf("      Found %d dependencies (e.g. %s)\n", len(dependencies), summarizeDeps(dependencies))
				depMap := make(map[string]string)
				for _, dep := range dependencies {
					v := resolveGoVersion(dep)
					depMap[dep] = v
					fmt.Printf("      - %s @ %s\n", dep, v)
				}
				createDevpack(root, "go", depMap)
				cmds.Setup = []string{"#DEVPACK_INSTALL"}
			}

			cmds.Run = "go run ."
			return env, cmds, name
		}

		// Check for Node/JS files
		dependencies := scanForNodeImports(codeFiles)
		if len(dependencies) > 0 {
			env.Type = "node"
			env.Version = ">=18.0.0"

			fmt.Printf("   📦 Identified %d unique packages:\n", len(dependencies))

			// Resolve versions
			depMap := make(map[string]string)
			for _, dep := range dependencies {
				v := resolveNodeVersion(root, dep)
				depMap[dep] = v
				fmt.Printf("      - %s: %s\n", dep, v)
			}

			createDevpack(root, "node", depMap)
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

	// 5. Check for Python (Classic detection)
	if exists(filepath.Join(root, "requirements.txt")) {
		env.Type = "python"
		env.Version = ">=3.9"
		cmds.Setup = []string{"pip install -r requirements.txt"}
		if exists(filepath.Join(root, "manage.py")) {
			cmds.Run = "python manage.py runserver"
		} else {
			cmds.Run = "python main.py"
		}
		return env, cmds, name
	}

	return env, cmds, name
}

// createDevpack writes the .devpack file
func createDevpack(root, envType string, deps map[string]string) {
	content := map[string]interface{}{
		"type":         envType,
		"dependencies": deps,
	}
	path := filepath.Join(root, "dependencies.devpack")
	bytes, _ := json.MarshalIndent(content, "", "  ")
	ioutil.WriteFile(path, bytes, 0644)
	fmt.Println("   📝 Generated dependencies.devpack")
}

// --- Node.js Logic ---

func scanForNodeImports(files []string) []string {
	deps := make(map[string]bool)
	requireRegex := regexp.MustCompile(`require\(['"]([^'"]+)['"]\)`)
	importRegex := regexp.MustCompile(`from ['"]([^'"]+)['"]`)
	dynamicRegex := regexp.MustCompile(`import\(['"]([^'"]+)['"]\)`)

	for _, file := range files {
		// Only scan JS/TS files
		if !strings.HasSuffix(file, ".js") && !strings.HasSuffix(file, ".ts") && !strings.HasSuffix(file, ".jsx") && !strings.HasSuffix(file, ".tsx") {
			continue
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		str := string(content)

		add := func(match []string) {
			if len(match) > 1 {
				name := match[1]
				if !isLocalImport(name) && !isBuiltinModule(name) {
					deps[getRootPackageName(name)] = true
				}
			}
		}

		for _, m := range requireRegex.FindAllStringSubmatch(str, -1) {
			add(m)
		}
		for _, m := range importRegex.FindAllStringSubmatch(str, -1) {
			add(m)
		}
		for _, m := range dynamicRegex.FindAllStringSubmatch(str, -1) {
			add(m)
		}
	}

	var result []string
	for d := range deps {
		result = append(result, d)
	}
	return result
}

func resolveNodeVersion(root, packageName string) string {
	// 1. Try node_modules (Truth)
	pkgPath := filepath.Join(root, "node_modules", packageName, "package.json")
	if content, err := ioutil.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(content, &pkg) == nil {
			return pkg.Version
		}
	}

	// 2. Try 'npm list' (Installed but maybe flattened)
	// Suppress output, just check result
	cmd := exec.Command("npm", "list", packageName, "--json", "--depth=0")
	if out, err := cmd.Output(); err == nil {
		var res struct {
			Dependencies map[string]struct {
				Version string `json:"version"`
			} `json:"dependencies"`
		}
		if json.Unmarshal(out, &res) == nil {
			if val, ok := res.Dependencies[packageName]; ok {
				return val.Version
			}
		}
	}

	// 3. Fallback
	return "latest"
}

// --- Go Logic ---

func resolveGoModVersion(root string) string {
	content, err := ioutil.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`go\s+([0-9]+\.[0-9]+)`)
	match := re.FindStringSubmatch(string(content))
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func scanForGoImports(files []string) []string {
	deps := make(map[string]bool)
	importRegex := regexp.MustCompile(`(?m)^\s*import\s*\(\s*((?:[^\)]+\s*)*)\)|^\s*import\s+"([^"]+)"`)

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		str := string(content)

		matches := importRegex.FindAllStringSubmatch(str, -1)
		for _, m := range matches {
			if m[2] != "" {
				// Single line import
				if !isStandardLib(m[2]) {
					deps[m[2]] = true
				}
			} else if m[1] != "" {
				// Block import
				lines := strings.Split(m[1], "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// Extract string inside quotes
					if start := strings.Index(line, "\""); start != -1 {
						if end := strings.LastIndex(line, "\""); end > start {
							pkg := line[start+1 : end]
							if !isStandardLib(pkg) {
								deps[pkg] = true
							}
						}
					}
				}
			}
		}
	}
	var res []string
	for d := range deps {
		res = append(res, d)
	}
	return res
}

func resolveGoVersion(pkgName string) string {
	// 1. Try 'go list'
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", pkgName)
	if out, err := cmd.Output(); err == nil {
		ver := strings.TrimSpace(string(out))
		if ver != "" {
			return ver
		}
	}
	return "latest"
}

func isStandardLib(pkg string) bool {
	// Heuristic: std lib usually doesn't have "." in the first part (e.g. "fmt", "net/http")
	// External pkgs usually are "github.com/...", "gopkg.in/..."
	parts := strings.Split(pkg, "/")
	if len(parts) > 0 && strings.Contains(parts[0], ".") {
		return false
	}
	// Edge case: "golang.org/x/..." is external but looks like domain
	// But generally, single word like "fmt" is std.
	// "net/http" has no domain.
	return true
}

// --- Helpers ---

func isLocalImport(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "@/")
}

func getRootPackageName(path string) string {
	if strings.HasPrefix(path, "@") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	} else {
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	return path
}

func isBuiltinModule(name string) bool {
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
