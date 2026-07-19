Let's reimplement and redesign the app.

Add tabs to the top. 
I want the app to be divided into 1. Code Editor for the selected workspace 2. Markdown Notes like obsidian  for this workspace 3. Sandboxes, mini projects that are using the same dependencies and are switched when needed.

Markdown will no longer be always on the right side of the code editor. However, it supports split pane if we need to code and read at the same time. The active markdown in the markdown tab will be shown and the active file code will be shown. This applies the same for sandboxes.

Let's say I have a project called project1. It is a golang project. I am coding and I want to experiment. So I got to the Sandboxes and create a mini sandbox to perform experiments and create a bunch of principles I needed.

### Where does the data of sandboxes be stored?
The files, contents, and structure of the sandboxes will be stored in a sqlite file. Which will be stored in the same chosen directory.
I am thinking of maybe creating a dedicated .crab folder to store the data. This will store the markdown and the sandboxes data. 

This means that for every workspace, there are is a completely different data. 

Markdowns are also stored in sqlite. The reason for this is because sqlite file is easy to transport and create backups since it is a single file. Markdown files when gets larger, gets really hard to backup because moving a file needs a memory.

### YAML Configuration
We will use YAML to tell the system how to setup or choose a sandbox. This way the system becomes super flexible. Just like how vscode, nodejs can be configured in many ways.

### Universal Environments
Somewhere in the computer, the system uses a directory to store its environments and dependencies. Create a settings page where we can change and see this location.

