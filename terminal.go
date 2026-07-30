package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) RunCommand(id string, commandName string, args []string, dir string) error {
	return a.RunCommandWithEnv(id, commandName, args, dir, nil)
}

func (a *App) RunCommandWithEnv(id string, commandName string, args []string, dir string, envVars map[string]string) error {
	a.processMutex.Lock()
	if existingCmd, exists := a.activeProcesses[id]; exists {
		if existingCmd.Process != nil {
			_ = existingCmd.Process.Kill()
		}
		delete(a.activeProcesses, id)
		delete(a.activeStdins, id)
	}

	cmd := exec.Command(commandName, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	if len(envVars) > 0 {
		env := os.Environ()
		for k, v := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	stdinPipe, err := cmd.StdinPipe()
	if err == nil {
		a.activeStdins[id] = stdinPipe
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		a.processMutex.Unlock()
		return err
	}
	cmd.Stderr = cmd.Stdout

	a.activeProcesses[id] = cmd
	a.processMutex.Unlock()

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
					"id":   id,
					"line": string(buf[:n]),
				})
			}
			if err != nil {
				break
			}
		}

		err := cmd.Wait()
		a.processMutex.Lock()
		delete(a.activeProcesses, id)
		delete(a.activeStdins, id)
		a.processMutex.Unlock()

		status := "0"
		if err != nil {
			status = err.Error()
		}
		runtime.EventsEmit(a.ctx, "terminal_status", map[string]interface{}{
			"id":     id,
			"status": status,
		})
	}()

	return nil
}

func (a *App) StopCommand(id string) error {
	a.processMutex.Lock()
	defer a.processMutex.Unlock()

	if stdin, exists := a.activeStdins[id]; exists {
		_ = stdin.Close()
		delete(a.activeStdins, id)
	}

	if cmd, exists := a.activeProcesses[id]; exists {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			delete(a.activeProcesses, id)
			return err
		}
	}
	return nil
}

func (a *App) StartTerminalSession(id string, dir string) error {
	a.processMutex.Lock()
	if existingCmd, exists := a.activeProcesses[id]; exists {
		if existingCmd.Process != nil {
			_ = existingCmd.Process.Kill()
		}
		delete(a.activeProcesses, id)
		delete(a.activeStdins, id)
	}

	shell := "bash"
	var shellArgs []string
	if os.PathSeparator == '\\' {
		shell = "cmd.exe"
		shellArgs = []string{"/q"}
	} else {
		if uShell := os.Getenv("SHELL"); uShell != "" {
			shell = uShell
		}
		// No -i flag: interactive mode over pipes sends SIGTTIN/SIGTTOU, suspending the process.
		shellArgs = []string{}
	}

	cmd := exec.Command(shell, shellArgs...)
	if dir != "" {
		cmd.Dir = dir
	}

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		a.processMutex.Unlock()
		return err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		a.processMutex.Unlock()
		return err
	}
	cmd.Stderr = cmd.Stdout

	a.activeProcesses[id] = cmd
	a.activeStdins[id] = stdinPipe
	a.processMutex.Unlock()

	if err := cmd.Start(); err != nil {
		a.processMutex.Lock()
		delete(a.activeProcesses, id)
		delete(a.activeStdins, id)
		a.processMutex.Unlock()
		return err
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				runtime.EventsEmit(a.ctx, "terminal_output", map[string]interface{}{
					"id":   id,
					"line": string(buf[:n]),
				})
			}
			if err != nil {
				break
			}
		}

		err := cmd.Wait()
		a.processMutex.Lock()
		delete(a.activeProcesses, id)
		delete(a.activeStdins, id)
		a.processMutex.Unlock()

		status := "0"
		if err != nil {
			status = err.Error()
		}
		runtime.EventsEmit(a.ctx, "terminal_status", map[string]interface{}{
			"id":     id,
			"status": status,
		})
	}()
	return nil
}

func (a *App) WriteTerminalInput(id string, input string) error {
	a.processMutex.Lock()
	stdin, exists := a.activeStdins[id]
	a.processMutex.Unlock()

	if !exists || stdin == nil {
		return errors.New("terminal session not active or stdin pipe closed")
	}

	if !strings.HasSuffix(input, "\n") {
		input += "\n"
	}

	_, err := stdin.Write([]byte(input))
	return err
}
