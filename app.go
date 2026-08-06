package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type App struct {
	ctx             context.Context
	activeProcesses map[string]*exec.Cmd
	activeStdins    map[string]io.WriteCloser
	processMutex    sync.Mutex
	db              *sql.DB
	cliPath         string
}

func NewApp() *App {
	return &App{
		activeProcesses: make(map[string]*exec.Cmd),
		activeStdins:    make(map[string]io.WriteCloser),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.ensureGlobalDBInitialized()
	_ = a.InitTemplates()
}

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

func (a *App) CloseDB() {
	if a.db != nil {
		_ = a.db.Close()
		a.db = nil
	}
}

// ensureColumn checks whether a column exists on a table and adds it via
// ALTER TABLE when missing, so databases created by older schema versions
// can be upgraded in place.
func (a *App) ensureColumn(table string, column string, ddl string) error {
	rows, err := a.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}

	_, err = a.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, ddl))
	return err
}

func (a *App) migrate() error {
	if a.db == nil {
		return errors.New("no active database connection")
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			config_yaml TEXT NOT NULL DEFAULT '',
			active_sandbox_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sandboxes (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			folder TEXT NOT NULL DEFAULT '',
			markdown_note TEXT NOT NULL DEFAULT '',
			html_note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
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

	// Upgrade databases created by older schema versions.
	for _, col := range []struct {
		table, column, ddl string
	}{
		{"workspaces", "config_yaml", "TEXT NOT NULL DEFAULT ''"},
		{"workspaces", "active_sandbox_id", "TEXT NOT NULL DEFAULT ''"},
		{"sandboxes", "folder", "TEXT NOT NULL DEFAULT ''"},
		{"sandboxes", "markdown_note", "TEXT NOT NULL DEFAULT ''"},
		{"sandboxes", "html_note", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := a.ensureColumn(col.table, col.column, col.ddl); err != nil {
			return err
		}
	}

	// Default Workspace Bootstrap
	var count int
	_ = a.db.QueryRow("SELECT COUNT(*) FROM workspaces WHERE id = 'default'").Scan(&count)
	if count == 0 {
		defaultConfig := `name: "Default Lab Workspace"
version: "1.0"
environment: "python"
setup: []
mappings:
  run: "python3 main.py"
  test: "pytest"
env_vars:
  PYTHONUNBUFFERED: "1"
`
		now := time.Now().Format(time.RFC3339)
		_, _ = a.db.Exec(
			"INSERT INTO workspaces (id, name, description, config_yaml, active_sandbox_id, created_at, updated_at) VALUES ('default', 'Default Lab Workspace', 'Primary experimentation workspace', ?, '', ?, ?)",
			defaultConfig, now, now,
		)

		// Create default initial sandbox
		sb, err := a.CreateSandboxInFolder("default", "Main Sandbox", "")
		if err == nil && sb != nil {
			_ = a.ActivateSandbox("default", sb.ID)
		}
	}

	return nil
}
