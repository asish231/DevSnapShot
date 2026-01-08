package metadata

// SnapshotMetadata represents the "brain" of the snapshot.
// It describes the environment, commands, and identity of the snapshot.
type SnapshotMetadata struct {
	// SchemaVersion allows for future non-breaking changes
	SchemaVersion string `json:"schema_version"`

	// Core Identity
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at"` // ISO 8601

	// Environment Requirements
	Environments []EnvironmentConfig `json:"environments"`

	// Execution Steps
	Commands LifecycleCommands `json:"commands"`

	// Secrets / Config
	RequiredVars []string `json:"required_vars,omitempty"`

	// Files to include (optional explicit list, or pattern)
	// If empty, defaults to all files in archive
	Manifest []string `json:"manifest,omitempty"`
}

type EnvironmentConfig struct {
	// Base platform hint, e.g., "node", "python", "go", "docker"
	Type string `json:"type"`

	// Version constraints, e.g. ">=18.0.0"
	Version string `json:"version,omitempty"`

	// Optional: Image to use if Type == "docker"
	Image string `json:"image,omitempty"`

	// Per-environment commands
	Setup []string `json:"setup,omitempty"`
	Run   string   `json:"run,omitempty"`
}

type LifecycleCommands struct {
	// Setup commands (e.g., "npm install")
	Setup []string `json:"setup,omitempty"`

	// Command to start the app/shell (e.g., "npm start")
	Run string `json:"run,omitempty"`

	// Automated verification command (e.g., "npm test")
	Test string `json:"test,omitempty"`
}
