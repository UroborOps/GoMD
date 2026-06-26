import React, { useState, useCallback, useMemo, useRef } from 'react';
import { FolderIcon, FileIcon, DownloadIcon, UploadIcon, FilePlusIcon, FolderPlusIcon, FolderUploadIcon, EditIcon, TrashIcon, DatabaseIcon, LockIcon, UnlockIcon } from './Icons';

interface SidebarProps {
  files: string[];
  folders: string[];
  currentFile: string | null;
  onSelectFile: (path: string) => void;
  onSelectFolder: (path: string) => void;
  onCreateFile: (path: string) => void;
  onDeleteFile: (path: string) => void;
  onCreateFolder: (path: string) => void;
  onDeleteFolder: (path: string) => void;
  onRename: (oldPath: string, newPath: string) => void;
  onUpload: (pathPrefix: string, files: FileList) => void;
  onToggle: () => void;
  isLocked: (path: string) => boolean;
  toggleLock: (path: string, locked: boolean) => void;
  config: { rag_enabled: boolean; qdrant_url: string; git_enabled?: boolean; git_remote?: string; s3_backup_enabled?: boolean; s3_endpoint?: string } | null;
}

type TreeNode = {
  name: string;
  path: string;
  type: 'file' | 'folder';
  children: Record<string, TreeNode>;
};

function buildTree(files: string[], folders: string[]) {
  const root: TreeNode = { name: 'root', path: '', type: 'folder', children: {} };

  const addPath = (path: string, type: 'file' | 'folder') => {
    const parts = path.split('/');
    let current = root;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      if (!part) continue;

      const isLast = i === parts.length - 1;
      const nodePath = parts.slice(0, i + 1).join('/');

      if (!current.children[part]) {
        current.children[part] = {
          name: part,
          path: nodePath,
          type: isLast ? type : 'folder',
          children: {}
        };
      } else if (isLast && type === 'file') {
        current.children[part].type = 'file'; // Upgrade intermediate folder to file if it was implicitly created
      }
      current = current.children[part];
    }
  };

  folders.forEach(f => addPath(f, 'folder'));
  files.forEach(f => addPath(f, 'file'));
  return root;
}

