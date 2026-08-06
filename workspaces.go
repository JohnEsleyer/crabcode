package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) GetWorkspaces() ([]Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, name, description, config_yaml, active_sandbox_id, created_at, updated_at FROM workspaces ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workspaces := make([]Workspace, 0)
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.ConfigYAML, &w.ActiveSandboxID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		w.RuntimePath = a.GetWorkspaceRuntimePath(w.ID)
		workspaces = append(workspaces, w)
	}
	return workspaces, nil
}

func (a *App) GetWorkspaceRuntimePath(workspaceID string) string {
	return filepath.Join(a.GetCrabRootDirectory(), "workspaces", workspaceID, "env")
}

func (a *App) CreateWorkspace(name string, description string, templateID string, customYAML string) (*Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	id := fmt.Sprintf("ws_%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	var configYAML string
	var selectedTemplate *TemplateSpec

	if customYAML != "" {
		configYAML = customYAML
	} else if templateID != "" {
		templates, _ := a.GetTemplates()
		for _, t := range templates {
			if t.ID == templateID {
				selectedTemplate = &t
				configYAML = t.RawYAML
				break
			}
		}
	}

	if configYAML == "" {
		configYAML = `name: "` + name + `"` + "\nversion: \"1.0\"\nenvironment: \"python\"\nmappings:\n  run: \"python3 main.py\"\n"
	}

	_, err := a.db.Exec(
		"INSERT INTO workspaces (id, name, description, config_yaml, active_sandbox_id, created_at, updated_at) VALUES (?, ?, ?, ?, '', ?, ?)",
		id, name, description, configYAML, now, now,
	)
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		ID:          id,
		Name:        name,
		Description: description,
		ConfigYAML:  configYAML,
		RuntimePath: a.GetWorkspaceRuntimePath(id),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create Default Entry Sandbox for this Workspace
	sb, err := a.createDefaultSandboxForWorkspace(id, name, selectedTemplate)
	if err == nil && sb != nil {
		_ = a.ActivateSandbox(id, sb.ID)
		ws.ActiveSandboxID = sb.ID
	}

	return ws, nil
}

func (a *App) createDefaultSandboxForWorkspace(workspaceID string, wsName string, tmpl *TemplateSpec) (*Sandbox, error) {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	markdownNote := "# " + wsName + " Sandbox\n\nShared environment sandbox experiment."
	htmlNote := "<h3>🧪 " + wsName + " Primary Sandbox Active</h3>"

	if tmpl != nil {
		if tmpl.Config.Notes.Markdown != "" {
			markdownNote = tmpl.Config.Notes.Markdown
		}
		if tmpl.Config.Notes.HTML != "" {
			htmlNote = tmpl.Config.Notes.HTML
		}
	}

	_, err := a.db.Exec(
		"INSERT INTO sandboxes (id, workspace_id, name, folder, markdown_note, html_note, created_at, updated_at) VALUES (?, ?, 'Main Sandbox', '', ?, ?, ?, ?)",
		id, workspaceID, markdownNote, htmlNote, now, now,
	)
	if err != nil {
		return nil, err
	}

	if tmpl != nil && len(tmpl.Files) > 0 {
		for _, f := range tmpl.Files {
			_ = a.SaveSandboxFile(id, f.Path, f.Content, f.IsDir)
		}
	} else {
		var cfg DeclarativeConfig
		_ = yaml.Unmarshal([]byte(a.GetWorkspaceConfigString(workspaceID)), &cfg)
		mainName, mainContent := getStarterFileForConfig(&cfg)
		_ = a.SaveSandboxFile(id, mainName, mainContent, false)
	}

	return &Sandbox{
		ID:           id,
		WorkspaceID:  workspaceID,
		Name:         "Main Sandbox",
		Folder:       "",
		MarkdownNote: markdownNote,
		HTMLNote:     htmlNote,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (a *App) GetWorkspaceConfigString(workspaceID string) string {
	var configYaml string
	_ = a.db.QueryRow("SELECT config_yaml FROM workspaces WHERE id = ?", workspaceID).Scan(&configYaml)
	return configYaml
}

func (a *App) SaveWorkspaceConfig(workspaceID string, configYAML string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE workspaces SET config_yaml = ?, updated_at = ? WHERE id = ?", configYAML, now, workspaceID)
	return err
}

func (a *App) DeleteWorkspace(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	if id == "default" {
		return errors.New("cannot delete default workspace")
	}

	sandboxes, _ := a.GetSandboxes(id)
	for _, s := range sandboxes {
		_ = a.DeleteSandbox(s.ID)
	}

	_ = os.RemoveAll(a.GetWorkspaceRuntimePath(id))
	_, err := a.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	return err
}

func (a *App) ActivateSandbox(workspaceID string, sandboxID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	var currentActiveID string
	_ = a.db.QueryRow("SELECT active_sandbox_id FROM workspaces WHERE id = ?", workspaceID).Scan(&currentActiveID)

	runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)
	_ = os.MkdirAll(runtimeDir, 0755)

	// 1. Flush & Save current active sandbox from disk back into database
	if currentActiveID != "" && currentActiveID != sandboxID {
		a.saveDiskToDatabase(currentActiveID, runtimeDir)
	}

	// 2. Clear runtime directory files
	entries, _ := os.ReadDir(runtimeDir)
	for _, entry := range entries {
		if entry.Name() == ".initialized" {
			continue // Keep workspace dependencies / setup state intact
		}
		_ = os.RemoveAll(filepath.Join(runtimeDir, entry.Name()))
	}

	// 3. Extract new target sandbox files from DB to runtime directory
	files, err := a.GetSandboxFiles(sandboxID)
	if err == nil {
		for _, f := range files {
			target := filepath.Join(runtimeDir, f.Path)
			if f.IsDir {
				_ = os.MkdirAll(target, 0755)
			} else {
				_ = os.MkdirAll(filepath.Dir(target), 0755)
				_ = os.WriteFile(target, []byte(f.Content), 0644)
			}
		}
	}

	// 4. Update Workspace active_sandbox_id pointer
	now := time.Now().Format(time.RFC3339)
	_, err = a.db.Exec("UPDATE workspaces SET active_sandbox_id = ?, updated_at = ? WHERE id = ?", sandboxID, now, workspaceID)
	return err
}

func (a *App) saveDiskToDatabase(sandboxID string, runtimeDir string) {
	now := time.Now().Format(time.RFC3339)
	_ = filepath.WalkDir(runtimeDir, func(diskPath string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		rel, _ := filepath.Rel(runtimeDir, diskPath)
		if rel == "." || rel == ".initialized" {
			return nil
		}

		isDirInt := 0
		if d.IsDir() {
			isDirInt = 1
		}

		if !d.IsDir() {
			data, readErr := os.ReadFile(diskPath)
			if readErr == nil {
				content := string(data)
				_, _ = a.db.Exec(`INSERT INTO sandbox_files (id, sandbox_id, path, content, is_dir, updated_at)
					VALUES (?, ?, ?, ?, ?, ?)
					ON CONFLICT(sandbox_id, path) DO UPDATE SET content=excluded.content, updated_at=excluded.updated_at`,
					fmt.Sprintf("%d", time.Now().UnixNano()), sandboxID, rel, content, isDirInt, now,
				)
			}
		}
		return nil
	})
}

func (a *App) BackupWorkspace(workspaceID string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	var ws Workspace
	err := a.db.QueryRow("SELECT id, name, description, config_yaml, active_sandbox_id, created_at, updated_at FROM workspaces WHERE id = ?", workspaceID).
		Scan(&ws.ID, &ws.Name, &ws.Description, &ws.ConfigYAML, &ws.ActiveSandboxID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return "", err
	}

	sandboxes, _ := a.GetSandboxes(workspaceID)
	var exportSandboxes []SandboxExportData
	for _, s := range sandboxes {
		files, _ := a.GetSandboxFiles(s.ID)
		exportSandboxes = append(exportSandboxes, SandboxExportData{
			Sandbox: s,
			Files:   files,
		})
	}

	backup := WorkspaceBackupData{
		Version:   "2.0",
		Workspace: ws,
		Sandboxes: exportSandboxes,
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) RestoreWorkspace(jsonContent string) (*Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	var backup WorkspaceBackupData
	if err := json.Unmarshal([]byte(jsonContent), &backup); err != nil {
		return nil, fmt.Errorf("invalid backup format: %w", err)
	}

	ws, err := a.CreateWorkspace(backup.Workspace.Name+" (Restored)", backup.Workspace.Description, "", backup.Workspace.ConfigYAML)
	if err != nil {
		return nil, err
	}

	for _, sbExport := range backup.Sandboxes {
		sbBytes, _ := json.Marshal(sbExport)
		_, _ = a.ImportSandbox(ws.ID, string(sbBytes))
	}

	return ws, nil
}
