package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func (a *App) GetTemplatesDirectory() string {
	return filepath.Join(a.GetCrabRootDirectory(), "templates")
}

func (a *App) InitTemplates() error {
	dir := a.GetTemplatesDirectory()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	defaultTemplates := map[string]string{
		"python.yaml": `name: "Python Environment"
version: "1.0"
environment: "python"
env_dir: "environments/python"
icon_color: "#3572A5"
setup: []
env_vars:
  PYTHONUNBUFFERED: "1"
files:
  - path: "main.py"
    content: "# Python Sandbox\nprint('Hello from CrabCode Python Sandbox!')\n"
build: []
run:
  command: "python3 main.py"
notes:
  markdown: "# Python Environment\n\nShared environment for dynamic Python script execution."
  html: "<h3>🐍 Python Sandbox Active</h3>"
`,
		"go.yaml": `name: "Go Standard Environment"
version: "1.0"
environment: "go"
env_dir: "environments/go"
icon_color: "#00ADD8"
setup: []
files:
  - path: "main.go"
    content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from CrabCode Go Sandbox!\")\n}\n"
run:
  command: "go run main.go"
notes:
  markdown: "# Go Sandbox Environment\n\nWrite fast system components in this shared Go environment."
  html: "<h3 style='color:#00ADD8;'>Go Compiler Environment Active</h3>"
`,
		"go-raylib.yaml": `name: "Go Raylib Engine"
version: "1.0"
environment: "go"
env_dir: "environments/go_raylib"
icon_color: "#00ADD8"
setup:
  - name: "Initialize Raylib Go Module"
    command: "go mod init raylib_app && go get github.com/gen2brain/raylib-go/raylib"
files:
  - path: "main.go"
    content: "package main\n\nimport rl \"github.com/gen2brain/raylib-go/raylib\"\n\nfunc main() {\n\trl.InitWindow(800, 450, \"Raylib Go Demo\")\n\tdefer rl.CloseWindow()\n\trl.SetTargetFPS(60)\n\tfor !rl.WindowShouldClose() {\n\t\trl.BeginDrawing()\n\t\trl.ClearBackground(rl.RayWhite)\n\t\trl.DrawText(\"Hello Raylib from CrabCode!\", 190, 200, 20, rl.LightGray)\n\t\trl.EndDrawing()\n\t}\n}\n"
run:
  command: "go run main.go"
notes:
  markdown: "# Go Raylib Shared Environment\n\nShared project environment for Go Raylib graphics & game experiments."
  html: "<h3 style='color:#00ADD8;'>🎨 Raylib Go Shared Environment</h3>"
`,
		"rust.yaml": `name: "Rust Core Environment"
version: "1.0"
environment: "rust"
env_dir: "environments/rust"
icon_color: "#DEA584"
setup: []
files:
  - path: "main.rs"
    content: "fn main() {\n    println!(\"Hello from CrabCode Rust Sandbox!\");\n}\n"
run:
  command: "rustc main.rs && ./main"
notes:
  markdown: "# Rust Sandbox Environment\n\nValidate memory management rules, traits, and complex data models safely."
  html: "<h3 style='color:#DEA584;'>Rust Core Active</h3>"
`,
		"rust-bevy.yaml": `name: "Rust Bevy Game Engine"
version: "1.0"
environment: "rust"
env_dir: "environments/rust_bevy"
icon_color: "#CE412B"
setup:
  - name: "Initialize Bevy Cargo project"
    command: "cargo init --vcs none && cargo add bevy"
files:
  - path: "src/main.rs"
    content: "use bevy::prelude::*;\n\nfn main() {\n    App::new()\n        .add_plugins(DefaultPlugins)\n        .add_systems(Startup, setup)\n        .run();\n}\n\nfn setup(mut commands: Commands) {\n    commands.spawn(Camera2dBundle::default());\n}\n"
run:
  command: "cargo run"
notes:
  markdown: "# Rust Bevy Engine\n\nShared environment template for 2D/3D Bevy game development."
  html: "<h3 style='color:#CE412B;'>🎮 Bevy Game Engine Shared Environment</h3>"
`,
		"java.yaml": `name: "Java Environment"
version: "1.0"
environment: "java"
env_dir: "environments/java"
icon_color: "#B07219"
setup: []
files:
  - path: "Main.java"
    content: "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello from CrabCode Java Sandbox!\");\n    }\n}\n"
run:
  command: "java Main.java"
notes:
  markdown: "# Java Sandbox Environment"
  html: "<h3 style='color:#B07219;'>☕ Java Virtual Machine Active</h3>"
`,
		"sql.yaml": `name: "SQL Database Environment"
version: "1.0"
environment: "sql"
env_dir: "environments/sql"
icon_color: "#00ADD8"
setup: []
files:
  - path: "main.sql"
    content: "-- SQL Sandbox\nCREATE TABLE IF NOT EXISTS demo (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT);\nINSERT INTO demo (message) VALUES ('Hello from CrabCode SQL Sandbox!');\nSELECT * FROM demo;\n"
run:
  command: "sqlite3 :memory: < main.sql"
notes:
  markdown: "# SQL Sandbox Environment\n\nRun in-memory SQLite relational queries and test database schema designs."
  html: "<h3>💾 SQL Database Engine Active</h3>"
`,
		"surrealdb.yaml": `name: "SurrealDB Environment"
version: "1.0"
environment: "surrealdb"
env_dir: "environments/surrealdb"
icon_color: "#FF00A0"
setup: []
files:
  - path: "main.surql"
    content: "-- SurrealDB Sandbox\nCREATE user SET name = 'CrabCode Developer', role = 'Admin';\nSELECT * FROM user;\n"
run:
  command: "surreal sql --endpoint memory --ns test --db test < main.surql"
notes:
  markdown: "# SurrealDB Sandbox Environment\n\nExecute SurrealQL queries for multi-model graph and document database design."
  html: "<h3 style='color:#FF00A0;'>⚡ SurrealDB Multi-Model Active</h3>"
`,
		"javascript.yaml": `name: "JavaScript Node Environment"
version: "1.0"
environment: "node"
env_dir: "environments/node"
icon_color: "#F7DF1E"
setup: []
files:
  - path: "index.js"
    content: "// JavaScript Sandbox\nconsole.log('Hello from CrabCode JavaScript Sandbox!');\n"
run:
  command: "node index.js"
notes:
  markdown: "# JavaScript Sandbox Environment"
  html: "<h3>Node.js Shared Environment Ready</h3>"
`,
		"typescript.yaml": `name: "TypeScript Environment"
version: "1.0"
environment: "node"
env_dir: "environments/typescript"
icon_color: "#3178C6"
setup: []
files:
  - path: "index.ts"
    content: "// TypeScript Sandbox\nconst greeting: string = 'Hello from CrabCode TypeScript Sandbox!';\nconsole.log(greeting);\n"
run:
  command: "npx tsx index.ts"
notes:
  markdown: "# TypeScript Sandbox Environment"
  html: "<h3>TypeScript Environment Ready</h3>"
`,
	}

	for fileName, content := range defaultTemplates {
		filePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			_ = os.WriteFile(filePath, []byte(content), 0644)
		}
	}

	return nil
}

