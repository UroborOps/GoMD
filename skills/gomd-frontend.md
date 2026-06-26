---
name: gomd-frontend
description: React frontend development patterns for GoMD — markdown rendering, mermaid, math, graph view, SSE integration, search
tags: [gomd, frontend, react]
---

# GoMD Frontend Skill

React + Vite patterns for building the GoMD frontend.

## SSE Hook

```typescript
import { useEffect, useRef, useState } from 'react';

interface SSEEvent {
  type: 'file_change' | 'file_deleted' | 'file_created' | 'index_ready' | 'config_changed';
  path?: string;
  timestamp: string;
}

export function useSSE(onEvent: (evt: SSEEvent) => void) {
  const sourceRef = useRef<EventSource | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const source = new EventSource('/events');
    sourceRef.current = source;

    source.onopen = () => setConnected(true);
    source.onmessage = (e) => {
      const evt: SSEEvent = JSON.parse(e.data);
      onEvent(evt);
    };
    source.onerror = () => {
      setConnected(false);
      // EventSource auto-reconnects
    };

    return () => {
      source.close();
    };
  }, [onEvent]);

  return { connected };
}
```

Usage:
```typescript
function App() {
  useSSE((evt) => {
    if (evt.type === 'file_change' && evt.path === currentFile) {
      // Re-fetch and update editor
      fetch(`/api/files/${encodeURIComponent(evt.path)}`)
        .then(r => r.json())
        .then(data => setFileContent(data.content));
    }
    if (evt.type === 'index_ready') {
      // Re-fetch graph data
      refetchGraphData();
    }
  });
}
```

## Markdown Rendering with markdown-it

```typescript
import MarkdownIt from 'markdown-it';
import mdMath from 'markdown-it-mathkatex';
import mdFrontmatter from 'markdown-it-front-matter';

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
});

// Plugins
md.use(mdFrontmatter, (fm: Record<string, any>) => {
  window.__frontmatter = fm;
});

md.use(mdMath, {
  inlineOpen: '$',
  inlineClose: '$',
  displayOpen: '$$',
  displayClose: '$$',
});

// Wiki link renderer — make [[links]] clickable
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const href = tokens[idx].attrs?.find(a => a[0] === 'href')?.[1] || '';
  if (href.startsWith('#') || href.startsWith('/')) {
    // Internal navigation
    return `<a href="#" data-path="${href.slice(1)}" onclick="navigateTo(this.dataset.path)">${self.renderToken(tokens, idx, options)}</a>`;
  }
  return self.renderToken(tokens, idx, options);
};
```

## Mermaid Integration

```typescript
import mermaid from 'mermaid';

mermaid.initialize({
  startOnLoad: false,
  theme: isDark ? 'dark' : 'default',
  securityLevel: 'loose',
});

export async function renderMermaid(container: HTMLElement, text: string) {
  const id = `mermaid-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const { svg } = await mermaid.render(id, text);
  container.innerHTML = svg;
}

// In a React component:
const mermaidRef = useRef<HTMLDivElement>(null);
const mermaidCode = useRef('');

useEffect(() => {
  mermaidCode.current = code;
}, [code]);

useEffect(() => {
  if (mermaidRef.current && mermaidCode.current) {
    renderMermaid(mermaidRef.current, mermaidCode.current);
  }
}, [theme]);
```

## Graph View with D3

```typescript
import * as d3 from 'd3';

interface GraphNode {
  id: string;
  label: string;
  tags?: string[];
  orphanIn?: boolean;
  orphanOut?: boolean;
}

interface GraphEdge {
  source: string;
  target: string;
}

