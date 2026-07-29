# SQLite Notes System

CrabCode features a built-in Markdown notebook stored directly within your project workspace's SQLite database (`.crab/crab.db`).

## Features

### 1. SQLite Storage
Notes are stored inside the `markdown_notes` database table. All content is stored locally inside your project folder, making notes easy to backup or version control alongside `.crab/crab.db`.

### 2. Creating and Searching Notes
- Switch to the **Notes** tab in the top navigation.
- Click the **+** button in the sidebar header to create a new note.
- Use the **Search notes...** input box to filter notes dynamically by title.

### 3. Editor & Rendered Preview Modes
- Toggle between **Edit** and **Preview** modes using the header button.
- Edit mode provides full CodeMirror markdown editing.
- Preview mode renders titles (`#`, `##`), bullet points (`-`), and paragraphs.

### 4. Saving & Shortcuts
- Press `Ctrl+S` or click **Save Note** to commit changes to SQLite.
