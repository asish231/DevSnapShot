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
func DetectProject(root string) ([]metadata.EnvironmentConfig, metadata.LifecycleCommands, string, []string) {
	// Defaults
	var envs []metadata.EnvironmentConfig
	cmds := metadata.LifecycleCommands{} // Kept for legacy/global or final override? Can stay empty.
	name := filepath.Base(root)

	// Global Scan for Code Files (for Env Guard & Sherlock)
	var codeFiles []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" || info.Name() == "build" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" || ext == ".go" || ext == ".py" {
			codeFiles = append(codeFiles, path)
		}
		return nil
	})

	// Env Guard
	requiredVars := removeDuplicates(scanForEnvVars(codeFiles))

	// 1. Check for Angular
	if exists(filepath.Join(root, "angular.json")) {
		env := metadata.EnvironmentConfig{
			Type:    "angular",
			Version: ">=14.0.0",
			Setup:   []string{"npm install"},
			Run:     "npm start",
		}
		if v := resolveNodeVersion(root, "@angular/core"); v != "" {
			env.Version = v
		}
		envs = append(envs, env)
	}

	// 2. Check for Node.js (package.json) - Only if not Angular (usually mutually exclusive but can coexist)
	// If angular.json exists, package.json definitely exists. We might not want to double detect.
	// Simple rule: Angular IMPLIES Node. If we have Angular, skip generic Node check?
	// Or maybe just let them coexist? Let's treat them as distinct for now, but usually they share package.json.
	// If Angular detected, we skip generic Node to avoid duplicate "npm install".
	alreadyNode := false
	for _, e := range envs {
		if e.Type == "angular" {
			alreadyNode = true
		}
	}

	if !alreadyNode && exists(filepath.Join(root, "package.json")) {
		env := metadata.EnvironmentConfig{
			Type:    "node",
			Version: ">=18.0.0",
			Setup:   []string{"npm install"},
			Run:     "npm start",
		}
		// TypeScript Enhancement
		if exists(filepath.Join(root, "tsconfig.json")) {
			env.Type = "node (TypeScript)"
			// If build script exists, we might want to run it, but 'npm start' is safer default.
		}
		envs = append(envs, env)
	}

	// 3. Check for PHP (composer.json)
	if exists(filepath.Join(root, "composer.json")) {
		env := metadata.EnvironmentConfig{
			Type:    "php",
			Version: ">=8.0",
			Setup:   []string{"composer install"},
			Run:     "php -S localhost:8000", // Default built-in server
		}
		// Try to find an entry point
		if exists(filepath.Join(root, "public/index.php")) {
			env.Run = "php -S localhost:8000 -t public"
		} else if exists(filepath.Join(root, "artisan")) {
			// Laravel
			env.Run = "php artisan serve"
		}
		envs = append(envs, env)
	}

	// 3. Check for Go (go.mod)
	if exists(filepath.Join(root, "go.mod")) {
		env := metadata.EnvironmentConfig{
			Type:  "go",
			Setup: []string{"go mod download"},
			Run:   "go run .",
		}
		if v := resolveGoModVersion(root); v != "" {
			env.Version = v
		} else {
			env.Version = "1.21"
		}
		envs = append(envs, env)
	}

	// 4. Check for Rust (Cargo.toml)
	if exists(filepath.Join(root, "Cargo.toml")) {
		env := metadata.EnvironmentConfig{
			Type:    "rust",
			Version: "1.70.0", // Safe default
			Setup:   []string{"cargo build"},
			Run:     "cargo run",
		}
		envs = append(envs, env)
	}

	// 5. Check for Java (pom.xml - Maven)
	if exists(filepath.Join(root, "pom.xml")) {
		env := metadata.EnvironmentConfig{
			Type:    "java",
			Version: "17",
			Setup:   []string{"mvn clean install"},
		}
		// Smart heuristic for run command
		if exists(filepath.Join(root, "src/main/resources/application.properties")) || exists(filepath.Join(root, "src/main/resources/application.yml")) {
			// Likely Spring Boot
			env.Run = "mvn spring-boot:run"
		} else {
			// Fallback: Try to find a produced JAR (heuristically assuming target/app.jar or java -jar)
			// But since we can't know the jar name before build, "java -jar target/*.jar" is tricky in raw shell without wildcard expansion support in 'execute'.
			// Safest default is to let user define it or use a standard convention.
			// Let's assume standard executable jar or allow manual override.
			env.Run = "java -jar target/app.jar"
		}
		envs = append(envs, env)
	}

	// 4. Sherlock Mode (If no manifests found for a language, try to detect it from code)
	// We check if we already have detected a language.
	hasGoEnv := false
	hasNodeEnv := false
	hasPyEnv := false
	for _, e := range envs {
		if e.Type == "go" {
			hasGoEnv = true
		}
		if e.Type == "node" || e.Type == "angular" {
			hasNodeEnv = true
		}
		if e.Type == "python" {
			hasPyEnv = true
		}
	}

	// Sherlock Go
	if !hasGoEnv {
		hasGoFile := false
		for _, f := range codeFiles {
			if strings.HasSuffix(f, ".go") {
				hasGoFile = true
				break
			}
		}

		if hasGoFile {
			env := metadata.EnvironmentConfig{Type: "go", Version: "1.21", Run: "go run ."}
			deps := scanForGoImports(codeFiles)
			if len(deps) > 0 {
				fmt.Printf("   🕵️  Sherlock (Go): Found %d dependencies. Generating devpack...\n", len(deps))
				depMap := make(map[string]string)
				for _, d := range deps {
					depMap[d] = resolveGoVersion(d)
				}
				createDevpack(root, "go", depMap, "go.devpack")
				env.Setup = []string{"#DEVPACK:go.devpack"}
			}
			envs = append(envs, env)
		}
	}

	// Sherlock Node
	if !hasNodeEnv {
		// Only check imports if no package.json found
		deps := scanForNodeImports(codeFiles)
		if len(deps) > 0 {
			env := metadata.EnvironmentConfig{Type: "node", Version: ">=18.0.0"}
			fmt.Printf("   �️  Sherlock (Node): Found %d dependencies. Generating devpack...\n", len(deps))
			depMap := make(map[string]string)
			for _, d := range deps {
				depMap[d] = resolveNodeVersion(root, d)
			}
			createDevpack(root, "node", depMap, "node.devpack")
			env.Setup = []string{"#DEVPACK:node.devpack"}

			// Guess run
			if exists(filepath.Join(root, "index.js")) {
				env.Run = "node index.js"
			} else {
				env.Run = "node " + filepath.Base(codeFiles[0])
			}

			envs = append(envs, env)
		}
	}

	// Sherlock Python (or Manifest Python)
	// Manifest check: requirements.txt
	if !hasPyEnv {
		if exists(filepath.Join(root, "requirements.txt")) {
			env := metadata.EnvironmentConfig{
				Type:    "python",
				Version: ">=3.9",
				Setup:   []string{"pip install -r requirements.txt"},
			}
			if exists(filepath.Join(root, "manage.py")) {
				env.Run = "python manage.py runserver"
			} else {
				env.Run = "python main.py"
			}
			envs = append(envs, env)
		} else {
			// Pure Sherlock Python
			hasPyFile := false
			for _, f := range codeFiles {
				if strings.HasSuffix(f, ".py") {
					hasPyFile = true
					break
				}
			}
			if hasPyFile {
				env := metadata.EnvironmentConfig{Type: "python", Version: "3.10"}
				deps := scanForPythonImports(codeFiles)
				if len(deps) > 0 {
					fmt.Printf("   �️  Sherlock (Python): Found %d dependencies. Generating devpack...\n", len(deps))
					depMap := make(map[string]string)
					for _, d := range deps {
						depMap[d] = resolvePythonVersion(d)
					}
					createDevpack(root, "python", depMap, "python.devpack")
					env.Setup = []string{"#DEVPACK:python.devpack"}
				} else {
					// No deps detected? Maybe just standard lib.
					// Don't add install command.
				}

				// Guess run
				if exists(filepath.Join(root, "main.py")) {
					env.Run = "python main.py"
				} else if exists(filepath.Join(root, "app.py")) {
					env.Run = "python app.py"
				} else {
					env.Run = "python " + filepath.Base(codeFiles[0])
				}

				envs = append(envs, env)
			}
		}
	}

	// Fallback if nothing detected
	if len(envs) == 0 {
		envs = append(envs, metadata.EnvironmentConfig{Type: "generic"})
	}

	return envs, cmds, name, requiredVars
}