export function renderGraph(container: HTMLElement, nodes: GraphNode[], edges: GraphEdge[]) {
  const width = container.clientWidth;
  const height = container.clientHeight;

  const svg = d3.select(container).append('svg')
    .attr('width', width).attr('height', height);

  const zoom = d3.zoom().on('zoom', (event) => {
    g.attr('transform', event.transform);
  });
  svg.call(zoom);

  const g = svg.append('g');

  const simulation = d3.forceSimulation(nodes)
    .force('link', d3.forceLink(edges).id(d => d.id).distance(120))
    .force('charge', d3.forceManyBody().strength(-300))
    .force('center', d3.forceCenter(width / 2, height / 2))
    .force('collision', d3.forceCollide().radius(40));

  // Links
  const link = g.selectAll('.link')
    .data(edges).join('line')
    .attr('class', 'link')
    .attr('stroke', '#666')
    .attr('stroke-opacity', 0.4);

  // Nodes
  const node = g.selectAll('.node')
    .data(nodes).join('g')
    .attr('class', 'node')
    .call(d3.drag()
      .on('start', dragStarted)
      .on('drag', dragged)
      .on('end', dragEnded));

  node.append('circle')
    .attr('r', 6)
    .attr('fill', d => {
      if (d.orphanIn && d.orphanOut) return '#888';
      if (d.tags?.length) return tagColor(d.tags[0]);
      return '#4a90d9';
    });

  node.append('text')
    .attr('dx', 10).attr('dy', '.35em')
    .text(d => d.label.length > 20 ? d.label.slice(0, 20) + '…' : d.label)
    .attr('font-size', '10px')
    .attr('fill', '#ccc')
    .style('pointer-events', 'none');

  node.on('click', (event, d) => navigateTo(d.id));

  simulation.nodes(nodes).on('tick', () => {
    link
      .attr('x1', d => (d.source as any).x)
      .attr('y1', d => (d.source as any).y)
      .attr('x2', d => (d.target as any).x)
      .attr('y2', d => (d.target as any).y);
    node.attr('transform', d => `translate(${d.x},${d.y})`);
  });

  simulation.alpha(1).restart();
}

// Drag handlers
function dragStarted(event: any, d: GraphNode) {
  if (!event.active) simulation.alphaTarget(0.3).restart();
  d.fx = d.x; d.fy = d.y;
}
function dragged(event: any, d: GraphNode) {
  d.fx = event.x; d.fy = event.y;
}
function dragEnded(event: any, d: GraphNode) {
  if (!event.active) simulation.alphaTarget(0);
  d.fx = null; d.fy = null;
}
```

## File Tree Component

```typescript
interface DirEntry {
  name: string;
  type: 'file' | 'directory';
  path: string;
  children?: DirEntry[];
}

export function FileTree({ entries, onSelect, selectedPath }: Props) {
  function renderNode(entry: DirEntry, depth: number) {
    const isExpanded = expandedDirs.has(entry.path);
    const isSelected = entry.path === selectedPath;

    if (entry.type === 'directory') {
      return (
        <div key={entry.path}>
          <div className="tree-folder" onClick={() => toggleDir(entry.path)} style={{ paddingLeft: depth * 16 }}>
            <span className="tree-icon">{isExpanded ? '📂' : '📁'}</span>
            <span>{entry.name}</span>
          </div>
          {isExpanded && entry.children?.map(child => renderNode(child, depth + 1))}
        </div>
      );
    }

    return (
      <div key={entry.path} className={`tree-file ${isSelected ? 'selected' : ''}`}
           onClick={() => onSelect(entry.path)} style={{ paddingLeft: depth * 16 + 16 }}>
        <span>📄</span>
        <span>{entry.name}</span>
      </div>
    );
  }

  return <div className="file-tree">{entries.map(e => renderNode(e, 0))}</div>;
}
```

## Styling Notes
- Use CSS custom properties for theming: `--bg-primary`, `--text-primary`, `--accent`, etc.
- Sidebar: fixed width (~260px), collapsible to ~48px (icons only)
- Main content: flex-grow, max-width for readability (~800px centered)
- Monaco editor: fill remaining height, no scrollbar on outer container
- Graph view: full viewport, no sidebar
