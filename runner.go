package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

func (a *App) GetWorkspaceConfig(workspaceID string) (*DeclarativeConfig, error) {
	var configYaml string
	err := a.db.QueryRow("SELECT config_yaml FROM workspaces WHERE id = ?", workspaceID).Scan(&configYaml)
	if err != nil {
		return nil, err
	}
	var cfg DeclarativeConfig
	if err := yaml.Unmarshal([]byte(configYaml), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (a *App) IsEnvironmentInitialized(workspaceID string) (bool, error) {
	runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)
	marker := filepath.Join(runtimeDir, ".initialized")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

func (a *App) InitializeEnvironment(workspaceID string) error {
	cfg, err := a.GetWorkspaceConfig(workspaceID)
	if err != nil {
		return err
	}

	runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return fmt.Errorf("failed to create environment folder: %w", err)
	}

	runtime.EventsEmit(a.ctx, "setup_output", map[string]interface{}{
		"workspaceId": workspaceID,
		"line":        fmt.Sprintf("[INIT] Preparing runtime for workspace '%s'...\n", workspaceID),
	})

	for _, step := range cfg.Setup {
		if step.Command == "" {
			continue
		}
		runtime.EventsEmit(a.ctx, "setup_output", map[string]interface{}{
			"workspaceId": workspaceID,
			"line":        fmt.Sprintf("[SETUP] Executing step '%s': %s\n", step.Name, step.Command),
		})

		if err := a.executeSyncCommandToTerm("setup", step.Command, runtimeDir, cfg.EnvVars); err != nil {
			runtime.EventsEmit(a.ctx, "setup_output", map[string]interface{}{
				"workspaceId": workspaceID,
				"line":        fmt.Sprintf("[ERROR] Setup step '%s' failed: %v\n", step.Name, err),
			})
			return fmt.Errorf("setup step '%s' failed: %w", step.Name, err)
		}
	}

	marker := filepath.Join(runtimeDir, ".initialized")
	_ = os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0644)

	runtime.EventsEmit(a.ctx, "setup_output", map[string]interface{}{
		"workspaceId": workspaceID,
		"line":        "[SUCCESS] Environment initialization completed successfully.\n",
	})

	return nil
}

func (a *App) RunSandbox(termID string, workspaceID string, action string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}

	cfg, err := a.GetWorkspaceConfig(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve workspace config: %w", err)
	}

	runtimeDir := a.GetWorkspaceRuntimePath(workspaceID)

	marker := filepath.Join(runtimeDir, ".initialized")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		return "", errors.New("ENV_NOT_INITIALIZED")
	}

	// Map UI action (run/build/test) to configured command
	cmdStr := cfg.Mappings.Run
	if action == "build" && cfg.Mappings.Build != "" {
		cmdStr = cfg.Mappings.Build
	} else if action == "test" && cfg.Mappings.Test != "" {
		cmdStr = cfg.Mappings.Test
	}

	if cmdStr == "" {
		return "", errors.New("no valid command mapped in workspace config")
	}

	shellOps := strings.Contains(cmdStr, "&&") || strings.Contains(cmdStr, ";") || strings.Contains(cmdStr, "|")

	if shellOps {
		shell := "sh"
		shellArg := "-c"
		if os.PathSeparator == '\\' {
			shell = "cmd"
			shellArg = "/c"
		}
		err = a.RunCommandWithEnv(termID, shell, []string{shellArg, cmdStr}, runtimeDir, cfg.EnvVars)
	} else {
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			return "", errors.New("command evaluates to empty sequence")
		}
		err = a.RunCommandWithEnv(termID, parts[0], parts[1:], runtimeDir, cfg.EnvVars)
	}

	return termID, err
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
		runtime.EventsEmit(a.ctx, "setup_output", map[string]interface{}{
			"line": string(output),
		})
	}
	return err
}
