package create

import "strings"

// summarizeDeps returns a short string representation of the dependencies
func summarizeDeps(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	limit := 3
	if len(deps) < limit {
		return strings.Join(deps, ", ")
	}
	return strings.Join(deps[:limit], ", ") + ", ..."
}
