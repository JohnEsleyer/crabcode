package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
