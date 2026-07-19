# CrabCode

CrabCode is a lightweight, dark-mode desktop editor and laboratory designed for organizing development projects, maintaining rich learning journals, and conducting isolated scratchpad experiments. 

Built on Go, Svelte, and Wails, CrabCode features a dual-engine layout that combines local directory management with a highly portable, SQLite-backed workspace filesystem.

---

## Key Features

1. **Integrated Workspace Editor**: Standard file-tree manager for opening, editing, and executing files (Python, Go, Node.js, Rust, HTML, CSS, JSON) within your local physical directory.
2. **Obsidian-Style Notes (SQLite-Backed)**: Local markdown notes stored entirely inside a workspace SQLite database (`.crab/crab.db`). This allows notebooks to be easily transported, backed up, and versioned as single portable binary files.
3. **Database-Backed Sandboxes**: Virtual mini-environments saved directly inside the workspace SQLite database. Experiment without cluttering your local drive. Running a sandbox compiles and extracts virtual assets temporarily before execution.
4. **Customizable Split-Pane Layout**: Side-by-side editing mode. Toggle a split pane inside your main Workspace tab to edit or preview SQLite Markdown Notes, or work on virtual Sandbox files alongside your local physical code.
5. **Dynamic Root Bootstrap Configuration**: Solve storage constraints by configuring CrabCode to run entirely off an external hard drive. Setting a custom path routes settings, toolchain environments, and scratchpads outside your primary system disk.

---

## Architectural Layout

```
├── main.go                     # Wails application entry point
├── app.go                      # Core backend API (Go-Svelte bridge, database, processes)
└── frontend/
    ├── src/
    │   ├── App.svelte          # Reactive Svelte interface & CodeMirror integrations
    │   ├── FileNode.svelte     # Recursive component for directory tree rendering
    │   └── main.js             # Frontend bootloader
    └── wailsjs/                # Auto-generated Go-to-Frontend bindings
```

### Data Storage Architecture

When you select a physical folder as your active workspace, CrabCode instantiates a hidden directory named `.crab` inside it.

```
your-project-folder/
├── .crab/
│   ├── crab.db                 # SQLite database storing notes, sandboxes, and virtual files
│   └── temp_sandboxes/         # Temporary cache directories for compilation runs
├── main.go                     # Your physical project files
└── package.json
```

---

## SQLite Database Schema

CrabCode utilizes a pure Go, CGO-free SQLite driver (`modernc.org/sqlite`) to store notes and virtual project assets inside `.crab/crab.db` with the following schema:

```sql
-- Markdown Notes (Obsidian-Style)
CREATE TABLE IF NOT EXISTS markdown_notes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Sandboxes Meta
CREATE TABLE IF NOT EXISTS sandboxes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config_yaml TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Sandbox Virtual Filesystem
CREATE TABLE IF NOT EXISTS sandbox_files (
    id TEXT PRIMARY KEY,
    sandbox_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    is_dir INTEGER NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(sandbox_id) REFERENCES sandboxes(id) ON DELETE CASCADE,
    UNIQUE(sandbox_id, path)
);
```

---

## Prerequisites & Installation

To run or build CrabCode from source, ensure your machine satisfies the following conditions:

* **Go**: Version 1.20 or newer
* **Node.js & npm**: For Svelte frontend compilation
* **Wails CLI**: Installed via:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Development Environment Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/crabcode.git
   cd crabcode
   ```
2. **Retrieve Go Dependencies**:
   ```bash
   go mod tidy
   ```
3. **Run in development mode** (launches hot-reloading window):
   ```bash
   wails dev
   ```

### Building the Application

Compile a production-ready, optimized native binary for your active operating system:

```bash
wails build
```

---

## Configuration & Usage

### 1. YAML Configuration for Sandboxes

Each virtual sandbox contains an editable `config_yaml` block that defines how scripts are packaged and executed. Modify this YAML file inside the **Sandboxes** interface:

```yaml
name: "Data Visualization Experiment"
environment: "python"
run_command: "python3 main.py"
dependencies:
  - "pandas"
  - "matplotlib"
```

* **run_command**: Tells the CrabCode compiler which script or execution engine to trigger when the run button is pressed. This command is executed in the extracted temporary sandbox folder.

### 2. Global Data Directory (`~/.crabcode`)

CrabCode maintains a global root directory (`~/.crabcode` by default) separate from your per-workspace folders:

```
~/.crabcode/
├── settings.json              # Global configuration (root path, preferences)
├── environments/              # Reserved for future managed runtimes (currently empty)
└── playground/                # Scratchpad templates for quick experiments
```

#### The `environments/` Directory

The `environments/` folder is a **planned structural directory**. It exists to reserve a location for future toolchain management features (e.g., downloading and isolating Python virtualenvs, Go SDKs, or Node versions). It is **not populated during normal use**.

Sandbox execution does not use this folder. Instead, sandboxes operate as follows:

1. All sandbox code and configuration is stored **inside SQLite** (`.crab/crab.db`)
2. When you click **Run**, CrabCode extracts the sandbox's virtual files to `.crab/temp_sandboxes/<id>/`
3. The `run_command` (e.g., `python3 main.py`, `go run main.go`, `node index.js`) is executed using **whatever language runtimes are already installed on your system `$PATH`**
4. After execution, the temp directory is cleaned up

The `environments/` folder would only be populated if a future feature adds managed runtime installation — for example, "install Python 3.12 into `environments/` and use that instead of your system Python."

### 3. Relocating Root Settings (`.crabcode`) to an External Drive

To migrate configuration files and universal environments to an external hard drive (e.g., to free up primary disk space):

1. Go to the **Settings** tab.
2. Under **CrabCode System Base Location**, enter or browse to your desired target path on the external drive (e.g., `/Volumes/ExternalDrive/.crabcode` or `D:\.crabcode`).
3. Save configurations.

#### Bootstrap Mechanics
CrabCode handles directory routing by maintaining a lightweight pointer file `~/.crabcode_root.txt` inside your system home directory. 

* When CrabCode starts, it reads `~/.crabcode_root.txt` to find the database and active settings directory.
* If empty or missing, it defaults back to your user home directory (`~/.crabcode`).

---

## License

This project is licensed under the MIT License - see the LICENSE file for details.