func (a *App) GetTemplates() ([]TemplateSpec, error) {
	_ = a.InitTemplates()
	dir := a.GetTemplatesDirectory()

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var specs []TemplateSpec
	for _, f := range files {
		if f.IsDir() || (!strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml")) {
			continue
		}

		filePath := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var cfg DeclarativeConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}

		id := strings.TrimSuffix(strings.TrimSuffix(f.Name(), ".yaml"), ".yml")
		color := cfg.IconColor
		if color == "" {
			color = "#ff5a36"
		}

		specs = append(specs, TemplateSpec{
			ID:          id,
			Name:        cfg.Name,
			Environment: cfg.Environment,
			IconColor:   color,
			Config:      cfg,
			RawYAML:     string(data),
		})
	}

	return specs, nil
}

func (a *App) GetPlaygroundDirectory() (string, error) {
	root := a.GetCrabRootDirectory()
	return filepath.Join(root, "playground"), nil
}

func (a *App) InitPlayground() error {
	dir, err := a.GetPlaygroundDirectory()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	templates := map[string]string{
		"scratch.py":    "# Python Scratchpad\nprint('Hello from the Python Playground!')\n",
		"scratch.js":    "// Javascript Scratchpad\nconsole.log('Hello from the JavaScript Playground!');\n",
		"scratch.go":    "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from the Go Playground!\")\n}\n",
		"scratch.rs":    "fn main() {\n    println!(\"Hello from the Rust Playground!\");\n}\n",
		"scratch.java":  "// Java Scratchpad\npublic class Scratch {\n    public static void main(String[] args) {\n        System.out.println(\"Hello from the Java Playground!\");\n    }\n}\n",
		"scratch.ts":    "// TypeScript Scratchpad\nconst greeting: string = \"Hello from the TypeScript Playground!\";\nconsole.log(greeting);\n",
		"scratch.sql":   "-- SQL Scratchpad\nCREATE TABLE IF NOT EXISTS scratchpad (id INTEGER PRIMARY KEY, message TEXT);\nINSERT INTO scratchpad (message) VALUES ('Hello from the SQL Playground!');\nSELECT * FROM scratchpad;\n",
		"scratch.dart":  "// Dart Scratchpad\nvoid main() {\n  print('Hello from the Dart Playground!');\n}\n",
		"scratch.surql": "-- SurrealDB Scratchpad\nCREATE user SET name = 'CrabCode', role = 'Developer';\nSELECT * FROM user;\n",
	}

	for fileName, content := range templates {
		filePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			_ = os.WriteFile(filePath, []byte(content), 0644)
		}
	}
	return nil
}
