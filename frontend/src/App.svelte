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
    RenamePath
  } from '../wailsjs/go/main/App';
  import FileNode from './FileNode.svelte';
  import { onMount, untrack } from 'svelte';
  import { FolderOpen, FilePlus, FolderPlus, Undo, Redo } from '@lucide/svelte';

  let currentFolder = $state('');
  let fileTree = $state([]);
  let expandedFolders = $state({});
  let folderContents = $state({});
  let activeFilePath = $state('');
  let activeFileName = $state('');
  let editorContent = $state('');
  let isSaving = $state(false);
  let statusMessage = $state('Ready');

  let autoSaveEnabled = $state(true);

  let toasts = $state([]);
  let modal = $state({ show: false, title: '', placeholder: '', value: '', onConfirm: null, onCancel: null });
  
  let contextMenu = $state({ show: false, x: 0, y: 0, node: null });

  let textareaEl = $state(null);
  let highlightEl = $state(null);
  let lineNumbersEl = $state(null);

  let lineCount = $derived(editorContent.split('\n').length);
  let charCount = $derived(editorContent.length);
  let fileExtension = $derived(
    activeFileName 
      ? (activeFileName.split('.').pop() || '').toUpperCase() 
      : 'TEXT'
  );

  let lastSavedContent = $state('');
  let activeFileUnsaved = $derived(activeFilePath !== '' && editorContent !== lastSavedContent);

  let history = $state([]);
  let historyIndex = $state(-1);
  let lastHistoryContent = '';
  let isUndoingOrRedoing = false;
  let historyTimeout;
  const MAX_HISTORY = 100;

  function highlight(code, ext) {
    if (!code) return '';
    let escaped = code
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');

    if (!ext) return escaped;
    const lang = ext.toLowerCase();

    let keywords = [];
    if (lang === 'go') {
      keywords = ['package', 'import', 'func', 'var', 'const', 'type', 'struct', 'interface', 'return', 'if', 'else', 'for', 'range', 'switch', 'case', 'default', 'select', 'chan', 'go', 'defer', 'nil', 'true', 'false', 'map', 'make', 'new'];
    } else if (['js', 'ts', 'svelte'].includes(lang)) {
      keywords = ['import', 'from', 'export', 'default', 'const', 'let', 'var', 'function', 'class', 'extends', 'return', 'if', 'else', 'for', 'while', 'do', 'switch', 'case', 'break', 'continue', 'new', 'this', 'true', 'false', 'null', 'undefined', 'async', 'await', 'try', 'catch', 'finally', 'throw'];
    } else if (lang === 'rs') {
      keywords = ['fn', 'let', 'mut', 'const', 'static', 'pub', 'use', 'mod', 'struct', 'enum', 'trait', 'impl', 'return', 'if', 'else', 'loop', 'while', 'for', 'in', 'match', 'as', 'ref', 'self', 'Self', 'true', 'false', 'unsafe', 'where', 'type'];
    } else if (lang === 'py') {
      keywords = ['def', 'class', 'import', 'from', 'as', 'return', 'if', 'elif', 'else', 'for', 'while', 'in', 'is', 'and', 'or', 'not', 'try', 'except', 'finally', 'raise', 'with', 'assert', 'pass', 'break', 'continue', 'lambda', 'None', 'True', 'False'];
    }

    let placeholders = [];
    let id = 0;
    function keep(text, className) {
      const ph = `___TOKEN_PH_${id++}___`;
      placeholders.push({ ph, text, className });
      return ph;
    }

    if (lang === 'py') {
      escaped = escaped.replace(/(#.*)/g, (m) => keep(m, 'hl-comment'));
    } else if (lang === 'html' || lang === 'svelte') {
      escaped = escaped.replace(/(&lt;!--[\s\S]*?--&gt;)/g, (m) => keep(m, 'hl-comment'));
    } else {
      escaped = escaped.replace(/(\/\*[\s\S]*?\*\/)/g, (m) => keep(m, 'hl-comment'));
      escaped = escaped.replace(/(\/\/.*)/g, (m) => keep(m, 'hl-comment'));
    }

    if (lang === 'py') {
      escaped = escaped.replace(/("""[\s\S]*?"""|'''[\s\S]*?''')/g, (m) => keep(m, 'hl-string'));
      escaped = escaped.replace(/('[^']*'|"[^"]*")/g, (m) => keep(m, 'hl-string'));
    } else if (['html', 'svelte', 'css'].includes(lang)) {
      escaped = escaped.replace(/('[^']*'|"[^"]*")/g, (m) => keep(m, 'hl-string'));
    } else {
      escaped = escaped.replace(/(`[\s\S]*?`)/g, (m) => keep(m, 'hl-string'));
      escaped = escaped.replace(/("(\\.|[^"\\])*"|'(\\.|[^'\\])*')/g, (m) => keep(m, 'hl-string'));
    }

    escaped = escaped.replace(/\b(\d+(\.\d+)?)\b/g, '<span class="hl-number">$1</span>');

    if (lang === 'css') {
      escaped = escaped.replace(/([a-zA-Z0-9_-]+)\s*:/g, '<span class="hl-keyword">$1</span>:');
      escaped = escaped.replace(/(#[a-zA-Z0-9_-]+|\.[a-zA-Z0-9_-]+)/g, '<span class="hl-type">$1</span>');
    } else if (lang === 'html' || lang === 'svelte') {
      escaped = escaped.replace(/&lt;(\/?[a-zA-Z0-9:-]+)/g, '&lt;<span class="hl-keyword">$1</span>');
      escaped = escaped.replace(/\s([a-zA-Z0-9:-]+)=/g, ' <span class="hl-type">$1</span>=');
    } else if (lang === 'json') {
      escaped = escaped.replace(/"([^"]+)":/g, '"<span class="hl-keyword">$1</span>":');
      escaped = escaped.replace(/\b(true|false|null)\b/g, '<span class="hl-type">$1</span>');
    } else {
      if (keywords.length > 0) {
        const kwRegex = new RegExp(`\\b(${keywords.join('|')})\\b`, 'g');
        escaped = escaped.replace(kwRegex, '<span class="hl-keyword">$1</span>');
      }
      escaped = escaped.replace(/\b([a-zA-Z0-9_]+)(?=\s*\()/g, '<span class="hl-function">$1</span>');
      const defaultTypes = ['int', 'float', 'string', 'bool', 'char', 'void', 'error', 'any', 'byte', 'rune', 'usize', 'u32', 'i32', 'u64', 'i64', 'f64', 'f32'];
      escaped = escaped.replace(new RegExp(`\\b(${defaultTypes.join('|')})\\b`, 'g'), '<span class="hl-type">$1</span>');
      escaped = escaped.replace(/\b([A-Z][a-zA-Z0-9_]*)\b/g, '<span class="hl-type">$1</span>');
    }

    for (let i = placeholders.length - 1; i >= 0; i--) {
      const { ph, text, className } = placeholders[i];
      escaped = escaped.replace(ph, `<span class="${className}">${text}</span>`);
    }

    return escaped;
  }

  let highlightedCode = $derived(highlight(editorContent, fileExtension));

  function syncScroll() {
    if (textareaEl && highlightEl) {
      highlightEl.scrollTop = textareaEl.scrollTop;
      highlightEl.scrollLeft = textareaEl.scrollLeft;
    }
    if (textareaEl && lineNumbersEl) {
      lineNumbersEl.scrollTop = textareaEl.scrollTop;
    }
  }

  let toastId = 0;
  function addToast(message, type = 'info', duration = 3000) {
    const id = toastId++;
    toasts = [...toasts, { id, message, type }];
    setTimeout(() => {
      toasts = toasts.filter(t => t.id !== id);
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

  let autoSaveTimeout;

  $effect(() => {
    const content = editorContent;
    const path = activeFilePath;
    
    if (autoSaveEnabled && path) {
      if (content !== untrack(() => lastSavedContent)) {
        if (autoSaveTimeout) clearTimeout(autoSaveTimeout);
        autoSaveTimeout = setTimeout(async () => {
          isSaving = true;
          try {
            await SaveFile(path, content);
            lastSavedContent = content;
            addToast('Auto-saved successfully', 'success', 1500);
          } catch (err) {
            addToast('Auto-save failed: ' + err, 'error');
          } finally {
            isSaving = false;
          }
        }, 1500);
      }
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

  $effect(() => {
    const _ = editorContent;
    setTimeout(syncScroll, 0);
  });

  async function chooseFolder() {
    try {
      const folder = await SelectFolder();
      if (folder) {
        currentFolder = folder;
        const contents = await ListDirectory(folder);
        sortNodes(contents);
        fileTree = contents;
        expandedFolders = {};
        folderContents = {};
        activeFilePath = '';
        activeFileName = '';
        editorContent = '';
        lastSavedContent = '';
        statusMessage = 'Opened directory: ' + folder;
        addToast('Workspace loaded successfully', 'success');
      }
    } catch (err) {
      statusMessage = 'Error opening folder: ' + err;
      addToast(err.toString(), 'error');
    }
  }

  async function createFile() {
    if (!currentFolder) return;
    const name = await openPrompt('Create New File', 'filename.js');
    if (!name) return;
    const path = currentFolder + '/' + name;
    try {
      await CreateFile(path);
      const contents = await ListDirectory(currentFolder);
      sortNodes(contents);
      fileTree = contents;
      statusMessage = 'Created file: ' + name;
      addToast('Created file "' + name + '"', 'success');
    } catch (err) {
      statusMessage = 'Error creating file: ' + err;
      addToast(err.toString(), 'error');
    }
  }

  async function createFolder() {
    if (!currentFolder) return;
    const name = await openPrompt('Create New Folder', 'components');
    if (!name) return;
    const path = currentFolder + '/' + name;
    try {
      await CreateDirectory(path);
      const contents = await ListDirectory(currentFolder);
      sortNodes(contents);
      fileTree = contents;
      statusMessage = 'Created folder: ' + name;
      addToast('Created folder "' + name + '"', 'success');
    } catch (err) {
      statusMessage = 'Error creating folder: ' + err;
      addToast(err.toString(), 'error');
    }
  }

  function sortNodes(nodes) {
    nodes.sort((a, b) => {
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;
      return a.name.localeCompare(b.name);
    });
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
        statusMessage = 'Error reading folder contents: ' + err;
        addToast(err.toString(), 'error');
      }
    }
  }

  async function openFile(node) {
    if (autoSaveTimeout) {
      clearTimeout(autoSaveTimeout);
      autoSaveTimeout = null;
    }
    if (historyTimeout) {
      clearTimeout(historyTimeout);
      historyTimeout = null;
    }
    try {
      const content = await ReadFile(node.path);
      activeFilePath = node.path;
      activeFileName = node.name;
      editorContent = content;
      lastSavedContent = content;

      history = [{
        content: content,
        selectionStart: 0,
        selectionEnd: 0
      }];
      historyIndex = 0;
      lastHistoryContent = content;

      statusMessage = 'Opened file: ' + node.name;
      addToast('Opened ' + node.name);
    } catch (err) {
      statusMessage = 'Error reading file: ' + err;
      addToast(err.toString(), 'error');
    }
  }

  async function saveCurrentFile() {
    if (!activeFilePath) return;
    isSaving = true;
    statusMessage = 'Saving file...';
    try {
      await SaveFile(activeFilePath, editorContent);
      lastSavedContent = editorContent;
      statusMessage = 'File saved successfully';
      addToast('Saved successfully', 'success');
      setTimeout(() => {
        if (statusMessage === 'File saved successfully') {
          statusMessage = 'Editing: ' + activeFileName;
        }
      }, 2000);
    } catch (err) {
      statusMessage = 'Error saving file: ' + err;
      addToast(err.toString(), 'error');
    } finally {
      isSaving = false;
    }
  }

  function recordHistoryState(immediate = false) {
    if (isUndoingOrRedoing) return;
    if (!textareaEl) return;
    if (editorContent === lastHistoryContent) return;

    const newState = {
      content: editorContent,
      selectionStart: textareaEl.selectionStart,
      selectionEnd: textareaEl.selectionEnd
    };

    const save = () => {
      const truncatedHistory = history.slice(0, historyIndex + 1);
      history = [...truncatedHistory, newState];
      if (history.length > MAX_HISTORY) {
        history.shift();
      }
      historyIndex = history.length - 1;
      lastHistoryContent = editorContent;
    };

    if (historyTimeout) clearTimeout(historyTimeout);

    if (immediate) {
      save();
    } else {
      historyTimeout = setTimeout(save, 800);
    }
  }

  function handleTextareaInput() {
    recordHistoryState(false);
  }

  function handleTextareaKeyDown(event) {
    if (event.key === ' ' || event.key === 'Enter' || event.key === 'Tab') {
      recordHistoryState(true);
    }
  }

  function undo() {
    if (historyIndex > 0) {
      if (editorContent !== lastHistoryContent) {
        recordHistoryState(true);
      }

      isUndoingOrRedoing = true;
      historyIndex--;
      const state = history[historyIndex];
      editorContent = state.content;
      lastHistoryContent = state.content;

      setTimeout(() => {
        if (textareaEl) {
          textareaEl.focus();
          textareaEl.setSelectionRange(state.selectionStart, state.selectionEnd);
          syncScroll();
        }
        isUndoingOrRedoing = false;
      }, 0);
    } else {
      addToast('Nothing to undo', 'info', 1000);
    }
  }

  function redo() {
    if (historyIndex < history.length - 1) {
      isUndoingOrRedoing = true;
      historyIndex++;
      const state = history[historyIndex];
      editorContent = state.content;
      lastHistoryContent = state.content;

      setTimeout(() => {
        if (textareaEl) {
          textareaEl.focus();
          textareaEl.setSelectionRange(state.selectionStart, state.selectionEnd);
          syncScroll();
        }
        isUndoingOrRedoing = false;
      }, 0);
    } else {
      addToast('Nothing to redo', 'info', 1000);
    }
  }

  function handleKeyDown(event) {
    const isMeta = event.ctrlKey || event.metaKey;
    if (isMeta && event.key === 's') {
      event.preventDefault();
      saveCurrentFile();
    } else if (isMeta && event.shiftKey && (event.key === 'z' || event.key === 'Z')) {
      event.preventDefault();
      redo();
    } else if (isMeta && event.key === 'z') {
      event.preventDefault();
      undo();
    } else if (isMeta && event.key === 'y') {
      event.preventDefault();
      redo();
    }
  }

  function handleNodeContextMenu(event, node) {
    contextMenu = {
      show: true,
      x: event.clientX,
      y: event.clientY,
      node: node
    };
  }

  function closeContextMenu() {
    contextMenu.show = false;
  }

  function getParentPath(path) {
    let cleanPath = path.replace(/[\\/]$/, '');
    const lastIdx = Math.max(cleanPath.lastIndexOf('/'), cleanPath.lastIndexOf('\\'));
    if (lastIdx === -1) return '';
    return cleanPath.substring(0, lastIdx);
  }

  async function refreshDirectory(dirPath) {
    try {
      const contents = await ListDirectory(dirPath);
      sortNodes(contents);
      if (dirPath === currentFolder) {
        fileTree = contents;
      } else {
        folderContents[dirPath] = contents;
      }
    } catch (err) {
      addToast('Error refreshing directory: ' + err, 'error');
    }
  }

  async function deleteNode(node) {
    const confirmDelete = confirm('Are you sure you want to delete "' + node.name + '"?');
    if (!confirmDelete) return;

    try {
      await DeletePath(node.path);
      addToast('Deleted "' + node.name + '"', 'success');

      delete folderContents[node.path];
      delete expandedFolders[node.path];

      const isWindows = node.path.includes('\\');
      const separator = isWindows ? '\\' : '/';
      if (
        activeFilePath === node.path || 
        activeFilePath.startsWith(node.path + separator)
      ) {
        activeFilePath = '';
        activeFileName = '';
        editorContent = '';
        lastSavedContent = '';
        history = [];
        historyIndex = -1;
      }

      const parentPath = getParentPath(node.path);
      if (parentPath) {
        await refreshDirectory(parentPath);
      } else {
        await refreshDirectory(currentFolder);
      }
    } catch (err) {
      addToast('Error deleting item: ' + err, 'error');
    }
  }

  async function renameNode(node) {
    const parentPath = getParentPath(node.path);
    const newName = await openPrompt('Rename', node.name, node.name);
    if (!newName || newName === node.name) return;

    const isWindows = node.path.includes('\\');
    const separator = isWindows ? '\\' : '/';
    const newPath = parentPath ? (parentPath + separator + newName) : (currentFolder + separator + newName);

    try {
      await RenamePath(node.path, newPath);
      addToast('Renamed to "' + newName + '"', 'success');

      if (activeFilePath === node.path) {
        activeFilePath = newPath;
        activeFileName = newName;
      } else if (activeFilePath.startsWith(node.path + separator)) {
        activeFilePath = newPath + activeFilePath.substring(node.path.length);
      }

      if (node.isDir) {
        if (expandedFolders[node.path]) {
          expandedFolders[newPath] = expandedFolders[node.path];
          folderContents[newPath] = folderContents[node.path];
        }
        delete folderContents[node.path];
        delete expandedFolders[node.path];
      }

      if (parentPath) {
        await refreshDirectory(parentPath);
      } else {
        await refreshDirectory(currentFolder);
      }
    } catch (err) {
      addToast('Error renaming item: ' + err, 'error');
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });
</script>

<svelte:window onclick={closeContextMenu} onkeydown={(e) => e.key === 'Escape' && closeContextMenu()} />

<div class="toast-container">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}">
      <span class="toast-message">{toast.message}</span>
    </div>
  {/each}
</div>

{#if contextMenu.show}
  <div 
    class="context-menu" 
    style="top: {contextMenu.y}px; left: {contextMenu.x}px;"
    onclick={(e) => e.stopPropagation()}
    role="menu"
    tabindex="-1"
  >
    <button class="context-item" onclick={() => { renameNode(contextMenu.node); closeContextMenu(); }}>
      Rename
    </button>
    <button class="context-item delete" onclick={() => { deleteNode(contextMenu.node); closeContextMenu(); }}>
      Delete
    </button>
  </div>
{/if}

{#if modal.show}
  <div class="modal-backdrop" onclick={modal.onCancel} role="presentation">
    <div class="modal-box" onclick={(e) => e.stopPropagation()} role="dialog" tabindex="-1" onkeydown={(e) => e.key === 'Escape' && modal.onCancel()}>
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

<div class="app-container">
  <aside class="sidebar">
    <div class="sidebar-header">
      <button class="open-btn" onclick={chooseFolder}>
        <FolderOpen size={14} />
        <span>Open Folder</span>
      </button>
    </div>

    {#if currentFolder}
      <div class="sidebar-toolbar">
        <button class="toolbar-btn" onclick={createFile} title="New File">
          <FilePlus size={14} />
        </button>
        <button class="toolbar-btn" onclick={createFolder} title="New Folder">
          <FolderPlus size={14} />
        </button>
      </div>
    {/if}
    <div class="file-tree-container">
      {#if currentFolder}
        <div class="project-title" title={currentFolder}>
          {currentFolder.split(/[\\/]/).pop()}
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
              onNodeContextMenu={handleNodeContextMenu}
            />
          {/each}
        </div>
      {:else}
        <div class="empty-tree-state">
          <p>No project workspace loaded.</p>
          <p class="subtitle">Open a folder to begin.</p>
        </div>
      {/if}
    </div>
  </aside>

  <main class="editor-panel">
    {#if activeFilePath}
      <div class="editor-header">
        <div class="file-info">
          <span class="active-file-name">{activeFileName}</span>
          <span class="active-file-path" title={activeFilePath}>{activeFilePath}</span>
        </div>
        <div class="header-actions">
          <button 
            class="header-action-btn" 
            onclick={undo} 
            disabled={historyIndex <= 0} 
            title="Undo (Ctrl+Z)"
          >
            <Undo size={14} />
          </button>
          <button 
            class="header-action-btn" 
            onclick={redo} 
            disabled={historyIndex >= history.length - 1} 
            title="Redo (Ctrl+Y)"
          >
            <Redo size={14} />
          </button>

          <label class="autosave-label">
            <input type="checkbox" class="autosave-checkbox" bind:checked={autoSaveEnabled} />
            <span>Auto Save</span>
          </label>
        </div>
      </div>

      <div class="editor-body">
        <div class="line-numbers" bind:this={lineNumbersEl}>
          {#each Array(lineCount) as _, i}
            <div class="line-num">{i + 1}</div>
          {/each}
        </div>
        <div class="editor-container-inner">
          <pre class="highlight-overlay" bind:this={highlightEl}>{@html highlightedCode}</pre>
          <textarea
            class="code-textarea"
            bind:value={editorContent}
            bind:this={textareaEl}
            onscroll={syncScroll}
            oninput={handleTextareaInput}
            onkeydown={handleTextareaKeyDown}
            spellcheck="false"
            autocomplete="off"
            autocorrect="off"
            autocapitalize="off"
          ></textarea>
        </div>
      </div>
    {:else}
      <div class="editor-empty-state">
        <div class="welcome-box">
          <h1>Welcome to CrabCode</h1>
          <p>A lightweight dark-mode editor designed for code management.</p>
          
          <div class="quick-actions">
            <button class="action-btn" onclick={chooseFolder}>
              <FolderOpen size={14} />
              <span>Select Folder</span>
            </button>
          </div>

          <div class="shortcuts">
            <div class="shortcut-item">
              <span class="key">Ctrl</span> + <span class="key">S</span>
              <span class="desc">Save active file</span>
            </div>
          </div>
        </div>
      </div>
    {/if}

    <footer class="status-bar">
      <div class="status-left">
        <span class="status-indicator"></span>
        <span class="status-text">{statusMessage}</span>
      </div>
      {#if activeFilePath}
        <div class="status-right">
          <span>Lines: {lineCount}</span>
          <span>Chars: {charCount}</span>
          <span class="lang-tag">{fileExtension}</span>
        </div>
      {/if}
    </footer>
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background-color: #0b0b0f;
    color: #e2e8f0;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    height: 100vh;
    overflow: hidden;
  }

  .app-container {
    display: flex;
    width: 100vw;
    height: 100vh;
    background-color: #0b0b0f;
  }

  .sidebar {
    width: 260px;
    min-width: 220px;
    max-width: 380px;
    background-color: #0f0f14;
    border-right: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .sidebar-header {
    padding: 16px;
    border-bottom: 1px solid #1a1a24;
  }

  .open-btn {
    background-color: #ff5a36;
    color: white;
    border: none;
    padding: 8px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: background-color 0.15s;
    width: 100%;
  }

  .open-btn:hover {
    background-color: #e04b28;
  }

  .sidebar-toolbar {
    display: flex;
    gap: 4px;
    padding: 8px 12px;
    border-bottom: 1px solid #1a1a24;
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
    justify-content: center;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
  }

  .toolbar-btn:hover {
    background-color: #2d3748;
    color: #edf2f7;
    border-color: #4a5568;
  }

  .file-tree-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 10px 4px;
  }

  .project-title {
    font-size: 11px;
    font-weight: 600;
    color: #a0aec0;
    padding: 6px 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tree-scroll {
    flex: 1;
    overflow-y: auto;
  }

  .tree-scroll::-webkit-scrollbar {
    width: 5px;
  }
  .tree-scroll::-webkit-scrollbar-track {
    background: transparent;
  }
  .tree-scroll::-webkit-scrollbar-thumb {
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
  }

  .empty-tree-state p {
    margin: 4px 0;
    font-size: 13px;
  }

  .empty-tree-state .subtitle {
    font-size: 11px;
    opacity: 0.7;
  }

  .editor-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #121217;
    height: 100%;
    overflow: hidden;
    position: relative;
  }

  .editor-header {
    height: 40px;
    padding: 0 16px;
    background-color: #0f0f14;
    border-bottom: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .header-action-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
  }

  .header-action-btn:hover:not(:disabled) {
    background-color: #2d3748;
    color: #edf2f7;
    border-color: #4a5568;
  }

  .header-action-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
    border-color: #1a1a24;
  }

  .autosave-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #a0aec0;
    cursor: pointer;
    user-select: none;
    margin-left: 8px;
  }

  .autosave-checkbox {
    accent-color: #ff5a36;
    cursor: pointer;
  }

  .file-info {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .active-file-name {
    font-size: 13px;
    font-weight: 500;
    color: #edf2f7;
  }

  .active-file-path {
    font-size: 11px;
    color: #4a5568;
    max-width: 350px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-left: 8px;
  }

  .editor-body {
    flex: 1;
    display: flex;
    overflow: hidden;
    background-color: #1e1e1e;
  }

  .line-numbers {
    padding-top: 16px;
    padding-bottom: 16px;
    width: 52px;
    background-color: #1e1e1e;
    border-right: 1px solid #2d2d2d;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    user-select: none;
    color: #858585;
    font-family: 'Fira Code', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
    padding-right: 12px;
    box-sizing: border-box;
    overflow: hidden;
  }

  .line-num {
    height: 1.6em;
    line-height: 1.6;
    display: flex;
    align-items: center;
  }

  .editor-container-inner {
    position: relative;
    flex: 1;
    height: 100%;
    overflow: hidden;
    background-color: #1e1e1e;
  }

  .code-textarea, .highlight-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 16px;
    font-family: 'Fira Code', 'JetBrains Mono', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
    tab-size: 4;
    -moz-tab-size: 4;
    white-space: pre;
    word-wrap: normal;
    overflow: auto;
    box-sizing: border-box;
    border: none;
    outline: none;
  }

  .code-textarea {
    color: transparent;
    background: transparent;
    caret-color: #ff5a36;
    z-index: 2;
    resize: none;
  }

  .highlight-overlay {
    color: #d4d4d4;
    z-index: 1;
    pointer-events: none;
    background-color: transparent;
  }

  :global(.hl-comment) {
    color: #6A9955;
    font-style: italic;
  }
  :global(.hl-string) {
    color: #CE9178;
  }
  :global(.hl-keyword) {
    color: #569CD6;
    font-weight: 500;
  }
  :global(.hl-type) {
    color: #4EC9B0;
  }
  :global(.hl-function) {
    color: #DCDCAA;
  }
  :global(.hl-number) {
    color: #B5CEA8;
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
    font-size: 22px;
    color: #edf2f7;
    margin-bottom: 8px;
    font-weight: 700;
  }

  .welcome-box p {
    font-size: 14px;
    color: #718096;
    margin-bottom: 24px;
    line-height: 1.5;
  }

  .quick-actions {
    margin-bottom: 32px;
  }

  .action-btn {
    background-color: #2d3748;
    color: #edf2f7;
    border: 1px solid #4a5568;
    padding: 8px 24px;
    border-radius: 6px;
    cursor: pointer;
    font-weight: 600;
    font-size: 13px;
    transition: background-color 0.15s, border-color 0.15s;
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .action-btn:hover {
    background-color: #4a5568;
    border-color: #718096;
  }

  .shortcuts {
    background-color: #0f0f14;
    border: 1px solid #1a1a24;
    border-radius: 8px;
    padding: 12px 16px;
    display: inline-block;
  }

  .shortcut-item {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    font-size: 12px;
  }

  .key {
    background-color: #2d3748;
    color: #edf2f7;
    border-radius: 4px;
    padding: 2px 6px;
    font-size: 11px;
    font-weight: 700;
    border-bottom: 2px solid #1a202c;
  }

  .desc {
    color: #718096;
    margin-left: 10px;
  }

  .status-bar {
    height: 24px;
    background-color: #0f0f14;
    border-top: 1px solid #1a1a24;
    padding: 0 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    color: #718096;
    user-select: none;
  }

  .status-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-indicator {
    width: 6px;
    height: 6px;
    background-color: #48bb78;
    border-radius: 50%;
  }

  .status-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .lang-tag {
    background-color: #2d3748;
    color: #cbd5e0;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 600;
    letter-spacing: 0.5px;
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
    background-color: #1a1a24;
    border-left: 4px solid #4a5568;
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

  .context-menu {
    position: fixed;
    background-color: #0f0f14;
    border: 1px solid #1a1a24;
    border-radius: 6px;
    padding: 4px 0;
    min-width: 120px;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5);
    z-index: 100000;
  }

  .context-item {
    width: 100%;
    background: none;
    border: none;
    color: #edf2f7;
    padding: 6px 12px;
    text-align: left;
    font-size: 13px;
    cursor: pointer;
    display: block;
    transition: background-color 0.1s;
  }

  .context-item:hover {
    background-color: #2d3748;
  }

  .context-item.delete {
    color: #ff5a36;
  }

  .context-item.delete:hover {
    background-color: #ff5a3622;
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
    background-color: #0f0f14;
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
    letter-spacing: 0.5px;
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
    transition: background-color 0.1s;
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
