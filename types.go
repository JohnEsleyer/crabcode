package main

// Declarative Infrastructure-as-Code Spec Structs
type SetupStep struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
	Dir     string `json:"dir" yaml:"dir"`
}

type TemplateFile struct {
	Path    string `json:"path" yaml:"path"`
	Content string `json:"content" yaml:"content"`
	IsDir   bool   `json:"isDir" yaml:"is_dir"`
}

type BuildStep struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
}

type RunSpec struct {
	Command string `json:"command" yaml:"command"`
}

type NotesSpec struct {
	Markdown string `json:"markdown" yaml:"markdown"`
	HTML     string `json:"html" yaml:"html"`
}

type DeclarativeConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Environment string            `json:"environment" yaml:"environment"`
	EnvDir      string            `json:"envDir" yaml:"env_dir"`
	IconColor   string            `json:"iconColor" yaml:"icon_color"`
	Setup       []SetupStep       `json:"setup" yaml:"setup"`
	EnvVars     map[string]string `json:"envVars" yaml:"env_vars"`
	Files       []TemplateFile    `json:"files" yaml:"files"`
	Build       []BuildStep       `json:"build" yaml:"build"`
	Run         RunSpec           `json:"run" yaml:"run"`
	Notes       NotesSpec         `json:"notes" yaml:"notes"`
}

type TemplateSpec struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Environment string            `json:"environment"`
	IconColor   string            `json:"iconColor"`
	Config      DeclarativeConfig `json:"config"`
	RawYAML     string            `json:"rawYaml"`
}

type Note struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Sandbox struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ConfigYAML   string `json:"configYaml"`
	MarkdownNote string `json:"markdownNote"`
	HTMLNote     string `json:"htmlNote"`
	Folder       string `json:"folder"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type SandboxFile struct {
	ID        string `json:"id"`
	SandboxID string `json:"sandboxId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	IsDir     bool   `json:"isDir"`
	UpdatedAt string `json:"updatedAt"`
}

type WorkspaceInfo struct {
	Path      string    `json:"path"`
	Notes     []Note    `json:"notes"`
	Sandboxes []Sandbox `json:"sandboxes"`
}

type GlobalSettings struct {
	CrabRootPath    string `json:"crabRootPath"`
	UniversalEnvDir string `json:"universalEnvDir"`
}

// FileNode represents a file or folder entry in the project directory
type FileNode struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// WorkspaceInitInfo represents initialization status of a workspace folder
type WorkspaceInitInfo struct {
	Path       string `json:"path"`
	HasDotCrab bool   `json:"hasDotCrab"`
	Exists     bool   `json:"exists"`
}
