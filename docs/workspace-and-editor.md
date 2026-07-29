# Workspace & Code Editor Guide

The **Workspace** tab serves as the primary file editing and code execution view in CrabCode.

## Key Features

### 1. Workspace Folder Selection
- Click **Select Workspace** in the top bar to open a native OS file dialog.
- Choosing a directory loads the file tree and initializes/connects to `.crab/crab.db`.

### 2. File Explorer Operations
In the left sidebar under Workspace mode:
- **New File**: Click the `+ File` button to create a new file in the root workspace folder.
- **New Folder**: Click the `+ Folder` button to create a directory.
- **Tree Navigation**: Click on folder names to toggle expansion. Click on any file to open it in the primary editor.
- **Unsaved Indicator**: Unsaved file edits are indicated with an orange dot next to the file name.

### 3. Code Mirror Editor Features
- **Keybindings**: Use `Ctrl+S` (or `Cmd+S` on macOS) to save the current file. Use `Tab` / `Shift+Tab` for indentation.
- **Syntax Highlighting**: Supports automatic language detection for Python, JavaScript, TypeScript, Go, Rust, C++, Java, HTML, CSS, JSON, and YAML.
- **Status Bar**: Displays line count, character count, and active language tag in the bottom right corner.

### 4. Running Code
Click the **Run** button in the header toolbar to execute the currently open file:
- **Python** (`.py`): Executed with `python3`
- **JavaScript** (`.js`): Executed with `node`
- **Go** (`.go`): Executed with `go run`
- **Java** (`.java`): Executed with `java`
- **TypeScript** (`.ts`): Executed with `npx tsx`
- **Dart** (`.dart`): Executed with `dart run`
- **SQL** (`.sql`): Executed with `sqlite3`

Output streams in real-time to the **Console Output** panel at the bottom of the editor. Click **Stop** to interrupt execution.

### 5. Split-Pane Mode
Click the **Split** button in the top action header to open a side-by-side pane:
- **Notes Mode**: Load and edit project markdown notes without switching away from your active code file.
- **Sandbox Mode**: Preview and edit virtual sandbox files side-by-side.