export default function Sidebar({
  files,
  folders,
  currentFile,
  onSelectFile,
  onSelectFolder,
  onCreateFile,
  onDeleteFile,
  onCreateFolder,
  onDeleteFolder,
  onRename,
  onUpload,
  onToggle,
  isLocked,
  toggleLock,
  config
}: SidebarProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({});
  const [selectedNode, setSelectedNode] = useState<{path: string, type: 'file' | 'folder' | 'root', name: string}>({ path: '', type: 'root', name: 'Vault' });
  const [uploadTarget, setUploadTarget] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const root = useMemo(() => buildTree(files, folders), [files, folders]);

  const toggleFolder = (path: string) => {
    setExpandedFolders(prev => ({ ...prev, [path]: !prev[path] }));
  };

  const handleCreate = (parentPath: string, type: 'file' | 'folder') => {
    const name = prompt(`Enter new ${type} name:`);
    if (!name) return;
    
    let fullPath = parentPath ? `${parentPath}/${name}` : name;
    if (type === 'file' && !fullPath.endsWith('.md')) {
      fullPath += '.md';
    }

    if (files.includes(fullPath) || folders.includes(fullPath)) {
      alert('A file or folder with that name already exists!');
      return;
    }

    if (type === 'file') onCreateFile(fullPath);
    else onCreateFolder(fullPath);
    
    if (parentPath) {
      setExpandedFolders(prev => ({ ...prev, [parentPath]: true }));
    }
  };

  const handleDelete = (node: {path: string, name: string, type: string}) => {
    if (confirm(`Are you sure you want to delete ${node.name}?`)) {
      if (node.type === 'file') onDeleteFile(node.path);
      else onDeleteFolder(node.path);
      if (selectedNode.path === node.path) {
        setSelectedNode({ path: '', type: 'root', name: 'Vault' });
      }
    }
  };

  const handleRename = (node: {path: string, name: string, type: string}) => {
    const newPath = prompt(`Enter new path for ${node.name}:`, node.path);
    if (!newPath || newPath === node.path) return;
    
    if (files.includes(newPath) || folders.includes(newPath)) {
      alert('A file or folder with that name already exists!');
      return;
    }
    
    onRename(node.path, newPath);
    if (selectedNode.path === node.path) {
      setSelectedNode({ path: newPath, name: newPath.split('/').pop() || '', type: node.type as 'file' | 'folder' | 'root' });
    }
  };

  const handleDragStart = (e: React.DragEvent, node: TreeNode) => {
    e.stopPropagation();
    e.dataTransfer.setData('application/gomd-path', node.path);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDrop = (e: React.DragEvent, targetNode: TreeNode | null) => {
    e.preventDefault();
    e.stopPropagation();
    
    const sourcePath = e.dataTransfer.getData('application/gomd-path');
    if (!sourcePath) return;

    let targetDir = '';
    if (targetNode) {
      if (targetNode.type === 'folder') {
        targetDir = targetNode.path;
      } else {
        const parts = targetNode.path.split('/');
        parts.pop();
        targetDir = parts.join('/');
      }
    }

    const sourceName = sourcePath.split('/').pop() || '';
    const newPath = targetDir ? `${targetDir}/${sourceName}` : sourceName;

    if (sourcePath !== newPath) {
      if (files.includes(newPath) || folders.includes(newPath)) {
        alert('A file or folder with that name already exists!');
        return;
      }
      onRename(sourcePath, newPath);
    }
  };

  const handleDownload = (path: string) => {
    window.location.href = `/api/download/${encodeURIComponent(path)}`;
  };

  const handleUploadClick = (path: string, type: 'file' | 'folder') => {
    setUploadTarget(path);
    if (type === 'file') {
      fileInputRef.current?.click();
    } else {
      folderInputRef.current?.click();
    }
  };

  const onFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onUpload(uploadTarget, e.target.files);
    }
    e.target.value = '';
  };

  const renderTree = (node: TreeNode, level: number = 0) => {
    const children = Object.values(node.children).sort((a, b) => {
      if (a.type !== b.type) return a.type === 'folder' ? -1 : 1;
      return a.name.localeCompare(b.name);
    });

    if (searchQuery) {
      const matches = children.filter(c => c.type === 'file' && c.name.toLowerCase().includes(searchQuery.toLowerCase()));
      if (level === 0) return matches.map(c => renderNode(c, 0));
      return null;
    }

    return children.map(child => (
      <div key={child.path}>
        {renderNode(child, level)}
        {child.type === 'folder' && expandedFolders[child.path] && (
          <div style={{ paddingLeft: '12px' }}>
            {renderTree(child, level + 1)}
          </div>
        )}
      </div>
    ));
  };

  const renderNode = (node: TreeNode, level: number) => {
    const isFile = node.type === 'file';
    const isActive = currentFile === node.path;
    const locked = isLocked(node.path);
    
    return (
      <div
        className={`file-tree-item ${isActive ? 'active' : ''}`}
        style={{ paddingLeft: `${level * 12 + 8}px`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
        onClick={() => {
          setSelectedNode({ path: node.path, name: node.name, type: node.type });
          if (isFile) {
            onSelectFile(node.path);
          } else {
            toggleFolder(node.path);
            onSelectFolder(node.path);
          }
        }}
        title={node.path}
        draggable
        onDragStart={(e) => handleDragStart(e, node)}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => handleDrop(e, node)}
      >
        <div style={{ display: 'flex', alignItems: 'center', overflow: 'hidden', flex: 1 }}>
          <span className="icon" style={{ marginRight: '6px', display: 'flex', alignItems: 'center' }}>
            {isFile ? (
              <span style={{ color: 'var(--text-secondary)' }}><FileIcon size={14} /></span>
            ) : (
              <span style={{ color: 'var(--accent)' }}><FolderIcon size={14} /></span>
            )}
          </span>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {node.name}
          </span>
        </div>
      </div>
    );
  };

  return (
    <div className="sidebar"
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => handleDrop(e, null)}
    >
      <input type="file" multiple ref={fileInputRef} style={{ display: 'none' }} onChange={onFileInputChange} />
      <input type="file" multiple {...{ webkitdirectory: "true", directory: "true" } as any} ref={folderInputRef} style={{ display: 'none' }} onChange={onFileInputChange} />
      
      <div className="sidebar-header">
        <h1 style={{ display: 'flex', alignItems: 'center', gap: '8px', margin: 0 }}>
          GoMD
          {config?.rag_enabled && (
            <span title="AI Enabled" style={{ fontSize: '10px', padding: '2px 4px', background: 'var(--accent)', color: 'white', borderRadius: '4px', verticalAlign: 'middle', fontWeight: 'bold' }}>AI</span>
          )}
        </h1>
        <button
          onClick={onToggle}
          style={{
            marginLeft: 'auto',
            background: 'transparent',
            border: '1px solid var(--border-color)',
            color: 'var(--text-secondary)',
            padding: '2px 6px',
            borderRadius: '4px',
            cursor: 'pointer',
            fontSize: '12px'
          }}
        >
          ✕
        </button>
      </div>

      <div style={{ padding: '8px' }}>
        <input
          type="text"
          placeholder="Search files..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{
            width: '100%',
            padding: '6px 8px',
            background: 'var(--bg-primary)',
            border: '1px solid var(--border-color)',
            borderRadius: '4px',
            color: 'var(--text-primary)',
            fontSize: '12px',
            marginBottom: '4px'
          }}
        />
      </div>

      <div className="sidebar-taskbar" style={{ 
        padding: '8px', 
        background: 'rgba(0, 0, 0, 0.15)', 
        borderBottom: '1px solid var(--border-color)',
        borderTop: '1px solid var(--border-color)',
        marginBottom: '8px'
      }}>
        <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '8px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center', gap: '4px' }}>
          <span style={{opacity: 0.7}}>TARGET</span> <span style={{ color: 'var(--text-primary)', fontWeight: 600, background: 'var(--bg-primary)', padding: '2px 6px', borderRadius: '4px', border: '1px solid var(--border-color)' }}>{selectedNode.name}</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px' }}>
          {(() => {
            const isSelLocked = isLocked(selectedNode.path);
            const isFile = selectedNode.type === 'file';
            const isRoot = selectedNode.type === 'root';
            const isFolder = selectedNode.type === 'folder';
            
            return (
              <>
                <button className="taskbar-btn" onClick={() => toggleLock(selectedNode.path, !isSelLocked)} title={isSelLocked ? "Unlock" : "Lock"}>
                  {isSelLocked ? <LockIcon size={12} /> : <UnlockIcon size={12} />} {isSelLocked ? 'Unlock' : 'Lock'}
                </button>
                
                {isFile && (
                  <button className="taskbar-btn" onClick={() => handleDownload(selectedNode.path)} title="Download">
                    <DownloadIcon size={12} /> Download
                  </button>
                )}
                
                {!isSelLocked && !isFile && (
                  <>
                    <button className="taskbar-btn" onClick={() => handleUploadClick(selectedNode.path, 'file')} title="Upload File"><UploadIcon size={12} /> Up File</button>
                    <button className="taskbar-btn" onClick={() => handleUploadClick(selectedNode.path, 'folder')} title="Upload Folder"><FolderUploadIcon size={12} /> Up Dir</button>
                    <button className="taskbar-btn" onClick={() => handleCreate(selectedNode.path, 'file')} title="New File"><FilePlusIcon size={12} /> New File</button>
                    <button className="taskbar-btn" onClick={() => handleCreate(selectedNode.path, 'folder')} title="New Folder"><FolderPlusIcon size={12} /> New Dir</button>
                  </>
                )}
                
                {!isSelLocked && !isRoot && (
                  <>
                    <button className="taskbar-btn" onClick={() => handleRename(selectedNode)} title="Rename / Move"><EditIcon size={12} /> Rename</button>
                    <button className="taskbar-btn danger" onClick={() => handleDelete(selectedNode)} title="Delete"><TrashIcon size={12} /> Delete</button>
                  </>
                )}
              </>
            );
          })()}
        </div>
      </div>

      <div className="file-tree" style={{ paddingBottom: '32px', flex: 1 }}>
        <div 
          className="file-tree-item"
          style={{ paddingLeft: '8px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', color: 'var(--text-primary)', fontWeight: 600, cursor: 'pointer' }}
          onClick={() => {
            setSelectedNode({ path: '', type: 'root', name: 'Vault' });
            onSelectFolder('');
          }}
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            if (!isLocked('')) handleDrop(e, null);
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <span style={{ color: 'var(--accent)', marginRight: '6px', display: 'flex', alignItems: 'center' }}>
              <DatabaseIcon size={14} />
            </span>
            Vault
          </div>
        </div>

        <div style={{ height: '1px', background: 'var(--border-color)', margin: '8px 12px' }} />

        {Object.keys(root.children).length === 0 ? (
          <div style={{ padding: '16px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '12px' }}>
            No files found
          </div>
        ) : (
          renderTree(root)
        )}
      </div>
    </div>
  );
}

const actionBtnStyle = {
  background: 'transparent',
  border: 'none',
  cursor: 'pointer',
  padding: '2px 4px',
  color: 'var(--text-secondary)'
};

