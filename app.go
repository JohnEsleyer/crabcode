package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// App struct
type App struct {
	ctx             context.Context
	activeProcesses map[string]*exec.Cmd
	activeStdins    map[string]io.WriteCloser
	processMutex    sync.Mutex
	workspacePath   string
	db              *sql.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		activeProcesses: make(map[string]*exec.Cmd),
		activeStdins:    make(map[string]io.WriteCloser),
	}
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.ensureGlobalDBInitialized()
	_ = a.InitPlayground()
	_ = a.InitTemplates()
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

// CloseDB closes active DB session if loaded
func (a *App) CloseDB() {
	if a.db != nil {
		_ = a.db.Close()
		a.db = nil
	}
}

// OpenWorkspace uses the global DB
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
	_, _ = a.db.Exec("ALTER TABLE sandboxes ADD COLUMN folder TEXT NOT NULL DEFAULT ''")

	return nil
}
