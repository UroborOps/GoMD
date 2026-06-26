import { useState, useEffect, useCallback } from 'react';
import { api } from '../lib/api';

export interface GraphNode {
  id: string;
  label: string;
  depth: number;
}

export interface GraphLink {
  source: string;
  target: string;
}

export function useGraph() {
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [links, setLinks] = useState<GraphLink[]>([]);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getGraph();
      setNodes(data.nodes || []);
      setLinks(data.edges || []);
    } catch {
      setNodes([]);
      setLinks([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return { nodes, links, loading, reload: load };
}