// createDevpack writes the .devpack file
func createDevpack(root, envType string, deps map[string]string, filename string) {
	content := map[string]interface{}{
		"type":         envType,
		"dependencies": deps,
	}
	path := filepath.Join(root, filename)
	bytes, _ := json.MarshalIndent(content, "", "  ")
	ioutil.WriteFile(path, bytes, 0644)
	fmt.Printf("   📝 Generated %s\n", filename)
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

// exists checks if a file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// --- Env Guard Logic ---

func scanForEnvVars(files []string) []string {
	vars := make(map[string]bool)

	// regexes
	// Node: process.env.API_KEY or process.env['API_KEY']
	nodeRegex := regexp.MustCompile(`process\.env\.([A-Z_0-9]+)|process\.env\['([A-Z_0-9]+)'\]`)
	// Go: os.Getenv("API_KEY") or os.LookupEnv("API_KEY")
	goRegex := regexp.MustCompile(`os\.(?:Getenv|LookupEnv)\("([A-Z_0-9]+)"\)`)
	// Python: os.environ.get("API_KEY") or os.getenv("API_KEY") or os.environ["API_KEY"]
	pyRegex := regexp.MustCompile(`os\.(?:environ\.get|getenv|environ\[")["']([A-Z_0-9]+)["']`)

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		str := string(content)

		for _, m := range nodeRegex.FindAllStringSubmatch(str, -1) {
			if len(m) > 1 && m[1] != "" {
				vars[m[1]] = true
			}
			if len(m) > 2 && m[2] != "" {
				vars[m[2]] = true
			}
		}
		for _, m := range goRegex.FindAllStringSubmatch(str, -1) {

			if len(m) > 1 {
				vars[m[1]] = true
			}
		}
		for _, m := range pyRegex.FindAllStringSubmatch(str, -1) {
			if len(m) > 1 {
				vars[m[1]] = true
			}
		}
	}

	var result []string
	for v := range vars {
		// Filter out common system ones if needed
		if v != "NODE_ENV" && v != "PATH" {
			result = append(result, v)
		}
	}
	return result
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == false {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}
	return result
}

