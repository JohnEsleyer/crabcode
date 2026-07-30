package main

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
