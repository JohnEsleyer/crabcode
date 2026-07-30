package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

// InitializeCrabFolder configures folders, environment directories, and dynamic templates
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
	_ = os.MkdirAll(filepath.Join(path, "playground"), 0755)
	_ = os.MkdirAll(filepath.Join(path, "sandboxes"), 0755)

	_ = a.InitTemplates()
	_ = a.ensureGlobalDBInitialized()

	return nil
}

// GetGlobalSettings retrieves user settings
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

// SaveGlobalSettings migrates system base directory if changed
func (a *App) SaveGlobalSettings(settings GlobalSettings) error {
	oldRoot := a.GetCrabRootDirectory()
	newRoot := filepath.Clean(settings.CrabRootPath)

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if oldRoot != newRoot && oldRoot != "" && newRoot != "" {
		if _, err := os.Stat(oldRoot); err == nil {
			_ = os.MkdirAll(newRoot, 0755)
			_ = copyDir(oldRoot, newRoot)
			_ = os.RemoveAll(oldRoot)
		}
	}

	bootstrapFile := filepath.Join(home, ".crabcode_root.txt")
	if newRoot != "" {
		if err := os.WriteFile(bootstrapFile, []byte(newRoot), 0644); err != nil {
			return fmt.Errorf("failed to save custom system bootstrap file: %w", err)
		}
	}

	_ = os.MkdirAll(newRoot, 0755)
	settings.CrabRootPath = newRoot
	settings.UniversalEnvDir = filepath.Join(newRoot, "environments")
	_ = os.MkdirAll(settings.UniversalEnvDir, 0755)

	settingsFile := filepath.Join(newRoot, "settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	_ = os.WriteFile(settingsFile, data, 0644)

	return a.ensureGlobalDBInitialized()
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
