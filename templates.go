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
icon_color: "#3572A5"
setup: []
mappings:
  run: "python3 main.py"
env_vars:
  PYTHONUNBUFFERED: "1"
files:
  - path: "main.py"
    content: "# Python Sandbox\nprint('Hello from CrabCode Python Sandbox!')\n"
`,
		"go.yaml": `name: "Go Standard Environment"
version: "1.0"
environment: "go"
icon_color: "#00ADD8"
setup: []
mappings:
  run: "go run main.go"
files:
  - path: "main.go"
    content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from CrabCode Go Sandbox!\")\n}\n"
`,
		"go-raylib.yaml": `name: "Go Raylib Engine"
version: "1.0"
environment: "go"
icon_color: "#00ADD8"
setup:
  - name: "Initialize Raylib Go Module"
    command: "go mod init raylib_app && go get github.com/gen2brain/raylib-go/raylib"
mappings:
  run: "go run main.go"
files:
  - path: "main.go"
    content: "package main\n\nimport rl \"github.com/gen2brain/raylib-go/raylib\"\n\nfunc main() {\n\trl.InitWindow(800, 450, \"Raylib Go Demo\")\n\tdefer rl.CloseWindow()\n\trl.SetTargetFPS(60)\n\tfor !rl.WindowShouldClose() {\n\t\trl.BeginDrawing()\n\t\trl.ClearBackground(rl.RayWhite)\n\t\trl.DrawText(\"Hello Raylib from CrabCode!\", 190, 200, 20, rl.LightGray)\n\t\trl.EndDrawing()\n\t}\n}\n"
`,
		"rust.yaml": `name: "Rust Core Environment"
version: "1.0"
environment: "rust"
icon_color: "#DEA584"
setup: []
mappings:
  run: "rustc main.rs && ./main"
files:
  - path: "main.rs"
    content: "fn main() {\n    println!(\"Hello from CrabCode Rust Sandbox!\");\n}\n"
`,
		"rust-bevy.yaml": `name: "Rust Bevy Game Engine"
version: "1.0"
environment: "rust"
icon_color: "#CE412B"
setup:
  - name: "Initialize Bevy Cargo project"
    command: "cargo init --vcs none && cargo add bevy"
mappings:
  run: "cargo run"
files:
  - path: "src/main.rs"
    content: "use bevy::prelude::*;\n\nfn main() {\n    App::new()\n        .add_plugins(DefaultPlugins)\n        .add_systems(Startup, setup)\n        .run();\n}\n\nfn setup(mut commands: Commands) {\n    commands.spawn(Camera2dBundle::default());\n}\n"
`,
		"java.yaml": `name: "Java Environment"
version: "1.0"
environment: "java"
icon_color: "#B07219"
setup: []
mappings:
  run: "java Main.java"
files:
  - path: "Main.java"
    content: "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello from CrabCode Java Sandbox!\");\n    }\n}\n"
`,
		"sql.yaml": `name: "SQL Database Environment"
version: "1.0"
environment: "sql"
icon_color: "#00ADD8"
setup: []
mappings:
  run: "sqlite3 :memory: < main.sql"
files:
  - path: "main.sql"
    content: "-- SQL Sandbox\nCREATE TABLE IF NOT EXISTS demo (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT);\nINSERT INTO demo (message) VALUES ('Hello from CrabCode SQL Sandbox!');\nSELECT * FROM demo;\n"
`,
		"surrealdb.yaml": `name: "SurrealDB Environment"
version: "1.0"
environment: "surrealdb"
icon_color: "#FF00A0"
setup: []
mappings:
  run: "surreal sql --endpoint memory --ns test --db test < main.surql"
files:
  - path: "main.surql"
    content: "-- SurrealDB Sandbox\nCREATE user SET name = 'CrabCode Developer', role = 'Admin';\nSELECT * FROM user;\n"
`,
		"javascript.yaml": `name: "JavaScript Node Environment"
version: "1.0"
environment: "node"
icon_color: "#F7DF1E"
setup: []
mappings:
  run: "node index.js"
files:
  - path: "index.js"
    content: "// JavaScript Sandbox\nconsole.log('Hello from CrabCode JavaScript Sandbox!');\n"
`,
		"typescript.yaml": `name: "TypeScript Environment"
version: "1.0"
environment: "node"
icon_color: "#3178C6"
setup: []
mappings:
  run: "npx tsx index.ts"
files:
  - path: "index.ts"
    content: "// TypeScript Sandbox\nconst greeting: string = 'Hello from CrabCode TypeScript Sandbox!';\nconsole.log(greeting);\n"
`,
	}

	for fileName, content := range defaultTemplates {
		filePath := filepath.Join(dir, fileName)

		// Refresh the factory template whenever the bundled content changes so
		// existing installations stay in sync with the current schema.
		if existing, err := os.ReadFile(filePath); err == nil && string(existing) == content {
			continue
		}
		_ = os.WriteFile(filePath, []byte(content), 0644)
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

		var parsed struct {
			DeclarativeConfig `yaml:",inline"`
			Files             []TemplateFile `yaml:"files"`
		}
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			continue
		}

		id := strings.TrimSuffix(strings.TrimSuffix(f.Name(), ".yaml"), ".yml")
		color := parsed.IconColor
		if color == "" {
			color = "#ff5a36"
		}

		specs = append(specs, TemplateSpec{
			ID:          id,
			Name:        parsed.Name,
			Environment: parsed.Environment,
			IconColor:   color,
			Config:      parsed.DeclarativeConfig,
			Files:       parsed.Files,
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
