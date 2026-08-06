package main

// ButtonMappings maps CrabCode UI actions to terminal/shell commands
type ButtonMappings struct {
	Run   string `json:"run" yaml:"run"`
	Build string `json:"build" yaml:"build"`
	Test  string `json:"test" yaml:"test"`
}

type SetupStep struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
}

type DeclarativeConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Environment string            `json:"environment" yaml:"environment"`
	IconColor   string            `json:"iconColor" yaml:"icon_color"`
	Setup       []SetupStep       `json:"setup" yaml:"setup"`
	EnvVars     map[string]string `json:"envVars" yaml:"env_vars"`
	Mappings    ButtonMappings    `json:"mappings" yaml:"mappings"`
	Notes       NotesSpec         `json:"notes" yaml:"notes"`
}

type NotesSpec struct {
	Markdown string `json:"markdown" yaml:"markdown"`
	HTML     string `json:"html" yaml:"html"`
}

type TemplateFile struct {
	Path    string `json:"path" yaml:"path"`
	Content string `json:"content" yaml:"content"`
	IsDir   bool   `json:"isDir" yaml:"is_dir"`
}

type TemplateSpec struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Environment string            `json:"environment"`
	IconColor   string            `json:"iconColor"`
	Config      DeclarativeConfig `json:"config"`
	Files       []TemplateFile    `json:"files"`
	RawYAML     string            `json:"rawYaml"`
}

type Workspace struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ConfigYAML      string `json:"configYaml"`
	RuntimePath     string `json:"runtimePath"`
	ActiveSandboxID string `json:"activeSandboxId"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type Sandbox struct {
	ID           string `json:"id"`
	WorkspaceID  string `json:"workspaceId"`
	Name         string `json:"name"`
	Folder       string `json:"folder"`
	MarkdownNote string `json:"markdownNote"`
	HTMLNote     string `json:"htmlNote"`
	IsActive     bool   `json:"isActive"`
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

type GlobalSettings struct {
	CrabRootPath    string `json:"crabRootPath"`
	UniversalEnvDir string `json:"universalEnvDir"`
}

type SandboxExportData struct {
	Sandbox Sandbox       `json:"sandbox"`
	Files   []SandboxFile `json:"files"`
}

type WorkspaceBackupData struct {
	Version   string              `json:"version"`
	Workspace Workspace           `json:"workspace"`
	Sandboxes []SandboxExportData `json:"sandboxes"`
}
