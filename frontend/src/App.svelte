<script>
  import './style.css';
  import './app.css';
  import {
    GetWorkspaces,
    CreateWorkspace,
    DeleteWorkspace,
    ExportSandbox,
    ImportSandbox,
    BackupWorkspace,
    RestoreWorkspace,
    GetSandboxes,
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
    StopCommand,
    StartTerminalSession,
    WriteTerminalInput,
    IsEnvironmentInitialized,
    InitializeEnvironment,
    GetGlobalSettings,
    GetTemplates
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import TerminalDrawer from './TerminalDrawer.svelte';
  import { onMount, onDestroy, untrack } from 'svelte';
  import {
    Play, Square, Settings, Database, BookOpen, Plus, Trash2, Columns,
    PanelLeft, ChevronLeft, Save, Sparkles, FileCode, Layers, Eye, Edit3, Folder,
    FolderPlus, Download, Upload, FlaskConical, Archive, FolderInput
  } from '@lucide/svelte';

  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState, Compartment } from '@codemirror/state';
  import { keymap } from '@codemirror/view';
  import { indentWithTab } from '@codemirror/commands';
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

  let activeTab = $state('lab'); // 'lab', 'workspaces', 'settings'

  // Workspace DB State
  let workspacesList = $state([]);
  let activeWorkspaceId = $state('default');
  let activeWorkspaceObj = $derived(workspacesList.find(w => w.id === activeWorkspaceId) || null);

  let statusMessage = $state('CrabCode Laboratory Ready');
  let showSidebar = $state(true);

  // Sandboxes State
  let sandboxesList = $state([]);
  let activeSandboxId = $state('');
  let activeSandboxName = $state('');
  let activeSandboxConfig = $state('');
  let lastSavedSandboxConfig = $state('');
  let activeSandboxMarkdownNote = $state('');
  let activeSandboxHTMLNote = $state('');
  let lastSavedSandboxMarkdownNote = $state('');
  let lastSavedSandboxHTMLNote = $state('');

  // Virtual Folders & Files State
  let sandboxSearchQuery = $state('');
  let virtualFolders = $state([]);
  let expandedVirtualFolders = $state({});

  let showSandboxExplorer = $state(true);
  let showSandboxSplit = $state(true);
  let sandboxSplitType = $state('markdown'); // 'markdown', 'html', 'yaml'
  let isSandboxMarkdownEditing = $state(false);
  let isSandboxHTMLEditing = $state(false);

  let sandboxFilesList = $state([]);
  let activeSandboxFilePath = $state('');
  let activeSandboxFileName = $state('');
  let activeSandboxFileContent = $state('');
  let lastSavedSandboxFileContent = $state('');

  let settingsCrabRootPath = $state('');
  let toasts = $state([]);
  let modal = $state({ show: false, title: '', placeholder: '', value: '', onConfirm: null, onCancel: null });
  let showCreateSandboxModal = $state(false);
  let newSandboxName = $state('');
  let selectedSandboxFolder = $state('');
  let selectedSandboxTemplate = $state('python');
  let availableTemplates = $state([]);

  let showCreateWorkspaceModal = $state(false);
  let newWorkspaceName = $state('');
  let newWorkspaceDesc = $state('');

  let showMoveSandboxModal = $state(false);
  let moveSandboxId = $state('');

  let showInitEnvModal = $state(false);
  let isInitializingEnv = $state(false);
  let isEnvInitialized = $state(true);

  let sandboxEditorContainer = $state(null);
  let sandboxView = null;
  const sandboxLanguageConf = new Compartment();

  let isConsoleOpen = $state(true);

  // Multi-terminal state
  let sandboxTerminals = $state([]);
  let activeSandboxTermId = $state('');

  let activeSandboxTerm = $derived(
    sandboxTerminals.find(t => t.id === activeSandboxTermId) || null
  );

  let bottomMode = $state('console');
  let consoleLogs = $state([]);
  let consoleStatus = $state('Ready');
  let isConsoleRunning = $state(false);
  let activeRunnerId = $state('');

  let isRunning = $derived(activeRunnerId !== '');

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
    '.cm-content': { caretColor: '#ff5a36', fontFamily: "'Fira Code', 'JetBrains Mono', monospace", fontSize: '13px', padding: '12px 0' },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#ff5a36', borderLeftWidth: '2px' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: '#ff5a3622 !important' },
    '.cm-gutters': { backgroundColor: '#14141a', color: '#6b7280', borderRight: '1px solid #20202b', fontSize: '13px', paddingTop: '12px', paddingBottom: '12px' },
    '.cm-gutterElement': { padding: '0 10px 0 12px' },
    '.cm-activeLine': { backgroundColor: '#ffffff03' }
  }, { dark: true });

  const crabHighlightStyle = HighlightStyle.define([
    { tag: t.keyword, color: '#569CD6', fontWeight: '600' },
    { tag: t.comment, color: '#6A9955', fontStyle: 'italic' },
    { tag: t.string, color: '#CE9178' },
    { tag: t.number, color: '#B5CEA8' },
    { tag: t.className, color: '#4EC9B0' },
    { tag: t.function(t.variableName), color: '#DCDCAA' },
    { tag: t.variableName, color: '#9CDCFE' },
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

  function updateSandboxEditor(content, fileName) {
    if (!sandboxView) return;
    sandboxView.setState(EditorState.create({
      doc: content,
      extensions: buildSandboxExtensions(fileName)
    }));
  }

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
    setTimeout(() => { toasts = toasts.filter(item => item.id !== id); }, duration);
  }

  function openPrompt(title, placeholder, defaultValue = '') {
    return new Promise((resolve) => {
      modal = {
        show: true, title, placeholder, value: defaultValue,
        onConfirm: (val) => { modal.show = false; resolve(val); },
        onCancel: () => { modal.show = false; resolve(null); }
      };
    });
  }

  async function loadWorkspaces() {
    try {
      workspacesList = await GetWorkspaces();
      if (workspacesList.length > 0 && !activeWorkspaceId) {
        activeWorkspaceId = workspacesList[0].id;
      }
      await loadSandboxesForWorkspace(activeWorkspaceId);
    } catch (err) {
      addToast('Failed to load workspaces: ' + String(err), 'error');
    }
  }

  async function loadSandboxesForWorkspace(wsId) {
    try {
      sandboxesList = await GetSandboxes(wsId);
      const foldersSet = new Set(sandboxesList.map(s => s.folder).filter(Boolean));
      virtualFolders = Array.from(foldersSet);

      if (sandboxesList.length > 0) {
        await selectSandbox(sandboxesList[0]);
      } else {
        clearActiveSandbox();
      }
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function switchWorkspace(wsId) {
    if (activeWorkspaceId === wsId) return;
    activeWorkspaceId = wsId;
    addToast('Switched lab workspace', 'info');
    await loadSandboxesForWorkspace(wsId);
  }

  async function handleCreateWorkspace() {
    if (!newWorkspaceName.trim()) return;
    try {
      const ws = await CreateWorkspace(newWorkspaceName.trim(), newWorkspaceDesc.trim());
      showCreateWorkspaceModal = false;
      newWorkspaceName = '';
      newWorkspaceDesc = '';
      workspacesList = [ws, ...workspacesList];
      await switchWorkspace(ws.id);
      addToast('Workspace created', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDeleteWorkspace(id) {
    if (id === 'default') {
      addToast('Default workspace cannot be deleted', 'error');
      return;
    }
    if (!confirm('Delete this workspace and all sandboxes inside it?')) return;
    try {
      await DeleteWorkspace(id);
      workspacesList = workspacesList.filter(w => w.id !== id);
      addToast('Workspace deleted', 'success');
      await switchWorkspace('default');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  function clearActiveSandbox() {
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

  async function refreshSandboxFiles() {
    if (!activeSandboxId) return;
    try {
      const files = await GetSandboxFiles(activeSandboxId);
      sandboxFilesList = files.filter(f => !f.isDir);
      if (activeSandboxFilePath) {
        const stillExists = sandboxFilesList.find(f => f.path === activeSandboxFilePath);
        if (!stillExists) {
          activeSandboxFilePath = '';
          activeSandboxFileName = '';
          activeSandboxFileContent = '';
          lastSavedSandboxFileContent = '';
        }
      }
      if (!activeSandboxFilePath && sandboxFilesList.length > 0) {
        const firstFile = sandboxFilesList[0];
        activeSandboxFilePath = firstFile.path;
        activeSandboxFileName = firstFile.path.split('/').pop() || firstFile.path;
        activeSandboxFileContent = firstFile.content;
        lastSavedSandboxFileContent = firstFile.content;
        updateSandboxEditor(firstFile.content, activeSandboxFileName);
      }
    } catch (_) {}
  }

  async function createVirtualFolder() {
    const folderName = await openPrompt('New Virtual Folder', 'UI Experiments');
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

  async function openCreateSandboxModal(defaultFolder = '') {
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
      const sandbox = await CreateSandboxInFolder(activeWorkspaceId, name, selectedSandboxTemplate, selectedSandboxFolder);
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
        addToast('Failed to save title: ' + String(err), 'error');
      }
    };

    clearTimeout(renameTimer);
    if (immediate) doSave();
    else renameTimer = setTimeout(doSave, 500);
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

    activeSandboxFilePath = '';
    activeSandboxFileName = '';
    activeSandboxFileContent = '';
    lastSavedSandboxFileContent = '';

    try {
      isEnvInitialized = await IsEnvironmentInitialized(sandbox.id);
    } catch (_) { isEnvInitialized = false; }

    if (!isEnvInitialized) showInitEnvModal = true;

    await refreshSandboxFiles();

    if (sandboxTerminals.length === 0) {
      try {
        const sandboxDir = await GetSandboxDirectory(activeSandboxId);
        const sbTermId = createSandboxTerminal('bash', sandboxDir);
        await StartTerminalSession(sbTermId, sandboxDir);
      } catch (_) {}
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
    const name = await openPrompt('New Virtual File', 'main.py');
    if (!name) return;
    try {
      await SaveSandboxFile(activeSandboxId, name, '# Sandbox virtual file\n', false);
      await refreshSandboxFiles();
      const created = sandboxFilesList.find(f => f.path === name);
      if (created) await selectSandboxFile(created);
      addToast('Virtual file created', 'success');
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
      await refreshSandboxFiles();
      addToast('Virtual file deleted', 'success');
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
        await refreshSandboxFiles();
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

      addToast('Saved to database', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleDeleteSandbox(id) {
    if (!confirm('Delete this experiment permanently?')) return;
    try {
      await DeleteSandbox(id);
      sandboxesList = sandboxesList.filter(s => s.id !== id);
      if (activeSandboxId === id) clearActiveSandbox();
      addToast('Sandbox deleted', 'success');
    } catch (err) {
      addToast(String(err), 'error');
    }
  }

  async function handleExportSandbox(id) {
    try {
      const jsonStr = await ExportSandbox(id);
      const blob = new Blob([jsonStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `sandbox_${id}.json`;
      a.click();
      addToast('Sandbox exported', 'success');
    } catch (err) { addToast(String(err), 'error'); }
  }

  async function handleImportSandboxFile(e) {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = async (evt) => {
      try {
        const imported = await ImportSandbox(activeWorkspaceId, evt.target.result);
        await loadSandboxesForWorkspace(activeWorkspaceId);
        await selectSandbox(imported);
        addToast('Sandbox imported', 'success');
      } catch (err) { addToast(String(err), 'error'); }
    };
    reader.readAsText(file);
  }

  async function handleBackupWorkspace(wsId) {
    try {
      const jsonStr = await BackupWorkspace(wsId);
      const blob = new Blob([jsonStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workspace_backup_${wsId}.json`;
      a.click();
      addToast('Workspace backup created', 'success');
    } catch (err) { addToast(String(err), 'error'); }
  }

  async function handleRestoreWorkspaceFile(e) {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = async (evt) => {
      try {
        const restored = await RestoreWorkspace(evt.target.result);
        await loadWorkspaces();
        await switchWorkspace(restored.id);
        addToast('Workspace restored', 'success');
      } catch (err) { addToast(String(err), 'error'); }
    };
    reader.readAsText(file);
  }

  function createSandboxTerminal(title = 'bash', dir = '') {
    const id = 'shell_sb_' + Date.now();
    const term = { id, logs: [], isRunning: true, title, inputBuffer: '', dir: dir };
    sandboxTerminals = [...sandboxTerminals, term];
    activeSandboxTermId = id;
    return id;
  }

  async function closeSandboxTerminal(id) {
    const term = sandboxTerminals.find(t => t.id === id);
    if (term && term.isRunning) {
      try { await StopCommand(id); } catch (_) {}
    }
    sandboxTerminals = sandboxTerminals.filter(t => t.id !== id);
    if (activeSandboxTermId === id) {
      activeSandboxTermId = sandboxTerminals.length > 0 ? sandboxTerminals[sandboxTerminals.length - 1].id : '';
    }
  }

  async function sendTerminalInput() {
    const term = activeSandboxTerm;
    if (!term || !term.inputBuffer) return;
    const input = term.inputBuffer.trim();
    WriteTerminalInput(term.id, input);
    term.logs = [...term.logs, '$ ' + input];
    if (term.logs.length > 1000) term.logs = term.logs.slice(-1000);
    term.inputBuffer = '';
  }

  async function runActiveCode() {
    bottomMode = 'console';
    isConsoleOpen = true;
    consoleStatus = 'Executing...';
    isConsoleRunning = true;

    if (!activeSandboxId) return;

    try {
      const isInit = await IsEnvironmentInitialized(activeSandboxId);
      if (!isInit) {
        showInitEnvModal = true;
        consoleStatus = 'Uninitialized Env';
        isConsoleRunning = false;
        return;
      }

      const runnerId = 'runner_sb_' + Date.now();
      activeRunnerId = runnerId;
      consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Running ${activeSandboxName}...]`, type: 'info' }];

      await handleSaveSandbox();
      await RunSandbox(runnerId, activeSandboxId, activeSandboxFilePath);
    } catch (err) {
      if (String(err).includes("ENV_NOT_INITIALIZED")) {
        showInitEnvModal = true;
        consoleStatus = 'Uninitialized Env';
      } else {
        consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Error] ${String(err)}`, type: 'error' }];
        consoleStatus = 'Error';
      }
      isConsoleRunning = false;
      activeRunnerId = '';
    }
  }

  async function stopActiveProcess() {
    if (activeRunnerId) {
      try { await StopCommand(activeRunnerId); } catch (_) {}
    }
    consoleLogs = [...consoleLogs, { time: Date.now(), text: '[Stopped]', type: 'error' }];
    consoleStatus = 'Stopped';
    isConsoleRunning = false;
    activeRunnerId = '';
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

  onMount(async () => {
    try {
      const settings = await GetGlobalSettings();
      settingsCrabRootPath = settings.crabRootPath;
    } catch (_) {}

    await loadWorkspaces();

    EventsOn('terminal_output', (data) => {
      if (data.id.startsWith('runner_')) {
        consoleLogs = [...consoleLogs, { time: Date.now(), text: data.line, type: 'out' }];
      } else if (data.id.startsWith('shell_')) {
        let sbIdx = sandboxTerminals.findIndex(t => t.id === data.id);
        if (sbIdx !== -1) {
          sandboxTerminals[sbIdx] = {
            ...sandboxTerminals[sbIdx],
            logs: [...sandboxTerminals[sbIdx].logs, data.line]
          };
        }
      }
    });

    EventsOn('terminal_status', (data) => {
      if (data.id.startsWith('runner_')) {
        consoleStatus = data.status === '0' ? 'Finished' : 'Error';
        isConsoleRunning = false;
        activeRunnerId = '';
      }
    });
  });

  onDestroy(() => {
    if (sandboxView) sandboxView.destroy();
  });
</script>

<div class="toast-container">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}"><span class="toast-message">{toast.message}</span></div>
  {/each}
</div>

{#if modal.show}
  <div class="modal-backdrop" onclick={modal.onCancel} role="presentation">
    <div class="modal-box" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">{modal.title}</div>
      <div class="modal-body">
        <input type="text" class="modal-input" placeholder={modal.placeholder} bind:value={modal.value}
          onkeydown={(e) => { if (e.key === 'Enter') modal.onConfirm(modal.value); }} />
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={modal.onCancel}>Cancel</button>
        <button class="modal-btn primary" onclick={() => modal.onConfirm(modal.value)}>Confirm</button>
      </div>
    </div>
  </div>
{/if}

{#if showCreateWorkspaceModal}
  <div class="modal-backdrop" onclick={() => showCreateWorkspaceModal = false} role="presentation">
    <div class="modal-box" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">Create Lab Workspace</div>
      <div class="modal-body">
        <label class="sandbox-modal-label" for="wsName">Workspace Name</label>
        <input type="text" id="wsName" class="modal-input" placeholder="e.g. React Experiments" bind:value={newWorkspaceName} />
        <label class="sandbox-modal-label" for="wsDesc">Description</label>
        <input type="text" id="wsDesc" class="modal-input" placeholder="Notes for this workspace..." bind:value={newWorkspaceDesc} />
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showCreateWorkspaceModal = false}>Cancel</button>
        <button class="modal-btn primary" onclick={handleCreateWorkspace}>Create</button>
      </div>
    </div>
  </div>
{/if}

{#if showCreateSandboxModal}
  <div class="modal-backdrop" onclick={() => showCreateSandboxModal = false} role="presentation">
    <div class="modal-box sandbox-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">New Sandbox Experiment</div>
      <div class="modal-body">
        <label class="sandbox-modal-label" for="sandboxName">Experiment Name</label>
        <input type="text" id="sandboxName" class="modal-input" placeholder="My Experiment" bind:value={newSandboxName} />
        <label class="sandbox-modal-label" for="tmpl">Template Spec</label>
        <div class="templates-grid">
          {#each availableTemplates as tmpl}
            <button class="template-option" class:selected={selectedSandboxTemplate === tmpl.id} onclick={() => selectedSandboxTemplate = tmpl.id}>
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
  <!-- Top Navigation Header -->
  <header class="top-header">
    <div class="top-header-left">
      <div class="logo-badge"><FlaskConical size={16} /> <span>CrabCode Lab</span></div>
      <div class="workspace-pill">
        <span class="ws-pill-label">CONTAINER:</span>
        <select class="ws-dropdown" value={activeWorkspaceId} onchange={(e) => switchWorkspace(e.target.value)}>
          {#each workspacesList as ws}
            <option value={ws.id}>{ws.name}</option>
          {/each}
        </select>
        <button class="ws-add-btn" onclick={() => showCreateWorkspaceModal = true} title="New Workspace Database"><Plus size={12} /></button>
      </div>
    </div>

    <nav class="top-tabs">
      <button class="top-tab" class:active={activeTab === 'lab'} onclick={() => activeTab = 'lab'}><FlaskConical size={13} /> Laboratory</button>
      <button class="top-tab" class:active={activeTab === 'workspaces'} onclick={() => activeTab = 'workspaces'}><Database size={13} /> Workspaces DB</button>
      <button class="top-tab" class:active={activeTab === 'settings'} onclick={() => activeTab = 'settings'}><Settings size={13} /> Settings</button>
    </nav>
  </header>

  <div class="app-body">
    {#if activeTab === 'lab'}
      <!-- Left Sidebar: Sandboxes Explorer -->
      {#if showSidebar}
        <aside class="sidebar">
          <div class="sidebar-header-row">
            <span class="sidebar-title">Experiments ({sandboxesList.length})</span>
            <div class="sidebar-actions">
              <button class="icon-btn" onclick={createVirtualFolder} title="New Folder"><FolderPlus size={13} /></button>
              <button class="icon-btn primary" onclick={() => openCreateSandboxModal('')} title="New Sandbox"><Plus size={13} /></button>
            </div>
          </div>

          <div class="sidebar-search-container">
            <input type="text" class="sidebar-search-input" placeholder="Filter experiments..." bind:value={sandboxSearchQuery} />
          </div>

          <div class="sandboxes-tree-scroll">
            {#each virtualFolders as folderName}
              {@const itemsInFolder = (folderGroupedSandboxes[folderName] || [])}
              <div class="virtual-folder-group">
                <div class="virtual-folder-header" onclick={() => expandedVirtualFolders[folderName] = !expandedVirtualFolders[folderName]} role="button" tabindex="0">
                  <span class="folder-arrow">{expandedVirtualFolders[folderName] ? '▼' : '▶'}</span>
                  <Folder size={13} class="v-folder-icon" />
                  <span class="v-folder-name">{folderName}</span>
                  <span class="v-folder-count">({itemsInFolder.length})</span>
                </div>

                {#if expandedVirtualFolders[folderName]}
                  <div class="folder-children-list">
                    {#each itemsInFolder as sandbox (sandbox.id)}
                      <div class="sb-item-row nested" class:active={activeSandboxId === sandbox.id} onclick={() => selectSandbox(sandbox)} role="button" tabindex="0">
                        <div class="sb-item-title"><FlaskConical size={12} class="sb-icon" /><span>{sandbox.name}</span></div>
                        <div class="sb-item-actions">
                          <button class="sb-action-btn" onclick={(e) => { e.stopPropagation(); handleExportSandbox(sandbox.id); }} title="Export"><Download size={11} /></button>
                          <button class="sb-action-btn danger" onclick={(e) => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }} title="Delete"><Trash2 size={11} /></button>
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}

            {#each (folderGroupedSandboxes[''] || []) as sandbox (sandbox.id)}
              <div class="sb-item-row" class:active={activeSandboxId === sandbox.id} onclick={() => selectSandbox(sandbox)} role="button" tabindex="0">
                <div class="sb-item-title"><FlaskConical size={13} class="sb-icon" /><span>{sandbox.name}</span></div>
                <div class="sb-item-actions">
                  <button class="sb-action-btn" onclick={(e) => { e.stopPropagation(); handleExportSandbox(sandbox.id); }} title="Export"><Download size={11} /></button>
                  <button class="sb-action-btn danger" onclick={(e) => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }} title="Delete"><Trash2 size={11} /></button>
                </div>
              </div>
            {/each}
          </div>
        </aside>
      {/if}

      <!-- Center Main Lab Workbench -->
      <main class="editor-panel">
        {#if activeSandboxId}
          <div class="sandbox-workbench">
            <!-- Sandbox Workspace Control Bar -->
            <div class="sandbox-control-bar">
              <div class="control-left">
                <button class="sidebar-toggle" onclick={() => showSidebar = !showSidebar} title="Toggle Sidebar">
                  <PanelLeft size={14} />
                </button>
                <input type="text" class="sandbox-title-input" value={activeSandboxName} oninput={(e) => handleAutoRenameSandbox(e.target.value)} />
              </div>
              <div class="control-actions">
                <button class="action-btn" onclick={() => showSandboxSplit = !showSandboxSplit}>
                  <Columns size={13} /> {showSandboxSplit ? 'Hide Split' : 'Show Split'}
                </button>
                <button class="action-btn" onclick={handleSaveSandbox} disabled={!activeSandboxUnsaved}>
                  <Save size={13} /> Save Lab
                </button>
                {#if !isRunning}
                  <button class="action-btn run" onclick={runActiveCode}><Play size={13} fill="#48bb78" stroke="none" /> Run Experiment</button>
                {:else}
                  <button class="action-btn stop" onclick={stopActiveProcess}><Square size={13} fill="#ff5a36" stroke="none" /> Stop</button>
                {/if}
              </div>
            </div>

            <!-- Working Split Area -->
            <div class="sandbox-working-split">
              <!-- Left: Virtual Files + Code Editor -->
              <div class="sandbox-main-area">
                {#if showSandboxExplorer}
                  <div class="v-files-sidebar">
                    <div class="v-files-header">
                      <span>VIRTUAL FILES</span>
                      <button class="icon-btn" onclick={handleCreateSandboxFile}><Plus size={12} /></button>
                    </div>
                    <div class="v-files-list">
                      {#each sandboxFilesList as vFile (vFile.path)}
                        <div class="v-file-row" class:active={activeSandboxFilePath === vFile.path}>
                          <button class="v-file-btn" onclick={() => selectSandboxFile(vFile)}>
                            <FileCode size={12} />
                            <span class="v-file-name">{vFile.path}</span>
                          </button>
                          <button class="v-file-delete" onclick={() => handleDeleteSandboxFile(vFile.path)}><Trash2 size={10} /></button>
                        </div>
                      {/each}
                    </div>
                  </div>
                {/if}

                <div class="sandbox-editor-wrapper">
                  {#if activeSandboxFilePath}
                    <div class="sandbox-cm-container" bind:this={sandboxEditorContainer}></div>
                  {:else}
                    <div class="empty-editor-hint">Select or create a virtual file to edit source code.</div>
                  {/if}
                </div>
              </div>

              <!-- Right Pane: Notes / HTML Canvas / YAML Spec -->
              {#if showSandboxSplit}
                <div class="sandbox-notes-pane">
                  <div class="notes-tab-bar">
                    <button class="notes-tab" class:active={sandboxSplitType === 'markdown'} onclick={() => sandboxSplitType = 'markdown'}><BookOpen size={12} /> Notes</button>
                    <button class="notes-tab" class:active={sandboxSplitType === 'html'} onclick={() => sandboxSplitType = 'html'}><Sparkles size={12} /> Live Canvas</button>
                    <button class="notes-tab" class:active={sandboxSplitType === 'yaml'} onclick={() => sandboxSplitType = 'yaml'}><Settings size={12} /> YAML Spec</button>
                  </div>

                  <div class="notes-pane-content">
                    {#if sandboxSplitType === 'markdown'}
                      <div class="split-pane">
                        <div class="pane-action-header">
                          <span class="pane-title">RESEARCH NOTES (MARKDOWN)</span>
                          <button class="edit-toggle-btn" onclick={() => isSandboxMarkdownEditing = !isSandboxMarkdownEditing}>
                            {#if isSandboxMarkdownEditing}<Eye size={11} /> View{:else}<Edit3 size={11} /> Edit{/if}
                          </button>
                        </div>
                        {#if isSandboxMarkdownEditing}
                          <textarea class="raw-textarea" bind:value={activeSandboxMarkdownNote} placeholder="Write markdown lab observations..."></textarea>
                        {:else}
                          <div class="markdown-preview">
                            {#each activeSandboxMarkdownNote.split('\n') as line}
                              {@const part = renderMarkdownLine(line)}
                              {#if part.type === 'h1'}<h1>{part.text}</h1>
                              {:else if part.type === 'h2'}<h2>{part.text}</h2>
                              {:else if part.type === 'h3'}<h3>{part.text}</h3>
                              {:else if part.type === 'quote'}<blockquote>{part.text}</blockquote>
                              {:else if part.type === 'li'}<li>{part.text}</li>
                              {:else if part.type === 'codeblock'}<pre class="codeblock"><code>{part.text}</code></pre>
                              {:else if part.type === 'br'}<br />
                              {:else}<p>{part.text}</p>{/if}
                            {/each}
                          </div>
                        {/if}
                      </div>

                    {:else if sandboxSplitType === 'html'}
                      <div class="split-pane">
                        <div class="pane-action-header">
                          <span class="pane-title">INTERACTIVE VISUAL CANVAS</span>
                          <button class="edit-toggle-btn" onclick={() => isSandboxHTMLEditing = !isSandboxHTMLEditing}>
                            {#if isSandboxHTMLEditing}<Eye size={11} /> Live{:else}<Edit3 size={11} /> Edit{/if}
                          </button>
                        </div>
                        {#if isSandboxHTMLEditing}
                          <textarea class="raw-textarea code-family" bind:value={activeSandboxHTMLNote} placeholder="HTML/SVG widget canvas..."></textarea>
                        {:else}
                          <div class="html-frame-wrapper">
                            <iframe title="Visual Frame" srcdoc={activeSandboxHTMLNote} sandbox="allow-scripts" class="html-iframe"></iframe>
                          </div>
                        {/if}
                      </div>

                    {:else if sandboxSplitType === 'yaml'}
                      <div class="split-pane">
                        <div class="pane-action-header"><span class="pane-title">DECLARATIVE YAML CONFIG</span></div>
                        <textarea class="raw-textarea code-family" bind:value={activeSandboxConfig}></textarea>
                      </div>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>

            <!-- Bottom Console / Terminal Drawer -->
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
                try {
                  const sandboxDir = await GetSandboxDirectory(activeSandboxId);
                  const id = createSandboxTerminal('bash', sandboxDir);
                  await StartTerminalSession(id, sandboxDir);
                } catch (_) {}
              }}
              onClearConsole={() => { consoleLogs = []; consoleStatus = 'Ready'; }}
              onMinimize={() => isConsoleOpen = false}
            />
          </div>
        {:else}
          <div class="empty-workbench">
            <FlaskConical size={48} style="color: #ff5a36;" />
            <h2>Laboratory Sandbox Manager</h2>
            <p>Select or create a sandbox experiment to begin prototyping.</p>
            <button class="modal-btn primary" onclick={() => openCreateSandboxModal('')}><Plus size={14} /> New Sandbox Experiment</button>
          </div>
        {/if}
      </main>

    {:else if activeTab === 'workspaces'}
      <!-- Workspaces Database View -->
      <div class="workspaces-db-view">
        <div class="wm-header">
          <div>
            <h2>Database Workspaces Manager</h2>
            <p>Organize isolated sandbox database containers, export backups, or restore labs.</p>
          </div>
          <div class="wm-actions">
            <label class="modal-btn secondary file-btn">
              <Upload size={13} /> Restore Workspace
              <input type="file" accept=".json" onchange={handleRestoreWorkspaceFile} hidden />
            </label>
            <label class="modal-btn secondary file-btn">
              <Upload size={13} /> Import Sandbox
              <input type="file" accept=".json" onchange={handleImportSandboxFile} hidden />
            </label>
            <button class="modal-btn primary" onclick={() => showCreateWorkspaceModal = true}><Plus size={13} /> New Workspace</button>
          </div>
        </div>

        <div class="workspaces-grid">
          {#each workspacesList as ws}
            <div class="workspace-card" class:active={ws.id === activeWorkspaceId}>
              <div class="ws-card-title-row">
                <h3>{ws.name}</h3>
                {#if ws.id !== 'default'}
                  <button class="icon-btn danger" onclick={() => handleDeleteWorkspace(ws.id)} title="Delete Workspace"><Trash2 size={13} /></button>
                {/if}
              </div>
              <p class="ws-card-desc">{ws.description || 'No description provided.'}</p>
              <div class="ws-card-footer">
                <button class="modal-btn secondary" onclick={() => handleBackupWorkspace(ws.id)}><Archive size={12} /> Backup</button>
                <button class="modal-btn primary" onclick={() => { switchWorkspace(ws.id); activeTab = 'lab'; }}>Select Container</button>
              </div>
            </div>
          {/each}
        </div>
      </div>

    {:else if activeTab === 'settings'}
      <div class="settings-view">
        <div class="settings-card">
          <h2>CrabCode System Path</h2>
          <p>Global SQLite database location:</p>
          <input type="text" class="modal-input" bind:value={settingsCrabRootPath} readonly />
        </div>
      </div>
    {/if}
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

  /* Header */
  .top-header {
    height: 44px;
    min-height: 44px;
    background-color: #0c0c12;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 14px;
    user-select: none;
    box-sizing: border-box;
  }

  .top-header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo-badge {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
    color: #ff5a36;
    font-size: 14px;
  }

  .workspace-pill {
    display: flex;
    align-items: center;
    gap: 6px;
    background: #12121a;
    padding: 2px 8px;
    border-radius: 6px;
    border: 1px solid #2d3748;
  }

  .ws-pill-label {
    font-size: 10px;
    font-weight: 700;
    color: #6b7280;
  }

  .ws-dropdown {
    background: transparent;
    border: none;
    color: #edf2f7;
    font-size: 12px;
    font-weight: 600;
    outline: none;
    cursor: pointer;
  }

  .ws-add-btn {
    background: none;
    border: none;
    color: #a0aec0;
    cursor: pointer;
    display: flex;
    align-items: center;
    padding: 2px;
  }

  .ws-add-btn:hover { color: #ff5a36; }

  .top-tabs {
    display: flex;
    gap: 4px;
    background-color: #0a0a0f;
    border: 1px solid #1a1a24;
    border-radius: 6px;
    padding: 3px;
  }

  .top-tab {
    background: none;
    border: 1px solid transparent;
    color: #9ca3af;
    padding: 4px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .top-tab.active {
    background-color: #ff5a3618;
    color: #edf2f7;
    border-color: #ff5a3644;
  }

  /* Body */
  .app-body {
    flex: 1;
    display: flex;
    flex-direction: row;
    overflow: hidden;
    height: calc(100vh - 44px);
  }

  /* Sidebar */
  .sidebar {
    width: 260px;
    min-width: 260px;
    max-width: 260px;
    background-color: #0c0c12;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-sizing: border-box;
  }

  .sidebar-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    border-bottom: 1px solid #1c1c28;
  }

  .sidebar-title {
    font-size: 11px;
    font-weight: 700;
    color: #6b7280;
    text-transform: uppercase;
  }

  .sidebar-actions {
    display: flex;
    gap: 4px;
  }

  .sidebar-search-container {
    padding: 8px 10px;
  }

  .sidebar-search-input {
    width: 100%;
    background-color: #12121a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    color: #edf2f7;
    padding: 6px 8px;
    font-size: 12px;
    outline: none;
    box-sizing: border-box;
  }

  .sandboxes-tree-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 4px 6px;
  }

  .virtual-folder-group {
    margin-bottom: 4px;
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

  .virtual-folder-header:hover { background-color: #1a1a24; color: #edf2f7; }

  .folder-arrow { font-size: 8px; color: #718096; }
  :global(.v-folder-icon) { color: #ff5a36; }
  .v-folder-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .v-folder-count { font-size: 10px; color: #718096; }

  .folder-children-list {
    padding-left: 12px;
  }

  .sb-item-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 8px;
    border-radius: 4px;
    cursor: pointer;
    margin-bottom: 2px;
    color: #a0aec0;
  }

  .sb-item-row:hover { background-color: #1a1a24; color: #edf2f7; }
  .sb-item-row.active { background-color: #ff5a3622; color: #ff5a36; font-weight: 600; }

  .sb-item-title {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
  }

  .sb-item-actions {
    display: flex;
    gap: 2px;
    opacity: 0;
  }

  .sb-item-row:hover .sb-item-actions { opacity: 1; }

  .sb-action-btn {
    background: none;
    border: none;
    color: #718096;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;
    align-items: center;
  }

  .sb-action-btn:hover { color: #edf2f7; background-color: #2d3748; }
  .sb-action-btn.danger:hover { color: #e53e3e; }

  /* Editor Panel Main */
  .editor-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #0b0b0f;
    min-width: 0;
    height: 100%;
    overflow: hidden;
  }

  .sandbox-workbench {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    overflow: hidden;
  }

  .sandbox-control-bar {
    height: 38px;
    min-height: 38px;
    background-color: #0e0e14;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px;
  }

  .control-left {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
  }

  .sidebar-toggle {
    background: none;
    border: none;
    color: #718096;
    cursor: pointer;
    display: flex;
    align-items: center;
    padding: 4px;
    border-radius: 4px;
  }

  .sidebar-toggle:hover { color: #edf2f7; background-color: #1a1a24; }

  .sandbox-title-input {
    background: transparent;
    border: 1px solid transparent;
    color: #edf2f7;
    font-size: 13px;
    font-weight: 600;
    outline: none;
    padding: 2px 6px;
    border-radius: 4px;
    width: 240px;
  }

  .sandbox-title-input:hover,
  .sandbox-title-input:focus {
    border-color: #2d3748;
    background-color: #12121a;
  }

  .control-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .action-btn {
    background-color: #1a1a24;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .action-btn:hover { color: #edf2f7; border-color: #4a5568; }
  .action-btn.run { background-color: #48bb7822; color: #48bb78; border-color: #48bb7844; }
  .action-btn.run:hover { background-color: #48bb7833; }
  .action-btn.stop { background-color: #ff5a3622; color: #ff5a36; border-color: #ff5a3644; }

  /* Split area */
  .sandbox-working-split {
    flex: 1;
    display: flex;
    flex-direction: row;
    min-height: 0;
    overflow: hidden;
  }

  .sandbox-main-area {
    flex: 1;
    display: flex;
    flex-direction: row;
    min-width: 0;
    background-color: #12121a;
  }

  .v-files-sidebar {
    width: 160px;
    min-width: 160px;
    background-color: #0a0a0f;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
  }

  .v-files-header {
    height: 28px;
    padding: 0 8px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid #1a1a24;
    font-size: 10px;
    font-weight: 700;
    color: #718096;
  }

  .v-files-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px;
  }

  .v-file-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 2px 4px;
    border-radius: 4px;
  }

  .v-file-row.active { background-color: #1a1a24; }

  .v-file-btn {
    background: none;
    border: none;
    color: #a0aec0;
    font-size: 11px;
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    text-align: left;
  }

  .v-file-row.active .v-file-btn { color: #ff5a36; font-weight: 600; }
  .v-file-delete { background: none; border: none; color: #718096; cursor: pointer; opacity: 0; padding: 2px; }
  .v-file-row:hover .v-file-delete { opacity: 1; }
  .v-file-delete:hover { color: #e53e3e; }

  .sandbox-editor-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    height: 100%;
    position: relative;
  }

  .sandbox-cm-container {
    width: 100%;
    height: 100%;
  }

  .empty-editor-hint {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #718096;
    font-size: 12px;
  }

  /* Notes Right Pane */
  .sandbox-notes-pane {
    width: 450px;
    min-width: 300px;
    background-color: #0d0d14;
    border-left: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
  }

  .notes-tab-bar {
    height: 32px;
    background-color: #08080c;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    padding: 0 8px;
    gap: 4px;
  }

  .notes-tab {
    background: none;
    border: none;
    color: #718096;
    font-size: 11px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .notes-tab.active { background-color: #12121a; color: #edf2f7; }

  .notes-pane-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .split-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  .pane-action-header {
    height: 28px;
    padding: 0 12px;
    background-color: #0a0a0f;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .pane-title { font-size: 10px; font-weight: 700; color: #6b7280; }

  .edit-toggle-btn {
    background: none;
    border: none;
    color: #a0aec0;
    font-size: 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .raw-textarea {
    flex: 1;
    width: 100%;
    background-color: #0b0b0f;
    border: none;
    color: #e2e8f0;
    padding: 12px;
    font-size: 12px;
    line-height: 1.5;
    outline: none;
    resize: none;
    box-sizing: border-box;
  }

  .raw-textarea.code-family {
    font-family: 'Fira Code', monospace;
    font-size: 11px;
    color: #34d399;
  }

  .markdown-preview {
    flex: 1;
    padding: 14px;
    overflow-y: auto;
    font-size: 13px;
    line-height: 1.6;
    color: #cbd5e0;
  }

  .markdown-preview h1 { font-size: 18px; color: #edf2f7; border-bottom: 1px solid #2d3748; padding-bottom: 4px; margin-top: 0; }
  .markdown-preview h2 { font-size: 15px; color: #edf2f7; margin-top: 12px; }
  .markdown-preview h3 { font-size: 13px; color: #ff5a36; margin-top: 10px; }
  .markdown-preview blockquote { border-left: 3px solid #ff5a36; margin: 8px 0; padding-left: 10px; color: #a0aec0; }
  .markdown-preview .codeblock { background: #07070a; padding: 8px; border-radius: 4px; font-family: monospace; font-size: 11px; color: #34d399; }

  .html-frame-wrapper {
    flex: 1;
    width: 100%;
    height: 100%;
    background: #fff;
  }

  .html-iframe {
    width: 100%;
    height: 100%;
    border: none;
  }

  /* Empty Workbench */
  .empty-workbench {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: #a0aec0;
  }

  .empty-workbench h2 { margin: 0; color: #edf2f7; font-size: 18px; }
  .empty-workbench p { margin: 0; font-size: 13px; color: #718096; }

  /* Workspaces Database View */
  .workspaces-db-view {
    flex: 1;
    padding: 24px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .wm-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    border-bottom: 1px solid #1a1a24;
    padding-bottom: 16px;
  }

  .wm-header h2 { margin: 0; font-size: 20px; color: #edf2f7; }
  .wm-header p { margin: 4px 0 0 0; font-size: 12px; color: #718096; }
  .wm-actions { display: flex; gap: 8px; }

  .workspaces-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }

  .workspace-card {
    background-color: #0f0f17;
    border: 1px solid #1a1a24;
    border-radius: 8px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .workspace-card.active { border-color: #ff5a3688; background-color: #ff5a3608; }

  .ws-card-title-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .ws-card-title-row h3 { margin: 0; font-size: 15px; color: #edf2f7; }
  .ws-card-desc { font-size: 12px; color: #a0aec0; margin: 0; flex: 1; }

  .ws-card-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 8px;
    padding-top: 10px;
    border-top: 1px solid #1a1a24;
  }

  /* Modals */
  .modal-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background-color: #000000aa;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .modal-box {
    background-color: #0f0f17;
    border: 1px solid #2d3748;
    border-radius: 8px;
    width: 380px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 10px 25px #00000088;
  }

  .modal-header {
    padding: 14px 16px;
    font-size: 14px;
    font-weight: 700;
    color: #edf2f7;
    border-bottom: 1px solid #1a1a24;
  }

  .modal-body {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .modal-input {
    background-color: #07070a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 8px 10px;
    color: #edf2f7;
    font-size: 13px;
    outline: none;
  }

  .modal-input:focus { border-color: #ff5a36; }

  .sandbox-modal-label {
    font-size: 11px;
    font-weight: 600;
    color: #a0aec0;
  }

  .templates-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .template-option {
    background-color: #07070a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: #a0aec0;
  }

  .template-option.selected { border-color: #ff5a36; background-color: #ff5a3614; color: #edf2f7; }

  .template-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .template-name { font-size: 12px; font-weight: 600; }

  .modal-footer {
    padding: 12px 16px;
    border-top: 1px solid #1a1a24;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .modal-btn {
    padding: 6px 12px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    border: none;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .modal-btn.primary { background-color: #ff5a36; color: #fff; }
  .modal-btn.primary:hover { background-color: #ff451d; }
  .modal-btn.secondary { background-color: #1a1a24; color: #a0aec0; border: 1px solid #2d3748; }
  .modal-btn.secondary:hover { color: #edf2f7; border-color: #4a5568; }

  .file-btn { display: inline-flex; align-items: center; cursor: pointer; }

  /* Utility Icon Buttons */
  .icon-btn {
    background: none;
    border: none;
    color: #718096;
    cursor: pointer;
    padding: 3px;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }

  .icon-btn:hover { color: #edf2f7; background-color: #1a1a24; }
  .icon-btn.primary { color: #ff5a36; }
  .icon-btn.danger:hover { color: #e53e3e; }
</style>
