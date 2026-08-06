package main

import (
	"database/sql"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
)

func TestLegacyCommandFromNode(t *testing.T) {
	legacyYAML := `name: "Go Standard Environment"
version: "1.0"
environment: "go"
icon_color: "#00ADD8"
setup: []
run:
  command: "go run main.go"
build: []
test:
  command: "go test ./..."
notes:
  markdown: "# Go Sandbox Environment"
  html: "<h3>Go</h3>"
files:
  - path: "main.go"
    content: "package main"
`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(legacyYAML), &root); err != nil {
		t.Fatal(err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		t.Fatal("expected document node")
	}
	mapping := root.Content[0]

	if got := legacyCommandFromNode(mapping, "run"); got != "go run main.go" {
		t.Fatalf("run = %q", got)
	}
	if got := legacyCommandFromNode(mapping, "build"); got != "" {
		t.Fatalf("build should be empty for `build: []`, got %q", got)
	}
	if got := legacyCommandFromNode(mapping, "test"); got != "go test ./..." {
		t.Fatalf("test = %q", got)
	}
}

func TestMigrateLegacyConfigs(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	legacyYAML := `name: "Go Standard Environment"
version: "1.0"
environment: "go"
icon_color: "#00ADD8"
setup: []
run:
  command: "go run main.go"
build: []
notes:
  markdown: "# Go Sandbox Environment"
  html: "<h3>Go</h3>"
files:
  - path: "main.go"
    content: "package main"
`

	if _, err := db.Exec(`CREATE TABLE workspaces (id TEXT PRIMARY KEY, name TEXT NOT NULL, config_yaml TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO workspaces (id, name, config_yaml) VALUES ('ws1', 'Golang Experiment', ?)", legacyYAML); err != nil {
		t.Fatal(err)
	}

	a := &App{db: db}
	if err := a.migrateLegacyWorkspaceConfigs(); err != nil {
		t.Fatal(err)
	}

	var cfgYML string
	if err := db.QueryRow("SELECT config_yaml FROM workspaces WHERE id = 'ws1'").Scan(&cfgYML); err != nil {
		t.Fatal(err)
	}

	var cfg DeclarativeConfig
	if err := yaml.Unmarshal([]byte(cfgYML), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Mappings.Run != "go run main.go" {
		t.Fatalf("migrated run mapping = %q", cfg.Mappings.Run)
	}
	if !strings.Contains(cfgYML, "mappings:") {
		t.Fatalf("config missing mappings key:\n%s", cfgYML)
	}
}
