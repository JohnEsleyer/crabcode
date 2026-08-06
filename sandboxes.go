package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (a *App) GetSandboxes(workspaceID string) ([]Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if workspaceID == "" {
		workspaceID = "default"
	}

	var activeID string
	_ = a.db.QueryRow("SELECT active_sandbox_id FROM workspaces WHERE id = ?", workspaceID).Scan(&activeID)

	rows, err := a.db.Query("SELECT id, workspace_id, name, COALESCE(folder, ''), markdown_note, html_note, created_at, updated_at FROM sandboxes WHERE workspace_id = ? ORDER BY updated_at DESC", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sandboxes := make([]Sandbox, 0)
	for rows.Next() {
		var s Sandbox
		if err := rows.Scan(&s.ID, &s.WorkspaceID, &s.Name, &s.Folder, &s.MarkdownNote, &s.HTMLNote, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.IsActive = (s.ID == activeID)
		sandboxes = append(sandboxes, s)
	}
	return sandboxes, nil
}

func (a *App) CreateSandboxInFolder(workspaceID string, name string, templateID string, folder string) (*Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if workspaceID == "" {
		workspaceID = "default"
	}

	templates, err := a.GetTemplates()
	if err != nil || len(templates) == 0 {
		return nil, fmt.Errorf("failed to load templates: %w", err)
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

	markdownNote := selectedTemplate.Config.Notes.Markdown
	htmlNote := selectedTemplate.Config.Notes.HTML

	_, err = a.db.Exec(
		"INSERT INTO sandboxes (id, workspace_id, name, folder, markdown_note, html_note, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, workspaceID, name, folder, markdownNote, htmlNote, now, now,
	)
	if err != nil {
		return nil, err
	}

	for _, f := range selectedTemplate.Files {
		_ = a.SaveSandboxFile(id, f.Path, f.Content, f.IsDir)
	}

	sb := &Sandbox{
		ID:           id,
		WorkspaceID:  workspaceID,
		Name:         name,
		Folder:       folder,
		MarkdownNote: markdownNote,
		HTMLNote:     htmlNote,
		IsActive:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return sb, nil
}

func (a *App) CreateSandbox(workspaceID string, name string, templateID string) (*Sandbox, error) {
	return a.CreateSandboxInFolder(workspaceID, name, templateID, "")
}

func (a *App) MoveSandbox(id string, folder string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET folder = ?, updated_at = ? WHERE id = ?", folder, now, id)
	return err
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
	_, _ = a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ?", id)
	_, err := a.db.Exec("DELETE FROM sandboxes WHERE id = ?", id)
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

func (a *App) GetSandboxFiles(sandboxID string) ([]SandboxFile, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, sandbox_id, path, content, is_dir, updated_at FROM sandbox_files WHERE sandbox_id = ?", sandboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]SandboxFile, 0)
	for rows.Next() {
		var f SandboxFile
		var isDirInt int
		if err := rows.Scan(&f.ID, &f.SandboxID, &f.Path, &f.Content, &isDirInt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.IsDir = (isDirInt == 1)
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

	_, err := a.db.Exec(`
		INSERT INTO sandbox_files (id, sandbox_id, path, content, is_dir, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(sandbox_id, path) DO UPDATE SET content=excluded.content, is_dir=excluded.is_dir, updated_at=excluded.updated_at`,
		fmt.Sprintf("%d", time.Now().UnixNano()), sandboxID, path, content, isDirInt, now,
	)

	// If this sandbox is active, sync immediately to disk
	var workspaceID, activeSandboxID string
	_ = a.db.QueryRow("SELECT workspace_id FROM sandboxes WHERE id = ?", sandboxID).Scan(&workspaceID)
	_ = a.db.QueryRow("SELECT active_sandbox_id FROM workspaces WHERE id = ?", workspaceID).Scan(&activeSandboxID)

	if sandboxID == activeSandboxID {
		runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)
		target := filepath.Join(runtimeDir, path)
		if isDir {
			_ = os.MkdirAll(target, 0755)
		} else {
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.WriteFile(target, []byte(content), 0644)
		}
	}

	return err
}

func (a *App) DeleteSandboxFile(sandboxID string, path string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	_, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path = ?", sandboxID, path)

	var workspaceID, activeSandboxID string
	_ = a.db.QueryRow("SELECT workspace_id FROM sandboxes WHERE id = ?", sandboxID).Scan(&workspaceID)
	_ = a.db.QueryRow("SELECT active_sandbox_id FROM workspaces WHERE id = ?", workspaceID).Scan(&activeSandboxID)

	if sandboxID == activeSandboxID {
		runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)
		_ = os.RemoveAll(filepath.Join(runtimeDir, path))
	}

	return err
}

func (a *App) ExportSandbox(sandboxID string) (string, error) {
	var s Sandbox
	err := a.db.QueryRow("SELECT id, workspace_id, name, COALESCE(folder, ''), markdown_note, html_note, created_at, updated_at FROM sandboxes WHERE id = ?", sandboxID).
		Scan(&s.ID, &s.WorkspaceID, &s.Name, &s.Folder, &s.MarkdownNote, &s.HTMLNote, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return "", err
	}

	files, _ := a.GetSandboxFiles(sandboxID)
	payload := SandboxExportData{
		Sandbox: s,
		Files:   files,
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	return string(b), err
}

func (a *App) ImportSandbox(workspaceID string, jsonContent string) (*Sandbox, error) {
	var payload SandboxExportData
	if err := json.Unmarshal([]byte(jsonContent), &payload); err != nil {
		return nil, err
	}

	sb, err := a.CreateSandboxInFolder(workspaceID, payload.Sandbox.Name+" (Imported)", "python", payload.Sandbox.Folder)
	if err != nil {
		return nil, err
	}

	for _, f := range payload.Files {
		_ = a.SaveSandboxFile(sb.ID, f.Path, f.Content, f.IsDir)
	}

	return sb, nil
}
