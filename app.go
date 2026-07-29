package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	activeProcesses map[string]*exec.Cmd
	processMutex    sync.Mutex
	workspacePath   string
	db              *sql.DB
}

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

// Structs for DB mapping
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

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		activeProcesses: make(map[string]*exec.Cmd),
	}
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.ensureGlobalDBInitialized()
	_ = a.InitPlayground()
	_ = a.InitTemplates()
}

// GetCrabRootDirectory resolves the system bootstrap target location
func (a *App) GetCrabRootDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".crabcode"
	}

	bootstrapFile := filepath.Join(home, ".crabcode_root.txt")
	data, err := os.ReadFile(bootstrapFile)
	if err == nil {
		customPath := strings.TrimSpace(string(data))
		if customPath != "" {
			return customPath
		}
	}

	return filepath.Join(home, ".crabcode")
}

// ensureGlobalDBInitialized ensures a single global SQLite database at <CrabRoot>/crabcode.db
func (a *App) ensureGlobalDBInitialized() error {
	root := a.GetCrabRootDirectory()
	_ = os.MkdirAll(root, 0755)

	dbPath := filepath.Join(root, "crabcode.db")
	if a.db != nil {
		_ = a.db.Close()
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	a.db = db

	return a.migrate()
}

// IsDirectoryEmpty evaluates if a given path is unpopulated or missing
func (a *App) IsDirectoryEmpty(path string) (bool, error) {
	if path == "" {
		return true, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// InitializeCrabFolder configures folders, environment directories, and dynamic templates
func (a *App) InitializeCrabFolder(path string) error {
	if path == "" {
		return errors.New("specified path is empty")
	}

	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	settings := GlobalSettings{
		CrabRootPath:    path,
		UniversalEnvDir: filepath.Join(path, "environments"),
	}

	settingsFile := filepath.Join(path, "settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsFile, data, 0644); err != nil {
		return err
	}

	_ = os.MkdirAll(settings.UniversalEnvDir, 0755)
	_ = os.MkdirAll(filepath.Join(path, "playground"), 0755)
	_ = os.MkdirAll(filepath.Join(path, "sandboxes"), 0755)

	_ = a.InitTemplates()
	_ = a.ensureGlobalDBInitialized()

	return nil
}

// DYNAMIC TEMPLATES ENGINE (<CrabRoot>/templates/*.yaml)

func (a *App) GetTemplatesDirectory() string {
	return filepath.Join(a.GetCrabRootDirectory(), "templates")
}

func (a *App) InitTemplates() error {
	dir := a.GetTemplatesDirectory()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	defaultTemplates := map[string]string{
		"python.yaml": `name: "Python Environment"
version: "1.0"
environment: "python"
icon_color: "#3572A5"
setup: []
env_vars:
  PYTHONUNBUFFERED: "1"
files:
  - path: "main.py"
    content: "# Python Sandbox\nprint('Hello from CrabCode Python Sandbox!')\n"
build: []
run:
  command: "python3 main.py"
notes:
  markdown: "# Python Environment\n\nUse this dynamic environment for algorithmic experiments or script execution."
  html: "<h3>🐍 Python Sandbox Active</h3>"
`,
		"go.yaml": `name: "Go Environment"
version: "1.0"
environment: "go"
icon_color: "#00ADD8"
setup: []
files:
  - path: "main.go"
    content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from CrabCode Go Sandbox!\")\n}\n"
run:
  command: "go run main.go"
notes:
  markdown: "# Go Sandbox Environment\n\nWrite fast, concurrent system components inside this SQLite environment."
  html: "<h3 style='color:#00ADD8;'>Go Compiler Environment Active</h3>"
`,
		"rust.yaml": `name: "Rust Environment"
version: "1.0"
environment: "rust"
icon_color: "#DEA584"
setup: []
files:
  - path: "main.rs"
    content: "fn main() {\n    println!(\"Hello from CrabCode Rust Sandbox!\");\n}\n"
run:
  command: "rustc main.rs && ./main"
notes:
  markdown: "# Rust Sandbox Environment\n\nValidate memory management rules, traits, and complex data models safely."
  html: "<h3 style='color:#DEA584;'>Rust Core Active</h3>"
`,
		"java.yaml": `name: "Java Environment"
version: "1.0"
environment: "java"
icon_color: "#B07219"
setup: []
files:
  - path: "Main.java"
    content: "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello from CrabCode Java Sandbox!\");\n    }\n}\n"
run:
  command: "java Main.java"
notes:
  markdown: "# Java Sandbox Environment"
  html: "<h3 style='color:#B07219;'>☕ Java Virtual Machine Active</h3>"
`,
		"sql.yaml": `name: "SQL Database Environment"
version: "1.0"
environment: "sql"
icon_color: "#00ADD8"
setup: []
files:
  - path: "main.sql"
    content: "-- SQL Sandbox\nCREATE TABLE IF NOT EXISTS demo (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT);\nINSERT INTO demo (message) VALUES ('Hello from CrabCode SQL Sandbox!');\nSELECT * FROM demo;\n"
run:
  command: "sqlite3 :memory: < main.sql"
notes:
  markdown: "# SQL Sandbox Environment\n\nRun in-memory SQLite relational queries and test database schema designs."
  html: "<h3>💾 SQL Database Engine Active</h3>"
`,
		"surrealdb.yaml": `name: "SurrealDB Environment"
version: "1.0"
environment: "surrealdb"
icon_color: "#FF00A0"
setup: []
files:
  - path: "main.surql"
    content: "-- SurrealDB Sandbox\nCREATE user SET name = 'CrabCode Developer', role = 'Admin';\nSELECT * FROM user;\n"
run:
  command: "surreal sql --endpoint memory --db test --ns test < main.surql"
notes:
  markdown: "# SurrealDB Sandbox Environment\n\nExecute SurrealQL queries for multi-model graph and document database design."
  html: "<h3 style='color:#FF00A0;'>⚡ SurrealDB Multi-Model Active</h3>"
`,
		"javascript.yaml": `name: "JavaScript Environment"
version: "1.0"
environment: "node"
icon_color: "#F7DF1E"
setup: []
files:
  - path: "index.js"
    content: "// JavaScript Sandbox\nconsole.log('Hello from CrabCode JavaScript Sandbox!');\n"
run:
  command: "node index.js"
notes:
  markdown: "# JavaScript Sandbox Environment"
  html: "<h3>Node.js Environment Ready</h3>"
`,
		"typescript.yaml": `name: "TypeScript Environment"
version: "1.0"
environment: "node"
icon_color: "#3178C6"
setup: []
files:
  - path: "index.ts"
    content: "// TypeScript Sandbox\nconst greeting: string = 'Hello from CrabCode TypeScript Sandbox!';\nconsole.log(greeting);\n"
run:
  command: "npx tsx index.ts"
notes:
  markdown: "# TypeScript Sandbox Environment"
  html: "<h3>TypeScript Environment Ready</h3>"
`,
	}

	for fileName, content := range defaultTemplates {
		filePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			_ = os.WriteFile(filePath, []byte(content), 0644)
		}
	}

	return nil
}

// GetTemplates scans <CrabRoot>/templates/ for all YAML templates dynamically
func (a *App) GetTemplates() ([]TemplateSpec, error) {
	_ = a.InitTemplates()
	dir := a.GetTemplatesDirectory()

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var specs []TemplateSpec
	for _, f := range files {
		if f.IsDir() || (!strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml")) {
			continue
		}

		filePath := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var cfg DeclarativeConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}

		id := strings.TrimSuffix(strings.TrimSuffix(f.Name(), ".yaml"), ".yml")
		color := cfg.IconColor
		if color == "" {
			color = "#ff5a36"
		}

		specs = append(specs, TemplateSpec{
			ID:          id,
			Name:        cfg.Name,
			Environment: cfg.Environment,
			IconColor:   color,
			Config:      cfg,
			RawYAML:     string(data),
		})
	}

	return specs, nil
}

// GetPlaygroundDirectory returns the global playground root directory inside CrabRoot
func (a *App) GetPlaygroundDirectory() (string, error) {
	root := a.GetCrabRootDirectory()
	return filepath.Join(root, "playground"), nil
}

// InitPlayground ensures the global playground environment files are prepared
func (a *App) InitPlayground() error {
	dir, err := a.GetPlaygroundDirectory()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	templates := map[string]string{
		"scratch.py":    "# Python Scratchpad\nprint('Hello from the Python Playground!')\n",
		"scratch.js":    "// Javascript Scratchpad\nconsole.log('Hello from the JavaScript Playground!');\n",
		"scratch.go":    "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from the Go Playground!\")\n}\n",
		"scratch.rs":    "fn main() {\n    println!(\"Hello from the Rust Playground!\");\n}\n",
		"scratch.java":  "// Java Scratchpad\npublic class Scratch {\n    public static void main(String[] args) {\n        System.out.println(\"Hello from the Java Playground!\");\n    }\n}\n",
		"scratch.ts":    "// TypeScript Scratchpad\nconst greeting: string = \"Hello from the TypeScript Playground!\";\nconsole.log(greeting);\n",
		"scratch.sql":   "-- SQL Scratchpad\nCREATE TABLE IF NOT EXISTS scratchpad (id INTEGER PRIMARY KEY, message TEXT);\nINSERT INTO scratchpad (message) VALUES ('Hello from the SQL Playground!');\nSELECT * FROM scratchpad;\n",
		"scratch.dart":  "// Dart Scratchpad\nvoid main() {\n  print('Hello from the Dart Playground!');\n}\n",
		"scratch.surql": "-- SurrealDB Scratchpad\nCREATE user SET name = 'CrabCode', role = 'Developer';\nSELECT * FROM user;\n",
	}

	for fileName, content := range templates {
		filePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			_ = os.WriteFile(filePath, []byte(content), 0644)
		}
	}
	return nil
}

