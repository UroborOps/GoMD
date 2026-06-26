import React, { useEffect, useState } from 'react';
import { api } from '../lib/api';
import { FolderIcon, FileIcon, LockIcon, UnlockIcon } from './Icons';

interface FolderViewProps {
  currentFolder: string;
  onSelectFile: (path: string) => void;
  onSelectFolder: (path: string) => void;
  refreshTrigger: number;
  isLocked: (path: string) => boolean;
  toggleLock: (path: string, locked: boolean) => void;
}

export default function FolderView({ currentFolder, onSelectFile, onSelectFolder, refreshTrigger, isLocked, toggleLock }: FolderViewProps) {
  const [items, setItems] = useState<{ directories: string[]; files: string[] }>({ directories: [], files: [] });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.listFolders(currentFolder)
      .then(data => {
        setItems({
          directories: (data.directories || []).map(d => d.replace(/\/$/, '')),
          files: data.files || []
        });
      })
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, [currentFolder, refreshTrigger]);

  if (loading) {
    return <div style={{ padding: '32px', color: 'var(--text-muted)' }}>Loading folder contents...</div>;
  }

  const isEmpty = items.directories.length === 0 && items.files.length === 0;

  return (
    <div style={{ padding: '32px', flex: 1, overflowY: 'auto' }}>
      <h2 style={{ marginBottom: '24px', fontWeight: 600 }}>{currentFolder || 'Vault Root'}</h2>
      
      {isEmpty ? (
        <div style={{ color: 'var(--text-muted)' }}>This folder is empty.</div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '16px' }}>
          {items.directories.map(dir => {
            const fullPath = currentFolder ? `${currentFolder}/${dir}` : dir;
            return (
              <div 
                key={dir} 
                onClick={() => onSelectFolder(fullPath)}
                style={{
                  padding: '16px',
                  border: '1px solid var(--border-color)',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  background: 'var(--bg-secondary)',
                  transition: 'background 0.2s',
                  position: 'relative'
                }}
                onMouseOver={(e) => e.currentTarget.style.background = 'var(--accent-dim)'}
                onMouseOut={(e) => e.currentTarget.style.background = 'var(--bg-secondary)'}
              >
                <div style={{ color: 'var(--accent)' }}>
                  <FolderIcon size={24} />
                </div>
                <div style={{ fontWeight: 500, color: 'var(--text-primary)', wordBreak: 'break-all' }}>
                  {dir}
                </div>
                <button 
                  onClick={(e) => { e.stopPropagation(); toggleLock(fullPath, !isLocked(fullPath)); }}
                  style={{ position: 'absolute', top: '8px', right: '8px', background: 'transparent', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', opacity: 0.5 }}
                  title={isLocked(fullPath) ? "Unlock" : "Lock"}
                >
                  {isLocked(fullPath) ? <LockIcon size={14} /> : <UnlockIcon size={14} />}
                </button>
              </div>
            );
          })}
          
          {items.files.map(file => {
            const fullPath = currentFolder ? `${currentFolder}/${file}` : file;
            return (
              <div 
                key={file} 
                onClick={() => onSelectFile(fullPath)}
                style={{
                  padding: '16px',
                  border: '1px solid var(--border-color)',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  background: 'var(--bg-primary)',
                  transition: 'background 0.2s',
                  position: 'relative'
                }}
                onMouseOver={(e) => e.currentTarget.style.background = 'var(--bg-secondary)'}
                onMouseOut={(e) => e.currentTarget.style.background = 'var(--bg-primary)'}
              >
                <div style={{ color: 'var(--text-muted)' }}>
                  <FileIcon size={24} />
                </div>
                <div style={{ fontWeight: 500, color: 'var(--text-primary)', wordBreak: 'break-all' }}>
                  {file}
                </div>
                <button 
                  onClick={(e) => { e.stopPropagation(); toggleLock(fullPath, !isLocked(fullPath)); }}
                  style={{ position: 'absolute', top: '8px', right: '8px', background: 'transparent', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', opacity: 0.5 }}
                  title={isLocked(fullPath) ? "Unlock" : "Lock"}
                >
                  {isLocked(fullPath) ? <LockIcon size={14} /> : <UnlockIcon size={14} />}
                </button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
