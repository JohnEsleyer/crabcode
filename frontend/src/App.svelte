<script>
  import './style.css';
  import './app.css';
  import {
    SelectFolder,
    ListDirectory,
    ReadFile,
    SaveFile,
    CreateFile,
    CreateDirectory,
    DeletePath,
    RenamePath,
    OpenWorkspace,
    GetNotes,
    CreateNote,
    SaveNote,
    DeleteNote,
    GetSandboxes,
    CreateSandbox,
    DeleteSandbox,
    SaveSandboxConfig,
    GetSandboxFiles,
    SaveSandboxFile,
    DeleteSandboxFile,
    RunSandbox,
    RunCommand,
    StopCommand,
    GetGlobalSettings,
    SaveGlobalSettings,
    IsDirectoryEmpty,
    InitializeCrabFolder
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import FileNode from './FileNode.svelte';
  import { onMount, onDestroy } from 'svelte';
  import {
    FolderOpen, FilePlus, FolderPlus, Play, Square,
    Terminal, FileText, Settings, Database, BookOpen, Plus, Trash2, Columns, Check
  } from '@lucide/svelte';

  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState, Compartment } from '@codemirror/state';
  import { keymap } from '@codemirror/view';
  import { indentWithTab, undo, redo, undoDepth, redoDepth } from '@codemirror/commands';
  import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
  import { tags as t } from '@lezer/highlight';

  import { javascript } from '@codemirror/lang-javascript';
  import { python } from '@codemirror/lang-python';
  import { go } from '@codemirror/lang-go';
  import { rust } from '@codemirror/lang-rust';
  import { html } from '@codemirror/lang-html';
  import { css } from '@codemirror/lang-css';
  import { json } from '@codemirror/lang-json';

  let activeTab = $state('workspace');

  let currentFolder = $state('');
  let fileTree = $state([]);
  let expandedFolders = $state({});
  let folderContents = $state({});
  let activeFilePath = $state('');
  let activeFileName = $state('');
  let editorContent = $state('');
  let lastSavedContent = $state('');
  let isSaving = $state(false);
  let statusMessage = $state('Ready');

  let isSplitPane = $state(false);
  let splitPaneType = $state('notes');

  let notesList = $state([]);
  let activeNoteId = $state('');
  let activeNoteTitle = $state('');
  let activeNoteContent = $state('');
  let lastSavedNoteContent = $state('');
  let noteSearchQuery = $state('');
  let isMarkdownEditing = $state(true);

  let sandboxesList = $state([]);
  let activeSandboxId = $state('');
  let activeSandboxName = $state('');
  let activeSandboxConfig = $state('');
  let lastSavedSandboxConfig = $state('');
  let sandboxFilesList = $state([]);
  let activeSandboxFilePath = $state('');
  let activeSandboxFileName = $state('');
  let activeSandboxFileContent = $state('');
  let lastSavedSandboxFileContent = $state('');
  let sandboxTabMode = $state('code');

  let settingsCrabRootPath = $state('');
  let isCrabFolderEmpty = $state(false);

  let toasts = $state([]);
  let modal = $state({ show: false, title: '', placeholder: '', value: '', onConfirm: null, onCancel: null });

  let workspaceEditorContainer = $state(null);
  let notesEditorContainer = $state(null);
  let sandboxEditorContainer = $state(null);
  let workspaceView = null;
  let notesView = null;
  let sandboxView = null;
  const workspaceLanguageConf = new Compartment();
  const sandboxLanguageConf = new Compartment();

  let canUndo = $state(false);
  let canRedo = $state(false);
  let consoleLogs = $state([]);
  let isRunning = $state(false);
  let isConsoleOpen = $state(true);
  let runningProcessId = $state('');
  const processTracker = { id: '' };

  let lineCount = $derived(editorContent.split('\n').length);
  let charCount = $derived(editorContent.length);
  let fileExtension = $derived(
    activeFileName ? (activeFileName.split('.').pop() || '').toUpperCase() : 'TEXT'
  );
  let activeFileUnsaved = $derived(activeFilePath !== '' && editorContent !== lastSavedContent);
  let activeNoteUnsaved = $derived(activeNoteId !== '' && activeNoteContent !== lastSavedNoteContent);
  let activeSandboxFileUnsaved = $derived(
    activeSandboxFilePath !== '' && activeSandboxFileContent !== lastSavedSandboxFileContent
  );
  let activeSandboxConfigUnsaved = $derived(
    activeSandboxId !== '' && activeSandboxConfig !== lastSavedSandboxConfig
  );
  let filteredNotes = $derived(
    notesList.filter(n => n.title.toLowerCase().includes(noteSearchQuery.toLowerCase()))
  );

  const crabCodeTheme = EditorView.theme({
    '&': { color: '#edf2f7', backgroundColor: '#14141a', height: '100%', width: '100%' },
    '.cm-content': {
      caretColor: '#ff5a36',
      fontFamily: "'Fira Code', 'JetBrains Mono', monospace",
      fontSize: '13px',
      padding: '16px 0'
    },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#ff5a36', borderLeftWidth: '2px' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
      backgroundColor: '#ff5a3622 !important'
    },
    '.cm-gutters': {
      backgroundColor: '#14141a',
      color: '#6b7280',
      borderRight: '1px solid #20202b',
      fontSize: '13px',
      paddingTop: '16px',
      paddingBottom: '16px'
    },
    '.cm-gutterElement': { padding: '0 12px 0 16px' },
    '.cm-activeLine': { backgroundColor: '#ffffff03' },
    '.cm-activeLineGutter': { backgroundColor: '#ffffff03', color: '#edf2f7' }
  }, { dark: true });

  const crabHighlightStyle = HighlightStyle.define([
    { tag: t.keyword, color: '#569CD6', fontWeight: '600' },
    { tag: t.comment, color: '#6A9955', fontStyle: 'italic' },
    { tag: t.string, color: '#CE9178' },
    { tag: t.number, color: '#B5CEA8' },
    { tag: t.className, color: '#4EC9B0' },
    { tag: t.typeName, color: '#4EC9B0' },
    { tag: t.function(t.variableName), color: '#DCDCAA' },
    { tag: t.definition(t.variableName), color: '#9CDCFE' },
    { tag: t.variableName, color: '#9CDCFE' },
    { tag: t.operator, color: '#D4D4D4' },
    { tag: t.propertyName, color: '#9CDCFE' },
    { tag: t.heading, color: '#ff5a36', fontWeight: 'bold' }
  ]);

  function getLanguageExtension(fileName) {
    if (!fileName) return [];
    const ext = (fileName.split('.').pop() || '').toLowerCase();
    switch (ext) {
      case 'js': case 'ts': case 'jsx': case 'tsx': return javascript();
      case 'py': return python();
      case 'go': return go();
      case 'rs': return rust();
      case 'html': case 'svelte': return html();
      case 'css': return css();
      case 'json': return json();
      case 'yaml': case 'yml': return [];
      default: return [];
    }
  }

  function buildWorkspaceExtensions(fileName) {
    return [
      basicSetup,
      keymap.of([indentWithTab]),
      crabCodeTheme,
      syntaxHighlighting(crabHighlightStyle),
      workspaceLanguageConf.of(getLanguageExtension(fileName)),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          editorContent = update.state.doc.toString();
          canUndo = undoDepth(update.state) > 0;
          canRedo = redoDepth(update.state) > 0;
        }
      })
    ];
  }

  function buildNotesExtensions() {
    return [
      basicSetup,
      keymap.of([indentWithTab]),
      crabCodeTheme,
      syntaxHighlighting(crabHighlightStyle),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          activeNoteContent = update.state.doc.toString();
        }
      })
    ];
  }

  function buildSandboxExtensions(fileName) {
    return [
      basicSetup,
      keymap.of([indentWithTab]),
      crabCodeTheme,
      syntaxHighlighting(crabHighlightStyle),
      sandboxLanguageConf.of(getLanguageExtension(fileName)),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          activeSandboxFileContent = update.state.doc.toString();
        }
      })
    ];
  }

  function updateWorkspaceEditor(content, fileName) {
    if (!workspaceView) return;
    workspaceView.setState(EditorState.create({
      doc: content,
      extensions: buildWorkspaceExtensions(fileName)
    }));
    canUndo = false;
    canRedo = false;
  }

  function updateNotesEditor(content) {
    if (!notesView) return;
    notesView.setState(EditorState.create({
      doc: content,
      extensions: buildNotesExtensions()
    }));
  }

  function updateSandboxEditor(content, fileName) {
    if (!sandboxView) return;
    sandboxView.setState(EditorState.create({
      doc: content,
      extensions: buildSandboxExtensions(fileName)
    }));
  }

  $effect(() => {
    if (workspaceEditorContainer && !workspaceView) {
      workspaceView = new EditorView({
        parent: workspaceEditorContainer,
        state: EditorState.create({
          doc: editorContent,
          extensions: buildWorkspaceExtensions(activeFileName)
        })
      });
    } else if (!workspaceEditorContainer && workspaceView) {
      workspaceView.destroy();
      workspaceView = null;
    }
  });

  $effect(() => {
    if (notesEditorContainer && !notesView) {
      notesView = new EditorView({
        parent: notesEditorContainer,
        state: EditorState.create({
          doc: activeNoteContent,
          extensions: buildNotesExtensions()
        })
      });
    } else if (!notesEditorContainer && notesView) {
      notesView.destroy();
      notesView = null;
    }
  });

  $effect(() => {
    if (sandboxEditorContainer && !sandboxView) {
      sandboxView = new EditorView({
        parent: sandboxEditorContainer,
        state: EditorState.create({
          doc: activeSandboxFileContent,
          extensions: buildSandboxExtensions(activeSandboxFileName)
        })
      });
    } else if (!sandboxEditorContainer && sandboxView) {
      sandboxView.destroy();
      sandboxView = null;
    }
  });

  $effect(() => {
    if (modal.show) {
      setTimeout(() => {
        const el = document.querySelector('.modal-input');
        if (el) el.focus();
      }, 50);
    }
  });

  let toastId = 0;
  function addToast(message, type = 'info', duration = 3000) {
    const id = toastId++;
    toasts = [...toasts, { id, message, type }];
    setTimeout(() => {
      toasts = toasts.filter(item => item.id !== id);
    }, duration);
  }

  function openPrompt(title, placeholder, defaultValue = '') {
    return new Promise((resolve) => {
      modal = {
        show: true,
        title,
        placeholder,
        value: defaultValue,
        onConfirm: (val) => {
          modal.show = false;
          resolve(val);
        },
        onCancel: () => {
          modal.show = false;
          resolve(null);
        }
      };
    });
  }

  function sortNodes(nodes) {
    nodes.sort((a, b) => {
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;
      return a.name.localeCompare(b.name);
    });
  }

  function getParentPath(path) {
    const cleanPath = path.replace(/[\\/]$/, '');
    const lastIdx = Math.max(cleanPath.lastIndexOf('/'), cleanPath.lastIndexOf('\\'));
    if (lastIdx === -1) return '';
    return cleanPath.substring(0, lastIdx);
  }

  async function refreshDirectory(dirPath) {
    const contents = await ListDirectory(dirPath);
    sortNodes(contents);
    if (dirPath === currentFolder) {
      fileTree = contents;
    } else {
      folderContents[dirPath] = contents;
    }
  }

  async function chooseFolder() {
    try {
      const folder = await SelectFolder();
      if (folder) {
        currentFolder = folder;
        const workspaceInfo = await OpenWorkspace(folder);
        notesList = workspaceInfo.notes || [];
        sandboxesList = workspaceInfo.sandboxes || [];
        const contents = await ListDirectory(folder);
        sortNodes(contents);
        fileTree = contents;
        expandedFolders = {};
        folderContents = {};
        activeFilePath = '';
        activeFileName = '';
        editorContent = '';
        lastSavedContent = '';
        statusMessage = 'Opened project: ' + folder;
        addToast('Workspace loaded successfully', 'success');
      }
    } catch (err) {
      statusMessage = 'Workspace error: ' + err;
      addToast(String(err), 'error');
    }
  }

  async function openFile(node) {
    try {
      const content = await ReadFile(node.path);
      activeFilePath = node.path;
      activeFileName = node.name;
      editorContent = content;
      lastSavedContent = content;
      updateWorkspaceEditor(content, node.name);
      statusMessage = 'Opened: ' + node.name;
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function saveWorkspaceFile() {
    if (!activeFilePath) return;
    isSaving = true;
    try {
      await SaveFile(activeFilePath, editorContent);
      lastSavedContent = editorContent;
      statusMessage = 'Saved: ' + activeFileName;
      addToast('File saved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    } finally {
      isSaving = false;
    }
  }

  async function createFile() {
    if (!currentFolder) return;
    const name = await openPrompt('Create File', 'main.go');
    if (!name) return;
    try {
      await CreateFile(currentFolder + '/' + name);
      await refreshDirectory(currentFolder);
      addToast('File created', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function createFolder() {
    if (!currentFolder) return;
    const name = await openPrompt('Create Folder', 'pkg');
    if (!name) return;
    try {
      await CreateDirectory(currentFolder + '/' + name);
      await refreshDirectory(currentFolder);
      addToast('Folder created', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function toggleFolder(path) {
    if (expandedFolders[path]) {
      expandedFolders[path] = false;
    } else {
      try {
        const contents = await ListDirectory(path);
        sortNodes(contents);
        folderContents[path] = contents;
        expandedFolders[path] = true;
      } catch (err) {
        addToast(String(err), 'error');
      }
    }
  }

  async function selectNote(note) {
    activeNoteId = note.id;
    activeNoteTitle = note.title;
    activeNoteContent = note.content;
    lastSavedNoteContent = note.content;
    isMarkdownEditing = true;
    updateNotesEditor(note.content);
  }

  async function handleCreateNote() {
    if (!currentFolder) {
      addToast('Select a workspace first', 'error');
      return;
    }
    const name = await openPrompt('New Note Title', 'Algorithms Design');
    if (!name) return;
    try {
      const created = await CreateNote(name);
      notesList = [created, ...notesList];
      await selectNote(created);
      addToast('Note created', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleSaveNote() {
    if (!activeNoteId) return;
    try {
      await SaveNote(activeNoteId, activeNoteTitle, activeNoteContent);
      lastSavedNoteContent = activeNoteContent;
      const index = notesList.findIndex(n => n.id === activeNoteId);
      if (index !== -1) {
        notesList[index].title = activeNoteTitle;
        notesList[index].content = activeNoteContent;
      }
      addToast('Note saved to SQLite', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDeleteNote(id) {
    if (!confirm('Delete this note permanently?')) return;
    try {
      await DeleteNote(id);
      notesList = notesList.filter(n => n.id !== id);
      if (activeNoteId === id) {
        activeNoteId = '';
        activeNoteTitle = '';
        activeNoteContent = '';
        lastSavedNoteContent = '';
      }
      addToast('Note deleted', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleCreateSandbox() {
    if (!currentFolder) {
      addToast('Select a workspace first', 'error');
      return;
    }
    const name = await openPrompt('Sandbox Project Name', 'Experiment 1');
    if (!name) return;
    try {
      const sandbox = await CreateSandbox(name, '');
      sandboxesList = [sandbox, ...sandboxesList];
      await selectSandbox(sandbox);
      addToast('Sandbox initialized', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function selectSandbox(sandbox) {
    activeSandboxId = sandbox.id;
    activeSandboxName = sandbox.name;
    activeSandboxConfig = sandbox.configYaml;
    lastSavedSandboxConfig = sandbox.configYaml;
    sandboxTabMode = 'code';
    await loadSandboxFiles();
  }

  async function loadSandboxFiles() {
    try {
      const files = await GetSandboxFiles(activeSandboxId);
      sandboxFilesList = files.filter(f => !f.isDir);
      if (sandboxFilesList.length > 0) {
        const firstFile = sandboxFilesList[0];
        activeSandboxFilePath = firstFile.path;
        activeSandboxFileName = firstFile.path.split('/').pop() || firstFile.path;
        activeSandboxFileContent = firstFile.content;
        lastSavedSandboxFileContent = firstFile.content;
        updateSandboxEditor(firstFile.content, activeSandboxFileName);
      } else {
        activeSandboxFilePath = '';
        activeSandboxFileName = '';
        activeSandboxFileContent = '';
        lastSavedSandboxFileContent = '';
      }
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function selectSandboxFile(file) {
    activeSandboxFilePath = file.path;
    activeSandboxFileName = file.path.split('/').pop() || file.path;
    activeSandboxFileContent = file.content;
    lastSavedSandboxFileContent = file.content;
    updateSandboxEditor(file.content, activeSandboxFileName);
  }

  async function handleCreateSandboxFile() {
    if (!activeSandboxId) return;
    const name = await openPrompt('New Sandbox File', 'main.py');
    if (!name) return;
    try {
      await SaveSandboxFile(activeSandboxId, name, '# virtual environment file\n', false);
      await loadSandboxFiles();
      const created = sandboxFilesList.find(f => f.path === name);
      if (created) await selectSandboxFile(created);
      addToast('Sandbox file added', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDeleteSandboxFile(path) {
    if (!activeSandboxId || !confirm('Delete this virtual file?')) return;
    try {
      await DeleteSandboxFile(activeSandboxId, path);
      if (activeSandboxFilePath === path) {
        activeSandboxFilePath = '';
        activeSandboxFileName = '';
        activeSandboxFileContent = '';
        lastSavedSandboxFileContent = '';
      }
      await loadSandboxFiles();
      addToast('Sandbox file deleted', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleSaveSandbox() {
    if (!activeSandboxId) return;
    try {
      if (sandboxTabMode === 'yaml') {
        await SaveSandboxConfig(activeSandboxId, activeSandboxConfig);
        lastSavedSandboxConfig = activeSandboxConfig;
        const index = sandboxesList.findIndex(s => s.id === activeSandboxId);
        if (index !== -1) {
          sandboxesList[index].configYaml = activeSandboxConfig;
        }
      } else if (activeSandboxFilePath) {
        await SaveSandboxFile(activeSandboxId, activeSandboxFilePath, activeSandboxFileContent, false);
        lastSavedSandboxFileContent = activeSandboxFileContent;
        await loadSandboxFiles();
      }
      addToast('Sandbox saved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDeleteSandbox(id) {
    if (!confirm('Delete this sandbox project permanently?')) return;
    try {
      await DeleteSandbox(id);
      sandboxesList = sandboxesList.filter(s => s.id !== id);
      if (activeSandboxId === id) {
        activeSandboxId = '';
        activeSandboxName = '';
        activeSandboxConfig = '';
        lastSavedSandboxConfig = '';
        sandboxFilesList = [];
        activeSandboxFilePath = '';
        activeSandboxFileName = '';
        activeSandboxFileContent = '';
        lastSavedSandboxFileContent = '';
      }
      addToast('Sandbox deleted', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function runActiveCode() {
    if (activeTab === 'workspace') {
      if (!activeFilePath) return;
      if (activeFileUnsaved) await saveWorkspaceFile();
      consoleLogs = [`[Running ${activeFileName}...]`];
      isRunning = true;
      isConsoleOpen = true;
      runningProcessId = activeFilePath;
      processTracker.id = activeFilePath;

      const ext = activeFileName.split('.').pop().toLowerCase();
      let runnerCmd = '';
      let runnerArgs = [];

      if (ext === 'py') {
        runnerCmd = 'python3';
        runnerArgs = [activeFilePath];
      } else if (ext === 'js') {
        runnerCmd = 'node';
        runnerArgs = [activeFilePath];
      } else if (ext === 'go') {
        runnerCmd = 'go';
        runnerArgs = ['run', activeFilePath];
      } else {
        consoleLogs = [...consoleLogs, `[Runner Error] No run configuration for ".${ext}" files.`];
        isRunning = false;
        runningProcessId = '';
        processTracker.id = '';
        return;
      }

      try {
        await RunCommand(activeFilePath, runnerCmd, runnerArgs, getParentPath(activeFilePath) || currentFolder);
      } catch (err) {
        consoleLogs = [...consoleLogs, `[Error] ${String(err)}`];
        isRunning = false;
        runningProcessId = '';
        processTracker.id = '';
      }
    } else if (activeTab === 'sandboxes') {
      if (!activeSandboxId) return;
      consoleLogs = [`[Extracting and executing sandbox: ${activeSandboxName}...]`];
      isRunning = true;
      isConsoleOpen = true;
      try {
        await handleSaveSandbox();
        const processId = await RunSandbox(activeSandboxId, activeSandboxFilePath);
        runningProcessId = processId;
        processTracker.id = processId;
      } catch (err) {
        consoleLogs = [...consoleLogs, `[Execution Error] ${String(err)}`];
        isRunning = false;
        runningProcessId = '';
        processTracker.id = '';
      }
    }
  }

  async function stopActiveProcess() {
    if (!runningProcessId) return;
    try {
      await StopCommand(runningProcessId);
      isRunning = false;
      consoleLogs = [...consoleLogs, '[Process stopped manually]'];
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function checkFolderState() {
    if (!settingsCrabRootPath) {
      isCrabFolderEmpty = true;
      return;
    }
    try {
      isCrabFolderEmpty = await IsDirectoryEmpty(settingsCrabRootPath);
    } catch (err) {
      isCrabFolderEmpty = true;
    }
  }

  async function initializeFolderStructure() {
    if (!settingsCrabRootPath) return;
    try {
      await InitializeCrabFolder(settingsCrabRootPath);
      isCrabFolderEmpty = false;
      addToast('CrabCode data folder initialized', 'success');
      await loadGlobalConfig();
    } catch (err) {
      addToast('Initialization failed: ' + String(err), 'error');
    }
  }

  async function loadGlobalConfig() {
    try {
      const settings = await GetGlobalSettings();
      settingsCrabRootPath = settings.crabRootPath;
      await checkFolderState();
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function saveGlobalConfig() {
    try {
      await SaveGlobalSettings({
        crabRootPath: settingsCrabRootPath,
        universalEnvDir: settingsCrabRootPath + '/environments'
      });
      addToast('Global settings saved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function browseCrabRootPath() {
    try {
      const dir = await SelectFolder();
      if (dir) {
        settingsCrabRootPath = dir;
        await checkFolderState();
      }
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  function scrollToConsoleBottom() {
    setTimeout(() => {
      const container = document.getElementById('console-output-view');
      if (container) container.scrollTop = container.scrollHeight;
    }, 40);
  }

  function handleKeyDown(event) {
    const isMeta = event.ctrlKey || event.metaKey;
    if (isMeta && event.key === 's') {
      event.preventDefault();
      if (activeTab === 'workspace') saveWorkspaceFile();
      else if (activeTab === 'notes') handleSaveNote();
      else if (activeTab === 'sandboxes') handleSaveSandbox();
    }
  }

  function renderMarkdownLine(line) {
    if (line.startsWith('# ')) return { type: 'h1', text: line.slice(2) };
    if (line.startsWith('## ')) return { type: 'h2', text: line.slice(3) };
    if (line.startsWith('- ')) return { type: 'li', text: line.slice(2) };
    if (line.trim() === '') return { type: 'br', text: '' };
    return { type: 'p', text: line };
  }

  onMount(() => {
    loadGlobalConfig();
    window.addEventListener('keydown', handleKeyDown);

    EventsOn('terminal_output', (data) => {
      if (data.id === processTracker.id) {
        consoleLogs = [...consoleLogs, data.line];
        scrollToConsoleBottom();
      }
    });

    EventsOn('terminal_status', (data) => {
      if (data.id === processTracker.id) {
        isRunning = false;
        consoleLogs = [...consoleLogs, `[Process exited: ${data.status}]`];
        scrollToConsoleBottom();
      }
    });

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });

  onDestroy(() => {
    if (workspaceView) workspaceView.destroy();
    if (notesView) notesView.destroy();
    if (sandboxView) sandboxView.destroy();
  });
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && modal.show && modal.onCancel()} />

<div class="toast-container">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}">
      <span class="toast-message">{toast.message}</span>
    </div>
  {/each}
</div>

{#if modal.show}
  <div class="modal-backdrop" onclick={modal.onCancel} role="presentation">
    <div
      class="modal-box"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && modal.onCancel()}
      role="dialog"
      tabindex="-1"
    >
      <div class="modal-header">{modal.title}</div>
      <div class="modal-body">
        <input
          type="text"
          class="modal-input"
          placeholder={modal.placeholder}
          bind:value={modal.value}
          onkeydown={(e) => {
            if (e.key === 'Enter') modal.onConfirm(modal.value);
            if (e.key === 'Escape') modal.onCancel();
          }}
        />
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={modal.onCancel}>Cancel</button>
        <button class="modal-btn primary" onclick={() => modal.onConfirm(modal.value)}>Confirm</button>
      </div>
    </div>
  </div>
{/if}

<div class="app-shell">
  <header class="top-header">
    <div class="top-header-left">
      <span class="app-brand">CrabCode</span>
      <button class="open-btn compact" onclick={chooseFolder}>
        <FolderOpen size={14} />
        <span>Select Workspace</span>
      </button>
      {#if currentFolder}
        <span class="workspace-badge" title={currentFolder}>
          {currentFolder.split(/[\\/]/).pop()}
        </span>
      {/if}
    </div>
    <nav class="top-tabs">
      <button
        class="top-tab"
        class:active={activeTab === 'workspace'}
        onclick={() => activeTab = 'workspace'}
      >
        <FolderOpen size={14} />
        <span>Workspace</span>
      </button>
      <button
        class="top-tab"
        class:active={activeTab === 'notes'}
        onclick={() => activeTab = 'notes'}
      >
        <BookOpen size={14} />
        <span>Notes</span>
      </button>
      <button
        class="top-tab"
        class:active={activeTab === 'sandboxes'}
        onclick={() => activeTab = 'sandboxes'}
      >
        <Database size={14} />
        <span>Sandboxes</span>
      </button>
      <button
        class="top-tab"
        class:active={activeTab === 'settings'}
        onclick={() => activeTab = 'settings'}
      >
        <Settings size={14} />
        <span>Settings</span>
      </button>
    </nav>
  </header>

  <div class="app-body">
    {#if activeTab !== 'settings'}
      <aside class="sidebar">
        <div class="file-tree-container">
          {#if activeTab === 'workspace'}
            {#if currentFolder}
              <div class="section-context-header">
                <span class="project-title" title={currentFolder}>
                  {currentFolder.split(/[\\/]/).pop()}
                </span>
                <div class="sidebar-toolbar">
                  <button class="toolbar-btn" onclick={createFile} title="New File">
                    <FilePlus size={13} />
                  </button>
                  <button class="toolbar-btn" onclick={createFolder} title="New Folder">
                    <FolderPlus size={13} />
                  </button>
                </div>
              </div>
              <div class="tree-scroll">
                {#each fileTree as node}
                  <FileNode
                    {node}
                    {openFile}
                    {toggleFolder}
                    {expandedFolders}
                    {folderContents}
                    {activeFilePath}
                    {activeFileUnsaved}
                  />
                {/each}
              </div>
            {:else}
              <div class="empty-tree-state">
                <p>No workspace loaded.</p>
                <p class="subtitle">Use Select Workspace to open a folder.</p>
              </div>
            {/if}
          {:else if activeTab === 'notes'}
            <div class="section-context-header">
              <span class="project-title">SQLite Notes</span>
              <button class="toolbar-btn primary" onclick={handleCreateNote} title="New Note">
                <Plus size={13} />
              </button>
            </div>
            <div class="notes-explorer">
              <input
                type="text"
                class="sidebar-search-input"
                placeholder="Search notes..."
                bind:value={noteSearchQuery}
              />
              <div class="notes-list-scroll">
                {#each filteredNotes as note (note.id)}
                  <div
                    class="sqlite-item-row"
                    class:active={activeNoteId === note.id}
                    onclick={() => selectNote(note)}
                    role="button"
                    tabindex="0"
                    onkeydown={(e) => e.key === 'Enter' && selectNote(note)}
                  >
                    <span class="item-title">{note.title}</span>
                    <button
                      class="item-delete-btn"
                      onclick={(e) => { e.stopPropagation(); handleDeleteNote(note.id); }}
                      title="Delete note"
                    >
                      <Trash2 size={12} />
                    </button>
                  </div>
                {/each}
                {#if filteredNotes.length === 0}
                  <div class="sidebar-empty-hint">No notes yet.</div>
                {/if}
              </div>
            </div>
          {:else if activeTab === 'sandboxes'}
            <div class="section-context-header">
              <span class="project-title">Virtual Sandboxes</span>
              <button class="toolbar-btn primary" onclick={handleCreateSandbox} title="New Sandbox">
                <Plus size={13} />
              </button>
            </div>
            <div class="sandboxes-list-scroll">
              {#each sandboxesList as sandbox (sandbox.id)}
                <div
                  class="sqlite-item-row"
                  class:active={activeSandboxId === sandbox.id}
                  onclick={() => selectSandbox(sandbox)}
                  role="button"
                  tabindex="0"
                  onkeydown={(e) => e.key === 'Enter' && selectSandbox(sandbox)}
                >
                  <span class="item-title">{sandbox.name}</span>
                  <button
                    class="item-delete-btn"
                    onclick={(e) => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }}
                    title="Delete sandbox"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              {/each}
              {#if sandboxesList.length === 0}
                <div class="sidebar-empty-hint">No sandboxes yet.</div>
              {/if}
            </div>
          {/if}
        </div>
      </aside>
    {/if}

    <main class="editor-panel" class:full-width={activeTab === 'settings'}>
      {#if activeTab === 'workspace'}
        {#if activeFilePath}
          <div class="editor-header">
            <div class="file-info">
              <span class="active-file-name">{activeFileName}</span>
              <span class="active-file-path" title={activeFilePath}>{activeFilePath}</span>
            </div>
            <div class="header-actions">
              {#if !isRunning}
                <button class="header-action-btn run" onclick={runActiveCode} title="Run Code">
                  <Play size={14} fill="#48bb78" stroke="none" />
                  <span>Run</span>
                </button>
              {:else}
                <button class="header-action-btn stop" onclick={stopActiveProcess} title="Stop">
                  <Square size={14} fill="#ff5a36" stroke="none" />
                  <span>Stop</span>
                </button>
              {/if}
              <button
                class="header-action-btn"
                onclick={saveWorkspaceFile}
                disabled={!activeFileUnsaved}
                title="Save (Ctrl+S)"
              >
                <Check size={14} />
                <span>Save</span>
              </button>
              <div class="divider">|</div>
              <button
                class="header-action-btn"
                class:active={isSplitPane}
                onclick={() => isSplitPane = !isSplitPane}
                title="Toggle Split Pane"
              >
                <Columns size={14} />
                <span>Split</span>
              </button>
              {#if isSplitPane}
                <select class="split-select" bind:value={splitPaneType}>
                  <option value="notes">Notes</option>
                  <option value="sandbox">Sandbox</option>
                </select>
              {/if}
            </div>
          </div>

          <div class="workspace-split">
            <div class="editor-body">
              <div class="editor-container-inner" bind:this={workspaceEditorContainer}></div>
            </div>

            {#if isSplitPane}
              <div class="split-view-container">
                {#if splitPaneType === 'notes'}
                  <div class="split-header-bar">
                    <span>
                      <FileText size={12} />
                      {activeNoteTitle || 'No note selected'}
                    </span>
                    <div class="split-controls">
                      <button class="split-btn" onclick={() => isMarkdownEditing = !isMarkdownEditing}>
                        {isMarkdownEditing ? 'Preview' : 'Edit'}
                      </button>
                      <button class="split-btn save" onclick={handleSaveNote} disabled={!activeNoteUnsaved}>
                        Save
                      </button>
                    </div>
                  </div>
                  <div class="split-body-content">
                    {#if activeNoteId}
                      {#if isMarkdownEditing}
                        <div class="notes-cm-container" bind:this={notesEditorContainer}></div>
                      {:else}
                        <div class="markdown-rendered-view">
                          {#each activeNoteContent.split('\n') as line}
                            {@const part = renderMarkdownLine(line)}
                            {#if part.type === 'h1'}
                              <h1>{part.text}</h1>
                            {:else if part.type === 'h2'}
                              <h2>{part.text}</h2>
                            {:else if part.type === 'li'}
                              <li>{part.text}</li>
                            {:else if part.type === 'br'}
                              <br />
                            {:else}
                              <p>{part.text}</p>
                            {/if}
                          {/each}
                        </div>
                      {/if}
                    {:else}
                      <div class="split-empty">Select a note from the Notes tab sidebar.</div>
                    {/if}
                  </div>
                {:else}
                  <div class="split-header-bar">
                    <span>{activeSandboxFileName || 'No sandbox file'}</span>
                    <button
                      class="split-btn save"
                      onclick={handleSaveSandbox}
                      disabled={!activeSandboxFileUnsaved}
                    >
                      Save
                    </button>
                  </div>
                  <div class="split-body-content">
                    {#if activeSandboxId && activeSandboxFilePath}
                      <div class="sandbox-cm-container" bind:this={sandboxEditorContainer}></div>
                    {:else}
                      <div class="split-empty">Select a sandbox and file from the Sandboxes tab.</div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/if}
          </div>

          {#if isConsoleOpen}
            <div class="console-drawer">
              <div class="console-header">
                <div class="console-title">
                  <Terminal size={12} />
                  <span>CONSOLE OUTPUT</span>
                </div>
                <div class="console-actions">
                  <button class="console-btn" onclick={() => consoleLogs = []}>Clear</button>
                  <button class="console-btn toggle" onclick={() => isConsoleOpen = false}>Minimize</button>
                </div>
              </div>
              <div class="console-body" id="console-output-view">
                {#each consoleLogs as log}
                  <div class="console-line">{log}</div>
                {/each}
              </div>
            </div>
          {:else}
            <button class="console-restore-bar" onclick={() => isConsoleOpen = true}>
              <Terminal size={12} />
              <span>Show Console</span>
            </button>
          {/if}
        {:else}
          <div class="editor-empty-state">
            <div class="welcome-box">
              <h1>Workspace</h1>
              <p>Open a project folder to edit code, run scripts, and optionally split-pane with notes or sandboxes.</p>
              <button class="action-btn primary" onclick={chooseFolder}>
                <FolderOpen size={14} />
                <span>Select Workspace</span>
              </button>
            </div>
          </div>
        {/if}

      {:else if activeTab === 'notes'}
        {#if activeNoteId}
          <div class="markdown-tab-panel">
            <div class="note-editing-header">
              <input type="text" class="note-title-input" bind:value={activeNoteTitle} placeholder="Note title" />
              <div class="note-header-actions">
                <button class="action-btn" onclick={() => isMarkdownEditing = !isMarkdownEditing}>
                  {isMarkdownEditing ? 'Preview' : 'Edit'}
                </button>
                <button class="action-btn primary" onclick={handleSaveNote} disabled={!activeNoteUnsaved}>
                  Save Note
                </button>
              </div>
            </div>
            <div class="note-editing-body">
              {#if isMarkdownEditing}
                <div class="notes-cm-container full" bind:this={notesEditorContainer}></div>
              {:else}
                <div class="markdown-rendered-view full">
                  {#each activeNoteContent.split('\n') as line}
                    {@const part = renderMarkdownLine(line)}
                    {#if part.type === 'h1'}
                      <h1>{part.text}</h1>
                    {:else if part.type === 'h2'}
                      <h2>{part.text}</h2>
                    {:else if part.type === 'li'}
                      <li>{part.text}</li>
                    {:else if part.type === 'br'}
                      <br />
                    {:else}
                      <p>{part.text}</p>
                    {/if}
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {:else}
          <div class="editor-empty-state">
            <div class="welcome-box">
              <h1>Notes</h1>
              <p>SQLite-backed markdown notebooks stored in your workspace `.crab/crab.db` file.</p>
              {#if currentFolder}
                <button class="action-btn primary" onclick={handleCreateNote}>
                  <Plus size={14} />
                  <span>Create Note</span>
                </button>
              {:else}
                <button class="action-btn" onclick={chooseFolder}>
                  <FolderOpen size={14} />
                  <span>Select Workspace First</span>
                </button>
              {/if}
            </div>
          </div>
        {/if}

      {:else if activeTab === 'sandboxes'}
        {#if activeSandboxId}
          <div class="sandbox-tab-panel">
            <div class="sandbox-header-controls">
              <div class="sandbox-metadata">
                <h2>{activeSandboxName}</h2>
                <span class="sandbox-id-badge">SQLITE VIRTUAL PROJECT</span>
              </div>
              <div class="sandbox-actions">
                <button
                  class="action-btn"
                  class:active={sandboxTabMode === 'code'}
                  onclick={() => sandboxTabMode = 'code'}
                >
                  Code
                </button>
                <button
                  class="action-btn"
                  class:active={sandboxTabMode === 'yaml'}
                  onclick={() => sandboxTabMode = 'yaml'}
                >
                  YAML Config
                </button>
                {#if !isRunning}
                  <button class="action-btn run" onclick={runActiveCode}>
                    <Play size={13} fill="#48bb78" stroke="none" />
                    <span>Run</span>
                  </button>
                {:else}
                  <button class="action-btn stop" onclick={stopActiveProcess}>
                    <Square size={13} fill="#ff5a36" stroke="none" />
                    <span>Stop</span>
                  </button>
                {/if}
                <button class="action-btn primary" onclick={handleSaveSandbox}>
                  Save
                </button>
              </div>
            </div>

            <div class="sandbox-working-split">
              {#if sandboxTabMode === 'code'}
                <div class="sandbox-virtual-explorer">
                  <div class="v-explorer-header">
                    <span>Virtual Files</span>
                    <button class="v-explorer-btn" onclick={handleCreateSandboxFile} title="Add file">
                      <Plus size={12} />
                    </button>
                  </div>
                  <div class="virtual-files-list">
                    {#each sandboxFilesList as vFile (vFile.path)}
                      <div class="v-file-row-wrap">
                        <button
                          class="v-file-row"
                          class:active={activeSandboxFilePath === vFile.path}
                          onclick={() => selectSandboxFile(vFile)}
                        >
                          <span>{vFile.path}</span>
                        </button>
                        <button
                          class="v-file-delete"
                          onclick={() => handleDeleteSandboxFile(vFile.path)}
                          title="Delete file"
                        >
                          <Trash2 size={11} />
                        </button>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}

              <div class="sandbox-editor-wrapper">
                {#if sandboxTabMode === 'code'}
                  {#if activeSandboxFilePath}
                    <div class="sandbox-cm-container full" bind:this={sandboxEditorContainer}></div>
                  {:else}
                    <div class="split-empty">No virtual file selected.</div>
                  {/if}
                {:else}
                  <div class="yaml-config-panel">
                    <div class="yaml-config-header">
                      <h3>Environment Config (YAML)</h3>
                      <p>
                        Define sandbox execution rules. Set <code>run_command</code> to control how
                        files are compiled and executed from the temp extraction directory.
                      </p>
                    </div>
                    <textarea
                      class="yaml-textarea"
                      bind:value={activeSandboxConfig}
                      placeholder={'name: "My Sandbox"\nenvironment: "python"\nrun_command: "python3 main.py"'}
                    ></textarea>
                    {#if activeSandboxConfigUnsaved}
                      <p class="yaml-unsaved-hint">Config has unsaved changes.</p>
                    {/if}
                  </div>
                {/if}
              </div>
            </div>

            {#if isConsoleOpen}
              <div class="console-drawer">
                <div class="console-header">
                  <div class="console-title">
                    <Terminal size={12} />
                    <span>SANDBOX CONSOLE</span>
                  </div>
                  <div class="console-actions">
                    <button class="console-btn" onclick={() => consoleLogs = []}>Clear</button>
                    <button class="console-btn toggle" onclick={() => isConsoleOpen = false}>Minimize</button>
                  </div>
                </div>
                <div class="console-body" id="console-output-view">
                  {#each consoleLogs as log}
                    <div class="console-line">{log}</div>
                  {/each}
                </div>
              </div>
            {:else}
              <button class="console-restore-bar" onclick={() => isConsoleOpen = true}>
                <Terminal size={12} />
                <span>Show Console</span>
              </button>
            {/if}
          </div>
        {:else}
          <div class="editor-empty-state">
            <div class="welcome-box">
              <h1>Sandboxes</h1>
              <p>Virtual mini-projects stored in SQLite. Extract to temp, run with YAML-defined commands.</p>
              {#if currentFolder}
                <button class="action-btn primary" onclick={handleCreateSandbox}>
                  <Plus size={14} />
                  <span>Create Sandbox</span>
                </button>
              {:else}
                <button class="action-btn" onclick={chooseFolder}>
                  <FolderOpen size={14} />
                  <span>Select Workspace First</span>
                </button>
              {/if}
            </div>
          </div>
        {/if}

      {:else if activeTab === 'settings'}
        <div class="settings-view-panel">
          <div class="settings-card">
            <h2>CrabCode System Base Location</h2>
            <p class="settings-desc">
              Specify the root storage location where configuration rules, workspace database overrides,
              global setups, and playbooks are placed. Perfect for moving your data directory to an
              external storage unit.
            </p>
            <div class="settings-field-group">
              <label for="crabRootPath">CrabCode Data Folder (.crabcode):</label>
              <div class="settings-input-row">
                <input
                  type="text"
                  id="crabRootPath"
                  class="settings-path-input"
                  bind:value={settingsCrabRootPath}
                  oninput={checkFolderState}
                  placeholder="~/.crabcode"
                />
                <button class="settings-browse-btn" onclick={browseCrabRootPath}>Browse</button>
              </div>
            </div>

            {#if isCrabFolderEmpty}
              <div class="initialization-banner">
                <div class="banner-content">
                  <span class="warning-icon">&#9888;&#65039;</span>
                  <div>
                    <h4>Uninitialized Directory</h4>
                    <p>This directory is currently empty or does not exist. Initialize the necessary environment folder structures and templates below.</p>
                  </div>
                </div>
                <button class="initialize-btn" onclick={initializeFolderStructure}>
                  Initialize Folder Structure
                </button>
              </div>
            {/if}

            <div class="settings-divider"></div>

            <h2>Universal Environments</h2>
            <p class="settings-desc">Environment and package toolchains are stored automatically inside your active CrabCode folder path.</p>

            <div class="derived-path-display">
              <span class="derived-label">Derived Environments Path:</span>
              <code class="derived-code">{settingsCrabRootPath ? `${settingsCrabRootPath}/environments` : 'Unconfigured'}</code>
            </div>

            <div class="settings-save-row">
              <button class="settings-save-btn" onclick={saveGlobalConfig} disabled={isCrabFolderEmpty}>Save Changes</button>
            </div>
          </div>
        </div>
      {/if}

      <footer class="status-bar">
        <div class="status-left">
          <span class="status-indicator" class:running={isRunning}></span>
          <span class="status-text">{statusMessage}</span>
        </div>
        {#if activeTab === 'workspace' && activeFilePath}
          <div class="status-right">
            <span>Lines: {lineCount}</span>
            <span>Chars: {charCount}</span>
            <span class="lang-tag">{fileExtension}</span>
          </div>
        {/if}
      </footer>
    </main>
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background-color: #0b0b0f;
    color: #e2e8f0;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    height: 100vh;
    overflow: hidden;
  }

  .app-shell {
    display: flex;
    flex-direction: column;
    width: 100vw;
    height: 100vh;
    background-color: #0b0b0f;
  }

  .top-header {
    height: 48px;
    min-height: 48px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    gap: 16px;
    user-select: none;
  }

  .top-header-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .app-brand {
    font-size: 14px;
    font-weight: 800;
    color: #ff5a36;
    letter-spacing: 0.3px;
    white-space: nowrap;
  }

  .open-btn.compact {
    background-color: #ff5a36;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: background-color 0.15s;
    white-space: nowrap;
  }

  .open-btn.compact:hover {
    background-color: #e04b28;
  }

  .workspace-badge {
    font-size: 11px;
    color: #9ca3af;
    background-color: #1a1a24;
    border: 1px solid #2d3748;
    padding: 4px 8px;
    border-radius: 4px;
    max-width: 160px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .top-tabs {
    display: flex;
    align-items: center;
    gap: 4px;
    background-color: #0a0a0f;
    border: 1px solid #1a1a24;
    border-radius: 8px;
    padding: 4px;
  }

  .top-tab {
    background: none;
    border: 1px solid transparent;
    color: #9ca3af;
    padding: 6px 14px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: background-color 0.1s, color 0.1s, border-color 0.1s;
    white-space: nowrap;
  }

  .top-tab:hover {
    color: #edf2f7;
    background-color: #1a1a24;
  }

  .top-tab.active {
    background-color: #ff5a3618;
    color: #edf2f7;
    border-color: #ff5a3644;
  }

  .app-body {
    flex: 1;
    display: flex;
    overflow: hidden;
    min-height: 0;
  }

  .sidebar {
    width: 280px;
    min-width: 240px;
    max-width: 360px;
    background-color: #0c0c12;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .file-tree-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 12px 6px;
  }

  .section-context-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 10px 10px 10px;
    border-bottom: 1px solid #1c1c28;
    margin-bottom: 8px;
  }

  .project-title {
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sidebar-toolbar {
    display: flex;
    gap: 4px;
  }

  .toolbar-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    transition: background-color 0.15s, color 0.15s;
  }

  .toolbar-btn:hover {
    background-color: #2d3748;
    color: #edf2f7;
  }

  .toolbar-btn.primary {
    border-color: #ff5a3655;
    color: #ff5a36;
  }

  .toolbar-btn.primary:hover {
    background-color: #ff5a3618;
  }

  .tree-scroll {
    flex: 1;
    overflow-y: auto;
  }

  .tree-scroll::-webkit-scrollbar,
  .notes-list-scroll::-webkit-scrollbar,
  .sandboxes-list-scroll::-webkit-scrollbar,
  .virtual-files-list::-webkit-scrollbar,
  .console-body::-webkit-scrollbar,
  .markdown-rendered-view::-webkit-scrollbar {
    width: 5px;
  }

  .tree-scroll::-webkit-scrollbar-thumb,
  .notes-list-scroll::-webkit-scrollbar-thumb,
  .sandboxes-list-scroll::-webkit-scrollbar-thumb,
  .virtual-files-list::-webkit-scrollbar-thumb,
  .console-body::-webkit-scrollbar-thumb,
  .markdown-rendered-view::-webkit-scrollbar-thumb {
    background-color: #2d3748;
    border-radius: 3px;
  }

  .empty-tree-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    text-align: center;
    padding: 20px;
    color: #718096;
    font-size: 13px;
  }

  .empty-tree-state .subtitle {
    font-size: 11px;
    opacity: 0.7;
    margin-top: 4px;
  }

  .sidebar-empty-hint {
    padding: 12px;
    font-size: 12px;
    color: #4a5568;
    text-align: center;
  }

  .notes-explorer {
    display: flex;
    flex-direction: column;
    flex: 1;
    gap: 8px;
    overflow: hidden;
  }

  .sidebar-search-input {
    background-color: #121218;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 6px 10px;
    color: #edf2f7;
    font-size: 12px;
    outline: none;
    margin: 0 6px;
  }

  .sidebar-search-input:focus {
    border-color: #ff5a36;
  }

  .notes-list-scroll,
  .sandboxes-list-scroll {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0 4px;
  }

  .sqlite-item-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 6px 10px;
    border-radius: 4px;
    cursor: pointer;
    color: #a0aec0;
    font-size: 13px;
    transition: background-color 0.15s, color 0.15s;
  }

  .sqlite-item-row:hover,
  .sqlite-item-row.active {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .sqlite-item-row.active {
    border-left: 2px solid #ff5a36;
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }

  .item-title {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .item-delete-btn {
    background: none;
    border: none;
    color: #718096;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.1s, background-color 0.1s;
  }

  .item-delete-btn:hover {
    color: #ff5a36;
    background-color: #ff5a361c;
  }

  .editor-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #121217;
    height: 100%;
    overflow: hidden;
    position: relative;
    min-width: 0;
  }

  .editor-panel.full-width {
    width: 100%;
  }

  .editor-header {
    height: 44px;
    padding: 0 16px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .file-info {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .active-file-name {
    font-size: 13px;
    font-weight: 600;
    color: #edf2f7;
    white-space: nowrap;
  }

  .active-file-path {
    font-size: 11px;
    color: #4a5568;
    max-width: 300px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-action-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
  }

  .header-action-btn:hover:not(:disabled),
  .header-action-btn.active {
    background-color: #2d3748;
    color: #edf2f7;
    border-color: #4a5568;
  }

  .header-action-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .header-action-btn.run {
    border-color: #48bb7855;
    color: #48bb78;
  }

  .header-action-btn.run:hover {
    background-color: #48bb781a;
  }

  .header-action-btn.stop {
    border-color: #ff5a3655;
    color: #ff5a36;
  }

  .header-action-btn.stop:hover {
    background-color: #ff5a361a;
  }

  .divider {
    color: #2d3748;
  }

  .split-select {
    background-color: #121217;
    border: 1px solid #2d3748;
    color: #edf2f7;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 12px;
    outline: none;
    cursor: pointer;
  }

  .workspace-split {
    flex: 1;
    display: flex;
    overflow: hidden;
    min-height: 0;
  }

  .editor-body {
    flex: 1;
    display: flex;
    overflow: hidden;
    background-color: #14141a;
    min-width: 0;
  }

  .editor-container-inner {
    position: relative;
    flex: 1;
    height: 100%;
    overflow: hidden;
  }

  .editor-container-inner :global(.cm-editor),
  .notes-cm-container :global(.cm-editor),
  .sandbox-cm-container :global(.cm-editor) {
    height: 100%;
    width: 100%;
  }

  .split-view-container {
    width: 420px;
    min-width: 300px;
    max-width: 560px;
    background-color: #0c0c12;
    border-left: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
  }

  .split-header-bar {
    height: 38px;
    padding: 0 12px;
    border-bottom: 1px solid #1a1a24;
    background-color: #0a0a0f;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    font-weight: 600;
    color: #a0aec0;
    gap: 8px;
  }

  .split-header-bar span {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .split-controls {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .split-btn {
    background-color: #1c1c28;
    color: #edf2f7;
    border: 1px solid #2d3748;
    padding: 2px 8px;
    font-size: 11px;
    border-radius: 4px;
    cursor: pointer;
  }

  .split-btn.save {
    background-color: #ff5a36;
    color: white;
    border: none;
  }

  .split-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .split-body-content {
    flex: 1;
    overflow: hidden;
    background-color: #14141a;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .notes-cm-container,
  .sandbox-cm-container {
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .notes-cm-container.full,
  .sandbox-cm-container.full {
    width: 100%;
    height: 100%;
  }

  .markdown-rendered-view {
    padding: 16px;
    overflow-y: auto;
    height: 100%;
    box-sizing: border-box;
    font-size: 13px;
    line-height: 1.6;
    color: #cbd5e0;
  }

  .markdown-rendered-view.full {
    background-color: #14141a;
  }

  .markdown-rendered-view h1 {
    font-size: 18px;
    color: #edf2f7;
    border-bottom: 1px solid #2d3748;
    padding-bottom: 4px;
    margin-top: 0;
  }

  .markdown-rendered-view h2 {
    font-size: 15px;
    color: #edf2f7;
    margin-top: 14px;
  }

  .markdown-rendered-view p {
    margin: 8px 0;
  }

  .split-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #4a5568;
    font-size: 13px;
    padding: 20px;
    text-align: center;
  }

  .markdown-tab-panel {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .note-editing-header {
    height: 48px;
    padding: 0 16px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .note-title-input {
    background: none;
    border: none;
    color: #edf2f7;
    font-size: 16px;
    font-weight: 700;
    outline: none;
    flex: 1;
    min-width: 0;
  }

  .note-header-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .note-editing-body {
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .sandbox-tab-panel {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .sandbox-header-controls {
    padding: 12px 16px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .sandbox-metadata h2 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: #edf2f7;
  }

  .sandbox-id-badge {
    font-size: 9px;
    background-color: #2b2b3c;
    color: #a0aec0;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    margin-top: 4px;
    display: inline-block;
    letter-spacing: 0.4px;
  }

  .sandbox-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }

  .action-btn {
    background-color: #1a1a24;
    color: #9ca3af;
    border: 1px solid #2d3748;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: background-color 0.15s, color 0.15s;
  }

  .action-btn:hover,
  .action-btn.active {
    background-color: #2d3748;
    color: #edf2f7;
  }

  .action-btn.primary {
    background-color: #ff5a36;
    color: white;
    border: none;
  }

  .action-btn.primary:hover {
    background-color: #e04b28;
  }

  .action-btn.run {
    border-color: #48bb7855;
    color: #48bb78;
  }

  .action-btn.run:hover {
    background-color: #48bb781a;
  }

  .action-btn.stop {
    border-color: #ff5a3655;
    color: #ff5a36;
  }

  .action-btn.stop:hover {
    background-color: #ff5a361a;
  }

  .action-btn:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .sandbox-working-split {
    flex: 1;
    display: flex;
    overflow: hidden;
    min-height: 0;
  }

  .sandbox-virtual-explorer {
    width: 220px;
    min-width: 180px;
    border-right: 1px solid #1a1a24;
    background-color: #0a0a0f;
    display: flex;
    flex-direction: column;
  }

  .v-explorer-header {
    height: 34px;
    padding: 0 10px;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    text-transform: uppercase;
  }

  .v-explorer-btn {
    background: none;
    border: none;
    color: #a0aec0;
    cursor: pointer;
    display: flex;
    align-items: center;
  }

  .v-explorer-btn:hover {
    color: #ff5a36;
  }

  .virtual-files-list {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .v-file-row-wrap {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .v-file-row {
    background: none;
    border: none;
    flex: 1;
    text-align: left;
    padding: 6px 8px;
    color: #a0aec0;
    font-size: 12px;
    border-radius: 4px;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .v-file-row:hover,
  .v-file-row.active {
    background-color: #1c1c28;
    color: #edf2f7;
  }

  .v-file-delete {
    background: none;
    border: none;
    color: #4a5568;
    cursor: pointer;
    padding: 4px;
    border-radius: 3px;
    display: flex;
    align-items: center;
  }

  .v-file-delete:hover {
    color: #ff5a36;
    background-color: #ff5a3618;
  }

  .sandbox-editor-wrapper {
    flex: 1;
    overflow: hidden;
    background-color: #14141a;
    min-width: 0;
  }

  .yaml-config-panel {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    box-sizing: border-box;
  }

  .yaml-config-header h3 {
    margin: 0 0 6px 0;
    color: #edf2f7;
    font-size: 16px;
  }

  .yaml-config-header p {
    margin: 0;
    color: #718096;
    font-size: 13px;
    line-height: 1.5;
  }

  .yaml-config-header code {
    color: #ff5a36;
    font-family: 'Fira Code', monospace;
    font-size: 12px;
  }

  .yaml-textarea {
    flex: 1;
    background-color: #0c0c12;
    border: 1px solid #2d3748;
    border-radius: 8px;
    color: #e2e8f0;
    font-family: 'Fira Code', monospace;
    font-size: 13px;
    padding: 16px;
    outline: none;
    resize: none;
    min-height: 200px;
  }

  .yaml-textarea:focus {
    border-color: #ff5a36;
  }

  .yaml-unsaved-hint {
    margin: 0;
    font-size: 12px;
    color: #ff5a36;
  }

  .console-drawer {
    height: 160px;
    background-color: #08080c;
    border-top: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
  }

  .console-header {
    background-color: #0c0c12;
    padding: 6px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #1a1a24;
  }

  .console-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    letter-spacing: 0.4px;
  }

  .console-actions {
    display: flex;
    gap: 12px;
  }

  .console-btn {
    background: none;
    border: none;
    color: #718096;
    font-size: 11px;
    cursor: pointer;
  }

  .console-btn:hover {
    color: #edf2f7;
  }

  .console-body {
    flex: 1;
    padding: 12px 16px;
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #cbd5e0;
    overflow-y: auto;
    background-color: #08080c;
  }

  .console-line {
    line-height: 1.5;
    white-space: pre-wrap;
  }

  .console-restore-bar {
    height: 28px;
    background-color: #0c0c12;
    border-top: 1px solid #1a1a24;
    color: #718096;
    font-size: 11px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    cursor: pointer;
    border-left: none;
    border-right: none;
    width: 100%;
    flex-shrink: 0;
  }

  .console-restore-bar:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .settings-view-panel {
    flex: 1;
    padding: 40px;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    overflow-y: auto;
    box-sizing: border-box;
  }

  .settings-card {
    background-color: #0c0c12;
    border: 1px solid #1a1a24;
    border-radius: 12px;
    width: 600px;
    max-width: 100%;
    padding: 32px;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5);
  }

  .settings-card h2 {
    margin: 0 0 8px 0;
    font-size: 18px;
    color: #edf2f7;
  }

  .settings-desc {
    color: #718096;
    font-size: 13px;
    line-height: 1.5;
    margin: 0 0 24px 0;
  }

  .settings-divider {
    height: 1px;
    background-color: #1a1a24;
    margin: 24px 0;
  }

  .initialization-banner {
    background-color: #2c1a16;
    border: 1px solid #ff5a3633;
    border-radius: 8px;
    padding: 16px;
    margin-top: 16px;
    margin-bottom: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .banner-content {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .warning-icon {
    font-size: 18px;
  }

  .banner-content h4 {
    margin: 0 0 4px 0;
    color: #ff5a36;
    font-size: 14px;
    font-weight: 700;
  }

  .banner-content p {
    margin: 0;
    color: #cbd5e0;
    font-size: 12px;
    line-height: 1.5;
  }

  .initialize-btn {
    background-color: #ff5a36;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px 16px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    align-self: flex-start;
    transition: background-color 0.1s;
  }

  .initialize-btn:hover {
    background-color: #e04b28;
  }

  .derived-path-display {
    background-color: #121217;
    border: 1px solid #1a1a24;
    border-radius: 6px;
    padding: 12px;
    margin-bottom: 24px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .derived-label {
    font-size: 11px;
    color: #718096;
    font-weight: 600;
  }

  .derived-code {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #48bb78;
  }

  .settings-field-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 24px;
  }

  .settings-field-group label {
    font-size: 12px;
    font-weight: 600;
    color: #edf2f7;
  }

  .settings-input-row {
    display: flex;
    gap: 8px;
  }

  .settings-path-input {
    flex: 1;
    background-color: #121217;
    border: 1px solid #2d3748;
    border-radius: 6px;
    padding: 8px 12px;
    color: #edf2f7;
    font-size: 13px;
    outline: none;
    min-width: 0;
  }

  .settings-path-input:focus {
    border-color: #ff5a36;
  }

  .settings-browse-btn {
    background-color: #2d3748;
    color: #edf2f7;
    border: none;
    border-radius: 6px;
    padding: 0 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
  }

  .settings-browse-btn:hover {
    background-color: #4a5568;
  }

  .settings-save-row {
    display: flex;
    justify-content: flex-end;
  }

  .settings-save-btn {
    background-color: #ff5a36;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px 24px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }

  .settings-save-btn:hover {
    background-color: #e04b28;
  }

  .editor-empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    padding: 40px;
    color: #718096;
    background-color: #121217;
  }

  .welcome-box {
    text-align: center;
    max-width: 420px;
  }

  .welcome-box h1 {
    font-size: 20px;
    color: #edf2f7;
    margin-bottom: 8px;
  }

  .welcome-box p {
    font-size: 13px;
    line-height: 1.6;
    margin-bottom: 20px;
  }

  .action-btn {
    margin: 0 auto;
  }

  .status-bar {
    height: 24px;
    background-color: #0c0c12;
    border-top: 1px solid #1a1a24;
    padding: 0 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    color: #6b7280;
    user-select: none;
    flex-shrink: 0;
  }

  .status-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-indicator {
    width: 6px;
    height: 6px;
    background-color: #4b5563;
    border-radius: 50%;
  }

  .status-indicator.running {
    background-color: #48bb78;
    box-shadow: 0 0 8px #48bb78;
  }

  .status-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .lang-tag {
    background-color: #1a1a24;
    color: #cbd5e0;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 600;
  }

  .toast-container {
    position: fixed;
    top: 20px;
    right: 20px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    z-index: 99999;
    pointer-events: none;
  }

  .toast {
    pointer-events: auto;
    background-color: #0c0c12;
    border-left: 4px solid #4b5563;
    border-radius: 6px;
    padding: 10px 16px;
    color: #edf2f7;
    font-size: 13px;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5);
    animation: slideIn 0.2s ease-out;
    min-width: 200px;
    max-width: 350px;
  }

  .toast.success {
    border-left-color: #48bb78;
  }

  .toast.error {
    border-left-color: #ff5a36;
  }

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 99998;
  }

  .modal-box {
    background-color: #0c0c12;
    border: 1px solid #1a1a24;
    border-radius: 12px;
    width: 400px;
    max-width: 90%;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.7);
    animation: modalScale 0.15s ease-out;
  }

  .modal-header {
    padding: 16px 20px;
    font-size: 14px;
    font-weight: 700;
    color: #edf2f7;
    border-bottom: 1px solid #1a1a24;
  }

  .modal-body {
    padding: 20px;
  }

  .modal-input {
    width: 100%;
    background-color: #121217;
    border: 1px solid #2d3748;
    border-radius: 6px;
    padding: 10px 12px;
    color: #edf2f7;
    font-size: 13px;
    outline: none;
    box-sizing: border-box;
  }

  .modal-input:focus {
    border-color: #ff5a36;
  }

  .modal-footer {
    padding: 16px 20px;
    border-top: 1px solid #1a1a24;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .modal-btn {
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 600;
    border-radius: 6px;
    cursor: pointer;
    border: none;
  }

  .modal-btn.primary {
    background-color: #ff5a36;
    color: white;
  }

  .modal-btn.primary:hover {
    background-color: #e04b28;
  }

  .modal-btn.secondary {
    background-color: #2d3748;
    color: #edf2f7;
  }

  .modal-btn.secondary:hover {
    background-color: #4a5568;
  }

  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
  }

  @keyframes modalScale {
    from { transform: scale(0.95); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }
</style>
