Here is a professional and structured `README.md` file detailing the intent, design philosophy, and system mechanics of CrabCode.

---

# CrabCode

CrabCode is a lightweight desktop-based **Local Code Laboratory and Playbook** designed for rapid, hands-on programming experiments. 

Unlike standard general-purpose code editors or heavy IDEs, CrabCode functions as a personal "Obsidian vault" for code. It provides a structured space to write, run, and document hundreds of micro-projects—such as testing a network protocol, verifying a data structure, or playing with an API—without the storage bloat and configuration friction of isolated project setups.

---

## The Intent

When learning software engineering through first principles, the best practice is to build small, isolated experiments. However, creating hundreds of mini-projects locally leads to two major friction points:
1. **Dependency Bloat:** Having 50 small Node, Go, or Rust projects results in gigabytes of redundant `node_modules`, package caches, and compiler `target/` directories.
2. **Context Fragmentation:** Code implementation is often separated from the developer's notes, lessons learned, and run instructions.

CrabCode is designed to address these challenges with a three-part architecture:

* **The Code Playbook (File-System as Source of Truth):** Your experiments are organized as clean directories on your local drive. They remain completely portable, standard source files that you can sync, backup, or version-control with Git.
* **Shared Runtimes (Zero-Bloat execution):** Instead of each experiment managing its own dependencies, CrabCode utilizes single, shared runtimes per language on your machine. The app dynamically manages compiling and executing code from your playbook using these centralized runtimes.
* **Integrated Notes & Execution Console:** A side-by-side layout lets you code on the left while reading and writing markdown notes (`README.md`) on the right. An integrated terminal drawer lets you run your script and observe stdout/stderr instantly.

---

## Core Features

### 1. Side-by-Side Notes (Code + Context)
Every folder in your playbook acts as an entry in your personal programming catalog. 
* **The Split Pane:** Open any source file to write code on the left, and document your learnings, configurations, or research on the right in Markdown.
* **Auto-Resolution:** CrabCode automatically detects and binds the nearest companion `README.md` to your active workspace file, ensuring notes are never separated from code.

### 2. Zero-Friction Temporary Playground
Sometimes you need to test a quick regex, a standard library function, or a single-file concept without naming or dedicating a project directory to it.
* **The Scratchpad:** Access a global temporary workspace with support for Python, JavaScript, and Go.
* **Throwaway Sandboxing:** Write your code, run it instantly, and let it save to a hidden, global cache folder that keeps your primary workspace directory clean.

### 3. Integrated Process Execution
No need to switch to an external terminal.
* Spawns, monitors, and stops execution processes in the background through the native Go execution engine.
* Streams execution outputs and errors directly to a collapsible bottom console.

---

## System Architecture & Tech Stack

CrabCode is built as a local desktop application using:
* **Backend:** [Wails v2](https://wails.io/) (Go) for secure native filesystem access, OS-dialog integration, and native background process control.
* **Frontend:** [Svelte 5](https://svelte.dev/) for a highly responsive, lightweight UI.
* **Editor Engine:** [CodeMirror 6](https://codemirror.net/) configured with custom syntax highlighting, dark mode themes, keymaps, and undo/redo histories.

### How Runtimes are Managed
* **Go:** Runs directly against the local `go` compiler. Projects can inherit a shared root Go module or `go.work` space.
* **Python:** Executes code against your default python interpreter.
* **Node.js:** Resolves package dependencies recursively by walking up to the root folder of your playbook, allowing you to maintain one single parent `node_modules` directory for hundreds of experiments.

---

## Limitations & Scope

* **Local Only:** CrabCode runs entirely on your local machine. It does not contain telemetry, cloud-saving mechanics, or remote hosting capabilities. Your data is yours.
* **Not for Production:** This tool is designed purely as an experimental laboratory and documentation workbook. It is not intended for building, packaging, or deploying production-grade applications.

---

## Getting Started

### Prerequisites
* Go 1.21+
* Node.js & npm (for frontend building)
* [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Running in Development Mode
To start the application with hot-reloading enabled for both Go and the Svelte frontend, run:
```bash
wails dev
```
