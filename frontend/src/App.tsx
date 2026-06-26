import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useVault } from './hooks/useVault';
import { useSSE } from './hooks/useSSE';
import Sidebar from './components/Sidebar';
import Editor from './components/Editor';
import Preview from './components/Preview';
import GraphView from './components/GraphView';
import SearchView from './components/SearchView';
import Backlinks from './components/Backlinks';
import FolderView from './components/FolderView';
import WelcomeTree from './components/WelcomeTree';
import CommandPalette from './components/CommandPalette';
import KeyboardShortcutsModal from './components/KeyboardShortcutsModal';
import { DatabaseIcon, EditIcon, GraphIcon, SearchIcon, FolderIcon, CloudIcon, GithubIcon, KeyboardIcon } from './components/Icons';
import './styles/index.css';

type View = 'editor' | 'graph' | 'search' | 'folder';

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const [view, setView] = useState<View>('editor');
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [splitMode, setSplitMode] = useState<'editor' | 'preview' | 'split'>('split');
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  
  const vault = useVault();
  const isCurrentlyLocked = vault.currentFile ? vault.isLocked(vault.currentFile) : false;
  const prevLockedRef = useRef(isCurrentlyLocked);

  useEffect(() => {
    if (prevLockedRef.current && !isCurrentlyLocked) {
      // Transition from locked -> unlocked
      setSplitMode('split');
    }
    prevLockedRef.current = isCurrentlyLocked;
  }, [isCurrentlyLocked]);
  
  const [theme, setTheme] = useState<string>(() => {
    return localStorage.getItem('theme') || 'dark';
  });
  
  const [customAccent, setCustomAccent] = useState<string>(() => {
    return localStorage.getItem('customAccent') || '';
  });

  useEffect(() => {
    document.body.className = `theme-${theme}`;
    localStorage.setItem('theme', theme);
    
    if (customAccent) {
      document.body.style.setProperty('--accent', customAccent);
      // Create a slightly transparent version for dim variant
      document.body.style.setProperty('--accent-dim', customAccent + 'b3');
    } else {
      document.body.style.removeProperty('--accent');
      document.body.style.removeProperty('--accent-dim');
    }
    localStorage.setItem('customAccent', customAccent);
  }, [theme, customAccent]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Intercept Ctrl+P or Cmd+P or Ctrl+Shift+P
      if ((e.ctrlKey || e.metaKey) && (e.key === 'p' || e.key === 'P')) {
        e.preventDefault();
        setPaletteOpen(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const { connected } = useSSE((event) => {
    if (event.type === 'updated' || event.type === 'created' || event.type === 'deleted') {
      vault.loadFiles();
      setRefreshTrigger(prev => prev + 1);
    }
  });

  // Navigate to selected file
  useEffect(() => {
    if (vault.currentFile && view === 'editor') {
      navigate(`/editor/${encodeURIComponent(vault.currentFile)}`, { replace: true });
    }
  }, [vault.currentFile, view, navigate]);

  // Handle initial load
  useEffect(() => {
    vault.loadFiles();
    vault.connectSSE();
    return () => vault.disconnectSSE();
  }, [vault]);

  const handleFileSelect = useCallback((path: string) => {
    vault.selectFile(path);
    setView('editor');
  }, [vault]);

  const handleFolderSelect = useCallback((path: string) => {
    vault.selectFolder(path);
    setView('folder');
  }, [vault]);

  const handleCreateFile = useCallback(async (path: string) => {
    await vault.createFile(path);
    setRefreshTrigger(prev => prev + 1);
  }, [vault]);

  const handleSave = useCallback(async (content?: string) => {
    if (vault.currentFile) {
      await vault.saveFile(vault.currentFile, content ?? vault.content);
    }
  }, [vault]);

  const handleDeleteFile = useCallback(async (path: string) => {
    await vault.deleteFile(path);
    setRefreshTrigger(prev => prev + 1);
  }, [vault]);

  const handleNewFile = useCallback(() => {
    setView('editor');
    const path = `new-${Date.now()}.md`;
    handleCreateFile(path);
  }, [handleCreateFile]);

  const renderBreadcrumbs = () => {
    const currentPath = view === 'folder' ? vault.currentFolder : (view === 'editor' ? vault.currentFile : '');
    if (currentPath === null || currentPath === undefined) return null;

    const parts = currentPath ? currentPath.split('/') : [];
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: 'var(--text-secondary)', marginLeft: '8px' }}>
        <span style={{ cursor: 'pointer' }} onClick={() => handleFolderSelect('')}>Vault</span>
        {parts.map((part, index) => {
          const path = parts.slice(0, index + 1).join('/');
          const isLast = index === parts.length - 1;
          const isFile = isLast && view === 'editor';
          return (
            <React.Fragment key={path}>
              <span>/</span>
              <span
                style={{ 
                  cursor: 'pointer', 
                  color: isLast ? 'var(--text-primary)' : 'inherit',
                  fontWeight: isLast ? 600 : 400
                }}
                onClick={() => isFile ? handleFileSelect(path) : handleFolderSelect(path)}
              >
                {part}
              </span>
            </React.Fragment>
          );
        })}
      </div>
    );
  };

  return (
    <div className="app-layout">
      <KeyboardShortcutsModal 
        isOpen={shortcutsOpen} 
        onClose={() => setShortcutsOpen(false)} 
      />
      <CommandPalette 
        isOpen={paletteOpen} 
        onClose={() => setPaletteOpen(false)} 
        commands={[
          { id: 'new-file', label: 'New File', action: handleNewFile },
          { id: 'toggle-sidebar', label: 'Toggle Sidebar', action: () => setSidebarOpen(prev => !prev) },
          { id: 'toggle-theme', label: `Switch to ${theme === 'dark' ? 'Light' : 'Dark'} Theme`, action: () => setTheme(theme === 'dark' ? 'light' : 'dark') },
          { id: 'view-graph', label: 'Go to Graph View', action: () => setView('graph') },
          { id: 'view-search', label: 'Search Vault...', action: () => setView('search') }
        ]} 
      />
      {sidebarOpen && (
        <Sidebar
          files={vault.files}
          folders={vault.folders}
          currentFile={vault.currentFile}
          onSelectFile={handleFileSelect}
          onSelectFolder={handleFolderSelect}
          onCreateFile={handleCreateFile}
          onDeleteFile={handleDeleteFile}
          onCreateFolder={vault.createFolder}
          onDeleteFolder={vault.deleteFolder}
          onRename={vault.renameNode}
          onUpload={vault.uploadFiles}
          onToggle={() => setSidebarOpen(false)}
          isLocked={vault.isLocked}
          toggleLock={vault.toggleLock}
          config={vault.config}
        />
      )}
      
      <div className="main-content">
        {/* Header */}
        <div style={{
          padding: '8px 16px',
          borderBottom: '1px solid var(--border-color)',
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          background: 'var(--bg-secondary)'
        }}>
          {!sidebarOpen && (
            <button
              onClick={() => setSidebarOpen(true)}
              style={{
                background: 'transparent',
                border: '1px solid var(--border-color)',
                color: 'var(--text-secondary)',
                padding: '4px 8px',
                borderRadius: '4px',
                cursor: 'pointer'
              }}
            >
              ☰
            </button>
          )}
          
          {renderBreadcrumbs()}

          {/* Split mode toggle (only in editor view) */}
          {view === 'editor' && vault.currentFile && (
            <div style={{ display: 'flex', gap: '4px', marginLeft: 'auto' }}>
              {(['editor', 'preview', 'split'] as const).map((m) => {
                const effectiveMode = isCurrentlyLocked ? 'preview' : splitMode;
                const isActive = effectiveMode === m;
                const isDisabled = isCurrentlyLocked && m !== 'preview';
                return (
                  <button
                    key={m}
                    onClick={() => { if (!isDisabled) setSplitMode(m); }}
                    disabled={isDisabled}
                    style={{
                      padding: '4px 8px',
                      background: isActive ? 'var(--accent-dim)' : 'transparent',
                      border: '1px solid var(--border-color)',
                      color: isActive ? 'white' : 'var(--text-secondary)',
                      borderRadius: '4px',
                      cursor: isDisabled ? 'not-allowed' : 'pointer',
                      fontSize: '11px',
                      opacity: isDisabled ? 0.3 : 1
                    }}
                  >
                    {m}
                  </button>
                );
              })}
            </div>
          )}
          
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginLeft: view === 'editor' && vault.currentFile ? '8px' : 'auto' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }} title="Custom Accent Color">
              <input 
                type="color" 
                value={customAccent || '#f14c4c'}
                onChange={(e) => setCustomAccent(e.target.value)}
                style={{
                  width: '20px',
                  height: '20px',
                  padding: '0',
                  border: '1px solid var(--border-color)',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  background: 'transparent'
                }}
              />
              {customAccent && (
                <button 
                  onClick={() => setCustomAccent('')}
                  title="Reset Accent Color"
                  style={{
                    background: 'transparent',
                    border: 'none',
                    color: 'var(--text-muted)',
                    cursor: 'pointer',
                    padding: '0 4px',
                    fontSize: '12px'
                  }}
                >
                  ✕
                </button>
              )}
            </div>
            <select
              value={theme}
              onChange={(e) => setTheme(e.target.value)}
              style={{
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                color: 'var(--text-primary)',
                borderRadius: '4px',
                padding: '4px 8px',
                fontSize: '12px',
                cursor: 'pointer',
                outline: 'none'
              }}
              title="Select Theme"
            >
              <option value="dark">Dark</option>
              <option value="light">Light</option>
              <option value="dracula">Dracula</option>
              <option value="monokai">Monokai</option>
              <option value="solarized-dark">Solarized Dark</option>
              <option value="solarized-light">Solarized Light</option>
            </select>
            
            <div style={{ display: 'flex', gap: '4px' }}>
              {(['editor', 'graph', 'search', 'folder'] as View[]).map((v) => {
                const Icon = v === 'editor' ? EditIcon : v === 'graph' ? GraphIcon : v === 'search' ? SearchIcon : FolderIcon;
                return (
                  <button
                    key={v}
                    onClick={() => setView(v)}
                    title={v.charAt(0).toUpperCase() + v.slice(1)}
                    style={{
                      padding: '4px 8px',
                      background: view === v ? 'var(--accent)' : 'transparent',
                      border: '1px solid var(--border-color)',
                      color: view === v ? 'white' : 'var(--text-secondary)',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center'
                    }}
                  >
                    <Icon size={14} />
                  </button>
                );
              })}
              
              <div style={{ width: '1px', background: 'var(--border-color)', margin: '0 4px' }} />
              
              <a
                href="https://github.com/nroitero/gomd"
                target="_blank"
                rel="noopener noreferrer"
                title="GoMD GitHub Repo"
                style={{
                  padding: '4px 8px',
                  background: 'transparent',
                  border: '1px solid var(--border-color)',
                  color: 'var(--text-secondary)',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center'
                }}
              >
                <GithubIcon size={14} />
              </a>

              {vault.config?.git_enabled && vault.config?.git_remote && (
                <a
                  href={vault.config.git_remote.replace(/\.git$/, '')}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="GitHub Backup Repo"
                  style={{
                    padding: '4px 8px',
                    background: 'rgba(36, 41, 46, 0.1)',
                    border: '1px solid rgba(36, 41, 46, 0.2)',
                    color: 'var(--text-primary)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}
                >
                  <GithubIcon size={14} />
                </a>
              )}

              <button
                onClick={() => setShortcutsOpen(true)}
                title="Keyboard Shortcuts"
                style={{
                  padding: '4px 8px',
                  background: 'transparent',
                  border: '1px solid var(--border-color)',
                  color: 'var(--text-secondary)',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center'
                }}
              >
                <KeyboardIcon size={14} />
              </button>
              <a
                href="/docs"
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  padding: '4px 12px',
                  background: 'transparent',
                  border: '1px solid var(--border-color)',
                  color: 'var(--text-secondary)',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontSize: '12px',
                  textDecoration: 'none',
                  display: 'flex',
                  alignItems: 'center'
                }}
              >
                API Docs ↗
              </a>
              {vault.config?.rag_enabled && vault.config?.qdrant_url && (
                <a
                  href={`${vault.config.qdrant_url.replace(/\/$/, '')}/dashboard`}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="Qdrant Dashboard"
                  style={{
                    padding: '4px 8px',
                    background: 'rgba(59, 130, 246, 0.1)',
                    border: '1px solid rgba(59, 130, 246, 0.2)',
                    color: '#3b82f6',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(59, 130, 246, 0.2)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(59, 130, 246, 0.1)'}
                >
                  <DatabaseIcon size={14} />
                </a>
              )}
              {vault.config?.s3_backup_enabled && vault.config?.s3_endpoint && (
                <a
                  href={vault.config.s3_endpoint}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="S3 Backend Dashboard"
                  style={{
                    padding: '4px 8px',
                    background: 'rgba(234, 88, 12, 0.1)',
                    border: '1px solid rgba(234, 88, 12, 0.2)',
                    color: '#ea580c',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(234, 88, 12, 0.2)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(234, 88, 12, 0.1)'}
                >
                  <CloudIcon size={14} />
                </a>
              )}
            </div>
          </div>

          {/* Live indicator */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginLeft: '8px' }}>
            {connected && (
              <>
                <div style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  background: 'var(--success)',
                  animation: 'pulse 2s infinite'
                }} title="Live Connection Active" />
              </>
            )}
          </div>
        </div>

        {/* Routes */}
        <Routes>
          <Route path="*" element={
            view === 'folder' ? (
              !vault.currentFolder ? (
                <WelcomeTree 
                  onSelectFile={handleFileSelect} 
                  onSelectFolder={handleFolderSelect} 
                  refreshTrigger={refreshTrigger} 
                  isLocked={vault.isLocked}
                  toggleLock={vault.toggleLock}
                />
              ) : (
                <FolderView 
                  currentFolder={vault.currentFolder}
                  onSelectFile={handleFileSelect}
                  onSelectFolder={handleFolderSelect}
                  refreshTrigger={refreshTrigger}
                  isLocked={vault.isLocked}
                  toggleLock={vault.toggleLock}
                />
              )
            ) : view === 'editor' ? (
              <EditorPane
                splitMode={splitMode}
                content={vault.content}
                isLoading={vault.isLoading}
                currentFile={vault.currentFile}
                onSave={handleSave}
                onNavigate={handleFileSelect}
                onNavigateFolder={handleFolderSelect}
                refreshTrigger={refreshTrigger}
                isLocked={vault.isLocked(vault.currentFile || '')}
                isLockedFn={vault.isLocked}
                toggleLock={vault.toggleLock}
              />
            ) : view === 'graph' ? (
              <GraphView onNodeClick={handleFileSelect} />
            ) : (
              <SearchView onSelectFile={handleFileSelect} />
            )
          } />
        </Routes>

        {/* Backlinks (only in editor mode) */}
        {view === 'editor' && vault.currentFile && (
          <Backlinks backlinks={vault.backlinks} onSelectFile={handleFileSelect} />
        )}

        {/* Status bar */}
        <div style={{
          padding: '4px 16px',
          fontSize: '11px',
          color: 'var(--text-muted)',
          borderTop: '1px solid var(--border-color)',
          display: 'flex',
          justifyContent: 'space-between',
          background: 'var(--bg-secondary)'
        }}>
          <span>{vault.files.length} files, {vault.folders.length} folders</span>
          <span>{view === 'folder' ? (vault.currentFolder || 'Vault Root') : (vault.currentFile || 'No file selected')}</span>
        </div>
      </div>
    </div>
  );
}

