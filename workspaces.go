package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (a *App) GetWorkspaces() ([]Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, name, description, created_at, updated_at FROM workspaces ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workspaces := make([]Workspace, 0)
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, nil
}

func (a *App) CreateWorkspace(name string, description string) (*Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	id := fmt.Sprintf("ws_%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	_, err := a.db.Exec(
		"INSERT INTO workspaces (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, name, description, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (a *App) DeleteWorkspace(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	if id == "default" {
		return errors.New("cannot delete default workspace")
	}

	// Delete all sandboxes associated with this workspace
	sandboxes, _ := a.GetSandboxes(id)
	for _, s := range sandboxes {
		_ = a.DeleteSandbox(s.ID)
	}

	_, err := a.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	return err
}

// ExportSandbox exports a sandbox and its virtual files into a JSON string
func (a *App) ExportSandbox(sandboxID string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	var s Sandbox
	err := a.db.QueryRow("SELECT id, workspace_id, name, config_yaml, markdown_note, html_note, COALESCE(folder, ''), created_at, updated_at FROM sandboxes WHERE id = ?", sandboxID).
		Scan(&s.ID, &s.WorkspaceID, &s.Name, &s.ConfigYAML, &s.MarkdownNote, &s.HTMLNote, &s.Folder, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return "", err
	}

	files, err := a.GetSandboxFiles(sandboxID)
	if err != nil {
		files = []SandboxFile{}
	}

	payload := SandboxExportData{
		Sandbox: s,
		Files:   files,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportSandbox imports a sandbox JSON string into the target workspace
func (a *App) ImportSandbox(targetWorkspaceID string, jsonContent string) (*Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	var payload SandboxExportData
	if err := json.Unmarshal([]byte(jsonContent), &payload); err != nil {
		return nil, fmt.Errorf("invalid sandbox export format: %w", err)
	}

	newID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)

	s := payload.Sandbox
	s.ID = newID
	s.WorkspaceID = targetWorkspaceID
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := a.db.Exec(
		"INSERT INTO sandboxes (id, workspace_id, name, config_yaml, markdown_note, html_note, folder, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.ID, s.WorkspaceID, s.Name, s.ConfigYAML, s.MarkdownNote, s.HTMLNote, s.Folder, now, now,
	)
	if err != nil {
		return nil, err
	}

	for _, f := range payload.Files {
		_ = a.SaveSandboxFile(s.ID, f.Path, f.Content, f.IsDir)
	}

	return &s, nil
}

// BackupWorkspace exports an entire workspace and all its sandboxes to a JSON backup
func (a *App) BackupWorkspace(workspaceID string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	var ws Workspace
	err := a.db.QueryRow("SELECT id, name, description, created_at, updated_at FROM workspaces WHERE id = ?", workspaceID).
		Scan(&ws.ID, &ws.Name, &ws.Description, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return "", err
	}

	sandboxes, err := a.GetSandboxes(workspaceID)
	if err != nil {
		sandboxes = []Sandbox{}
	}

	var exportSandboxes []SandboxExportData
	for _, s := range sandboxes {
		files, _ := a.GetSandboxFiles(s.ID)
		exportSandboxes = append(exportSandboxes, SandboxExportData{
			Sandbox: s,
			Files:   files,
		})
	}

	backup := WorkspaceBackupData{
		Version:   "1.0",
		Workspace: ws,
		Sandboxes: exportSandboxes,
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RestoreWorkspace imports a workspace backup JSON payload
func (a *App) RestoreWorkspace(jsonContent string) (*Workspace, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	var backup WorkspaceBackupData
	if err := json.Unmarshal([]byte(jsonContent), &backup); err != nil {
		return nil, fmt.Errorf("invalid workspace backup format: %w", err)
	}

	ws, err := a.CreateWorkspace(backup.Workspace.Name+" (Restored)", backup.Workspace.Description)
	if err != nil {
		return nil, err
	}

	for _, sbExport := range backup.Sandboxes {
		sbBytes, _ := json.Marshal(sbExport)
		_, _ = a.ImportSandbox(ws.ID, string(sbBytes))
	}

	return ws, nil
}
