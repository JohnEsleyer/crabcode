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

// ResolveAndCheckWorkspace checks whether target directory exists and contains a .crab folder
func (a *App) ResolveAndCheckWorkspace(dir string, baseDir string) (*WorkspaceInitInfo, error) {
	target := dir
	if target == "" || target == "." {
		target = baseDir
	} else if !filepath.IsAbs(target) {
		if baseDir != "" {
			target = filepath.Join(baseDir, target)
		}
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	dotCrabPath := filepath.Join(absPath, ".crab")
	crabInfo, crabErr := os.Stat(dotCrabPath)
	hasDotCrab := crabErr == nil && crabInfo.IsDir()

	return &WorkspaceInitInfo{
		Path:       absPath,
		HasDotCrab: hasDotCrab,
		Exists:     true,
	}, nil
}

// InitializeDotCrab creates the .crab directory inside the target path
func (a *App) InitializeDotCrab(dir string) error {
	dotCrabPath := filepath.Join(dir, ".crab")
	return os.MkdirAll(dotCrabPath, 0755)
}

// GetCLIOpenPath returns and clears any workspace path passed via command line arguments
func (a *App) GetCLIOpenPath() string {
	a.processMutex.Lock()
	defer a.processMutex.Unlock()
	p := a.cliPath
	a.cliPath = ""
	return p
}