interface EditorPaneProps {
  splitMode: 'editor' | 'preview' | 'split';
  content: string;
  isLoading: boolean;
  currentFile: string | null;
  onSave: (content?: string) => Promise<void>;
  onNavigate: (path: string) => void;
  onNavigateFolder: (path: string) => void;
  refreshTrigger: number;
  isLocked: boolean;
  isLockedFn: (path: string) => boolean;
  toggleLock: (path: string, locked: boolean) => void;
}

function EditorPane({ splitMode, content, isLoading, currentFile, onSave, onNavigate, onNavigateFolder, refreshTrigger, isLocked, isLockedFn, toggleLock }: EditorPaneProps) {
  if (!currentFile) {
    return (
      <WelcomeTree 
        onSelectFile={onNavigate} 
        onSelectFolder={onNavigateFolder} 
        refreshTrigger={refreshTrigger} 
        isLocked={isLockedFn}
        toggleLock={toggleLock}
      />
    );
  }

  if (isLoading) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div className="shimmer" style={{ width: '200px', height: '20px' }} />
      </div>
    );
  }

  const effectiveSplitMode = isLocked ? 'preview' : splitMode;
  const editorStyle = { flex: effectiveSplitMode === 'editor' ? 1 : effectiveSplitMode === 'split' ? 1 : 0, width: effectiveSplitMode === 'editor' ? '100%' : '50%' };
  const previewStyle = { flex: effectiveSplitMode === 'preview' ? 1 : effectiveSplitMode === 'split' ? 1 : 0, width: effectiveSplitMode === 'preview' ? '100%' : '50%' };

  return (
    <div style={{
      flex: 1,
      display: 'flex',
      overflow: 'hidden',
      flexDirection: effectiveSplitMode === 'editor' || effectiveSplitMode === 'preview' ? 'column' : 'row'
    }}>
      {(effectiveSplitMode === 'editor' || effectiveSplitMode === 'split') && (
        <Editor
          style={editorStyle}
          value={content}
          onChange={(newContent) => {
            onSave(newContent);
          }}
          currentFile={currentFile}
        />
      )}
      {(effectiveSplitMode === 'preview' || effectiveSplitMode === 'split') && (
        <Preview style={previewStyle} content={content} onNavigate={onNavigate} />
      )}
    </div>
  );
}
