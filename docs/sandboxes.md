# Virtual Sandboxes Guide

CrabCode Virtual Sandboxes allow you to build, document, and execute isolated mini-projects entirely inside your workspace SQLite database using a **Declarative Infrastructure-as-Code (IaC)** approach.

## How Sandboxes Work

1. **Virtual Filesystem**: All sandbox files are virtual entries stored in the `sandbox_files` SQLite table.
2. **IaC Execution Lifecycle**:
   - **Setup Phase**: CrabCode executes `setup` steps (defined in the IaC config) inside `<CrabRoot>/environments/` to prepare toolchains.
   - **Extraction**: All virtual files are extracted into `.crab/temp_sandboxes/<sandbox_id>/`.
   - **Build Phase**: `build` steps (compilation checks, etc.) run in the temp directory.
   - **Run Phase**: The `run.command` is executed with configured `env_vars`.
   - Terminal output streams live to the Sandbox Console drawer.

---

## Declarative IaC YAML Format

Every sandbox uses a Terraform-inspired YAML spec with four lifecycle phases:

```yaml
name: "Python Data Science Environment"
version: "1.0"
environment: "python"
icon_color: "#3572A5"

# 1. SETUP — executed inside <CrabRoot>/environments/
setup:
  - name: "Initialize Python Virtual Environment"
    command: "python3 -m venv env"
    dir: "environments/python"
  - name: "Install Dependencies"
    command: "env/bin/pip install requests pandas"
    dir: "environments/python"

# 2. ENVIRONMENT VARIABLES
env_vars:
  PYTHONUNBUFFERED: "1"
  CRABCODE_ENV: "sandbox"

# 3. VIRTUAL FILES injected into the sandbox
files:
  - path: "main.py"
    content: "print('Hello from IaC Sandbox!')"

# 4. BUILD — compilation or pre-run checks
build:
  - name: "Compile Check"
    command: "python3 -m py_compile main.py"

# 5. RUN — the main execution command
run:
  command: "python3 main.py"

# 6. NOTES — documentation embedded in the sandbox
notes:
  markdown: "# Python Environment\n\nExperimental data science workspace."
  html: "<h3>Python Sandbox Active</h3>"
```

You can customize `run.command` to include shell operators (`&&`, `;`, `|`) or custom compilation scripts.

---

## Dynamic File-Based Template System

Templates are **no longer hardcoded** in the Go source. Instead, CrabCode scans `<CrabRoot>/templates/*.yaml` at runtime.

### How It Works

1. On startup, CrabCode initializes `~/.crabcode/templates/` with default `.yaml` template files.
2. When you open the **Create Virtual Sandbox** modal, CrabCode calls `GetTemplates()` which reads every `.yaml` file in that directory.
3. Each template dynamically populates the UI grid — no recompilation needed.

### Adding a Custom Template

Simply drop a new `.yaml` file into the templates folder:

```bash
# Example: ~/.crabcode/templates/docker_postgres.yaml
```

See the [Custom Template Example](#custom-template-example-docker-postgresql) below for a full walkthrough.

### Default Templates

CrabCode ships with these pre-built templates in `~/.crabcode/templates/`:

| File | Environment |
|---|---|
| `python.yaml` | Python 3 |
| `go.yaml` | Go |
| `rust.yaml` | Rust |
| `javascript.yaml` | Node.js |
| `typescript.yaml` | TypeScript (tsx) |
| `sql.yaml` | SQLite |
| `surrealdb.yaml` | SurrealDB |

---

## Interactive HTML Canvas & Markdown Notes

In the sandbox split pane, switch between:
1. **Markdown Note**: Document the purpose, parameters, and results of your sandbox experiment.
2. **HTML Canvas**: Write HTML/CSS/JavaScript that renders live in an isolated preview iframe. Perfect for dynamic data visualizations, concepts, or UI prototypes.
3. **YAML Config**: Edit the IaC execution parameters directly.

---

## Generating Sandboxes with AI

CrabCode includes a prompt template helper to easily copy a prompt for AI assistants (ChatGPT, Claude, etc.):
1. Click the **Guide** icon (`?`) in the Sandboxes header.
2. Click **Copy** in the AI Prompt Template box.
3. Paste the prompt into your preferred AI model to generate compatible sandbox file structures and YAML configs.

---

## Custom Template Example: Docker PostgreSQL

Create `~/.crabcode/templates/docker_postgres.yaml`:

```yaml
name: "Docker PostgreSQL Environment"
version: "1.0"
environment: "postgres"
icon_color: "#336791"

setup:
  - name: "Pull PostgreSQL Docker Container"
    command: "docker pull postgres:alpine"
    dir: "environments/postgres"

env_vars:
  PGUSER: "postgres"
  PGPASSWORD: "secretpassword"

files:
  - path: "schema.sql"
    content: |
      CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(50));
      INSERT INTO users (name) VALUES ('CrabCode IaC Developer');
      SELECT * FROM users;

build: []

run:
  command: "docker run --rm -i -e POSTGRES_PASSWORD=$PGPASSWORD postgres:alpine psql -U postgres -d postgres < schema.sql"

notes:
  markdown: "# Docker PostgreSQL IaC Environment\n\nRuns containerized PostgreSQL queries."
  html: "<h3>🐘 Docker PostgreSQL Ready</h3>"
```

CrabCode will detect this file automatically and display **Docker PostgreSQL Environment** in the template picker the next time you create a sandbox.
