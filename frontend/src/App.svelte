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
    OpenWorkspace,
    GetSandboxes,
    CreateSandbox,
    CreateSandboxInFolder,
    MoveSandbox,
    RenameSandbox,
    DeleteSandbox,
    SaveSandboxConfig,
    SaveSandboxNotes,
    GetSandboxFiles,
    SaveSandboxFile,
    DeleteSandboxFile,
    GetSandboxDirectory,
    RunSandbox,
    RunCommand,
    StopCommand,
    StartTerminalSession,
    WriteTerminalInput,
    IsEnvironmentInitialized,
    InitializeEnvironment,
    GetGlobalSettings,
    SaveGlobalSettings,
    IsDirectoryEmpty,
    InitializeCrabFolder,
    GetTemplates
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import FileNode from './FileNode.svelte';
  import TerminalDrawer from './TerminalDrawer.svelte';
  import { onMount, onDestroy, untrack } from 'svelte';
  import {
    FolderOpen, FilePlus, FolderPlus, Play, Square,
    Terminal, Settings, Database, BookOpen, Plus, Trash2, Columns, Check,
    PanelLeft, ChevronLeft, ChevronDown, ChevronRight, Save, Sparkles, FileCode, Layers, Eye, Edit3, Folder, FolderPlus as AddFolderIcon, FolderInput
  } from '@lucide/svelte';

  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState, Compartment } from '@codemirror/state';
  import { keymap } from '@codemirror/view';
  import { indentWithTab, undoDepth, redoDepth } from '@codemirror/commands';
  import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
  import { tags as t } from '@lezer/highlight';

  import { javascript } from '@codemirror/lang-javascript';
  import { python } from '@codemirror/lang-python';
  import { go } from '@codemirror/lang-go';
  import { rust } from '@codemirror/lang-rust';
  import { html } from '@codemirror/lang-html';
  import { css } from '@codemirror/lang-css';
  import { json } from '@codemirror/lang-json';
  import { java } from '@codemirror/lang-java';
  import { cpp } from '@codemirror/lang-cpp';
  import { sql } from '@codemirror/lang-sql';
  import { markdown } from '@codemirror/lang-markdown';

  let activeTab = $state('workspace');

  let currentFolder = $state('');
  let fileTree = $state([]);
  let expandedFolders = $state({});
  let folderContents = $state({});
  let activeFilePath = $state('');
  let activeFileName = $state('');
  let editorContent = $state('');
  let lastSavedContent = $state('');
  let statusMessage = $state('Ready');

  let showSidebar = $state(true);

  let sandboxesList = $state([]);
  let activeSandboxId = $state('');
  let activeSandboxName = $state('');
  let activeSandboxConfig = $state('');
  let lastSavedSandboxConfig = $state('');
  let activeSandboxMarkdownNote = $state('');
  let activeSandboxHTMLNote = $state('');
  let lastSavedSandboxMarkdownNote = $state('');
  let lastSavedSandboxHTMLNote = $state('');

  // Sandbox File System State
  let sandboxSearchQuery = $state('');
  let virtualFolders = $state([]);
  let expandedVirtualFolders = $state({});

  let showSandboxExplorer = $state(true);
  let showSandboxSplit = $state(true);
  let sandboxSplitType = $state('html');
  let isSandboxMarkdownEditing = $state(false);
  let isSandboxHTMLEditing = $state(false);

  let sandboxFilesList = $state([]);
  let activeSandboxFilePath = $state('');
  let activeSandboxFileName = $state('');
  let activeSandboxFileContent = $state('');
  let lastSavedSandboxFileContent = $state('');

  let settingsCrabRootPath = $state('');
  let isCrabFolderEmpty = $state(false);

  let toasts = $state([]);
  let modal = $state({ show: false, title: '', placeholder: '', value: '', onConfirm: null, onCancel: null });
  let showCreateSandboxModal = $state(false);
  let newSandboxName = $state('');
  let selectedSandboxFolder = $state('');
  let selectedSandboxTemplate = $state('python');
  let availableTemplates = $state([]);

  let showMoveSandboxModal = $state(false);
  let moveSandboxId = $state('');
  let moveNewFolderName = $state('');

  let showInitEnvModal = $state(false);
  let isInitializingEnv = $state(false);
  let isEnvInitialized = $state(true);

  let workspaceEditorContainer = $state(null);
  let sandboxEditorContainer = $state(null);
  let workspaceView = null;
  let sandboxView = null;
  const workspaceLanguageConf = new Compartment();
  const sandboxLanguageConf = new Compartment();

  let canUndo = $state(false);
  let canRedo = $state(false);
  let isConsoleOpen = $state(true);

  // Multi-terminal state
  let workspaceTerminals = $state([]);
  let activeWorkspaceTermId = $state('');
  let sandboxTerminals = $state([]);
  let activeSandboxTermId = $state('');

  let activeWorkspaceTerm = $derived(
    workspaceTerminals.find(t => t.id === activeWorkspaceTermId) || null
  );
  let activeSandboxTerm = $derived(
    sandboxTerminals.find(t => t.id === activeSandboxTermId) || null
  );
  // Bottom Drawer State (Console vs Terminal)
  let bottomMode = $state('console');
  let consoleLogs = $state([]);
  let consoleStatus = $state('Ready');
  let isConsoleRunning = $state(false);
  let activeRunnerId = $state('');

  let isRunning = $derived(activeRunnerId !== '');

  let activeFileUnsaved = $derived(activeFilePath !== '' && editorContent !== lastSavedContent);
  let activeSandboxFileUnsaved = $derived(
    activeSandboxFilePath !== '' && activeSandboxFileContent !== lastSavedSandboxFileContent
  );
  let activeSandboxConfigUnsaved = $derived(
    activeSandboxId !== '' && activeSandboxConfig !== lastSavedSandboxConfig
  );
  let activeSandboxNotesUnsaved = $derived(
    activeSandboxId !== '' && (
      activeSandboxMarkdownNote !== lastSavedSandboxMarkdownNote ||
      activeSandboxHTMLNote !== lastSavedSandboxHTMLNote
    )
  );
  let activeSandboxUnsaved = $derived(
    activeSandboxFileUnsaved || activeSandboxConfigUnsaved || activeSandboxNotesUnsaved
  );

  // Chronologically ordered sandboxes filtered by search
  let filteredSandboxes = $derived(
    sandboxesList
      .filter(s => s.name.toLowerCase().includes(sandboxSearchQuery.toLowerCase()) || (s.folder || '').toLowerCase().includes(sandboxSearchQuery.toLowerCase()))
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
  );

  let folderGroupedSandboxes = $derived.by(() => {
    const groups = { '': [] };
    virtualFolders.forEach(f => { groups[f] = []; });
    filteredSandboxes.forEach(s => {
      const f = s.folder || '';
      if (!groups[f]) groups[f] = [];
      groups[f].push(s);
    });
    return groups;
  });

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
      case 'js': case 'jsx': case 'tsx': return javascript();
      case 'ts': return javascript({ typescript: true });
      case 'py': return python();
      case 'go': return go();
      case 'rs': return rust();
      case 'java': return java();
      case 'cpp': case 'c': case 'cc': case 'h': case 'hpp': return cpp();
      case 'sql': case 'surql': return sql();
      case 'md': case 'markdown': return markdown();
      case 'html': case 'svelte': return html();
      case 'css': return css();
      case 'json': return json();
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
      const initialDoc = untrack(() => editorContent);
      const initialName = untrack(() => activeFileName);
      workspaceView = new EditorView({
        parent: workspaceEditorContainer,
        state: EditorState.create({
          doc: initialDoc,
          extensions: buildWorkspaceExtensions(initialName)
        })
      });
    } else if (!workspaceEditorContainer && workspaceView) {
      workspaceView.destroy();
      workspaceView = null;
    }
  });

  $effect(() => {
    if (sandboxEditorContainer && !sandboxView) {
      const initialDoc = untrack(() => activeSandboxFileContent);
      const initialName = untrack(() => activeSandboxFileName);
      sandboxView = new EditorView({
        parent: sandboxEditorContainer,
        state: EditorState.create({
          doc: initialDoc,
          extensions: buildSandboxExtensions(initialName)
        })
      });
    } else if (!sandboxEditorContainer && sandboxView) {
      sandboxView.destroy();
      sandboxView = null;
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
        sandboxesList = workspaceInfo.sandboxes || [];

        const foldersSet = new Set(sandboxesList.map(s => s.folder).filter(Boolean));
        virtualFolders = Array.from(foldersSet);

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
        if (workspaceTerminals.length === 0) {
          const wsTermId = createWorkspaceTerminal('shell', 'bash');
          try { await StartTerminalSession(wsTermId, folder); } catch (_) {}
        }
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
    try {
      await SaveFile(activeFilePath, editorContent);
      lastSavedContent = editorContent;
      statusMessage = 'Saved: ' + activeFileName;
      addToast('File saved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
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

  // VIRTUAL SANDBOX FOLDER CREATION & MOVEMENT
  async function createVirtualFolder() {
    const folderName = await openPrompt('New Virtual Folder', 'Game Engine Experiments');
    if (!folderName) return;
    const cleanFolder = folderName.trim();
    if (!cleanFolder) return;
    if (!virtualFolders.includes(cleanFolder)) {
      virtualFolders = [...virtualFolders, cleanFolder];
      expandedVirtualFolders[cleanFolder] = true;
      addToast(`Folder '${cleanFolder}' created`, 'success');
    }
  }

  function openMoveSandboxModal(sandboxId) {
    moveSandboxId = sandboxId;
    moveNewFolderName = '';
    showMoveSandboxModal = true;
  }

  async function confirmMoveSandbox(targetFolder) {
    showMoveSandboxModal = false;
    const id = moveSandboxId;
    moveSandboxId = '';
    try {
      await MoveSandbox(id, targetFolder);
      const si = sandboxesList.findIndex(s => s.id === id);
      if (si !== -1) {
        if (targetFolder && !virtualFolders.includes(targetFolder)) {
          virtualFolders = [...virtualFolders, targetFolder];
        }
        sandboxesList[si].folder = targetFolder;
      }
      addToast('Sandbox moved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  function handleDragStart(e, sandboxId) {
    e.dataTransfer.setData('text/plain', sandboxId);
    e.dataTransfer.effectAllowed = 'move';
  }

  function handleDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  }

  async function handleDropOnFolder(e, folderName) {
    e.preventDefault();
    const sandboxId = e.dataTransfer.getData('text/plain');
    if (!sandboxId) return;
    try {
      await MoveSandbox(sandboxId, folderName);
      const si = sandboxesList.findIndex(s => s.id === sandboxId);
      if (si !== -1) sandboxesList[si].folder = folderName;
      if (folderName && !virtualFolders.includes(folderName)) {
        virtualFolders = [...virtualFolders, folderName];
      }
      addToast('Sandbox moved', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDropOnRoot(e) {
    e.preventDefault();
    const sandboxId = e.dataTransfer.getData('text/plain');
    if (!sandboxId) return;
    try {
      await MoveSandbox(sandboxId, '');
      const si = sandboxesList.findIndex(s => s.id === sandboxId);
      if (si !== -1) sandboxesList[si].folder = '';
      addToast('Sandbox moved to root', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function openCreateSandboxModal(defaultFolder = '') {
    if (!currentFolder) {
      addToast('Select a workspace first', 'error');
      return;
    }
    newSandboxName = '';
    selectedSandboxFolder = defaultFolder;
    try {
      availableTemplates = await GetTemplates();
      if (availableTemplates.length > 0) {
        selectedSandboxTemplate = availableTemplates[0].id;
      }
    } catch (err) {
      addToast('Failed to load templates: ' + String(err), 'error');
    }
    showCreateSandboxModal = true;
  }

  async function confirmCreateSandbox() {
    const name = newSandboxName.trim() || 'Experiment';
    showCreateSandboxModal = false;
    try {
      const sandbox = await CreateSandboxInFolder(name, selectedSandboxTemplate, selectedSandboxFolder);
      sandboxesList = [sandbox, ...sandboxesList];
      await selectSandbox(sandbox);
      addToast('Sandbox created', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  let renameTimer = null;
  function handleAutoRenameSandbox(newName, immediate = false) {
    if (!activeSandboxId) return;
    activeSandboxName = newName;
    const cleanName = newName.trim() || 'Untitled Sandbox';

    const si = sandboxesList.findIndex(s => s.id === activeSandboxId);
    if (si !== -1) {
      sandboxesList[si].name = cleanName;
    }

    const doSave = async () => {
      try {
        await RenameSandbox(activeSandboxId, cleanName);
      } catch (err) {
        addToast('Failed to auto-save sandbox title: ' + String(err), 'error');
      }
    };

    clearTimeout(renameTimer);
    if (immediate) {
      doSave();
    } else {
      renameTimer = setTimeout(doSave, 500);
    }
  }

  async function checkSandboxEnvState(sandboxId) {
    try {
      isEnvInitialized = await IsEnvironmentInitialized(sandboxId);
    } catch (err) {
      isEnvInitialized = false;
    }
  }

  async function triggerInitializeEnvironment() {
    if (!activeSandboxId) return;
    isInitializingEnv = true;
    try {
      await InitializeEnvironment(activeSandboxId);
      isEnvInitialized = true;
      showInitEnvModal = false;
      addToast('Shared project environment initialized', 'success');
    } catch (err) {
      addToast('Failed to initialize environment: ' + String(err), 'error');
    } finally {
      isInitializingEnv = false;
    }
  }

  async function selectSandbox(sandbox) {
    if (activeSandboxId === sandbox.id) return;
    activeSandboxId = sandbox.id;
    activeSandboxName = sandbox.name;
    activeSandboxConfig = sandbox.configYaml;
    lastSavedSandboxConfig = sandbox.configYaml;

    activeSandboxMarkdownNote = sandbox.markdownNote || '';
    activeSandboxHTMLNote = sandbox.htmlNote || '';
    lastSavedSandboxMarkdownNote = sandbox.markdownNote || '';
    lastSavedSandboxHTMLNote = sandbox.htmlNote || '';

    isSandboxMarkdownEditing = false;
    isSandboxHTMLEditing = false;

    // Clear active file selection when switching sandboxes
    activeSandboxFilePath = '';
    activeSandboxFileName = '';
    activeSandboxFileContent = '';
    lastSavedSandboxFileContent = '';

    await checkSandboxEnvState(sandbox.id);
    if (!isEnvInitialized) {
      showInitEnvModal = true;
    }

    await loadSandboxFiles();

    if (sandboxTerminals.length === 0) {
      const sbTermId = createSandboxTerminal('shell', 'bash');
      try {
        const sandboxDir = await GetSandboxDirectory(activeSandboxId);
        await StartTerminalSession(sbTermId, sandboxDir);
      } catch (_) {}
    }
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
      if (activeSandboxFilePath && activeSandboxFileUnsaved) {
        await SaveSandboxFile(activeSandboxId, activeSandboxFilePath, activeSandboxFileContent, false);
        lastSavedSandboxFileContent = activeSandboxFileContent;
        await loadSandboxFiles();
      }

      if (activeSandboxConfigUnsaved) {
        await SaveSandboxConfig(activeSandboxId, activeSandboxConfig);
        lastSavedSandboxConfig = activeSandboxConfig;
        const si = sandboxesList.findIndex(s => s.id === activeSandboxId);
        if (si !== -1) sandboxesList[si].configYaml = activeSandboxConfig;
      }

      if (activeSandboxNotesUnsaved) {
        await SaveSandboxNotes(activeSandboxId, activeSandboxMarkdownNote, activeSandboxHTMLNote);
        lastSavedSandboxMarkdownNote = activeSandboxMarkdownNote;
        lastSavedSandboxHTMLNote = activeSandboxHTMLNote;
        const si = sandboxesList.findIndex(s => s.id === activeSandboxId);
        if (si !== -1) {
          sandboxesList[si].markdownNote = activeSandboxMarkdownNote;
          sandboxesList[si].htmlNote = activeSandboxHTMLNote;
        }
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
        activeSandboxMarkdownNote = '';
        activeSandboxHTMLNote = '';
        lastSavedSandboxMarkdownNote = '';
        lastSavedSandboxHTMLNote = '';
      }
      addToast('Sandbox deleted', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  function createWorkspaceTerminal(title = 'bash') {
    const id = 'shell_ws_' + Date.now();
    const term = { id, logs: [], isRunning: true, title, inputBuffer: '' };
    workspaceTerminals = [...workspaceTerminals, term];
    activeWorkspaceTermId = id;
    return id;
  }

  function createSandboxTerminal(title = 'bash') {
    const id = 'shell_sb_' + Date.now();
    const term = { id, logs: [], isRunning: true, title, inputBuffer: '' };
    sandboxTerminals = [...sandboxTerminals, term];
    activeSandboxTermId = id;
    return id;
  }

  async function closeWorkspaceTerminal(id) {
    const term = workspaceTerminals.find(t => t.id === id);
    if (term && term.isRunning) {
      try { await StopCommand(id); } catch (_) {}
    }
    workspaceTerminals = workspaceTerminals.filter(t => t.id !== id);
    if (activeWorkspaceTermId === id) {
      activeWorkspaceTermId = workspaceTerminals.length > 0
        ? workspaceTerminals[workspaceTerminals.length - 1].id
        : '';
    }
  }

  async function closeSandboxTerminal(id) {
    const term = sandboxTerminals.find(t => t.id === id);
    if (term && term.isRunning) {
      try { await StopCommand(id); } catch (_) {}
    }
    sandboxTerminals = sandboxTerminals.filter(t => t.id !== id);
    if (activeSandboxTermId === id) {
      activeSandboxTermId = sandboxTerminals.length > 0
        ? sandboxTerminals[sandboxTerminals.length - 1].id
        : '';
    }
  }

  function sendTerminalInput() {
    const term = activeTab === 'workspace' ? activeWorkspaceTerm : activeSandboxTerm;
    if (!term || !term.inputBuffer) return;
    const input = term.inputBuffer;
    WriteTerminalInput(term.id, input);
    term.logs.push('$ ' + input);
    if (term.logs.length > 1000) term.logs = term.logs.slice(-1000);
    term.inputBuffer = '';
    scrollToConsoleBottom();
  }

  async function runActiveCode() {
    bottomMode = 'console';
    isConsoleOpen = true;
    consoleStatus = 'Executing...';
    isConsoleRunning = true;

    if (activeTab === 'workspace') {
      if (!activeFilePath) return;
      if (activeFileUnsaved) await saveWorkspaceFile();

      const runnerId = 'runner_ws_' + Date.now();
      activeRunnerId = runnerId;
      consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Running ${activeFileName}...]`, type: 'info' }];

      const ext = activeFileName.split('.').pop().toLowerCase();
      let runnerCmd = '';
      let runnerArgs = [];

      if (ext === 'py') { runnerCmd = 'python3'; runnerArgs = [activeFilePath]; }
      else if (ext === 'js') { runnerCmd = 'node'; runnerArgs = [activeFilePath]; }
      else if (ext === 'go') { runnerCmd = 'go'; runnerArgs = ['run', activeFilePath]; }
      else if (ext === 'rs') { runnerCmd = 'sh'; runnerArgs = ['-c', `rustc "${activeFilePath}" -o main && ./main`]; }
      else if (ext === 'java') { runnerCmd = 'java'; runnerArgs = [activeFilePath]; }
      else if (ext === 'cpp' || ext === 'c') { runnerCmd = 'sh'; runnerArgs = ['-c', `g++ "${activeFilePath}" -o main && ./main`]; }
      else if (ext === 'ts') { runnerCmd = 'npx'; runnerArgs = ['tsx', activeFilePath]; }
      else if (ext === 'sql') { runnerCmd = 'sh'; runnerArgs = ['-c', `sqlite3 :memory: < "${activeFilePath}"`]; }
      else {
        consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Runner Error] No run rule for ".${ext}" files.`, type: 'error' }];
        consoleStatus = 'Error';
        isConsoleRunning = false;
        activeRunnerId = '';
        scrollToConsoleBottom();
        return;
      }

      try {
        await RunCommand(runnerId, runnerCmd, runnerArgs, getParentPath(activeFilePath) || currentFolder);
      } catch (err) {
        consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Error] ${String(err)}`, type: 'error' }];
        consoleStatus = 'Error';
        isConsoleRunning = false;
        activeRunnerId = '';
        scrollToConsoleBottom();
      }
    } else if (activeTab === 'sandboxes') {
      if (!activeSandboxId) return;

      await checkSandboxEnvState(activeSandboxId);
      if (!isEnvInitialized) {
        showInitEnvModal = true;
        consoleStatus = 'Uninitialized Env';
        isConsoleRunning = false;
        return;
      }

      const runnerId = 'runner_sb_' + Date.now();
      activeRunnerId = runnerId;
      consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Executing ${activeSandboxName} via YAML config...]`, type: 'info' }];

      try {
        await handleSaveSandbox();
        await RunSandbox(runnerId, activeSandboxId, activeSandboxFilePath);
      } catch (err) {
        if (String(err).includes("ENV_NOT_INITIALIZED")) {
          showInitEnvModal = true;
          consoleStatus = 'Uninitialized Env';
        } else {
          consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Execution Error] ${String(err)}`, type: 'error' }];
          consoleStatus = 'Error';
        }
        isConsoleRunning = false;
        activeRunnerId = '';
        scrollToConsoleBottom();
      }
    }
  }

  async function stopActiveProcess() {
    if (activeRunnerId) {
      try { await StopCommand(activeRunnerId); } catch (_) {}
    }
    consoleLogs = [...consoleLogs, { time: Date.now(), text: '[Process stopped]', type: 'error' }];
    consoleStatus = 'Stopped';
    isConsoleRunning = false;
    activeRunnerId = '';
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
      const cContainer = document.getElementById('console-output-view');
      if (cContainer) cContainer.scrollTop = cContainer.scrollHeight;
      const tContainer = document.getElementById('terminal-output-view');
      if (tContainer) tContainer.scrollTop = tContainer.scrollHeight;
    }, 40);
  }

  function handleKeyDown(event) {
    const isMeta = event.ctrlKey || event.metaKey;
    if (isMeta && event.key === 's') {
      event.preventDefault();
      if (activeTab === 'workspace') saveWorkspaceFile();
      else if (activeTab === 'sandboxes') handleSaveSandbox();
    }
  }

  function renderMarkdownLine(line) {
    if (line.startsWith('# ')) return { type: 'h1', text: line.slice(2) };
    if (line.startsWith('## ')) return { type: 'h2', text: line.slice(3) };
    if (line.startsWith('### ')) return { type: 'h3', text: line.slice(4) };
    if (line.startsWith('> ')) return { type: 'quote', text: line.slice(2) };
    if (line.startsWith('- ') || line.startsWith('* ')) return { type: 'li', text: line.slice(2) };
    if (line.trim().startsWith('```')) return { type: 'codeblock', text: line.trim() };
    if (line.trim() === '') return { type: 'br', text: '' };
    return { type: 'p', text: line };
  }

  let pendingTerminalOutputs = [];
  let terminalFlushTimer = null;

  function flushTerminalOutputs() {
    terminalFlushTimer = null;
    if (pendingTerminalOutputs.length === 0) return;

    const items = pendingTerminalOutputs;
    pendingTerminalOutputs = [];

    let updatedLogs = [...consoleLogs];
    let wsTerminals = [...workspaceTerminals];
    let sbTerminals = [...sandboxTerminals];

    for (const data of items) {
      if (data.id.startsWith('runner_')) {
        updatedLogs.push({ time: Date.now(), text: data.line, type: 'out' });
      } else if (data.id.startsWith('shell_')) {
        let wsIdx = wsTerminals.findIndex(t => t.id === data.id);
        if (wsIdx !== -1) {
          wsTerminals[wsIdx] = {
            ...wsTerminals[wsIdx],
            logs: [...wsTerminals[wsIdx].logs, data.line]
          };
        }
        let sbIdx = sbTerminals.findIndex(t => t.id === data.id);
        if (sbIdx !== -1) {
          sbTerminals[sbIdx] = {
            ...sbTerminals[sbIdx],
            logs: [...sbTerminals[sbIdx].logs, data.line]
          };
        }
      }
    }

    if (updatedLogs.length > 1000) updatedLogs = updatedLogs.slice(-1000);

    wsTerminals.forEach(t => { if (t.logs.length > 1000) t.logs = t.logs.slice(-1000); });
    sbTerminals.forEach(t => { if (t.logs.length > 1000) t.logs = t.logs.slice(-1000); });

    consoleLogs = updatedLogs;
    workspaceTerminals = wsTerminals;
    sandboxTerminals = sbTerminals;

    scrollToConsoleBottom();
  }

  function queueTerminalOutput(data) {
    pendingTerminalOutputs.push(data);
    if (!terminalFlushTimer) {
      terminalFlushTimer = requestAnimationFrame(flushTerminalOutputs);
    }
  }

  onMount(() => {
    loadGlobalConfig();
    window.addEventListener('keydown', handleKeyDown);

    EventsOn('terminal_output', (data) => {
      queueTerminalOutput(data);
    });

    EventsOn('terminal_status', (data) => {
      if (data.id.startsWith('runner_')) {
        if (data.status === '0') {
          consoleStatus = 'Finished';
        } else {
          consoleStatus = 'Error';
          consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Process exited: ${data.status}]`, type: 'error' }];
        }
        isConsoleRunning = false;
        activeRunnerId = '';
        scrollToConsoleBottom();
      } else if (data.id.startsWith('shell_')) {
        let wsIdx = workspaceTerminals.findIndex(t => t.id === data.id);
        if (wsIdx !== -1) workspaceTerminals[wsIdx].isRunning = false;
        let sbIdx = sandboxTerminals.findIndex(t => t.id === data.id);
        if (sbIdx !== -1) sandboxTerminals[sbIdx].isRunning = false;
      }
    });

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });

  onDestroy(() => {
    if (workspaceView) workspaceView.destroy();
    if (sandboxView) sandboxView.destroy();
  });
</script>

<svelte:window onkeydown={(e) => {
  if (e.key === 'Escape') {
    if (modal.show) modal.onCancel();
    if (showCreateSandboxModal) showCreateSandboxModal = false;
    if (showInitEnvModal) showInitEnvModal = false;
  }
}} />

<div class="toast-container">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}">
      <span class="toast-message">{toast.message}</span>
    </div>
  {/each}
</div>

{#if modal.show}
  <div class="modal-backdrop" onclick={modal.onCancel} role="presentation">
    <div class="modal-box" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
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

{#if showInitEnvModal}
  <div class="modal-backdrop" onclick={() => showInitEnvModal = false} role="presentation">
    <div class="modal-box guide-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">Initialize Shared Environment</div>
      <div class="modal-body guide-body">
        <p>This sandbox relies on a shared project environment that has not been initialized yet.</p>
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showInitEnvModal = false} disabled={isInitializingEnv}>Cancel</button>
        <button class="modal-btn primary" onclick={triggerInitializeEnvironment} disabled={isInitializingEnv}>
          {isInitializingEnv ? 'Initializing...' : 'Initialize Environment'}
        </button>
      </div>
    </div>
  </div>
{/if}

{#if showMoveSandboxModal}
  <div class="modal-backdrop" onclick={() => showMoveSandboxModal = false} role="presentation">
    <div class="modal-box move-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">Move Sandbox</div>
      <div class="modal-body">
        <div class="move-folder-list">
          <button class="move-folder-option" onclick={() => confirmMoveSandbox('')}>
            <Layers size={16} />
            <span class="move-folder-label">Root (No Folder)</span>
            <span class="move-folder-count">({(folderGroupedSandboxes[''] || []).length})</span>
          </button>
          {#each virtualFolders as folderName}
            <button class="move-folder-option" onclick={() => confirmMoveSandbox(folderName)}>
              <Folder size={16} class="v-folder-icon" />
              <span class="move-folder-label">{folderName}</span>
              <span class="move-folder-count">({(folderGroupedSandboxes[folderName] || []).length})</span>
            </button>
          {/each}
          <div class="move-new-folder-section">
            <input
              type="text"
              class="modal-input"
              placeholder="Create new folder..."
              bind:value={moveNewFolderName}
              onkeydown={(e) => {
                if (e.key === 'Enter' && moveNewFolderName.trim()) {
                  confirmMoveSandbox(moveNewFolderName.trim());
                }
              }}
            />
            <button
              class="modal-btn primary create-folder-btn"
              disabled={!moveNewFolderName.trim()}
              onclick={() => moveNewFolderName.trim() && confirmMoveSandbox(moveNewFolderName.trim())}
            >
              <Plus size={12} /> Create & Move
            </button>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showMoveSandboxModal = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if showCreateSandboxModal}
  <div class="modal-backdrop" onclick={() => showCreateSandboxModal = false} role="presentation">
    <div class="modal-box sandbox-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">Create Virtual Sandbox</div>
      <div class="modal-body">
        <label class="sandbox-modal-label" for="sandboxName">Sandbox Name</label>
        <input
          type="text"
          id="sandboxName"
          class="modal-input sandbox-name-input"
          placeholder="My Experiment"
          bind:value={newSandboxName}
          onkeydown={(e) => { if (e.key === 'Enter') confirmCreateSandbox(); }}
        />
        <label class="sandbox-modal-label" for="sandboxTemplate">Template</label>
        <div class="templates-grid">
          {#each availableTemplates as tmpl}
            <button
              class="template-option"
              class:selected={selectedSandboxTemplate === tmpl.id}
              onclick={() => selectedSandboxTemplate = tmpl.id}
            >
              <span class="template-dot" style="background-color: {tmpl.iconColor}"></span>
              <span class="template-name">{tmpl.name}</span>
            </button>
          {/each}
        </div>
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showCreateSandboxModal = false}>Cancel</button>
        <button class="modal-btn primary" onclick={confirmCreateSandbox}>Create Sandbox</button>
      </div>
    </div>
  </div>
{/if}

<div class="app-shell">
  <header class="top-header">
    <div class="top-header-left">
      {#if activeTab !== 'settings'}
        <button class="sidebar-toggle-btn" class:active={showSidebar} onclick={() => showSidebar = !showSidebar}>
          <PanelLeft size={16} />
        </button>
      {/if}
      <span class="app-brand">CrabCode</span>
      <button class="open-btn compact" onclick={chooseFolder}>
        <FolderOpen size={14} />
        <span>Select Workspace</span>
      </button>
      {#if currentFolder}
        <span class="workspace-badge" title={currentFolder}>{currentFolder.split(/[\\/]/).pop()}</span>
      {/if}
    </div>
    <nav class="top-tabs">
      <button class="top-tab" class:active={activeTab === 'workspace'} onclick={() => activeTab = 'workspace'}><FolderOpen size={14} /> Workspace</button>
      <button class="top-tab" class:active={activeTab === 'sandboxes'} onclick={() => activeTab = 'sandboxes'}><Database size={14} /> Sandboxes</button>
      <button class="top-tab" class:active={activeTab === 'settings'} onclick={() => activeTab = 'settings'}><Settings size={14} /> Settings</button>
    </nav>
  </header>

  <div class="app-body">
    {#if showSidebar && activeTab !== 'settings'}
      <aside class="sidebar">
        <div class="file-tree-container">
          {#if activeTab === 'workspace'}
            {#if currentFolder}
              <div class="section-context-header">
                <span class="project-title">{currentFolder.split(/[\\/]/).pop()}</span>
                <div class="sidebar-toolbar">
                  <button class="toolbar-btn" onclick={createFile} title="New File"><FilePlus size={13} /></button>
                  <button class="toolbar-btn" onclick={createFolder} title="New Folder"><FolderPlus size={13} /></button>
                  <button class="toolbar-btn" onclick={() => showSidebar = false} title="Hide Sidebar"><ChevronLeft size={13} /></button>
                </div>
              </div>
              <div class="tree-scroll">
                {#each fileTree as node}
                  <FileNode {node} {openFile} {toggleFolder} {expandedFolders} {folderContents} {activeFilePath} {activeFileUnsaved} />
                {/each}
              </div>
            {:else}
              <div class="empty-tree-state"><p>No workspace loaded.</p></div>
            {/if}
          {:else if activeTab === 'sandboxes'}
            <div class="section-context-header">
              <span class="project-title">Sandboxes</span>
              <div class="sidebar-toolbar">
                <button class="toolbar-btn" onclick={createVirtualFolder} title="New Virtual Folder"><AddFolderIcon size={13} /></button>
                <button class="toolbar-btn primary" onclick={() => openCreateSandboxModal('')} title="New Sandbox"><Plus size={13} /></button>
              </div>
            </div>

            <div class="sandbox-fs-explorer">
              <input type="text" class="sidebar-search-input" placeholder="Search sandboxes..." bind:value={sandboxSearchQuery} />
              <div class="sandboxes-list-scroll">

                {#each virtualFolders as folderName}
                  {@const itemsInFolder = (folderGroupedSandboxes[folderName] || [])}
                  <div class="virtual-folder-group">
                    <div
                      class="virtual-folder-header"
                      onclick={() => expandedVirtualFolders[folderName] = !expandedVirtualFolders[folderName]}
                      ondragover={handleDragOver}
                      ondrop={(e) => handleDropOnFolder(e, folderName)}
                      role="button"
                      tabindex="0"
                    >
                      <span class="folder-arrow">{expandedVirtualFolders[folderName] ? '▼' : '▶'}</span>
                      <Folder size={13} class="v-folder-icon" />
                      <span class="v-folder-name">{folderName}</span>
                      <span class="v-folder-count">({itemsInFolder.length})</span>
                      <button class="add-sub-sandbox-btn" onclick={(e) => { e.stopPropagation(); openCreateSandboxModal(folderName); }} title="Add sandbox to this folder">
                        <Plus size={11} />
                      </button>
                    </div>

                    {#if expandedVirtualFolders[folderName]}
                      <div class="folder-children-list">
                        {#each itemsInFolder as sandbox (sandbox.id)}
                          <div class="sqlite-item-row nested" class:active={activeSandboxId === sandbox.id} onclick={() => selectSandbox(sandbox)} draggable="true" ondragstart={(e) => handleDragStart(e, sandbox.id)} role="button" tabindex="0">
                            <div class="item-title-wrap">
                              <Layers size={12} class="item-icon" />
                              <span class="item-title">{sandbox.name}</span>
                            </div>
                            <button class="item-move-btn" onclick={(e) => { e.stopPropagation(); openMoveSandboxModal(sandbox.id); }} title="Move to folder"><FolderInput size={11} /></button>
                            <button class="item-delete-btn" onclick={(e) => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }}><Trash2 size={11} /></button>
                          </div>
                        {/each}
                      </div>
                    {/if}
                  </div>
                {/each}

                <div class="root-sandbox-drop-zone" ondragover={handleDragOver} ondrop={handleDropOnRoot}>
                {#each (folderGroupedSandboxes[''] || []) as sandbox (sandbox.id)}
                  <div class="sqlite-item-row" class:active={activeSandboxId === sandbox.id} onclick={() => selectSandbox(sandbox)} draggable="true" ondragstart={(e) => handleDragStart(e, sandbox.id)} role="button" tabindex="0">
                    <div class="item-title-wrap">
                      <Layers size={13} class="item-icon" />
                      <span class="item-title">{sandbox.name}</span>
                    </div>
                    <button class="item-move-btn" onclick={(e) => { e.stopPropagation(); openMoveSandboxModal(sandbox.id); }} title="Move to folder"><FolderInput size={11} /></button>
                    <button class="item-delete-btn" onclick={(e) => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }}><Trash2 size={12} /></button>
                  </div>
                {/each}
                </div>
                <div class="root-drop-hint" ondragover={handleDragOver} ondrop={handleDropOnRoot}>Drop here to move to root</div>

                {#if filteredSandboxes.length === 0}
                  <div class="sidebar-empty-hint">No sandboxes match search.</div>
                {/if}

              </div>
            </div>
          {/if}
        </div>
      </aside>
    {/if}

    <main class="editor-panel" class:full-width={activeTab === 'settings' || !showSidebar}>
      {#if activeTab === 'workspace'}
        {#if activeFilePath}
          <div class="editor-header">
            <div class="file-info"><span class="active-file-name">{activeFileName}</span></div>
            <div class="header-actions">
              {#if !isRunning}
                <button class="header-action-btn run" onclick={runActiveCode} title="Run Code">
                  <Play size={14} fill="#48bb78" stroke="none" /> <span>Run</span>
                </button>
              {:else}
                <button class="header-action-btn stop" onclick={stopActiveProcess} title="Stop">
                  <Square size={14} fill="#ff5a36" stroke="none" /> <span>Stop</span>
                </button>
              {/if}
              <button class="header-action-btn" onclick={saveWorkspaceFile} disabled={!activeFileUnsaved}>
                <Check size={14} /> <span>Save</span>
              </button>
            </div>
          </div>
          <div class="workspace-split">
            <div class="editor-body"><div class="editor-container-inner" bind:this={workspaceEditorContainer}></div></div>
          </div>

          {#if isConsoleOpen}
            <TerminalDrawer
              bind:bottomMode
              bind:consoleLogs
              bind:consoleStatus
              bind:isConsoleRunning
              bind:terminals={workspaceTerminals}
              bind:activeTermId={activeWorkspaceTermId}
              onSendTerminalInput={sendTerminalInput}
              onCloseTerminal={closeWorkspaceTerminal}
              onAddTerminal={async () => {
                const id = createWorkspaceTerminal('bash');
                try { await StartTerminalSession(id, currentFolder); } catch (_) {}
              }}
              onClearConsole={() => { consoleLogs = []; consoleStatus = 'Ready'; }}
              onMinimize={() => isConsoleOpen = false}
            />
          {:else}
            <button class="console-restore-bar" onclick={() => isConsoleOpen = true}>
              <Terminal size={12} /> <span>Show Console & Terminal</span>
            </button>
          {/if}
        {:else}
          <div class="editor-empty-state"><div class="welcome-box"><h1>Workspace</h1></div></div>
        {/if}

      {:else if activeTab === 'sandboxes'}
        {#if activeSandboxId}
          <div class="sandbox-tab-panel">
            <div class="sandbox-header-controls">
              <input type="text" class="sandbox-title-input" value={activeSandboxName} oninput={(e) => handleAutoRenameSandbox(e.target.value)} />
              <div class="sandbox-actions">
                <button
                  class="action-btn"
                  class:active={showSandboxExplorer}
                  onclick={() => showSandboxExplorer = !showSandboxExplorer}
                  title="Toggle Virtual Files Sidebar"
                >
                  <Columns size={13} />
                  <span>Files</span>
                </button>
                <button
                  class="action-btn"
                  class:active={showSandboxSplit}
                  onclick={() => showSandboxSplit = !showSandboxSplit}
                  title="Toggle Notes/Inspector Split"
                >
                  <BookOpen size={13} />
                  <span>Notes & Config</span>
                </button>
                {#if !isRunning}
                  <button class="action-btn run" onclick={runActiveCode}><Play size={13} fill="#48bb78" stroke="none" /> Run</button>
                {:else}
                  <button class="action-btn stop" onclick={stopActiveProcess}><Square size={13} fill="#ff5a36" stroke="none" /> Stop</button>
                {/if}
                <button class="action-btn primary" onclick={handleSaveSandbox} disabled={!activeSandboxUnsaved}><Save size={13} /> Save</button>
              </div>
            </div>

            <div class="sandbox-working-split">
              <div class="sandbox-main-area" style="flex: 1; display: flex; overflow: hidden; min-width: 0;">
                {#if showSandboxExplorer}
                  <div class="sandbox-virtual-explorer">
                    <div class="v-explorer-header">
                      <span>Virtual Files</span>
                      <button class="v-explorer-btn" onclick={handleCreateSandboxFile}><Plus size={13} /></button>
                    </div>
                    <div class="virtual-files-list">
                      {#each sandboxFilesList as vFile (vFile.path)}
                        <div class="v-file-row-wrap" class:active={activeSandboxFilePath === vFile.path}>
                          <button class="v-file-row" onclick={() => selectSandboxFile(vFile)}>
                            <FileCode size={13} class="v-file-icon" />
                            <span class="v-file-name">{vFile.path}</span>
                          </button>
                          <button class="v-file-delete" onclick={() => handleDeleteSandboxFile(vFile.path)}>
                            <Trash2 size={11} />
                          </button>
                        </div>
                      {/each}
                    </div>
                  </div>
                {/if}

                <div class="sandbox-editor-wrapper">
                  {#if activeSandboxFilePath}
                    <div class="sandbox-cm-container full" bind:this={sandboxEditorContainer}></div>
                  {/if}
                </div>
              </div>

              <!-- RESTORED SANDBOX NOTES, HTML CANVAS & YAML CONFIG SPLIT PANEL -->
              {#if showSandboxSplit}
                <div class="sandbox-notes-pane">
                  <div class="sandbox-notes-header">
                    <div class="tabs">
                      <button
                        class="notes-tab-btn"
                        class:active={sandboxSplitType === 'markdown'}
                        onclick={() => sandboxSplitType = 'markdown'}
                      >
                        <BookOpen size={12} />
                        Markdown Note
                      </button>
                      <button
                        class="notes-tab-btn"
                        class:active={sandboxSplitType === 'html'}
                        onclick={() => sandboxSplitType = 'html'}
                      >
                        <Sparkles size={12} />
                        HTML Canvas
                      </button>
                      <button
                        class="notes-tab-btn"
                        class:active={sandboxSplitType === 'yaml'}
                        onclick={() => sandboxSplitType = 'yaml'}
                      >
                        <Settings size={12} />
                        YAML Config
                      </button>
                    </div>
                  </div>

                  <div class="sandbox-notes-body">
                    {#if sandboxSplitType === 'markdown'}
                      <div class="split-pane-wrapper">
                        <div class="pane-action-row">
                          <span class="pane-title">MD Documentation Note</span>
                          <button
                            class="split-btn"
                            onclick={() => isSandboxMarkdownEditing = !isSandboxMarkdownEditing}
                          >
                            {#if isSandboxMarkdownEditing}
                              <Eye size={11} /> Preview
                            {:else}
                              <Edit3 size={11} /> Edit
                            {/if}
                          </button>
                        </div>
                        {#if isSandboxMarkdownEditing}
                          <textarea
                            class="notes-raw-textarea"
                            bind:value={activeSandboxMarkdownNote}
                            placeholder="Document the design goals, math proofs, or logic of this experiment..."
                          ></textarea>
                        {:else}
                          <div class="markdown-rendered-view text-view-box">
                            {#each activeSandboxMarkdownNote.split('\n') as line}
                              {@const part = renderMarkdownLine(line)}
                              {#if part.type === 'h1'}
                                <h1>{part.text}</h1>
                              {:else if part.type === 'h2'}
                                <h2>{part.text}</h2>
                              {:else if part.type === 'h3'}
                                <h3>{part.text}</h3>
                              {:else if part.type === 'quote'}
                                <blockquote>{part.text}</blockquote>
                              {:else if part.type === 'li'}
                                <li>{part.text}</li>
                              {:else if part.type === 'codeblock'}
                                <pre class="inline-codeblock"><code>{part.text}</code></pre>
                              {:else if part.type === 'br'}
                                <br />
                              {:else}
                                <p>{part.text}</p>
                              {/if}
                            {/each}
                            {#if !activeSandboxMarkdownNote}
                              <p class="empty-notif">No documentation note added yet.</p>
                            {/if}
                          </div>
                        {/if}
                      </div>

                    {:else if sandboxSplitType === 'html'}
                      <div class="split-pane-wrapper">
                        <div class="pane-action-row">
                          <span class="pane-title">Interactive Visual Canvas</span>
                          <button
                            class="split-btn"
                            onclick={() => isSandboxHTMLEditing = !isSandboxHTMLEditing}
                          >
                            {#if isSandboxHTMLEditing}
                              <Eye size={11} /> View Live
                            {:else}
                              <Edit3 size={11} /> Edit HTML
                            {/if}
                          </button>
                        </div>
                        {#if isSandboxHTMLEditing}
                          <textarea
                            class="notes-raw-textarea code-family"
                            bind:value={activeSandboxHTMLNote}
                            placeholder="Insert custom HTML widgets, SVG graphics, or JavaScript drawing loops..."
                          ></textarea>
                        {:else}
                          <div class="html-preview-frame-container">
                            <iframe
                              title="Concept Interactive Visualization Frame"
                              srcdoc={activeSandboxHTMLNote}
                              sandbox="allow-scripts"
                              class="html-canvas-iframe"
                            ></iframe>
                          </div>
                        {/if}
                      </div>

                    {:else if sandboxSplitType === 'yaml'}
                      <div class="yaml-config-panel">
                        <div class="yaml-config-header">
                          <h3>Environment Rules (YAML)</h3>
                          <p>
                            Configure execution rules. Set <code>run.command</code> to dictate build and run steps.
                          </p>
                        </div>
                        <textarea
                          class="yaml-textarea"
                          bind:value={activeSandboxConfig}
                          placeholder={'name: "My Sandbox"\nenvironment: "python"\nrun:\n  command: "python3 main.py"'}
                        ></textarea>
                        {#if activeSandboxConfigUnsaved}
                          <p class="yaml-unsaved-hint">Config has unsaved changes.</p>
                        {/if}
                      </div>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>

            {#if isConsoleOpen}
              <TerminalDrawer
                bind:bottomMode
                bind:consoleLogs
                bind:consoleStatus
                bind:isConsoleRunning
                bind:terminals={sandboxTerminals}
                bind:activeTermId={activeSandboxTermId}
                onSendTerminalInput={sendTerminalInput}
                onCloseTerminal={closeSandboxTerminal}
                onAddTerminal={async () => {
                  const id = createSandboxTerminal('bash');
                  try {
                    const sandboxDir = await GetSandboxDirectory(activeSandboxId);
                    await StartTerminalSession(id, sandboxDir);
                  } catch (_) {}
                }}
                onClearConsole={() => { consoleLogs = []; consoleStatus = 'Ready'; }}
                onMinimize={() => isConsoleOpen = false}
              />
            {:else}
              <button class="console-restore-bar" onclick={() => isConsoleOpen = true}>
                <Terminal size={12} /> <span>Show Console & Terminal</span>
              </button>
            {/if}
          </div>
        {/if}

      {:else if activeTab === 'settings'}
        <div class="settings-view-panel">
          <div class="settings-card">
            <h2>CrabCode System Base Location</h2>
            <div class="settings-input-row">
              <input type="text" class="settings-path-input" bind:value={settingsCrabRootPath} oninput={checkFolderState} />
              <button class="settings-browse-btn" onclick={browseCrabRootPath}>Browse</button>
            </div>
            {#if isCrabFolderEmpty}
              <button class="initialize-btn" onclick={initializeFolderStructure}>Initialize Folder Structure</button>
            {/if}
            <button class="settings-save-btn" onclick={saveGlobalConfig} disabled={isCrabFolderEmpty}>Save Changes</button>
          </div>
        </div>
      {/if}

      <footer class="status-bar">
        <div class="status-left">
          <span class="status-indicator" class:running={isRunning}></span>
          <span class="status-text">{statusMessage}</span>
        </div>
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
    font-size: 15px;
    font-weight: 800;
    color: #ff5a36;
    letter-spacing: 0.3px;
    white-space: nowrap;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .sidebar-toggle-btn {
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

  .sidebar-toggle-btn:hover,
  .sidebar-toggle-btn.active {
    background-color: #2d3748;
    color: #edf2f7;
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

  .notes-explorer {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .sidebar-search-input {
    background-color: #12121a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    color: #edf2f7;
    padding: 6px 10px;
    font-size: 12px;
    outline: none;
    margin: 0 6px 8px 6px;
    font-family: 'Inter', sans-serif;
  }

  .sidebar-search-input:focus {
    border-color: #ff5a36;
  }

  .notes-list-scroll,
  .sandboxes-list-scroll {
    flex: 1;
    overflow-y: auto;
  }

  .sqlite-item-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    cursor: pointer;
    font-size: 13px;
    color: #9ca3af;
    border-radius: 4px;
    margin: 1px 6px;
    transition: background-color 0.1s;
  }

  .sqlite-item-row:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .sqlite-item-row.active {
    background-color: #ff5a3611;
    color: #edf2f7;
    border-left: 2px solid #ff5a36;
  }

  .item-title-wrap {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    flex: 1;
  }

  .item-icon {
    flex-shrink: 0;
    color: #6b7280;
  }

  .item-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .item-delete-btn {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.1s, color 0.1s;
  }

  .sqlite-item-row:hover .item-delete-btn {
    opacity: 1;
  }

  .item-delete-btn:hover {
    color: #ff5a36;
  }

  .sidebar-empty-hint {
    padding: 20px;
    text-align: center;
    color: #4a5568;
    font-size: 12px;
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

  .editor-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background-color: #0b0b0f;
    min-width: 0;
  }

  .editor-panel.full-width {
    width: 100%;
  }

  .editor-header {
    height: 40px;
    min-height: 40px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
  }

  .file-info {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .active-file-name {
    font-size: 13px;
    font-weight: 600;
    color: #edf2f7;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-action-btn {
    background: none;
    border: none;
    color: #9ca3af;
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: background-color 0.15s, color 0.15s;
  }

  .header-action-btn:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .header-action-btn.run {
    color: #48bb78;
  }

  .header-action-btn.run:hover {
    background-color: #48bb7818;
  }

  .header-action-btn.stop {
    color: #ff5a36;
  }

  .header-action-btn.stop:hover {
    background-color: #ff5a3618;
  }

  .header-action-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
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
    flex-direction: column;
    overflow: hidden;
    min-width: 0;
  }

  .editor-container-inner {
    flex: 1;
    overflow-y: auto;
  }

  .editor-empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .welcome-box {
    text-align: center;
    color: #718096;
  }

  .welcome-box h1 {
    margin: 0 0 8px 0;
    font-size: 24px;
    color: #9ca3af;
    font-weight: 600;
  }

  .console-restore-bar {
    height: 28px;
    background-color: #0c0c12;
    border-top: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    color: #6b7280;
    font-size: 11px;
    cursor: pointer;
    border-bottom: none;
    border-left: none;
    border-right: none;
    width: 100%;
    transition: color 0.15s, background-color 0.15s;
    flex-shrink: 0;
  }

  .console-restore-bar:hover {
    color: #edf2f7;
    background-color: #12121a;
  }

  .toast-container {
    position: fixed;
    bottom: 40px;
    right: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    z-index: 9999;
    pointer-events: none;
  }

  .toast {
    background-color: #1a1a26;
    border: 1px solid #2d3748;
    border-radius: 6px;
    padding: 8px 16px;
    color: #edf2f7;
    font-size: 12px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.4);
    pointer-events: all;
    animation: slideIn 0.2s ease-out;
  }

  .toast.success {
    border-left: 3px solid #48bb78;
  }

  .toast.error {
    border-left: 3px solid #ff5a36;
  }

  .toast.info {
    border-left: 3px solid #569CD6;
  }

  .toast-message {
    line-height: 1.4;
  }

  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
  }

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9998;
  }

  .modal-box {
    background-color: #14141c;
    border: 1px solid #2d3748;
    border-radius: 8px;
    width: 420px;
    max-width: 90vw;
    box-shadow: 0 8px 32px rgba(0,0,0,0.5);
  }

  .modal-header {
    padding: 16px 20px 8px 20px;
    font-size: 15px;
    font-weight: 700;
    color: #edf2f7;
    border-bottom: 1px solid #1a1a26;
  }

  .modal-body {
    padding: 16px 20px;
  }

  .modal-input {
    width: 100%;
    background-color: #12121a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    color: #edf2f7;
    padding: 8px 12px;
    font-size: 13px;
    outline: none;
    font-family: 'Inter', sans-serif;
    box-sizing: border-box;
  }

  .modal-input:focus {
    border-color: #ff5a36;
  }

  .modal-footer {
    padding: 12px 20px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    border-top: 1px solid #1a1a26;
  }

  .modal-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #9ca3af;
    padding: 6px 14px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    transition: background-color 0.15s, color 0.15s;
  }

  .modal-btn:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .modal-btn.primary {
    background-color: #ff5a36;
    border-color: #ff5a36;
    color: white;
  }

  .modal-btn.primary:hover {
    background-color: #e04b28;
  }

  .modal-btn.secondary:hover {
    background-color: #2d3748;
  }

  .move-modal {
    width: 400px;
  }

  .move-folder-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 300px;
    overflow-y: auto;
  }

  .move-folder-option {
    display: flex;
    align-items: center;
    gap: 10px;
    background: none;
    border: 1px solid #2d3748;
    border-radius: 6px;
    padding: 10px 12px;
    color: #9ca3af;
    font-size: 13px;
    cursor: pointer;
    transition: background-color 0.15s, border-color 0.15s, color 0.15s;
    width: 100%;
    text-align: left;
  }

  .move-folder-option:hover {
    background-color: #1a1a24;
    border-color: #4a5568;
    color: #edf2f7;
  }

  .move-folder-option .v-folder-icon {
    color: #ff8c73;
    flex-shrink: 0;
  }

  .move-folder-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .move-folder-count {
    font-size: 11px;
    color: #6b7280;
    flex-shrink: 0;
  }

  .move-new-folder-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
    padding-top: 12px;
    border-top: 1px solid #1a1a26;
  }

  .create-folder-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
  }

  .root-sandbox-drop-zone {
    min-height: 24px;
  }

  .root-drop-hint {
    font-size: 11px;
    color: #4a5568;
    text-align: center;
    padding: 6px;
    border: 1px dashed #2d3748;
    border-radius: 4px;
    margin: 4px 8px;
    transition: border-color 0.15s, color 0.15s;
  }

  .root-drop-hint:hover,
  .root-drop-hint--active {
    border-color: #ff5a3655;
    color: #9ca3af;
  }

  .guide-modal {
    width: 520px;
  }

  .guide-body {
    max-height: 60vh;
    overflow-y: auto;
    font-size: 13px;
    line-height: 1.6;
    color: #cbd5e0;
  }

  .guide-body code {
    background-color: #1a1a26;
    padding: 1px 4px;
    border-radius: 3px;
    font-size: 12px;
    color: #ff5a36;
  }

  .sandbox-modal {
    width: 480px;
  }

  .sandbox-modal-label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: #9ca3af;
    margin-bottom: 6px;
    margin-top: 12px;
  }

  .sandbox-modal-label:first-child {
    margin-top: 0;
  }

  .templates-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
    max-height: 200px;
    overflow-y: auto;
    margin-top: 6px;
  }

  .template-option {
    display: flex;
    align-items: center;
    gap: 6px;
    background-color: #12121a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 6px 10px;
    cursor: pointer;
    color: #9ca3af;
    font-size: 12px;
    transition: border-color 0.15s, background-color 0.15s;
  }

  .template-option:hover {
    border-color: #4a5568;
    background-color: #1a1a26;
  }

  .template-option.selected {
    border-color: #ff5a36;
    background-color: #ff5a3611;
  }

  .template-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .template-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .settings-view-panel {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding: 40px;
    overflow-y: auto;
  }

  .settings-card {
    background-color: #0c0c12;
    border: 1px solid #1a1a24;
    border-radius: 8px;
    padding: 32px;
    max-width: 600px;
    width: 100%;
  }

  .settings-card h2 {
    margin: 0 0 8px 0;
    font-size: 18px;
    font-weight: 600;
    color: #edf2f7;
  }

  .settings-input-row {
    display: flex;
    gap: 8px;
    margin: 16px 0;
  }

  .settings-path-input {
    flex: 1;
    background-color: #12121a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    color: #edf2f7;
    padding: 8px 12px;
    font-size: 13px;
    outline: none;
    font-family: 'Inter', sans-serif;
  }

  .settings-path-input:focus {
    border-color: #ff5a36;
  }

  .settings-browse-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #9ca3af;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    transition: background-color 0.15s, color 0.15s;
    white-space: nowrap;
  }

  .settings-browse-btn:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .initialize-btn {
    background-color: #ff5a36;
    border: none;
    color: white;
    padding: 8px 20px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    transition: background-color 0.15s;
    margin-bottom: 16px;
  }

  .initialize-btn:hover {
    background-color: #e04b28;
  }

  .settings-save-btn {
    background-color: #ff5a36;
    border: none;
    color: white;
    padding: 10px 24px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 600;
    transition: background-color 0.15s;
  }

  .settings-save-btn:hover {
    background-color: #e04b28;
  }

  .settings-save-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .status-bar {
    height: 24px;
    min-height: 24px;
    background-color: #0c0c12;
    border-top: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    font-size: 11px;
    color: #6b7280;
  }

  .status-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-indicator {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: #4b5563;
    flex-shrink: 0;
  }

  .status-indicator.running {
    background-color: #48bb78;
    box-shadow: 0 0 4px #48bb78;
  }

  .status-text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Sandbox tab styles */
  .sandbox-tab-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  .sandbox-header-controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    min-height: 40px;
    gap: 12px;
  }

  .sandbox-title-input {
    background: transparent;
    border: 1px solid transparent;
    color: #edf2f7;
    font-size: 14px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    outline: none;
    font-family: 'Inter', sans-serif;
    min-width: 200px;
  }

  .sandbox-title-input:hover {
    border-color: #2d3748;
  }

  .sandbox-title-input:focus {
    border-color: #ff5a36;
    background-color: #12121a;
  }

  .sandbox-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .action-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #9ca3af;
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: background-color 0.15s, color 0.15s;
    white-space: nowrap;
  }

  .action-btn:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .action-btn.run {
    color: #48bb78;
  }

  .action-btn.run:hover {
    background-color: #48bb7818;
  }

  .action-btn.stop {
    color: #ff5a36;
  }

  .action-btn.stop:hover {
    background-color: #ff5a3618;
  }

  .action-btn.primary {
    color: #ff5a36;
    border-color: #ff5a3655;
  }

  .action-btn.primary:hover {
    background-color: #ff5a3618;
  }

  .action-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .sandbox-working-split {
    flex: 1;
    display: flex;
    overflow: hidden;
    min-height: 0;
  }

  .sandbox-virtual-explorer {
    width: 200px;
    min-width: 160px;
    max-width: 260px;
    background-color: #0c0c12;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
  }

  .v-explorer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    letter-spacing: 0.3px;
    border-bottom: 1px solid #1a1a24;
    flex-shrink: 0;
  }

  .v-explorer-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 2px 6px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    transition: background-color 0.15s, color 0.15s;
  }

  .v-explorer-btn:hover {
    background-color: #2d3748;
    color: #edf2f7;
  }

  .virtual-files-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
  }

  .v-file-row-wrap {
    display: flex;
    align-items: center;
    padding: 3px 8px;
    cursor: pointer;
    transition: background-color 0.1s;
  }

  .v-file-row-wrap:hover {
    background-color: #1a1a24;
  }

  .v-file-row-wrap.active {
    background-color: #ff5a3611;
    border-left: 2px solid #ff5a36;
  }

  .v-file-row {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
    background: none;
    border: none;
    color: #9ca3af;
    font-size: 12px;
    cursor: pointer;
    padding: 2px 0;
  }

  .v-file-icon {
    flex-shrink: 0;
    color: #6b7280;
  }

  .v-file-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .v-file-delete {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.1s, color 0.1s;
  }

  .v-file-row-wrap:hover .v-file-delete {
    opacity: 1;
  }

  .v-file-delete:hover {
    color: #ff5a36;
  }

  .sandbox-editor-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 0;
  }

  .sandbox-cm-container.full {
    flex: 1;
    overflow: hidden;
  }

  /* Restored notes pane */
  .sandbox-notes-pane {
    width: 320px;
    min-width: 260px;
    max-width: 400px;
    background-color: #0c0c12;
    border-left: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .sandbox-notes-header {
    border-bottom: 1px solid #1a1a24;
    padding: 4px 8px;
    flex-shrink: 0;
  }

  .sandbox-notes-header .tabs {
    display: flex;
    gap: 2px;
  }

  .notes-tab-btn {
    background: none;
    border: 1px solid transparent;
    color: #6b7280;
    padding: 6px 10px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: background-color 0.1s, color 0.1s;
    white-space: nowrap;
  }

  .notes-tab-btn:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .notes-tab-btn.active {
    background-color: #ff5a3611;
    color: #ff5a36;
    border-color: #ff5a3633;
  }

  .sandbox-notes-body {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .split-pane-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .pane-action-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    border-bottom: 1px solid #1a1a24;
    flex-shrink: 0;
  }

  .pane-title {
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .split-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #9ca3af;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 10px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: background-color 0.15s, color 0.15s;
  }

  .split-btn:hover {
    background-color: #2d3748;
    color: #edf2f7;
  }

  .notes-raw-textarea {
    flex: 1;
    background-color: #12121a;
    border: none;
    color: #e2e8f0;
    padding: 12px;
    font-size: 12px;
    font-family: 'Inter', sans-serif;
    resize: none;
    outline: none;
    line-height: 1.5;
  }

  .notes-raw-textarea.code-family {
    font-family: 'Fira Code', 'JetBrains Mono', monospace;
    font-size: 11px;
  }

  .markdown-rendered-view {
    flex: 1;
    overflow-y: auto;
    padding: 12px;
    font-size: 12px;
    line-height: 1.5;
    color: #cbd5e0;
  }

  .markdown-rendered-view h1 { font-size: 16px; margin: 0 0 8px 0; color: #edf2f7; }
  .markdown-rendered-view h2 { font-size: 14px; margin: 0 0 6px 0; color: #edf2f7; }
  .markdown-rendered-view h3 { font-size: 13px; margin: 0 0 4px 0; color: #edf2f7; }
  .markdown-rendered-view p { margin: 0 0 6px 0; }
  .markdown-rendered-view blockquote {
    border-left: 3px solid #ff5a36;
    padding-left: 12px;
    margin: 8px 0;
    color: #9ca3af;
    font-style: italic;
  }
  .markdown-rendered-view li { margin: 2px 0 2px 16px; }
  .markdown-rendered-view pre.inline-codeblock {
    background-color: #1a1a26;
    padding: 8px;
    border-radius: 4px;
    overflow-x: auto;
    margin: 8px 0;
  }
  .markdown-rendered-view pre.inline-codeblock code {
    font-family: 'Fira Code', 'JetBrains Mono', monospace;
    font-size: 11px;
    color: #e2e8f0;
  }

  .text-view-box {
    border: none;
    background-color: transparent;
  }

  .empty-notif {
    color: #6b7280;
    font-style: italic;
    font-size: 11px;
  }

  .html-preview-frame-container {
    flex: 1;
    overflow: hidden;
    background-color: #ffffff;
  }

  .html-canvas-iframe {
    width: 100%;
    height: 100%;
    border: none;
    display: block;
  }

  .yaml-config-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .yaml-config-header {
    padding: 8px 12px;
    border-bottom: 1px solid #1a1a24;
    flex-shrink: 0;
  }

  .yaml-config-header h3 {
    margin: 0 0 4px 0;
    font-size: 12px;
    font-weight: 700;
    color: #edf2f7;
  }

  .yaml-config-header p {
    margin: 0;
    font-size: 11px;
    color: #6b7280;
    line-height: 1.4;
  }

  .yaml-config-header code {
    background-color: #1a1a26;
    padding: 1px 4px;
    border-radius: 3px;
    font-size: 10px;
    color: #ff5a36;
  }

  .yaml-textarea {
    flex: 1;
    background-color: #12121a;
    border: none;
    color: #e2e8f0;
    padding: 12px;
    font-size: 11px;
    font-family: 'Fira Code', 'JetBrains Mono', monospace;
    resize: none;
    outline: none;
    line-height: 1.6;
    tab-size: 2;
  }

  .yaml-unsaved-hint {
    padding: 6px 12px;
    margin: 0;
    font-size: 10px;
    color: #ff5a36;
    background-color: #ff5a3611;
    border-top: 1px solid #ff5a3622;
    flex-shrink: 0;
  }

  .action-btn.active {
    border-color: #ff5a3655;
    color: #ff5a36;
    background-color: #ff5a3611;
  }

  .sandbox-fs-explorer {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
    gap: 8px;
  }

  .virtual-folder-group {
    display: flex;
    flex-direction: column;
    margin-bottom: 2px;
  }

  .virtual-folder-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    border-radius: 4px;
    cursor: pointer;
    color: #a0aec0;
    font-size: 12px;
    font-weight: 600;
  }

  .virtual-folder-header:hover {
    background-color: #1a1a24;
    color: #edf2f7;
  }

  .folder-arrow {
    font-size: 8px;
    color: #718096;
  }

  :global(.v-folder-icon) {
    color: #ff8c73;
  }

  .v-folder-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .v-folder-count {
    font-size: 10px;
    color: #6b7280;
  }

  .add-sub-sandbox-btn {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    padding: 2px;
  }

  .add-sub-sandbox-btn:hover {
    color: #ff5a36;
  }

  .folder-children-list {
    padding-left: 14px;
    border-left: 1px solid #1c1c28;
    margin-left: 12px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .sqlite-item-row.nested {
    padding: 4px 8px;
    font-size: 12px;
  }

  .item-move-btn {
    background: none;
    border: none;
    color: #4a5568;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
  }

  .item-move-btn:hover {
    color: #ff8c73;
    background-color: #ff8c7318;
  }
</style>
