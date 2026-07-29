# Global Settings & Environments

The **Settings** tab configures system-wide storage, environmental toolchains, and scratchpad defaults.

## Directory Layout

```
~/.crabcode/                     (Default System Root)
├── settings.json                 (Root & environment paths)
├── crabcode.db                   (Global SQLite database)
├── templates/                    (Dynamic IaC template files)
│   ├── python.yaml
│   ├── go.yaml
│   ├── rust.yaml
│   ├── java.yaml
│   ├── javascript.yaml
│   ├── typescript.yaml
│   ├── sql.yaml
│   └── surrealdb.yaml
├── sandboxes/                    (Persisted sandbox files on disk)
│   └── <sandbox_id>/
│       ├── config.yaml
│       ├── main.py
│       └── ...
├── environments/                 (Universal package toolchains)
├── temp_sandboxes/               (Extracted runtime execution folder)
│   └── <sandbox_id>/
└── playground/                   (Global code scratchpads)
    ├── scratch.py
    ├── scratch.js
    ├── scratch.go
    ├── scratch.rs
    ├── scratch.java
    ├── scratch.ts
    ├── scratch.sql
    └── scratch.dart
```

## Configuration Options

### 1. CrabCode Data Folder (.crabcode)
- Default path: `~/.crabcode`
- **Custom Bootstrap Target**: Specify a custom storage directory (e.g., an external drive or custom folder). CrabCode persists custom paths to `~/.crabcode_root.txt`.
- **Initialization**: If the specified path is empty or unpopulated, click **Initialize Folder Structure** to auto-generate default settings, template files, playground files, and environment directories.

### 2. Templates Directory
- Path: `<CrabCode Root>/templates/`
- Contains `.yaml` files that define sandbox environment configurations in Declarative IaC format.
- **No hardcoded templates**: Adding a new `.yaml` file here immediately makes it available in the **Create Virtual Sandbox** UI — no recompilation required.
- See [Virtual Sandboxes Guide](sandboxes.md#dynamic-file-based-template-system) for the full specification.

### 3. Universal Environments Directory
- Path: `<CrabCode Root>/environments`
- Setup steps in IaC configs execute inside subdirectories of this folder (e.g., `environments/python/`, `environments/postgres/`).
- Package toolchains and dependencies managed by CrabCode reside here.

### 4. Global Playground Scratchpads
Upon startup or initialization, CrabCode builds a global `playground/` folder inside your CrabCode root path populated with multi-language scratchpad starter files:
- `scratch.py`, `scratch.js`, `scratch.go`, `scratch.rs`, `scratch.java`, `scratch.ts`, `scratch.sql`, `scratch.dart`, `scratch.surql`
