import React, { useState, useEffect } from 'react';
import { api, TreeNode } from '../lib/api';
import { FolderIcon, FileIcon, LockIcon, UnlockIcon } from './Icons';

interface WelcomeTreeProps {
  onSelectFile: (path: string) => void;
  onSelectFolder: (path: string) => void;
  refreshTrigger: number;
  isLocked: (path: string) => boolean;
  toggleLock: (path: string, locked: boolean) => void;
}

export default function WelcomeTree({ onSelectFile, onSelectFolder, refreshTrigger, isLocked, toggleLock }: WelcomeTreeProps) {
  const [tree, setTree] = useState<TreeNode | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.getTree()
      .then(setTree)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [refreshTrigger]);

  if (loading) {
    return (
      <div style={{ padding: '32px', textAlign: 'center', color: 'var(--text-muted)' }}>
        Loading vault statistics...
      </div>
    );
  }

  if (!tree) {
    return null;
  }

  // Flatten tree for table view
  const flattenTree = (node: TreeNode, depth: number = 0): { node: TreeNode, depth: number }[] => {
    let result = [{ node, depth }];
    if (node.children) {
      // Sort: folders first, then files, both alphabetically
      const sortedChildren = Object.values(node.children).sort((a, b) => {
        if (a.type !== b.type) return a.type === 'folder' ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
      for (const child of sortedChildren) {
        result = result.concat(flattenTree(child, depth + 1));
      }
    }
    return result;
  };

  const rows = flattenTree(tree);

  const formatSize = (bytes?: number) => {
    if (bytes === undefined) return '-';
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div style={{ flex: 1, padding: '24px', overflowY: 'auto' }}>
      <h2 style={{ margin: '0 0 16px 0', color: 'var(--text-primary)', fontWeight: 600 }}>Vault Overview</h2>
      <div style={{
        background: 'var(--bg-secondary)',
        borderRadius: '8px',
        border: '1px solid var(--border-color)',
        overflow: 'hidden'
      }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ background: 'var(--bg-primary)', borderBottom: '1px solid var(--border-color)', textAlign: 'left', color: 'var(--text-muted)' }}>
              <th style={{ padding: '12px 16px', fontWeight: 500, width: '40px' }}></th>
              <th style={{ padding: '12px 16px', fontWeight: 500 }}>Name</th>
              <th style={{ padding: '12px 16px', fontWeight: 500 }}>Size</th>
              <th style={{ padding: '12px 16px', fontWeight: 500 }}>Lines</th>
              <th style={{ padding: '12px 16px', fontWeight: 500 }}>Chars</th>
              <th style={{ padding: '12px 16px', fontWeight: 500 }}>Last Modified</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(({ node, depth }) => {
              const locked = isLocked(node.path);
              return (
              <tr 
                key={node.path || 'root'} 
                style={{ 
                  borderBottom: '1px solid var(--border-color)',
                  cursor: 'pointer',
                  background: 'var(--bg-secondary)'
                }}
                className="hover-bg-primary"
                onClick={() => node.type === 'file' ? onSelectFile(node.path) : onSelectFolder(node.path)}
              >
                <td style={{ padding: '8px 16px', textAlign: 'center' }}>
                  <button 
                    onClick={(e) => { e.stopPropagation(); toggleLock(node.path, !locked); }}
                    style={{ background: 'transparent', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', opacity: 0.5 }}
                    title={locked ? "Unlock" : "Lock"}
                  >
                    {locked ? <LockIcon size={14} /> : <UnlockIcon size={14} />}
                  </button>
                </td>
                <td style={{ padding: '8px 16px', display: 'flex', alignItems: 'center', gap: '8px', paddingLeft: (16 + depth * 16) + 'px' }}>
                  <span style={{ color: node.type === 'folder' ? 'var(--accent)' : 'var(--text-muted)', display: 'flex' }}>
                    {node.type === 'folder' ? <FolderIcon size={14} /> : <FileIcon size={14} />}
                  </span>
                  <span style={{ color: 'var(--text-primary)' }}>{node.name}</span>
                </td>
                <td style={{ padding: '8px 16px', color: 'var(--text-secondary)' }}>{node.type === 'file' ? formatSize(node.size) : '-'}</td>
                <td style={{ padding: '8px 16px', color: 'var(--text-secondary)' }}>{node.type === 'file' ? node.lines : '-'}</td>
                <td style={{ padding: '8px 16px', color: 'var(--text-secondary)' }}>{node.type === 'file' ? node.chars : '-'}</td>
                <td style={{ padding: '8px 16px', color: 'var(--text-secondary)' }}>{node.type === 'file' ? node.modTime : '-'}</td>
              </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      
      {/* Add a global style for hover effect since we don't use inline hover styles well without state */}
      <style>{`
        .hover-bg-primary:hover {
          background: var(--bg-primary) !important;
        }
      `}</style>
    </div>
  );
}
