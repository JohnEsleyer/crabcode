package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) GetSandboxes() ([]Sandbox, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, name, config_yaml, markdown_note, html_note, COALESCE(folder, ''), created_at, updated_at FROM sandboxes ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sandboxes := make([]Sandbox, 0)
	for rows.Next() {
		var s Sandbox
		if err := rows.Scan(&s.ID, &s.Name, &s.ConfigYAML, &s.MarkdownNote, &s.HTMLNote, &s.Folder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, s)
	}
	return sandboxes, nil
}

func (a *App) CreateSandboxInFolder(name string, templateID string, folder string) (*Sandbox, error) {
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
		"INSERT INTO sandboxes (id, name, config_yaml, markdown_note, html_note, folder, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, name, configYaml, markdownNote, htmlNote, folder, now, now,
	)
	if err != nil {
		return nil, err
	}

	for _, f := range cfg.Files {
		_ = a.SaveSandboxFile(id, f.Path, f.Content, f.IsDir)
	}

	return &Sandbox{
		ID:           id,
		Name:         name,
		ConfigYAML:   configYaml,
		MarkdownNote: markdownNote,
		HTMLNote:     htmlNote,
		Folder:       folder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (a *App) CreateSandbox(name string, templateID string) (*Sandbox, error) {
	return a.CreateSandboxInFolder(name, templateID, "")
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

func (a *App) SaveSandboxNotes(id string, markdownNote string, htmlNote string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec("UPDATE sandboxes SET markdown_note = ?, html_note = ?, updated_at = ? WHERE id = ?", markdownNote, htmlNote, now, id)
	return err
}

func (a *App) syncSandboxFromDisk(sandboxID string) error {
	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return err
	}
	envDir := a.resolveEnvDir(cfg)

	if _, statErr := os.Stat(envDir); os.IsNotExist(statErr) {
		return nil
	}

	rows, err := a.db.Query("SELECT path, is_dir FROM sandbox_files WHERE sandbox_id = ?", sandboxID)
	if err != nil {
		return err
	}
	dbFiles := make(map[string]bool)
	for rows.Next() {
		var path string
		var isDir int
		if err := rows.Scan(&path, &isDir); err != nil {
			rows.Close()
			return err
		}
		dbFiles[path] = isDir == 1
	}
	rows.Close()

	now := time.Now().Format(time.RFC3339)
	diskFiles := make(map[string]bool)

	_ = filepath.WalkDir(envDir, func(diskPath string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		rel, _ := filepath.Rel(envDir, diskPath)
		if rel == "." {
			return nil
		}
		diskFiles[rel] = d.IsDir()

		isDirInt := 0
		if d.IsDir() {
			isDirInt = 1
		}

		if _, exists := dbFiles[rel]; !exists {
			id := fmt.Sprintf("%d", time.Now().UnixNano())
			content := ""
			if !d.IsDir() {
				data, readErr := os.ReadFile(diskPath)
				if readErr == nil {
					content = string(data)
				}
			}
			_, _ = a.db.Exec(
				"INSERT INTO sandbox_files (id, sandbox_id, path, content, is_dir, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
				id, sandboxID, rel, content, isDirInt, now,
			)
		} else if !d.IsDir() {
			data, readErr := os.ReadFile(diskPath)
			if readErr != nil {
				return nil
			}
			content := string(data)
			var dbContent string
			_ = a.db.QueryRow("SELECT content FROM sandbox_files WHERE sandbox_id = ? AND path = ?", sandboxID, rel).Scan(&dbContent)
			if dbContent != content {
				_, _ = a.db.Exec("UPDATE sandbox_files SET content = ?, updated_at = ? WHERE sandbox_id = ? AND path = ?",
					content, now, sandboxID, rel)
			}
		}
		return nil
	})

	for dbPath := range dbFiles {
		if !diskFiles[dbPath] {
			a.deleteSandboxFileDisk(sandboxID, dbPath)
			_, _ = a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path = ?", sandboxID, dbPath)
			_, _ = a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path LIKE ?", sandboxID, dbPath+"/%")
		}
	}

	return nil
}

func (a *App) GetSandboxFiles(sandboxID string) ([]SandboxFile, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	_ = a.syncSandboxFromDisk(sandboxID)

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
		f.IsDir = isDirInt == 1
		files = append(files, f)
	}
	return files, nil
}

func (a *App) writeSandboxFileDisk(sandboxID string, path string, content string, isDir bool) {
	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return
	}
	envDir := a.resolveEnvDir(cfg)
	target := filepath.Join(envDir, path)
	if isDir {
		_ = os.MkdirAll(target, 0755)
	} else {
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		_ = os.WriteFile(target, []byte(content), 0644)
	}
}

func (a *App) deleteSandboxFileDisk(sandboxID string, path string) {
	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return
	}
	envDir := a.resolveEnvDir(cfg)
	target := filepath.Join(envDir, path)
	_ = os.RemoveAll(target)
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

	a.writeSandboxFileDisk(sandboxID, path, content, isDir)

	return nil
}

func (a *App) DeleteSandboxFile(sandboxID string, path string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	a.deleteSandboxFileDisk(sandboxID, path)

	if _, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path = ?", sandboxID, path); err != nil {
		return err
	}

	prefix := path + "/"
	_, err := a.db.Exec("DELETE FROM sandbox_files WHERE sandbox_id = ? AND path LIKE ?", sandboxID, prefix+"%")
	return err
}
