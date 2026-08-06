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
    GetSandboxFiles,
    SaveSandboxFile,
    DeleteSandboxFile,
    GetFileNotes,
    AddFileNote,
    UpdateFileNote,
    DeleteFileNote,
    SaveSandboxNotes,
    GetWorkspaceRuntimePath,
    ActivateSandbox,
    SaveWorkspaceConfig,
    RunSandbox,
    StopCommand,
    StartTerminalSession,
    WriteTerminalInput,
    IsEnvironmentInitialized,
    InitializeEnvironment,
    GetGlobalSettings,
    GetTemplates,
    ResetAndReinitializeEverything
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import TerminalDrawer from './TerminalDrawer.svelte';
  import { onMount, onDestroy, untrack } from 'svelte';
  import {
    Play, Square, Settings, Database, BookOpen, Plus, Trash2, Columns,
    PanelLeft, Save, Sparkles, FileCode, Eye, Edit3, Folder,
    FolderPlus, Download, Upload, FlaskConical, Archive, AlertTriangle, RefreshCw, CheckCircle2,
    Loader2, Cpu, FileText, ChevronDown
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

  // Workspace DB State & Dropdown UI State
  let workspacesList = $state([]);
  let activeWorkspaceId = $state('default');
  let activeWorkspaceObj = $derived(workspacesList.find(w => w.id === activeWorkspaceId) || null);
  let isWsDropdownOpen = $state(false);

  // Sidebar Expandable / Resizable state
  let showSidebar = $state(true);
  let sidebarWidth = $state(260);
  let isSidebarResizing = $state(false);

  // Sandboxes State
  let sandboxesList = $state([]);
  let activeSandboxId = $state('');
  let activeSandboxName = $state('');
  let activeSandboxConfig = $state('');
  let lastSavedSandboxConfig = $state('');
  let activeSandboxHTMLNote = $state('');
  let lastSavedSandboxHTMLNote = $state('');

  let activeFileNotes = $state([]);
  let editingNoteIds = $state({});

  let isActivatingSandbox = $state(false);

  // Virtual Folders & Files State
  let sandboxSearchQuery = $state('');
  let virtualFolders = $state([]);
  let expandedVirtualFolders = $state({});

  let showSandboxExplorer = $state(true);
  let showSandboxSplit = $state(true);
  let sandboxSplitType = $state('markdown'); // 'markdown', 'html', 'yaml'
  let isSandboxHTMLEditing = $state(false);

  let sandboxFilesList = $state([]);
  let activeSandboxFilePath = $state('');
  let activeSandboxFileName = $state('');
  let activeSandboxFileContent = $state('');
  let lastSavedSandboxFileContent = $state('');

  let settingsCrabRootPath = $state('');
  let toasts = $state([]);
  let modal = $state({ show: false, title: '', placeholder: '', value: '', onConfirm: null, onCancel: null });
  
  // Workspace Creation Modal State
  let showCreateWorkspaceModal = $state(false);
  let wsConfigMode = $state('template'); // 'template' or 'custom'
  let newWorkspaceName = $state('');
  let newWorkspaceDesc = $state('');
  let selectedWsTemplate = $state('python');
  let customWsConfigYaml = $state(`name: "Custom Workspace"
version: "1.0"
environment: "python"
setup: []
mappings:
  run: "python3 main.py"
  test: "pytest"
env_vars:
  PYTHONUNBUFFERED: "1"
`);
  let availableTemplates = $state([]);

  // Sandbox Creation Modal State
  let showCreateSandboxModal = $state(false);
  let newSandboxName = $state('');
  let selectedSandboxFolder = $state('');

  // Environment Initialization Indicator & Modal State
  let showInitEnvModal = $state(false);
  let isInitializingEnv = $state(false);
  let isEnvInitialized = $state(true);
  let setupLogs = $state([]);

  let showResetConfirmModal = $state(false);
  let isResetting = $state(false);

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
  let systemLogs = $state([]);
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
    activeSandboxId !== '' && activeSandboxHTMLNote !== lastSavedSandboxHTMLNote
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
      keymap.of([
        indentWithTab,
        {
          key: 'Mod-s',
          run: () => {
            handleSaveSandbox();
            return true;
          }
        }
      ]),
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

  function startSidebarResize(e) {
    isSidebarResizing = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    function onMouseMove(event) {
      const deltaX = event.clientX - startX;
      sidebarWidth = Math.max(180, Math.min(500, startWidth + deltaX));
    }

    function onMouseUp() {
      isSidebarResizing = false;
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    }

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  function handleGlobalKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault();
      handleSaveSandbox();
    }
  }

  function handleWindowClick(e) {
    if (isWsDropdownOpen && !e.target.closest('.workspace-dropdown-wrapper')) {
      isWsDropdownOpen = false;
    }
  }

  let toastId = 0;
  function addToast(message, type = 'info', duration = 2500) {
    const id = toastId++;
    toasts = [...toasts, { id, message, type }];
    setTimeout(() => { toasts = toasts.filter(item => item.id !== id); }, duration);
  }

  function logSystem(text, type = 'info') {
    systemLogs = [...systemLogs, { time: Date.now(), text, type }];
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
      logSystem('Failed to load workspaces: ' + String(err), 'error');
    }
  }

  async function loadSandboxesForWorkspace(wsId) {
    try {
      sandboxesList = await GetSandboxes(wsId);
      const foldersSet = new Set(sandboxesList.map(s => s.folder).filter(Boolean));
      virtualFolders = Array.from(foldersSet);

      if (sandboxesList.length > 0) {
        const activeSb = sandboxesList.find(s => s.isActive) || sandboxesList[0];
        await selectSandbox(activeSb);
      } else {
        clearActiveSandbox();
      }
    } catch (err) {
      logSystem(String(err), 'error');
    }
  }

  async function switchWorkspace(wsId) {
    if (activeWorkspaceId === wsId) return;
    activeWorkspaceId = wsId;
    logSystem(`Switched active workspace runtime to '${wsId}'`, 'info');
    await loadSandboxesForWorkspace(wsId);
  }

  async function openCreateWorkspaceModal() {
    newWorkspaceName = '';
    newWorkspaceDesc = '';
    wsConfigMode = 'template';
    try {
      availableTemplates = await GetTemplates();
      if (availableTemplates.length > 0) {
        selectedWsTemplate = availableTemplates[0].id;
      }
    } catch (err) {
      logSystem('Failed to load templates: ' + String(err), 'error');
    }
    showCreateWorkspaceModal = true;
  }

  async function handleCreateWorkspace() {
    if (!newWorkspaceName.trim()) return;
    try {
      const tmplId = (wsConfigMode === 'template') ? selectedWsTemplate : '';
      const customYaml = (wsConfigMode === 'custom') ? customWsConfigYaml : '';

      const ws = await CreateWorkspace(newWorkspaceName.trim(), newWorkspaceDesc.trim(), tmplId, customYaml);
      showCreateWorkspaceModal = false;
      workspacesList = [ws, ...workspacesList];
      await switchWorkspace(ws.id);
      addToast('Workspace created', 'success');
    } catch (err) {
      logSystem('Workspace creation failed: ' + String(err), 'error');
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
      logSystem('Delete workspace error: ' + String(err), 'error');
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
    activeSandboxHTMLNote = '';
    lastSavedSandboxHTMLNote = '';
    activeFileNotes = [];
    editingNoteIds = {};
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
          activeFileNotes = [];
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
      await refreshActiveFileNotes();
    } catch (_) {}
  }

  async function refreshActiveFileNotes() {
    if (!activeSandboxId || !activeSandboxFilePath) {
      activeFileNotes = [];
      return;
    }
    try {
      activeFileNotes = await GetFileNotes(activeSandboxId, activeSandboxFilePath);
    } catch (_) {
      activeFileNotes = [];
    }
  }

  async function handleAddFileNote() {
    if (!activeSandboxId || !activeSandboxFilePath) return;
    try {
      const newNote = await AddFileNote(
        activeSandboxId,
        activeSandboxFilePath,
        `Note Entry #${activeFileNotes.length + 1}`,
        `# Notes for ${activeSandboxFileName}\n\nWrite observations here...`
      );
      editingNoteIds[newNote.id] = true;
      await refreshActiveFileNotes();
      addToast('Note appended', 'success');
    } catch (err) {
      logSystem('Failed to add note: ' + String(err), 'error');
    }
  }

  async function handleSaveFileNote(note) {
    try {
      await UpdateFileNote(note.id, note.title, note.content);
      editingNoteIds[note.id] = false;
      await refreshActiveFileNotes();
      addToast('Note saved', 'success');
    } catch (err) {
      logSystem('Failed to update note: ' + String(err), 'error');
    }
  }

  async function handleDeleteFileNote(noteId) {
    if (!confirm('Delete this markdown note entry?')) return;
    try {
      await DeleteFileNote(noteId);
      await refreshActiveFileNotes();
      addToast('Note deleted', 'success');
    } catch (err) {
      logSystem('Failed to delete note: ' + String(err), 'error');
    }
  }

  async function createVirtualFolder() {
    const folderName = await openPrompt('New Virtual Folder', 'UI Experiments');
    if (!folderName) return;
    const cleanFolder = folderName.trim();
    if (!cleanFolder) return;
    if (!virtualFolders.includes(cleanFolder)) {
      virtualFolders = [...virtualFolders, cleanFolder];
      expandedVirtualFolders[cleanFolder] = true;
    }
  }

  async function openCreateSandboxModal(defaultFolder = '') {
    newSandboxName = '';
    selectedSandboxFolder = defaultFolder;
    showCreateSandboxModal = true;
  }

  async function confirmCreateSandbox() {
    const name = newSandboxName.trim() || 'Experiment';
    showCreateSandboxModal = false;
    try {
      const sandbox = await CreateSandboxInFolder(activeWorkspaceId, name, selectedSandboxFolder);
      sandboxesList = [sandbox, ...sandboxesList];
      await selectSandbox(sandbox);
      addToast('Sandbox experiment created', 'success');
    } catch (err) {
      logSystem('Failed to create sandbox: ' + String(err), 'error');
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
        logSystem('Failed to save title: ' + String(err), 'error');
      }
    };

    clearTimeout(renameTimer);
    if (immediate) doSave();
    else renameTimer = setTimeout(doSave, 500);
  }

  async function selectSandbox(sandbox) {
    if (activeSandboxId === sandbox.id && sandbox.isActive) return;

    isActivatingSandbox = true;
    try {
      await ActivateSandbox(activeWorkspaceId, sandbox.id);
    } catch (err) {
      logSystem('Activation error: ' + String(err), 'error');
    } finally {
      isActivatingSandbox = false;
    }

    sandboxesList = sandboxesList.map(s => ({
      ...s,
      isActive: (s.id === sandbox.id)
    }));

    activeSandboxId = sandbox.id;
    activeSandboxName = sandbox.name;

    const wsConfig = activeWorkspaceObj?.configYaml || '';
    activeSandboxConfig = wsConfig;
    lastSavedSandboxConfig = wsConfig;

    activeSandboxHTMLNote = sandbox.htmlNote || '';
    lastSavedSandboxHTMLNote = sandbox.htmlNote || '';

    isSandboxHTMLEditing = false;

    activeSandboxFilePath = '';
    activeSandboxFileName = '';
    activeSandboxFileContent = '';
    lastSavedSandboxFileContent = '';

    try {
      isEnvInitialized = await IsEnvironmentInitialized(activeWorkspaceId);
    } catch (_) { isEnvInitialized = false; }

    await refreshSandboxFiles();

    if (sandboxTerminals.length === 0) {
      try {
        const sandboxDir = await GetWorkspaceRuntimePath(activeWorkspaceId);
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
    await refreshActiveFileNotes();
  }

  async function handleCreateSandboxFile() {
    if (!activeSandboxId) return;
    const name = await openPrompt('New Sandbox File', 'main.py');
    if (!name) return;
    try {
      await SaveSandboxFile(activeSandboxId, name, '# Sandbox source code\n', false);
      await refreshSandboxFiles();
      const created = sandboxFilesList.find(f => f.path === name);
      if (created) await selectSandboxFile(created);
    } catch (err) {
      logSystem('Failed to create file: ' + String(err), 'error');
    }
  }

  async function handleDeleteSandboxFile(path) {
    if (!activeSandboxId || !confirm('Delete this file?')) return;
    try {
      await DeleteSandboxFile(activeSandboxId, path);
      if (activeSandboxFilePath === path) {
        activeSandboxFilePath = '';
        activeSandboxFileName = '';
        activeSandboxFileContent = '';
        lastSavedSandboxFileContent = '';
      }
      await refreshSandboxFiles();
    } catch (err) {
      logSystem('Failed to delete file: ' + String(err), 'error');
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
        await SaveWorkspaceConfig(activeWorkspaceId, activeSandboxConfig);
        lastSavedSandboxConfig = activeSandboxConfig;
      }

      if (activeSandboxNotesUnsaved) {
        await SaveSandboxNotes(activeSandboxId, '', activeSandboxHTMLNote);
        lastSavedSandboxHTMLNote = activeSandboxHTMLNote;
        const si = sandboxesList.findIndex(s => s.id === activeSandboxId);
        if (si !== -1) {
          sandboxesList[si].htmlNote = activeSandboxHTMLNote;
        }
      }

      addToast('Saved', 'success');
    } catch (err) {
      logSystem('Save error: ' + String(err), 'error');
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
      logSystem('Delete sandbox error: ' + String(err), 'error');
    }
  }

  async function triggerInitializeEnvironment() {
    isInitializingEnv = true;
    setupLogs = [];
    try {
      logSystem(`Starting environment setup for workspace '${activeWorkspaceId}'...`, 'info');
      await InitializeEnvironment(activeWorkspaceId);
      isEnvInitialized = true;
      showInitEnvModal = false;
      addToast('Environment Initialized', 'success');
      logSystem('Workspace environment initialized successfully.', 'success');
    } catch (err) {
      logSystem('Environment initialization failed: ' + String(err), 'error');
    } finally {
      isInitializingEnv = false;
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
    } catch (err) { logSystem(String(err), 'error'); }
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
      } catch (err) { logSystem(String(err), 'error'); }
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
      addToast('Backup created', 'success');
    } catch (err) { logSystem(String(err), 'error'); }
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
      } catch (err) { logSystem(String(err), 'error'); }
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
      const isInit = await IsEnvironmentInitialized(activeWorkspaceId);
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
      await RunSandbox(runnerId, activeWorkspaceId, activeSandboxFilePath);
    } catch (err) {
      if (String(err).includes("ENV_NOT_INITIALIZED")) {
        showInitEnvModal = true;
        consoleStatus = 'Uninitialized Env';
      } else {
        consoleLogs = [...consoleLogs, { time: Date.now(), text: `[Error] ${String(err)}`, type: 'error' }];
        logSystem(`Run Error: ${String(err)}`, 'error');
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

  async function confirmResetSystem() {
    isResetting = true;
    try {
      await ResetAndReinitializeEverything();
      showResetConfirmModal = false;
      addToast('Reset complete', 'success');
      consoleLogs = [];
      systemLogs = [];
      consoleStatus = 'Ready';
      isConsoleRunning = false;
      activeRunnerId = '';
      sandboxTerminals = [];
      clearActiveSandbox();
      activeWorkspaceId = 'default';
      await loadWorkspaces();
    } catch (err) {
      logSystem('Reset error: ' + String(err), 'error');
    } finally {
      isResetting = false;
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

  onMount(async () => {
    window.addEventListener('keydown', handleGlobalKeydown);
    window.addEventListener('click', handleWindowClick);

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

    EventsOn('setup_output', (data) => {
      if (data.line) {
        setupLogs = [...setupLogs, data.line];
        logSystem(data.line, 'info');
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
    window.removeEventListener('keydown', handleGlobalKeydown);
    window.removeEventListener('click', handleWindowClick);
    if (sandboxView) sandboxView.destroy();
  });
</script>

<!-- Clean Bottom-Right Toast Notifications Container -->
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

<!-- Environment Initialization Loading Modal -->
{#if showInitEnvModal}
  <div class="modal-backdrop" onclick={() => { if (!isInitializingEnv) showInitEnvModal = false; }} role="presentation">
    <div class="modal-box env-init-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">
        <Cpu size={16} class="icon-spin-color" /> Environment Initialization
      </div>
      <div class="modal-body">
        <p class="guide-text">
          Workspace <strong>{activeWorkspaceObj?.name}</strong> needs to initialize its environment dependencies before execution.
        </p>

        {#if isInitializingEnv}
          <div class="setup-loading-box">
            <div class="loading-status-row">
              <Loader2 size={16} class="spinner" />
              <span>Initializing workspace runtime...</span>
            </div>
            <div class="setup-console-box">
              {#each setupLogs as line}
                <div class="setup-line">{line}</div>
              {/each}
            </div>
          </div>
        {:else}
          <p class="guide-subtext">Click below to execute environment setup steps.</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showInitEnvModal = false} disabled={isInitializingEnv}>Dismiss</button>
        <button class="modal-btn primary" onclick={triggerInitializeEnvironment} disabled={isInitializingEnv}>
          {#if isInitializingEnv}
            <Loader2 size={13} class="spinner" /> Initializing...
          {:else}
            Initialize Environment
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Workspace Creation Modal -->
{#if showCreateWorkspaceModal}
  <div class="modal-backdrop" onclick={() => showCreateWorkspaceModal = false} role="presentation">
    <div class="modal-box ws-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">Create Workspace Container</div>
      <div class="modal-body">
        <label class="sandbox-modal-label" for="wsName">Workspace Name</label>
        <input type="text" id="wsName" class="modal-input" placeholder="e.g. Raylib Go Engine Lab" bind:value={newWorkspaceName} />
        
        <label class="sandbox-modal-label" for="wsDesc">Description</label>
        <input type="text" id="wsDesc" class="modal-input" placeholder="Shared runtime description..." bind:value={newWorkspaceDesc} />

        <div class="config-mode-toggle">
          <button class="mode-btn" class:active={wsConfigMode === 'template'} onclick={() => wsConfigMode = 'template'}>
            Select Environment Template
          </button>
          <button class="mode-btn" class:active={wsConfigMode === 'custom'} onclick={() => wsConfigMode = 'custom'}>
            Custom Configuration YAML
          </button>
        </div>

        {#if wsConfigMode === 'template'}
          <label class="sandbox-modal-label">Environment Spec</label>
          <div class="templates-grid">
            {#each availableTemplates as tmpl}
              <button class="template-option" class:selected={selectedWsTemplate === tmpl.id} onclick={() => selectedWsTemplate = tmpl.id}>
                <span class="template-dot" style="background-color: {tmpl.iconColor}"></span>
                <span class="template-name">{tmpl.name}</span>
              </button>
            {/each}
          </div>
        {:else}
          <label class="sandbox-modal-label">Declarative Workspace Specification (YAML)</label>
          <textarea class="raw-textarea code-family yaml-editor" bind:value={customWsConfigYaml}></textarea>
        {/if}
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showCreateWorkspaceModal = false}>Cancel</button>
        <button class="modal-btn primary" onclick={handleCreateWorkspace}>Create Workspace</button>
      </div>
    </div>
  </div>
{/if}

<!-- Sandbox Creation Modal -->
{#if showCreateSandboxModal}
  <div class="modal-backdrop" onclick={() => showCreateSandboxModal = false} role="presentation">
    <div class="modal-box" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header">New Sandbox Experiment</div>
      <div class="modal-body">
        <p class="guide-text">
          Sandboxes inherit environment dependencies from <strong>{activeWorkspaceObj?.name}</strong>.
        </p>
        <label class="sandbox-modal-label" for="sandboxName">Experiment Name</label>
        <input type="text" id="sandboxName" class="modal-input" placeholder="e.g. Physics Demo" bind:value={newSandboxName} />
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showCreateSandboxModal = false}>Cancel</button>
        <button class="modal-btn primary" onclick={confirmCreateSandbox}>Create Sandbox</button>
      </div>
    </div>
  </div>
{/if}

{#if showResetConfirmModal}
  <div class="modal-backdrop" onclick={() => showResetConfirmModal = false} role="presentation">
    <div class="modal-box danger-modal" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
      <div class="modal-header danger-header"><AlertTriangle size={18} /> Reset System</div>
      <div class="modal-body">
        <p class="guide-text danger-text">
          Are you sure you want to completely reset CrabCode?
        </p>
        <p class="guide-subtext">
          This will wipe all SQLite databases, workspaces, sandboxes, and restore default settings.
        </p>
      </div>
      <div class="modal-footer">
        <button class="modal-btn secondary" onclick={() => showResetConfirmModal = false} disabled={isResetting}>Cancel</button>
        <button class="modal-btn danger-btn" onclick={confirmResetSystem} disabled={isResetting}>
          <RefreshCw size={13} /> {isResetting ? 'Resetting...' : 'Yes, Reset Everything'}
        </button>
      </div>
    </div>
  </div>
{/if}

<div class="app-shell">
  <!-- Top Navigation Header with Custom Workspace Dropdown -->
  <header class="top-header">
    <div class="top-header-left">
      <div class="logo-badge"><FlaskConical size={16} /> <span>CrabCode Lab</span></div>

      <!-- Custom Dark Workspace Dropdown -->
      <div class="workspace-dropdown-wrapper">
        <button class="workspace-pill-btn" onclick={() => isWsDropdownOpen = !isWsDropdownOpen}>
          <span class="ws-pill-label">WORKSPACE:</span>
          <span class="ws-pill-name">{activeWorkspaceObj?.name || 'Select Workspace'}</span>
          <ChevronDown size={12} class="ws-chevron {isWsDropdownOpen ? 'open' : ''}" />
        </button>

        {#if isWsDropdownOpen}
          <div class="workspace-dropdown-menu" onclick={(e) => e.stopPropagation()} role="menu" tabindex="-1">
            <div class="ws-menu-header">
              <span>WORKSPACES ({workspacesList.length})</span>
              <button class="icon-btn primary" onclick={() => { isWsDropdownOpen = false; openCreateWorkspaceModal(); }} title="Create Workspace">
                <Plus size={12} />
              </button>
            </div>

            <div class="ws-menu-list">
              {#each workspacesList as ws}
                <button
                  class="ws-menu-item"
                  class:active={ws.id === activeWorkspaceId}
                  onclick={() => { switchWorkspace(ws.id); isWsDropdownOpen = false; }}
                >
                  <div class="ws-item-left">
                    <span class="ws-item-name">{ws.name}</span>
                    {#if ws.description}
                      <span class="ws-item-desc">{ws.description}</span>
                    {/if}
                  </div>
                  {#if ws.id === activeWorkspaceId}
                    <CheckCircle2 size={12} class="ws-active-check" />
                  {/if}
                </button>
              {/each}
            </div>
          </div>
        {/if}
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
      <!-- Resizable & Expandable Left Sidebar -->
      {#if showSidebar}
        <aside class="sidebar" style="width: {sidebarWidth}px; min-width: {sidebarWidth}px; max-width: {sidebarWidth}px;">
          <div
            class="sidebar-resize-handle"
            onmousedown={startSidebarResize}
            class:resizing={isSidebarResizing}
            title="Drag to resize sidebar"
            role="separator"
            tabindex="0"
          ></div>

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
                        <div class="sb-item-title">
                          <FlaskConical size={12} class="sb-icon" />
                          <span>{sandbox.name}</span>
                          {#if sandbox.isActive}
                            <span class="active-badge" title="Active Sandbox Code">ACTIVE</span>
                          {/if}
                        </div>
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
                <div class="sb-item-title">
                  <FlaskConical size={13} class="sb-icon" />
                  <span>{sandbox.name}</span>
                  {#if sandbox.isActive}
                    <span class="active-badge" title="Active Sandbox Code">ACTIVE</span>
                  {/if}
                </div>
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
            <div class="sandbox-control-bar">
              <div class="control-left">
                <button class="sidebar-toggle" onclick={() => showSidebar = !showSidebar} title="Toggle Sidebar">
                  <PanelLeft size={14} />
                </button>
                <input type="text" class="sandbox-title-input" value={activeSandboxName} oninput={(e) => handleAutoRenameSandbox(e.target.value)} />
                <span class="runtime-badge"><CheckCircle2 size={11} /> {activeWorkspaceObj?.name}</span>
              </div>
              <div class="control-actions">
                <button class="action-btn" onclick={() => showSandboxSplit = !showSandboxSplit}>
                  <Columns size={13} /> {showSandboxSplit ? 'Hide Split' : 'Show Split'}
                </button>
                <button class="action-btn" class:has-unsaved={activeSandboxUnsaved} onclick={handleSaveSandbox} title="Save (Ctrl+S / Cmd+S)">
                  <Save size={13} />
                  <span>Save Lab</span>
                  {#if activeSandboxUnsaved}
                    <span class="unsaved-dot-inline" title="Unsaved changes">●</span>
                  {/if}
                </button>
                {#if !isRunning}
                  <button class="action-btn run" onclick={runActiveCode}><Play size={13} fill="#48bb78" stroke="none" /> Run Experiment</button>
                {:else}
                  <button class="action-btn stop" onclick={stopActiveProcess}><Square size={13} fill="#ff5a36" stroke="none" /> Stop</button>
                {/if}
              </div>
            </div>

            <!-- Working Split Area -->
            <div class="sandbox-working-split" class:relative-container={isActivatingSandbox}>
              {#if isActivatingSandbox}
                <div class="activation-overlay">
                  <Loader2 size={24} class="spinner" />
                  <span>Syncing sandbox files...</span>
                </div>
              {/if}

              <!-- Left: Virtual Files + Code Editor -->
              <div class="sandbox-main-area">
                {#if showSandboxExplorer}
                  <div class="v-files-sidebar">
                    <div class="v-files-header">
                      <span>FILES</span>
                      <button class="icon-btn" onclick={handleCreateSandboxFile}><Plus size={12} /></button>
                    </div>
                    <div class="v-files-list">
                      {#each sandboxFilesList as vFile (vFile.path)}
                        {@const isUnsaved = (vFile.path === activeSandboxFilePath && activeSandboxFileUnsaved)}
                        <div class="v-file-row" class:active={activeSandboxFilePath === vFile.path}>
                          <button class="v-file-btn" onclick={() => selectSandboxFile(vFile)}>
                            <FileCode size={12} />
                            <span class="v-file-name">{vFile.path}</span>
                            {#if isUnsaved}
                              <span class="unsaved-dot" title="Unsaved changes">●</span>
                            {/if}
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
                    <div class="empty-editor-hint">Select or create a file to edit source code.</div>
                  {/if}
                </div>
              </div>

              <!-- Right Pane: Notes / HTML Canvas / YAML Spec -->
              {#if showSandboxSplit}
                <div class="sandbox-notes-pane">
                  <div class="notes-tab-bar">
                    <button class="notes-tab" class:active={sandboxSplitType === 'markdown'} onclick={() => sandboxSplitType = 'markdown'}>
                      <BookOpen size={12} /> Markdown Notes ({activeFileNotes.length})
                    </button>
                    <button class="notes-tab" class:active={sandboxSplitType === 'html'} onclick={() => sandboxSplitType = 'html'}>
                      <Sparkles size={12} /> Live Canvas
                    </button>
                    <button class="notes-tab" class:active={sandboxSplitType === 'yaml'} onclick={() => sandboxSplitType = 'yaml'}>
                      <Settings size={12} /> Workspace YAML
                    </button>
                  </div>

                  <div class="notes-pane-content">
                    {#if sandboxSplitType === 'markdown'}
                      <div class="split-pane">
                        <div class="pane-action-header">
                          <span class="pane-title">NOTES FOR {activeSandboxFileName || 'FILE'} (FIFO ORDER)</span>
                          <button class="edit-toggle-btn primary" onclick={handleAddFileNote} disabled={!activeSandboxFilePath}>
                            <Plus size={11} /> Add Note Entry
                          </button>
                        </div>

                        <div class="fifo-notes-scroll">
                          {#if activeFileNotes.length === 0}
                            <div class="empty-notes-hint">
                              <FileText size={20} />
                              <span>No markdown notes for <strong>{activeSandboxFileName || 'this file'}</strong> yet.</span>
                              <button class="modal-btn secondary" onclick={handleAddFileNote} disabled={!activeSandboxFilePath}>
                                <Plus size={12} /> Add First Markdown Note
                              </button>
                            </div>
                          {/if}

                          {#each activeFileNotes as note, index (note.id)}
                            <div class="file-note-card">
                              <div class="note-card-header">
                                <span class="note-index-badge">#{index + 1}</span>
                                <span class="note-title-display">{note.title || 'Note Entry'}</span>
                                <div class="note-header-actions">
                                  <button class="edit-toggle-btn" onclick={() => editingNoteIds[note.id] = !editingNoteIds[note.id]}>
                                    {#if editingNoteIds[note.id]}<Eye size={11} /> View{:else}<Edit3 size={11} /> Edit{/if}
                                  </button>
                                  <button class="icon-btn danger" onclick={() => handleDeleteFileNote(note.id)}><Trash2 size={11} /></button>
                                </div>
                              </div>

                              {#if editingNoteIds[note.id]}
                                <div class="note-edit-box">
                                  <input type="text" class="modal-input note-title-input" bind:value={note.title} placeholder="Note Title" />
                                  <textarea class="raw-textarea note-textarea" bind:value={note.content} placeholder="Markdown notes..."></textarea>
                                  <button class="modal-btn primary mini-btn" onclick={() => handleSaveFileNote(note)}>
                                    <Save size={11} /> Save Note Entry
                                  </button>
                                </div>
                              {:else}
                                <div class="markdown-preview">
                                  {#each note.content.split('\n') as line}
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
                          {/each}
                        </div>
                      </div>

                    {:else if sandboxSplitType === 'html'}
                      <div class="split-pane">
                        <div class="pane-action-header">
                          <span class="pane-title">LIVE CANVAS</span>
                          <button class="edit-toggle-btn" onclick={() => isSandboxHTMLEditing = !isSandboxHTMLEditing}>
                            {#if isSandboxHTMLEditing}<Eye size={11} /> Live{:else}<Edit3 size={11} /> Edit{/if}
                          </button>
                        </div>
                        {#if isSandboxHTMLEditing}
                          <textarea class="raw-textarea code-family" bind:value={activeSandboxHTMLNote} placeholder="HTML/SVG canvas..."></textarea>
                        {:else}
                          <div class="html-frame-wrapper">
                            <iframe title="Visual Frame" srcdoc={activeSandboxHTMLNote} sandbox="allow-scripts" class="html-iframe"></iframe>
                          </div>
                        {/if}
                      </div>

                    {:else if sandboxSplitType === 'yaml'}
                      <div class="split-pane">
                        <div class="pane-action-header"><span class="pane-title">WORKSPACE SPEC (YAML)</span></div>
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
              bind:systemLogs
              bind:consoleStatus
              bind:isConsoleRunning
              bind:terminals={sandboxTerminals}
              bind:activeTermId={activeSandboxTermId}
              onSendTerminalInput={sendTerminalInput}
              onCloseTerminal={closeSandboxTerminal}
              onAddTerminal={async () => {
                try {
                  const sandboxDir = await GetWorkspaceRuntimePath(activeWorkspaceId);
                  const id = createSandboxTerminal('bash', sandboxDir);
                  await StartTerminalSession(id, sandboxDir);
                } catch (_) {}
              }}
              onClearConsole={() => {
                if (bottomMode === 'syslog') systemLogs = [];
                else { consoleLogs = []; consoleStatus = 'Ready'; }
              }}
              onMinimize={() => isConsoleOpen = false}
            />
          </div>
        {:else}
          <div class="empty-workbench">
            <FlaskConical size={48} style="color: #ff5a36;" />
            <h2>Laboratory Workbench Ready</h2>
            <p>Select or create a sandbox experiment to begin coding.</p>
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
            <p>Organize workspace runtime database containers and configurations.</p>
          </div>
          <div class="wm-actions">
            <label class="modal-btn secondary file-btn">
              <Upload size={13} /> Restore Workspace
              <input type="file" accept=".json" onchange={handleRestoreWorkspaceFile} hidden />
            </label>
            <button class="modal-btn primary" onclick={openCreateWorkspaceModal}><Plus size={13} /> New Workspace</button>
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
                <button class="modal-btn primary" onclick={() => { switchWorkspace(ws.id); activeTab = 'lab'; }}>Select Workspace</button>
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

          <div class="danger-zone">
            <h3><AlertTriangle size={14} /> Factory Reset &amp; Reinitialization</h3>
            <p class="danger-desc">
              Wipe all SQLite database containers, workspace environments, sandboxes, and restore CrabCode to default state.
            </p>
            <button class="modal-btn danger-btn" onclick={() => showResetConfirmModal = true}>
              <RefreshCw size={13} /> Reset System
            </button>
          </div>
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

  /* Non-intrusive Toast Container in Bottom-Right Corner */
  .toast-container {
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }

  .toast {
    pointer-events: auto;
    background-color: #12121a;
    border: 1px solid #2d3748;
    color: #edf2f7;
    padding: 8px 14px;
    border-radius: 6px;
    font-size: 12px;
    box-shadow: 0 4px 12px #00000088;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .toast.error { border-color: #e53e3e88; background-color: #1c1012; color: #fc8181; }
  .toast.success { border-color: #48bb7888; background-color: #0f1c14; color: #68d391; }
  .toast.info { border-color: #3178c688; background-color: #101622; color: #63b3ed; }

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

  /* Custom Workspace Popover Dropdown */
  .workspace-dropdown-wrapper {
    position: relative;
    user-select: none;
  }

  .workspace-pill-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #12121a;
    padding: 4px 10px;
    border-radius: 6px;
    border: 1px solid #2d3748;
    color: #edf2f7;
    cursor: pointer;
    font-size: 12px;
    transition: border-color 0.15s, background-color 0.15s;
  }

  .workspace-pill-btn:hover {
    border-color: #ff5a3688;
    background-color: #1a1a26;
  }

  .ws-pill-label {
    font-size: 10px;
    font-weight: 800;
    color: #6b7280;
    letter-spacing: 0.5px;
  }

  .ws-pill-name {
    font-weight: 600;
    color: #edf2f7;
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.ws-chevron) {
    color: #718096;
    transition: transform 0.2s;
  }

  :global(.ws-chevron.open) {
    transform: rotate(180deg);
  }

  .workspace-dropdown-menu {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    width: 280px;
    background-color: #0f0f18;
    border: 1px solid #2d3748;
    border-radius: 8px;
    box-shadow: 0 10px 25px #000000aa;
    z-index: 1000;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .ws-menu-header {
    height: 30px;
    padding: 0 10px;
    background-color: #09090e;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 10px;
    font-weight: 800;
    color: #6b7280;
    letter-spacing: 0.5px;
  }

  .ws-menu-list {
    max-height: 220px;
    overflow-y: auto;
    padding: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .ws-menu-item {
    background: none;
    border: none;
    padding: 8px 10px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: space-between;
    text-align: left;
    color: #a0aec0;
    transition: background-color 0.1s, color 0.1s;
  }

  .ws-menu-item:hover {
    background-color: #1a1a26;
    color: #edf2f7;
  }

  .ws-menu-item.active {
    background-color: #ff5a3618;
    color: #ff5a36;
  }

  .ws-item-left {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
  }

  .ws-item-name {
    font-size: 12px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ws-item-desc {
    font-size: 10px;
    color: #6b7280;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.ws-active-check) {
    color: #48bb78;
    flex-shrink: 0;
  }

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

  /* Expandable / Resizable Sidebar */
  .sidebar {
    position: relative;
    width: 260px;
    min-width: 260px;
    max-width: 260px;
    background-color: #0c0c12;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-sizing: border-box;
    transition: width 0.05s ease-out;
  }

  .sidebar-resize-handle {
    position: absolute;
    top: 0; right: -3px; bottom: 0;
    width: 6px;
    cursor: ew-resize;
    z-index: 100;
    background: transparent;
  }

  .sidebar-resize-handle:hover,
  .sidebar-resize-handle.resizing {
    background-color: #ff5a36aa;
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
    flex: 1;
  }

  .active-badge {
    font-size: 9px;
    font-weight: 800;
    background-color: #48bb7822;
    color: #48bb78;
    border: 1px solid #48bb7844;
    padding: 1px 4px;
    border-radius: 3px;
    margin-left: 4px;
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
    width: 200px;
  }

  .sandbox-title-input:hover,
  .sandbox-title-input:focus {
    border-color: #2d3748;
    background-color: #12121a;
  }

  .runtime-badge {
    font-size: 10px;
    color: #48bb78;
    background-color: #48bb7814;
    border: 1px solid #48bb7833;
    padding: 2px 6px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 4px;
    font-weight: 600;
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
  .action-btn.has-unsaved { border-color: #ff5a36aa; color: #edf2f7; }
  .action-btn.run { background-color: #48bb7822; color: #48bb78; border-color: #48bb7844; }
  .action-btn.run:hover { background-color: #48bb7833; }
  .action-btn.stop { background-color: #ff5a3622; color: #ff5a36; border-color: #ff5a3644; }

  /* Unsaved Dot Indicator */
  .unsaved-dot {
    color: #ff5a36;
    font-size: 10px;
    margin-left: auto;
    filter: drop-shadow(0 0 3px #ff5a3688);
  }

  .unsaved-dot-inline {
    color: #ff5a36;
    font-size: 10px;
    margin-left: 2px;
    filter: drop-shadow(0 0 3px #ff5a3688);
  }

  /* Split area */
  .sandbox-working-split {
    flex: 1;
    display: flex;
    flex-direction: row;
    min-height: 0;
    overflow: hidden;
    position: relative;
  }

  .activation-overlay {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    background-color: #0b0b0fdd;
    z-index: 50;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    color: #ff5a36;
    font-size: 13px;
    font-weight: 600;
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

  .edit-toggle-btn.primary {
    color: #ff5a36;
    font-weight: 600;
  }

  .edit-toggle-btn:hover { color: #edf2f7; }

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

  .yaml-editor {
    height: 160px;
    border: 1px solid #2d3748;
    border-radius: 4px;
  }

  .markdown-preview {
    padding: 12px;
    font-size: 12px;
    line-height: 1.6;
    color: #cbd5e0;
  }

  .markdown-preview h1 { font-size: 16px; color: #edf2f7; border-bottom: 1px solid #2d3748; padding-bottom: 4px; margin-top: 0; }
  .markdown-preview h2 { font-size: 14px; color: #edf2f7; margin-top: 12px; }
  .markdown-preview h3 { font-size: 12px; color: #ff5a36; margin-top: 10px; }
  .markdown-preview blockquote { border-left: 3px solid #ff5a36; margin: 8px 0; padding-left: 10px; color: #a0aec0; }
  .markdown-preview .codeblock { background: #07070a; padding: 8px; border-radius: 4px; font-family: monospace; font-size: 11px; color: #34d399; }

  .fifo-notes-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .empty-notes-hint {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px 20px;
    color: #6b7280;
    font-size: 12px;
    gap: 10px;
    text-align: center;
  }

  .file-note-card {
    background-color: #08080d;
    border: 1px solid #1a1a24;
    border-radius: 6px;
    overflow: hidden;
  }

  .note-card-header {
    height: 28px;
    background-color: #0f0f18;
    border-bottom: 1px solid #1a1a24;
    padding: 0 10px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .note-index-badge {
    font-size: 10px;
    font-weight: 800;
    color: #ff5a36;
    background-color: #ff5a3618;
    padding: 1px 5px;
    border-radius: 3px;
  }

  .note-title-display {
    font-size: 11px;
    font-weight: 700;
    color: #edf2f7;
    flex: 1;
    margin-left: 8px;
  }

  .note-header-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .note-edit-box {
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .note-title-input {
    font-size: 12px;
    padding: 4px 8px;
  }

  .note-textarea {
    height: 120px;
    border: 1px solid #2d3748;
    border-radius: 4px;
  }

  .mini-btn {
    align-self: flex-end;
    padding: 4px 8px;
    font-size: 11px;
  }

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

  /* Environment Initialization Loading Modal Box */
  .setup-loading-box {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
  }

  .loading-status-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    font-weight: 600;
    color: #ff5a36;
  }

  .setup-console-box {
    background-color: #07070a;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 8px 10px;
    max-height: 140px;
    overflow-y: auto;
    font-family: 'Fira Code', monospace;
    font-size: 11px;
    color: #34d399;
  }

  .setup-line {
    line-height: 1.4;
    white-space: pre-wrap;
  }

  :global(.spinner) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
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
    width: 420px;
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
    display: flex;
    align-items: center;
    gap: 8px;
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

  .config-mode-toggle {
    display: flex;
    gap: 4px;
    background-color: #07070a;
    border: 1px solid #2d3748;
    padding: 3px;
    border-radius: 6px;
    margin-top: 4px;
  }

  .mode-btn {
    flex: 1;
    background: none;
    border: none;
    color: #a0aec0;
    font-size: 11px;
    font-weight: 600;
    padding: 6px;
    border-radius: 4px;
    cursor: pointer;
  }

  .mode-btn.active {
    background-color: #ff5a361a;
    color: #ff5a36;
  }

  .sandbox-modal-label {
    font-size: 11px;
    font-weight: 600;
    color: #a0aec0;
  }

  .templates-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    max-height: 180px;
    overflow-y: auto;
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

  .template-name { font-size: 11px; font-weight: 600; }

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

  .settings-view {
    flex: 1;
    padding: 40px;
    display: flex;
    justify-content: center;
    overflow-y: auto;
  }

  .settings-card {
    background: #0c0c12;
    border: 1px solid #1a1a24;
    border-radius: 8px;
    padding: 24px;
    width: 520px;
    height: fit-content;
  }

  .settings-card h2 {
    margin: 0 0 8px 0;
    font-size: 16px;
    color: #edf2f7;
  }

  .settings-card > p {
    font-size: 12px;
    color: #718096;
    margin: 0 0 6px 0;
  }

  .danger-zone {
    margin-top: 32px;
    padding-top: 20px;
    border-top: 1px solid #2d374844;
  }

  .danger-zone h3 {
    margin: 0 0 8px 0;
    font-size: 13px;
    color: #fc8181;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .danger-desc {
    font-size: 12px;
    color: #a0aec0;
    line-height: 1.5;
    margin: 0 0 16px 0;
  }

  .danger-modal { border-color: #e53e3e44; }
  .danger-header { color: #fc8181; display: flex; align-items: center; gap: 8px; }
  .danger-text { color: #feb2b2; }

  .modal-btn.danger-btn { background-color: #e53e3e; color: #fff; }
  .modal-btn.danger-btn:hover { background-color: #c53030; }

  .guide-text { font-size: 12px; color: #a0aec0; line-height: 1.5; }
  .guide-subtext { font-size: 11px; color: #718096; line-height: 1.4; margin-top: 6px; }
</style>