// FileNode represents a file or folder entry in the project directory
type FileNode struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// SelectFolder opens a native OS folder dialog to choose a project directory
func (a *App) SelectFolder() (string, error) {
	options := runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	}
	return runtime.OpenDirectoryDialog(a.ctx, options)
}

// ListDirectory lists files and folders inside the provided path without recursing
func (a *App) ListDirectory(path string) ([]FileNode, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	nodes := make([]FileNode, 0)
	for _, file := range files {
		nodes = append(nodes, FileNode{
			Name:  file.Name(),
			Path:  filepath.Join(path, file.Name()),
			IsDir: file.IsDir(),
		})
	}
	return nodes, nil
}

// ReadFile reads the contents of the given file path
func (a *App) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveFile writes contents to the specified file path
func (a *App) SaveFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateFile creates an empty file at the given path
func (a *App) CreateFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// CreateDirectory creates a directory (and any parents) at the given path
func (a *App) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// DeletePath deletes the file or folder at the specified path recursively
func (a *App) DeletePath(path string) error {
	return os.RemoveAll(path)
}

// RenamePath renames or moves a file or folder from oldPath to newPath
func (a *App) RenamePath(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// CloseDB closes active DB session if loaded
func (a *App) CloseDB() {
	if a.db != nil {
		_ = a.db.Close()
		a.db = nil
	}
}

// OpenWorkspace uses the global DB (no per-workspace .crab/)
func (a *App) OpenWorkspace(path string) (*WorkspaceInfo, error) {
	a.workspacePath = path
	_ = a.ensureGlobalDBInitialized()

	notes, err := a.GetNotes()
	if err != nil {
		notes = []Note{}
	}
	sandboxes, err := a.GetSandboxes()
	if err != nil {
		sandboxes = []Sandbox{}
	}

	return &WorkspaceInfo{
		Path:      path,
		Notes:     notes,
		Sandboxes: sandboxes,
	}, nil
}

func (a *App) migrate() error {
	if a.db == nil {
		return errors.New("no active database connection")
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS markdown_notes (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sandboxes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			config_yaml TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sandbox_files (
			id TEXT PRIMARY KEY,
			sandbox_id TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			is_dir INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(sandbox_id) REFERENCES sandboxes(id) ON DELETE CASCADE,
			UNIQUE(sandbox_id, path)
		);`,
	}

	for _, query := range queries {
		if _, err := a.db.Exec(query); err != nil {
			return err
		}
	}

	_, _ = a.db.Exec("ALTER TABLE sandboxes ADD COLUMN markdown_note TEXT NOT NULL DEFAULT ''")
	_, _ = a.db.Exec("ALTER TABLE sandboxes ADD COLUMN html_note TEXT NOT NULL DEFAULT ''")

	return nil
}

// NOTE CRUD OPERATIONS

func (a *App) GetNotes() ([]Note, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, title, content, created_at, updated_at FROM markdown_notes ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (a *App) CreateNote(title string) (*Note, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf("# %s\n\nWrite your markdown observations here.", title)

	_, err := a.db.Exec(
		"INSERT INTO markdown_notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, title, content, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a *App) SaveNote(id string, title string, content string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec(
		"UPDATE markdown_notes SET title = ?, content = ?, updated_at = ? WHERE id = ?",
		title, content, now, id,
	)
	return err
}

func (a *App) DeleteNote(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	_, err := a.db.Exec("DELETE FROM markdown_notes WHERE id = ?", id)
	return err
}

// SANDBOX OPERATIONS

func (a *App) GetSandboxes() ([]Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, name, config_yaml, markdown_note, html_note, created_at, updated_at FROM sandboxes ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sandboxes []Sandbox
	for rows.Next() {
		var s Sandbox
		if err := rows.Scan(&s.ID, &s.Name, &s.ConfigYAML, &s.MarkdownNote, &s.HTMLNote, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, s)
	}
	return sandboxes, nil
}

// CreateSandbox creates a sandbox from a dynamic template ID
func (a *App) CreateSandbox(name string, templateID string) (*Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	templates, err := a.GetTemplates()
	if err != nil || len(templates) == 0 {
		return nil, fmt.Errorf("failed to load template specifications: %w", err)
	}

	var selectedTemplate TemplateSpec
	found := false
	for _, t := range templates {
		if t.ID == templateID {
			selectedTemplate = t
			found = true
			break
		}
	}
	if !found {
		selectedTemplate = templates[0]
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	cfg := selectedTemplate.Config
	cfg.Name = name
	configBytes, _ := yaml.Marshal(cfg)
	configYaml := string(configBytes)

	markdownNote := cfg.Notes.Markdown
	htmlNote := cfg.Notes.HTML

	_, err = a.db.Exec(
		"INSERT INTO sandboxes (id, name, config_yaml, markdown_note, html_note, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, name, configYaml, markdownNote, htmlNote, now, now,
	)
	if err != nil {
		return nil, err
	}

	// Write sandbox config directory in <CrabRoot>/sandboxes/<id>/
	sandboxDir := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", id)
	_ = os.MkdirAll(sandboxDir, 0755)
	_ = os.WriteFile(filepath.Join(sandboxDir, "config.yaml"), []byte(configYaml), 0644)

	for _, f := range cfg.Files {
		_ = a.SaveSandboxFile(id, f.Path, f.Content, f.IsDir)
	}

	return &Sandbox{
		ID:           id,
		Name:         name,
		ConfigYAML:   configYaml,
		MarkdownNote: markdownNote,
		HTMLNote:     htmlNote,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (a *App) RenameSandbox(id string, name string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET name = ?, updated_at = ? WHERE id = ?", name, now, id)
	return err
}

func (a *App) DeleteSandbox(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	if _, err := a.db.Exec("DELETE FROM sandboxes WHERE id = ?", id); err != nil {
		return err
	}
	_, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ?", id)

	sandboxDir := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", id)
	_ = os.RemoveAll(sandboxDir)

	return err
}

func (a *App) SaveSandboxConfig(id string, configYaml string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET config_yaml = ?, updated_at = ? WHERE id = ?", configYaml, now, id)

	sandboxDir := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", id)
	_ = os.MkdirAll(sandboxDir, 0755)
	_ = os.WriteFile(filepath.Join(sandboxDir, "config.yaml"), []byte(configYaml), 0644)

	return err
}

func (a *App) SaveSandboxNotes(id string, markdownNote string, htmlNote string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET markdown_note = ?, html_note = ?, updated_at = ? WHERE id = ?", markdownNote, htmlNote, now, id)
	return err
}

// VIRTUAL SANDBOX FILE OPERATIONS

func (a *App) GetSandboxFiles(sandboxID string) ([]SandboxFile, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, sandbox_id, path, content, is_dir, updated_at FROM sandbox_files WHERE sandbox_id = ?", sandboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []SandboxFile
	for rows.Next() {
		var f SandboxFile
		var isDirInt int
		if err := rows.Scan(&f.ID, &f.SandboxID, &f.Path, &f.Content, &isDirInt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.IsDir = isDirInt == 1
		files = append(files, f)
	}
	return files, nil
}

func (a *App) SaveSandboxFile(sandboxID string, path string, content string, isDir bool) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	isDirInt := 0
	if isDir {
		isDirInt = 1
	}

	result, err := a.db.Exec(
		"UPDATE sandbox_files SET content = ?, is_dir = ?, updated_at = ? WHERE sandbox_id = ? AND path = ?",
		content, isDirInt, now, sandboxID, path,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		_, err = a.db.Exec(
			"INSERT INTO sandbox_files (id, sandbox_id, path, content, is_dir, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			id, sandboxID, path, content, isDirInt, now,
		)
		if err != nil {
			return err
		}
	}

	// Persist sandbox file in <CrabRoot>/sandboxes/<id>/<path>
	fileOnDisk := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", sandboxID, path)
	if isDir {
		_ = os.MkdirAll(fileOnDisk, 0755)
	} else {
		_ = os.MkdirAll(filepath.Dir(fileOnDisk), 0755)
		_ = os.WriteFile(fileOnDisk, []byte(content), 0644)
	}

	return nil
}

func (a *App) DeleteSandboxFile(sandboxID string, path string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	if _, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path = ?", sandboxID, path); err != nil {
		return err
	}

	prefix := path + "/"
	_, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path LIKE ?", sandboxID, prefix+"%")

	fileOnDisk := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", sandboxID, path)
	_ = os.RemoveAll(fileOnDisk)

	return err
}

// DECLARATIVE EXECUTION ENGINE

func (a *App) RunSandbox(sandboxID string, activeFilePath string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	var configYaml string
	err := a.db.QueryRow("SELECT config_yaml FROM sandboxes WHERE id = ?", sandboxID).Scan(&configYaml)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve sandbox environment configurations: %w", err)
	}

	var cfg DeclarativeConfig
	if err := yaml.Unmarshal([]byte(configYaml), &cfg); err != nil {
		return "", fmt.Errorf("failed to parse declarative sandbox YAML configuration: %w", err)
	}

	crabRoot := a.GetCrabRootDirectory()

	// 1. EXECUTE DECLARATIVE SETUP STEPS inside <CrabRoot>/environments/
	for _, step := range cfg.Setup {
		if step.Command == "" {
			continue
		}
		targetDir := crabRoot
		if step.Dir != "" {
			targetDir = filepath.Join(crabRoot, step.Dir)
		}
		_ = os.MkdirAll(targetDir, 0755)
		_ = a.executeSyncCommand(step.Command, targetDir, cfg.EnvVars)
	}

	// 2. EXTRACT VIRTUAL FILES TO TEMP COMPILATION DIRECTORY INSIDE SYSTEM BASE LOCATION
	files, err := a.GetSandboxFiles(sandboxID)
	if err != nil {
		return "", fmt.Errorf("failed to load sandbox assets from database: %w", err)
	}

	tempDir := filepath.Join(crabRoot, "temp_sandboxes", sandboxID)
	_ = os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to generate sandbox compilation directory: %w", err)
	}

	for _, f := range files {
		targetPath := filepath.Join(tempDir, f.Path)
		if f.IsDir {
			_ = os.MkdirAll(targetPath, 0755)
		} else {
			dir := filepath.Dir(targetPath)
			_ = os.MkdirAll(dir, 0755)
			_ = os.WriteFile(targetPath, []byte(f.Content), 0644)
		}
	}

	// 3. EXECUTE DECLARATIVE BUILD STEPS
	for _, bStep := range cfg.Build {
		if bStep.Command == "" {
			continue
		}
		_ = a.executeSyncCommand(bStep.Command, tempDir, cfg.EnvVars)
	}

	// 4. EXECUTE RUN COMMAND
	runCmd := cfg.Run.Command
	if runCmd == "" {
		return "", errors.New("no valid 'run.command' specified in declarative sandbox rules")
	}

	processID := "sandbox_" + sandboxID
	shellOps := strings.Contains(runCmd, "&&") || strings.Contains(runCmd, ";") || strings.Contains(runCmd, "|") || strings.Contains(runCmd, "<") || strings.Contains(runCmd, ">")

	if shellOps {
		shell := "sh"
		shellArg := "-c"
		if os.PathSeparator == '\\' {
			shell = "cmd"
			shellArg = "/c"
		}
		err = a.RunCommandWithEnv(processID, shell, []string{shellArg, runCmd}, tempDir, cfg.EnvVars)
	} else {
		cmdParts := strings.Fields(runCmd)
		if len(cmdParts) == 0 {
			return "", errors.New("run.command evaluates to empty sequence")
		}
		runnerBin := cmdParts[0]
		runnerArgs := cmdParts[1:]
		err = a.RunCommandWithEnv(processID, runnerBin, runnerArgs, tempDir, cfg.EnvVars)
	}

	if err != nil {
		return "", err
	}

	return processID, nil
}

func (a *App) executeSyncCommand(cmdStr string, dir string, envVars map[string]string) error {
	shell := "sh"
	shellArg := "-c"
	if os.PathSeparator == '\\' {
		shell = "cmd"
		shellArg = "/c"
	}

	cmd := exec.Command(shell, shellArg, cmdStr)
	cmd.Dir = dir

	env := os.Environ()
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
			"id":   "setup",
			"line": string(output),
		})
	}
	return err
}

// GLOBAL SETTINGS & SYSTEM BASE LOCATION MIGRATION

func (a *App) GetGlobalSettings() (GlobalSettings, error) {
	root := a.GetCrabRootDirectory()
	_ = os.MkdirAll(root, 0755)

	settingsFile := filepath.Join(root, "settings.json")
	var settings GlobalSettings
	settings.CrabRootPath = root
	settings.UniversalEnvDir = filepath.Join(root, "environments")

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		_ = a.SaveGlobalSettings(settings)
		return settings, nil
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, nil
	}

	settings.CrabRootPath = root
	settings.UniversalEnvDir = filepath.Join(root, "environments")
	return settings, nil
}

// SaveGlobalSettings migrates system base directory if changed
func (a *App) SaveGlobalSettings(settings GlobalSettings) error {
	oldRoot := a.GetCrabRootDirectory()
	newRoot := filepath.Clean(settings.CrabRootPath)

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// If path changed, migrate existing data from old to new location
	if oldRoot != newRoot && oldRoot != "" && newRoot != "" {
		if _, err := os.Stat(oldRoot); err == nil {
			_ = os.MkdirAll(newRoot, 0755)
			_ = copyDir(oldRoot, newRoot)
			_ = os.RemoveAll(oldRoot)
		}
	}

	bootstrapFile := filepath.Join(home, ".crabcode_root.txt")
	if newRoot != "" {
		if err := os.WriteFile(bootstrapFile, []byte(newRoot), 0644); err != nil {
			return fmt.Errorf("failed to save custom system bootstrap file: %w", err)
		}
	}

	_ = os.MkdirAll(newRoot, 0755)
	settings.CrabRootPath = newRoot
	settings.UniversalEnvDir = filepath.Join(newRoot, "environments")
	_ = os.MkdirAll(settings.UniversalEnvDir, 0755)

	settingsFile := filepath.Join(newRoot, "settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	_ = os.WriteFile(settingsFile, data, 0644)

	// Re-initialize DB connection at new base path
	return a.ensureGlobalDBInitialized()
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func (a *App) RunCommand(id string, commandName string, args []string, dir string) error {
	return a.RunCommandWithEnv(id, commandName, args, dir, nil)
}

func (a *App) RunCommandWithEnv(id string, commandName string, args []string, dir string, envVars map[string]string) error {
	a.processMutex.Lock()
	if existingCmd, exists := a.activeProcesses[id]; exists {
		if existingCmd.Process != nil {
			_ = existingCmd.Process.Kill()
		}
		delete(a.activeProcesses, id)
	}

	cmd := exec.Command(commandName, args...)
	cmd.Dir = dir

	if len(envVars) > 0 {
		env := os.Environ()
		for k, v := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		a.processMutex.Unlock()
		return err
	}
	cmd.Stderr = cmd.Stdout

	a.activeProcesses[id] = cmd
	a.processMutex.Unlock()

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			text := scanner.Text()
			runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
				"id":   id,
				"line": text,
			})
		}

		err := cmd.Wait()
		a.processMutex.Lock()
		delete(a.activeProcesses, id)
		a.processMutex.Unlock()

		status := "0"
		if err != nil {
			status = err.Error()
		}
		runtime.EventsEmit(a.ctx, "terminal_status", map[string]interface{}{
			"id":     id,
			"status": status,
		})
	}()

	return nil
}

func (a *App) StopCommand(id string) error {
	a.processMutex.Lock()
	defer a.processMutex.Unlock()

	if cmd, exists := a.activeProcesses[id]; exists {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			delete(a.activeProcesses, id)
			return err
		}
	}
	return nil
}