// scanForPythonImports finds 'import X' or 'from X import Y'
func scanForPythonImports(files []string) []string {
	deps := make(map[string]bool)
	// regex: from X import Y  OR  import X
	// strict import: ^import\s+([a-zA-Z0-9_]+)
	// from import: ^from\s+([a-zA-Z0-9_]+)\s+import

	importRegex := regexp.MustCompile(`(?m)^(?:import\s+([a-zA-Z0-9_]+)|from\s+([a-zA-Z0-9_]+)\s+import)`)

	stdLib := map[string]bool{
		"os": true, "sys": true, "math": true, "json": true, "time": true, "random": true,
		"datetime": true, "re": true, "subprocess": true, "pathlib": true, "typing": true,
		"collections": true, "itertools": true, "functools": true, "logging": true,
		"threading": true, "multiprocessing": true, "socket": true, "email": true,
		"argparse": true, "shutil": true, "glob": true, "pickle": true, "copy": true,
		"hashlib": true, "base64": true, "uuid": true, "csv": true, "io": true, "requests": false,
	}

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		matches := importRegex.FindAllStringSubmatch(string(content), -1)
		for _, m := range matches {
			pkg := ""
			if m[1] != "" {
				pkg = m[1]
			} else if m[2] != "" {
				pkg = m[2]
			}

			if pkg != "" && !stdLib[pkg] {
				deps[pkg] = true
			}
		}
	}

	var result []string
	for d := range deps {
		result = append(result, d)
	}
	return result
}

func resolvePythonVersion(pkg string) string {
	// Try pip show
	cmd := exec.Command("pip", "show", pkg)
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Version: ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
			}
		}
	}
	return "latest"
}
