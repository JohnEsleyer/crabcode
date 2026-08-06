<script>
  import { Terminal, Code2, Plus, X, Trash2, Copy, CornerDownLeft, SquareTerminal, Activity } from '@lucide/svelte';

  let {
    bottomMode = $bindable('console'),
    consoleLogs = $bindable([]),
    systemLogs = $bindable([]),
    consoleStatus = $bindable('Ready'),
    isConsoleRunning = false,
    terminals = $bindable([]),
    activeTermId = $bindable(''),
    onSendTerminalInput,
    onCloseTerminal,
    onAddTerminal,
    onClearConsole,
    onMinimize
  } = $props();

  let activeTerm = $derived(terminals.find(t => t.id === activeTermId) || null);
  let drawerHeight = $state(220);
  let isResizing = $state(false);

  function startResize(e) {
    isResizing = true;
    const startY = e.clientY;
    const startHeight = drawerHeight;

    function onMouseMove(event) {
      const deltaY = startY - event.clientY;
      drawerHeight = Math.max(120, Math.min(600, startHeight + deltaY));
    }

    function onMouseUp() {
      isResizing = false;
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    }

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  function formatTime(t) {
    if (!t) return '';
    const d = new Date(t);
    return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function getStatusColor(status) {
    if (status === 'Executing...') return '#ff5a36';
    if (status === 'Finished') return '#48bb78';
    if (status === 'Error') return '#e53e3e';
    if (status === 'Stopped') return '#ecc94b';
    if (status === 'Uninitialized Env') return '#ecc94b';
    return '#718096';
  }

  async function handleCopyConsole() {
    const logsToCopy = (bottomMode === 'syslog') ? systemLogs : consoleLogs;
    const text = logsToCopy.map(l => l.text || l).join('\n');
    try {
      await navigator.clipboard.writeText(text);
    } catch (_) {}
  }
</script>

<div class="console-drawer multi-term" style="height: {drawerHeight}px;">
  <div
    class="resize-handle"
    onmousedown={startResize}
    class:resizing={isResizing}
    title="Drag to resize bottom panel"
    role="separator"
    tabindex="0"
  ></div>

  <div class="panel-main-area">
    <div class="panel-header">
      <div class="header-mode-tabs">
        <button
          class="mode-tab-btn"
          class:active={bottomMode === 'console'}
          onclick={() => bottomMode = 'console'}
        >
          <Code2 size={13} />
          <span>Console Output</span>
          {#if isConsoleRunning}
            <span class="active-dot running" title="Executing code..."></span>
          {/if}
        </button>
        <button
          class="mode-tab-btn"
          class:active={bottomMode === 'syslog'}
          onclick={() => bottomMode = 'syslog'}
        >
          <Activity size={13} />
          <span>System Logs</span>
          {#if systemLogs.length > 0}
            <span class="log-count-badge">{systemLogs.length}</span>
          {/if}
        </button>
        <button
          class="mode-tab-btn"
          class:active={bottomMode === 'terminal'}
          onclick={() => bottomMode = 'terminal'}
        >
          <Terminal size={13} />
          <span>Terminal</span>
          {#if terminals.some(t => t.isRunning)}
            <span class="active-dot shell"></span>
          {/if}
        </button>
      </div>

      <div class="panel-actions">
        {#if bottomMode === 'console' || bottomMode === 'syslog'}
          <button class="panel-btn" onclick={handleCopyConsole} title="Copy output">
            <Copy size={12} /> Copy
          </button>
          <button class="panel-btn" onclick={onClearConsole} title="Clear logs">
            <Trash2 size={12} /> Clear
          </button>
        {/if}
        <button class="panel-btn toggle" onclick={onMinimize}>Minimize</button>
      </div>
    </div>

    {#if bottomMode === 'console'}
      <div class="console-body" id="console-output-view">
        {#if consoleLogs.length > 0}
          <div class="console-status-bar" style="background-color: {getStatusColor(consoleStatus)}">
            {consoleStatus}
          </div>
        {/if}
        {#each consoleLogs as log}
          <div class="console-line log-type-{log.type || 'info'}">
            {#if log.time}
              <span class="log-time">[{formatTime(log.time)}]</span>
            {/if}
            <span class="log-text">{log.text || log}</span>
          </div>
        {/each}
        {#if consoleLogs.length === 0}
          <div class="console-empty-hint">
            <Code2 size={16} />
            <span>Console output clean. Click 'Run Experiment' to execute code.</span>
          </div>
        {/if}
      </div>

    {:else if bottomMode === 'syslog'}
      <div class="console-body" id="syslog-output-view">
        {#each systemLogs as log}
          <div class="console-line log-type-{log.type || 'info'}">
            {#if log.time}
              <span class="log-time">[{formatTime(log.time)}]</span>
            {/if}
            <span class="log-text">{log.text || log}</span>
          </div>
        {/each}
        {#if systemLogs.length === 0}
          <div class="console-empty-hint">
            <Activity size={16} />
            <span>No system background logs generated yet.</span>
          </div>
        {/if}
      </div>

    {:else}
      <div class="terminal-body-wrap">
        <div class="console-body" id="terminal-output-view">
          {#if activeTerm}
            {#each activeTerm.logs as log}
              <div class="console-line">{log}</div>
            {/each}
          {:else}
            <div class="console-empty-hint">
              <SquareTerminal size={16} />
              <span>No active terminal session. Click '+' to open shell.</span>
            </div>
          {/if}
        </div>

        <div class="terminal-input-bar">
          <span class="terminal-prompt">$</span>
          <input
            type="text"
            class="terminal-input-field"
            placeholder="Type command..."
            value={activeTerm?.inputBuffer || ''}
            oninput={(e) => {
              if (activeTerm) activeTerm.inputBuffer = e.target.value;
            }}
            onkeydown={(e) => {
              if (e.key === 'Enter') onSendTerminalInput();
            }}
            disabled={!activeTerm}
          />
          <button class="terminal-send-btn" onclick={onSendTerminalInput} disabled={!activeTerm}>
            <CornerDownLeft size={12} />
          </button>
        </div>
      </div>
    {/if}
  </div>

  {#if bottomMode === 'terminal'}
    <div class="terminal-tabs-sidebar">
      <div class="tabs-sidebar-header">
        <span>SHELL SESSIONS</span>
        <button class="add-term-tab-btn" onclick={onAddTerminal} title="New Terminal Session">
          <Plus size={12} />
        </button>
      </div>
      {#each terminals as term (term.id)}
        <div
          class="terminal-tab-item"
          class:active={activeTermId === term.id}
          onclick={() => activeTermId = term.id}
          role="button"
          tabindex="0"
          onkeydown={(e) => e.key === 'Enter' && (activeTermId = term.id)}
        >
          <div class="tab-item-left">
            <span class="tab-status-dot" class:running={term.isRunning} class:stopped={!term.isRunning}></span>
            <Terminal size={10} />
            <span class="tab-title">{term.title}</span>
          </div>
          <button
            class="tab-close-btn"
            onclick={(e) => { e.stopPropagation(); onCloseTerminal(term.id); }}
            title="Close session"
          >
            <X size={10} />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .console-drawer.multi-term {
    display: flex;
    flex-direction: row;
    background-color: #08080c;
    border-top: 1px solid #1a1a24;
    flex-shrink: 0;
    position: relative;
  }

  .resize-handle {
    position: absolute;
    top: -4px;
    left: 0;
    right: 0;
    height: 8px;
    cursor: ns-resize;
    z-index: 10;
    background: transparent;
  }

  .resize-handle:hover,
  .resize-handle.resizing {
    background-color: #ff5a36aa;
  }

  .panel-main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    overflow: hidden;
  }

  .panel-header {
    height: 34px;
    background-color: #0c0c12;
    padding: 0 12px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #1a1a24;
    flex-shrink: 0;
  }

  .header-mode-tabs {
    display: flex;
    gap: 4px;
  }

  .mode-tab-btn {
    background: none;
    border: 1px solid transparent;
    color: #718096;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: color 0.15s, background-color 0.15s;
  }

  .mode-tab-btn:hover {
    color: #edf2f7;
    background-color: #1a1a24;
  }

  .mode-tab-btn.active {
    color: #ff5a36;
    background-color: #ff5a3614;
    border-color: #ff5a3633;
  }

  .log-count-badge {
    background-color: #2d3748;
    color: #a0aec0;
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 10px;
  }

  .active-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  .active-dot.running {
    background-color: #48bb78;
    box-shadow: 0 0 6px #48bb78;
  }

  .active-dot.shell {
    background-color: #3178c6;
  }

  .panel-actions {
    display: flex;
    gap: 8px;
  }

  .panel-btn {
    background: none;
    border: 1px solid #2d3748;
    color: #a0aec0;
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .panel-btn:hover {
    color: #edf2f7;
    background-color: #1a1a24;
  }

  .terminal-body-wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
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

  .console-status-bar {
    display: inline-block;
    padding: 2px 10px;
    border-radius: 3px;
    font-size: 10px;
    font-weight: 700;
    color: #fff;
    margin-bottom: 8px;
    letter-spacing: 0.3px;
    text-transform: uppercase;
  }

  .console-line {
    line-height: 1.5;
    white-space: pre-wrap;
  }

  .log-time {
    color: #4a5568;
    margin-right: 6px;
    font-size: 11px;
  }

  .log-type-error .log-text { color: #fc8181; }
  .log-type-success .log-text { color: #68d391; }
  .log-type-out .log-text { color: #e2e8f0; }
  .log-type-info .log-text { color: #a0aec0; }

  .console-empty-hint {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #4a5568;
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    gap: 8px;
  }

  .terminal-input-bar {
    height: 32px;
    background-color: #0c0c12;
    border-top: 1px solid #1a1a24;
    display: flex;
    align-items: center;
    padding: 0 8px;
    gap: 6px;
    flex-shrink: 0;
  }

  .terminal-prompt {
    color: #48bb78;
    font-family: 'Fira Code', monospace;
    font-size: 13px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .terminal-input-field {
    flex: 1;
    background-color: #121218;
    border: 1px solid #2d3748;
    border-radius: 4px;
    padding: 4px 8px;
    color: #edf2f7;
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    outline: none;
    min-width: 0;
  }

  .terminal-input-field:focus {
    border-color: #ff5a36;
  }

  .terminal-input-field:disabled {
    opacity: 0.3;
  }

  .terminal-send-btn {
    background: none;
    border: 1px solid #2d3748;
    border-radius: 4px;
    color: #a0aec0;
    cursor: pointer;
    padding: 4px 6px;
    display: flex;
    align-items: center;
  }

  .terminal-send-btn:hover:not(:disabled) {
    background-color: #3178c61a;
    color: #3178c6;
  }

  .terminal-send-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .terminal-tabs-sidebar {
    width: 140px;
    min-width: 140px;
    max-width: 180px;
    background-color: #0a0a10;
    border-left: 1px solid #1a1a24;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .tabs-sidebar-header {
    height: 34px;
    padding: 0 8px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 10px;
    font-weight: 700;
    color: #6b7280;
    letter-spacing: 0.3px;
    border-bottom: 1px solid #1a1a24;
    flex-shrink: 0;
  }

  .add-term-tab-btn {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    display: flex;
    align-items: center;
    padding: 2px;
    border-radius: 3px;
  }

  .add-term-tab-btn:hover {
    color: #edf2f7;
    background-color: #1a1a24;
  }

  .terminal-tab-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 8px;
    cursor: pointer;
    color: #9ca3af;
    font-size: 11px;
    border-bottom: 1px solid #12121c;
    gap: 4px;
  }

  .terminal-tab-item:hover {
    background-color: #12121c;
    color: #edf2f7;
  }

  .terminal-tab-item.active {
    background-color: #1a1a26;
    color: #edf2f7;
    border-left: 2px solid #ff5a36;
  }

  .tab-item-left {
    display: flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    flex: 1;
    overflow: hidden;
  }

  .tab-status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .tab-status-dot.running { background-color: #48bb78; }
  .tab-status-dot.stopped { background-color: #4b5563; }

  .tab-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tab-close-btn {
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
  }

  .terminal-tab-item:hover .tab-close-btn { opacity: 1; }
  .tab-close-btn:hover { color: #ff5a36; }
</style>
