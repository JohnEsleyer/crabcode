package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetSandboxDirectory(sandboxID string) (string, error) {
	dir := filepath.Join(a.GetCrabRootDirectory(), "sandboxes", sandboxID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) getSandboxConfig(sandboxID string) (*DeclarativeConfig, error) {
	var configYaml string
	err := a.db.QueryRow("SELECT config_yaml FROM sandboxes WHERE id = ?", sandboxID).Scan(&configYaml)
	if err != nil {
		return nil, err
	}
	var cfg DeclarativeConfig
	if err := yaml.Unmarshal([]byte(configYaml), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (a *App) resolveEnvDir(cfg *DeclarativeConfig) string {
	crabRoot := a.GetCrabRootDirectory()
	envFolderName := cfg.EnvDir
	if envFolderName == "" {
		envFolderName = cfg.Environment
	}
	if !strings.HasPrefix(envFolderName, "environments/") && !strings.HasPrefix(envFolderName, "environments\\") {
		envFolderName = filepath.Join("environments", envFolderName)
	}
	return filepath.Join(crabRoot, envFolderName)
}

func (a *App) IsEnvironmentInitialized(sandboxID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}
	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return false, err
	}
	envDir := a.resolveEnvDir(cfg)
	marker := filepath.Join(envDir, ".initialized")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

func (a *App) InitializeEnvironment(sandboxID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return err
	}

	envDir := a.resolveEnvDir(cfg)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create shared environment folder: %w", err)
	}

	for _, step := range cfg.Setup {
		if step.Command == "" {
			continue
		}
		targetDir := envDir
		if step.Dir != "" {
			targetDir = filepath.Join(a.GetCrabRootDirectory(), step.Dir)
		}
		_ = os.MkdirAll(targetDir, 0755)
		if err := a.executeSyncCommand(step.Command, targetDir, cfg.EnvVars); err != nil {
			return fmt.Errorf("setup step '%s' failed: %w", step.Name, err)
		}
	}

	marker := filepath.Join(envDir, ".initialized")
	_ = os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0644)
	return nil
}

func (a *App) RunSandbox(termID string, sandboxID string, activeFilePath string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	cfg, err := a.getSandboxConfig(sandboxID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve sandbox config: %w", err)
	}

	envDir := a.resolveEnvDir(cfg)

	marker := filepath.Join(envDir, ".initialized")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		return "", errors.New("ENV_NOT_INITIALIZED")
	}

	files, err := a.GetSandboxFiles(sandboxID)
	if err != nil {
		return "", fmt.Errorf("failed to load virtual source files: %w", err)
	}

	// 1. Extract virtual files into shared project directory
	for _, f := range files {
		targetPath := filepath.Join(envDir, f.Path)
		if f.IsDir {
			_ = os.MkdirAll(targetPath, 0755)
		} else {
			_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
			_ = os.WriteFile(targetPath, []byte(f.Content), 0644)
		}
	}

	// 2. Execute YAML build steps (if any) streaming to termID
	for _, bStep := range cfg.Build {
		if bStep.Command == "" {
			continue
		}
		runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
			"id":   termID,
			"line": fmt.Sprintf("[Build Step: %s]\n", bStep.Name),
		})
		_ = a.executeSyncCommandToTerm(termID, bStep.Command, envDir, cfg.EnvVars)
	}

	// 3. Execute YAML run command
	runCmd := cfg.Run.Command
	if runCmd == "" {
		return "", errors.New("no valid 'run.command' specified in declarative sandbox rules")
	}

	shellOps := strings.Contains(runCmd, "&&") || strings.Contains(runCmd, ";") || strings.Contains(runCmd, "|") || strings.Contains(runCmd, "<") || strings.Contains(runCmd, ">")

	if shellOps {
		shell := "sh"
		shellArg := "-c"
		if os.PathSeparator == '\\' {
			shell = "cmd"
			shellArg = "/c"
		}
		err = a.RunCommandWithEnv(termID, shell, []string{shellArg, runCmd}, envDir, cfg.EnvVars)
	} else {
		cmdParts := strings.Fields(runCmd)
		if len(cmdParts) == 0 {
			return "", errors.New("run.command evaluates to empty sequence")
		}
		runnerBin := cmdParts[0]
		runnerArgs := cmdParts[1:]
		err = a.RunCommandWithEnv(termID, runnerBin, runnerArgs, envDir, cfg.EnvVars)
	}

	if err != nil {
		return "", err
	}

	return termID, nil
}

func (a *App) executeSyncCommandToTerm(termID string, cmdStr string, dir string, envVars map[string]string) error {
	shell := "sh"
	shellArg := "-c"
	if os.PathSeparator == '\\' {
		shell = "cmd"
		shellArg = "/c"
	}

	cmd := exec.Command(shell, shellArg, cmdStr)
	cmd.Dir = dir

	env := os.Environ()
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
			"id":   termID,
			"line": string(output),
		})
	}
	return err
}

func (a *App) executeSyncCommand(cmdStr string, dir string, envVars map[string]string) error {
	return a.executeSyncCommandToTerm("setup", cmdStr, dir, envVars)
}
