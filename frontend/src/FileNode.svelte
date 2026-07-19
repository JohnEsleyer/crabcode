<script>
  import FileNode from './FileNode.svelte';
  import { Folder, File, Circle } from '@lucide/svelte';
  let { 
    node, 
    openFile, 
    toggleFolder, 
    expandedFolders, 
    folderContents, 
    activeFilePath, 
    activeFileUnsaved = false
  } = $props();
</script>

<div class="tree-item" style="padding-left: 12px;">
  {#if node.isDir}
    <div 
      class="folder-row" 
      onclick={() => toggleFolder(node.path)}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && toggleFolder(node.path)}
    >
      <span class="icon">{expandedFolders[node.path] ? '▼' : '▶'}</span>
      <Folder class="folder-icon" size={14} />
      <span class="name">{node.name}</span>
    </div>
    {#if expandedFolders[node.path]}
      <div class="folder-children">
        {#each folderContents[node.path] || [] as child}
          <FileNode 
            node={child} 
            {openFile} 
            {toggleFolder} 
            {expandedFolders} 
            {folderContents} 
            {activeFilePath}
            {activeFileUnsaved}
          />
        {/each}
      </div>
    {/if}
  {:else}
    <div 
      class="file-row" 
      class:active={activeFilePath === node.path}
      onclick={() => openFile(node)}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && openFile(node)}
    >
      <File class="file-icon" size={14} />
      <span class="name">{node.name}</span>
      {#if activeFilePath === node.path && activeFileUnsaved}
        <Circle class="unsaved-dot" size={8} fill="#ff5a36" stroke="none" />
      {/if}
    </div>
  {/if}
</div>

<style>
  .tree-item {
    font-family: 'Inter', sans-serif;
    font-size: 13px;
    user-select: none;
  }
  .folder-row, .file-row {
    display: flex;
    align-items: center;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 4px;
    color: #a0aec0;
    gap: 6px;
    transition: background-color 0.15s, color 0.15s;
    margin: 1px 0;
  }
  .folder-row:hover, .file-row:hover {
    background-color: #2d3748;
    color: #edf2f7;
  }
  .file-row.active {
    background-color: #ff5a3611;
    color: #edf2f7;
    border-left: 2px solid #ff5a36;
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }
  .icon {
    font-size: 8px;
    width: 10px;
    color: #718096;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  :global(.folder-icon), :global(.file-icon) {
    flex-shrink: 0;
    color: #718096;
  }
  .name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }
  .folder-children {
    border-left: 1px solid #2d3748;
    margin-left: 15px;
  }
</style>
