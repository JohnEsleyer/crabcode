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

// Structs for DB mapping
type Note struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Sandbox struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ConfigYAML string `json:"configYaml"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
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
	_ = a.InitPlayground()
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

// InitializeCrabFolder configures folders and environment directories
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

	playgroundDir := filepath.Join(path, "playground")
	_ = os.MkdirAll(playgroundDir, 0755)

	templates := map[string]string{
		"scratch.py": "# Python Scratchpad\nprint('Hello from the Python Playground!')\n",
		"scratch.js": "// Javascript Scratchpad\nconsole.log('Hello from the JavaScript Playground!');\n",
		"scratch.go": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from the Go Playground!\")\n}\n",
		"scratch.rs": "fn main() {\n    println!(\"Hello from the Rust Playground!\");\n}\n",
	}

	for fileName, content := range templates {
		filePath := filepath.Join(playgroundDir, fileName)
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}

	return nil
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
		"scratch.py": "# Python Scratchpad\nprint('Hello from the Python Playground!')\n",
		"scratch.js": "// Javascript Scratchpad\nconsole.log('Hello from the JavaScript Playground!');\n",
		"scratch.go": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from the Go Playground!\")\n}\n",
		"scratch.rs": "fn main() {\n    println!(\"Hello from the Rust Playground!\");\n}\n",
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
		if file.Name() == ".crab" {
			continue
		}
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

// OpenWorkspace initializes the workspace directory, DB, and migrates tables
func (a *App) OpenWorkspace(path string) (*WorkspaceInfo, error) {
	a.CloseDB()
	a.workspacePath = path

	crabDir := filepath.Join(path, ".crab")
	if err := os.MkdirAll(crabDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(crabDir, "crab.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	a.db = db

	if err := a.migrate(); err != nil {
		return nil, err
	}

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

	rows, err := a.db.Query("SELECT id, name, config_yaml, created_at, updated_at FROM sandboxes ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sandboxes []Sandbox
	for rows.Next() {
		var s Sandbox
		if err := rows.Scan(&s.ID, &s.Name, &s.ConfigYAML, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, s)
	}
	return sandboxes, nil
}

func (a *App) CreateSandbox(name string, configYaml string) (*Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	if configYaml == "" {
		configYaml = fmt.Sprintf("name: \"%s\"\nenvironment: \"python\"\nrun_command: \"python3 main.py\"\n", name)
	}

	_, err := a.db.Exec(
		"INSERT INTO sandboxes (id, name, config_yaml, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, name, configYaml, now, now,
	)
	if err != nil {
		return nil, err
	}

	_ = a.SaveSandboxFile(id, "main.py", "# Virtual Sandbox Entrypoint\nprint('Hello from your portable database sandbox!')\n", false)

	return &Sandbox{
		ID:         id,
		Name:       name,
		ConfigYAML: configYaml,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (a *App) DeleteSandbox(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	if _, err := a.db.Exec("DELETE FROM sandboxes WHERE id = ?", id); err != nil {
		return err
	}
	_, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ?", id)
	return err
}

func (a *App) SaveSandboxConfig(id string, configYaml string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET config_yaml = ?, updated_at = ? WHERE id = ?", configYaml, now, id)
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
	return err
}

func parseRunCommand(yamlStr string) (string, error) {
	lines := strings.Split(yamlStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "run_command:") {
			val := strings.TrimPrefix(line, "run_command:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", errors.New("no valid 'run_command' specified in sandbox configuration rules")
}

// RunSandbox extracts virtual SQLite files to a temporary disk location and executes standard runners
func (a *App) RunSandbox(sandboxID string, activeFilePath string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	var configYaml string
	err := a.db.QueryRow("SELECT config_yaml FROM sandboxes WHERE id = ?", sandboxID).Scan(&configYaml)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve sandbox environment configurations: %w", err)
	}

	files, err := a.GetSandboxFiles(sandboxID)
	if err != nil {
		return "", fmt.Errorf("failed to load sandbox assets from database: %w", err)
	}

	tempDir := filepath.Join(a.workspacePath, ".crab", "temp_sandboxes", sandboxID)
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

	runCmd, err := parseRunCommand(configYaml)
	if err != nil {
		return "", err
	}

	cmdParts := strings.Fields(runCmd)
	if len(cmdParts) == 0 {
		return "", errors.New("run_command parameter evaluates to empty sequence")
	}

	runnerBin := cmdParts[0]
	runnerArgs := cmdParts[1:]

	processID := "sandbox_" + sandboxID
	err = a.RunCommand(processID, runnerBin, runnerArgs, tempDir)
	if err != nil {
		return "", err
	}

	return processID, nil
}

// GLOBAL SETTINGS MANAGEMENT

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

func (a *App) SaveGlobalSettings(settings GlobalSettings) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	bootstrapFile := filepath.Join(home, ".crabcode_root.txt")
	if settings.CrabRootPath != "" {
		settings.CrabRootPath = filepath.Clean(settings.CrabRootPath)
		if err := os.WriteFile(bootstrapFile, []byte(settings.CrabRootPath), 0644); err != nil {
			return fmt.Errorf("failed to save custom system bootstrap file: %w", err)
		}
	} else {
		_ = os.Remove(bootstrapFile)
		settings.CrabRootPath = filepath.Join(home, ".crabcode")
	}

	if err := os.MkdirAll(settings.CrabRootPath, 0755); err != nil {
		return fmt.Errorf("failed to instantiate directory on dynamic root drive: %w", err)
	}

	settings.UniversalEnvDir = filepath.Join(settings.CrabRootPath, "environments")
	_ = os.MkdirAll(settings.UniversalEnvDir, 0755)

	settingsFile := filepath.Join(settings.CrabRootPath, "settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, data, 0644)
}

// RunCommand executes a command and streams stdout/stderr via Wails events
func (a *App) RunCommand(id string, commandName string, args []string, dir string) error {
	a.processMutex.Lock()
	if existingCmd, exists := a.activeProcesses[id]; exists {
		if existingCmd.Process != nil {
			_ = existingCmd.Process.Kill()
		}
		delete(a.activeProcesses, id)
	}

	cmd := exec.Command(commandName, args...)
	cmd.Dir = dir

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

// StopCommand kills a running process by ID
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
