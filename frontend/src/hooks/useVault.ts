import { useState, useCallback, useRef, useEffect } from 'react';
import { api } from '../lib/api';

export interface FileIndex {
  path: string;
  content: string;
  frontmatter: Record<string, unknown>;
  title: string;
  tags: string[];
}

interface FileEvent {
  type: 'created' | 'updated' | 'deleted';
  path: string;
}

export function useVault() {
  const [files, setFiles] = useState<string[]>([]);
  const [folders, setFolders] = useState<string[]>([]);
  const [currentFile, setCurrentFile] = useState<string | null>(null);
  const [currentFolder, setCurrentFolder] = useState<string | null>(null);
  const [content, setContent] = useState('');
  const [fileIndex, setFileIndex] = useState<Record<string, FileIndex>>({});
  const [backlinks, setBacklinks] = useState<Array<{ file: string; heading?: string; alias?: string }>>([]);
  const [locks, setLocks] = useState<Record<string, boolean>>({});
  const [config, setConfig] = useState<{ rag_enabled: boolean; qdrant_url: string; vault_path: string; theme: string; git_enabled: boolean; git_remote: string; s3_backup_enabled: boolean; s3_endpoint: string } | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isLive, setIsLive] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  const loadFiles = useCallback(async () => {
    try {
      const data = await api.listFiles();
      setFiles(data.files || []);
      setFolders(data.folders || []);
    } catch {
      setFiles([]);
      setFolders([]);
    }
  }, []);

  const loadLocks = useCallback(async () => {
    try {
      const lockData = await api.getLocks();
      setLocks(lockData);
    } catch {
      setLocks({});
    }
  }, []);

  const loadConfig = useCallback(async () => {
    try {
      const cfg = await api.getConfig();
      setConfig(cfg);
    } catch {
      setConfig(null);
    }
  }, []);

  // Initial load
  useEffect(() => {
    loadLocks();
    loadConfig();
  }, [loadLocks, loadConfig]);

  const isLocked = useCallback((path: string) => {
    if (!path && path !== '') return false;
    let current = path;
    while (current !== '.' && current !== '') {
      if (locks[current] !== undefined) return locks[current];
      const lastSlash = current.lastIndexOf('/');
      if (lastSlash === -1) {
        current = '';
      } else {
        current = current.substring(0, lastSlash);
      }
    }
    return locks[''] || false;
  }, [locks]);

  const toggleLock = useCallback(async (path: string, locked: boolean) => {
    try {
      await api.setLock(path, locked);
      // Optimistic update
      setLocks(prev => {
        if (path === '') {
          return { '': locked };
        }
        return { ...prev, [path]: locked };
      });
    } catch (err) {
      console.error('Failed to set lock:', err);
    }
  }, []);

  const selectFile = useCallback(async (path: string) => {
    setIsLoading(true);
    try {
      const data = await api.getFile(path);
      setCurrentFile(path);
      setContent(data.content);
      setFileIndex((prev) => ({ ...prev, [path]: data }));
      // Load backlinks
      const bl = await api.getBacklinks(path);
      setBacklinks(bl.backlinks);
    } catch {
      setCurrentFile(path);
      setContent('');
      setBacklinks([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const saveFile = useCallback(async (path: string, newContent: string) => {
    if (!path) return;
    setContent(newContent);
    try {
      await api.updateFile(path, newContent);
    } catch (err) {
      console.error('Failed to save:', err);
    }
  }, []);

  const createFile = useCallback(async (path: string, content: string = '') => {
    try {
      await api.createFile(path, content);
      await loadFiles();
      selectFile(path);
    } catch (err) {
      console.error('Failed to create:', err);
    }
  }, [loadFiles, selectFile]);

  const deleteFile = useCallback(async (path: string) => {
    try {
      await api.deleteFile(path);
      await loadFiles();
      if (currentFile === path) {
        setCurrentFile(null);
        setContent('');
      }
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  }, [loadFiles, currentFile]);

  const createFolder = useCallback(async (path: string) => {
    try {
      await api.createFolder(path);
      await loadFiles();
    } catch (err) {
      console.error('Failed to create folder:', err);
    }
  }, [loadFiles]);

  const deleteFolder = useCallback(async (path: string) => {
    try {
      await api.deleteFolder(path);
      await loadFiles();
    } catch (err) {
      console.error('Failed to delete folder:', err);
    }
  }, [loadFiles]);

  const renameNode = useCallback(async (oldPath: string, newPath: string) => {
    try {
      await api.renameNode(oldPath, newPath);
      await loadFiles();
      if (currentFile === oldPath) {
        selectFile(newPath);
      } else if (currentFile?.startsWith(oldPath + '/')) {
        const remaining = currentFile.substring(oldPath.length);
        selectFile(newPath + remaining);
      }
    } catch (err) {
      console.error('Failed to rename:', err);
    }
  }, [loadFiles, currentFile, selectFile]);

  const selectFolder = useCallback((path: string) => {
    setCurrentFolder(path);
  }, []);

  const uploadFiles = useCallback(async (pathPrefix: string, files: FileList | File[]) => {
    try {
      await api.upload(pathPrefix, files);
      await loadFiles();
    } catch (err) {
      console.error('Failed to upload:', err);
    }
  }, [loadFiles]);

  // SSE for live updates
  const connectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }
    const es = new EventSource('/events');
    es.onmessage = (evt) => {
      try {
        const event: FileEvent = JSON.parse(evt.data);
        setIsLive(true);
        if (event.type === 'created' || event.type === 'updated') {
          loadFiles();
          if (event.path === currentFile) {
            api.getFile(event.path).then((data) => {
              setContent(data.content);
              setFileIndex((prev) => ({ ...prev, [event.path]: data }));
            });
          }
        } else if (event.type === 'deleted') {
          loadFiles();
          if (currentFile === event.path) {
            setCurrentFile(null);
            setContent('');
          }
        }
      } catch {
        // ignore parse errors
      }
    };
    
    const handleEvent = (evt: MessageEvent) => {
      try {
        const event = JSON.parse(evt.data);
        if (event.type === 'created' || event.type === 'updated') {
          loadFiles();
          if (event.path === currentFile) {
            api.getFile(event.path).then((data) => {
              setContent(data.content);
              setFileIndex((prev) => ({ ...prev, [event.path]: data }));
            });
          }
        } else if (event.type === 'deleted') {
          loadFiles();
          if (currentFile === event.path) {
            setCurrentFile(null);
            setContent('');
          }
        } else if (event.type === 'config_changed' && event.path === 'locks') {
          loadLocks();
        }
      } catch {}
    };

    es.addEventListener('file_change', handleEvent);
    es.addEventListener('file_deleted', handleEvent);
    es.addEventListener('file_created', handleEvent);
    es.addEventListener('config_changed', handleEvent);

    es.onerror = () => setIsLive(false);
    eventSourceRef.current = es;
  }, [loadFiles, loadLocks, currentFile]);

  const disconnectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsLive(false);
  }, []);

  return {
    files,
    folders,
    currentFile,
    currentFolder,
    content,
    fileIndex,
    backlinks,
    isLoading,
    isLive,
    selectFile,
    selectFolder,
    saveFile,
    createFile,
    deleteFile,
    createFolder,
    deleteFolder,
    renameNode,
    uploadFiles,
    loadFiles,
    connectSSE,
    disconnectSSE,
    setContent,
    locks,
    config,
    isLocked,
    toggleLock,
  };
}
