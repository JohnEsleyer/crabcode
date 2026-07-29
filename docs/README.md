# CrabCode Documentation

Welcome to the CrabCode documentation. CrabCode is a modern desktop development workspace built with Go, Wails, Svelte, and CodeMirror. It combines traditional file-tree editing with SQLite-backed Markdown notes and isolated virtual sandboxes.

## Documentation Index

1. [Workspace & Code Editor](workspace-and-editor.md)
   - Opening project folders and file management
   - Syntax highlighting and shortcuts
   - Executing single files and split-pane views
2. [SQLite Notes System](notes-system.md)
   - Creating and searching markdown notebooks
   - Markdown preview mode
   - Split view integration
3. [Virtual Sandboxes](sandboxes.md)
   - Declarative Infrastructure-as-Code (IaC) YAML format
   - Setup, build, and run lifecycle phases
   - Dynamic file-based template system (no hardcoded templates)
   - Virtual file management inside SQLite
   - Interactive HTML visual canvas & Markdown notes
   - AI Assistant Prompt workflows
4. [Global Settings & Environments](global-settings.md)
   - System root folder (`.crabcode`)
   - Dynamic templates directory (`templates/`)
   - Environment toolchains
   - Global playground scratchpads

---

## Quick Start

1. Launch CrabCode.
2. Click **Select Workspace** in the top navigation header and choose your project directory.
3. CrabCode automatically initializes a local SQLite database at `.crab/crab.db` inside your workspace folder.
4. Create files in the workspace explorer or switch to **Notes** / **Sandboxes** to build isolated experiments and record project documentation.
