import { useRef, useEffect, useState, useMemo } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { useGraph } from '../hooks/useGraph';

interface GraphViewProps {
  onNodeClick?: (path: string) => void;
}

export default function GraphView({ onNodeClick }: GraphViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const { nodes, links, loading } = useGraph();
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const handleResize = () => {
      if (containerRef.current) {
        setDimensions({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight
        });
      }
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Format data for ForceGraph2D
  const graphData = useMemo(() => {
    return {
      nodes: nodes.map(n => ({ id: n.id, name: n.label, val: n.depth > 0 ? 1 : 2 })),
      links: links.map(l => ({ source: l.source, target: l.target }))
    };
  }, [nodes, links]);

  // Read CSS variables for colors to match themes
  const [colors, setColors] = useState({ accent: '#4a9eff', text: '#e1e1e1', bg: '#1e1e1e', link: '#333333' });
  
  useEffect(() => {
    // This effect updates colors when the body class changes (theme switch)
    const updateColors = () => {
      const style = getComputedStyle(document.body);
      setColors({
        accent: style.getPropertyValue('--accent').trim() || '#4a9eff',
        text: style.getPropertyValue('--text-primary').trim() || '#e1e1e1',
        bg: style.getPropertyValue('--bg-primary').trim() || '#1e1e1e',
        link: style.getPropertyValue('--border-color').trim() || '#333333'
      });
    };
    
    updateColors();
    
    // Setup a small observer to watch for theme class changes on body
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName === 'class') {
          updateColors();
        }
      });
    });
    
    observer.observe(document.body, { attributes: true });
    return () => observer.disconnect();
  }, []);

  if (loading) {
    return (
      <div className="graph-view" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div className="shimmer" style={{ width: '200px', height: '20px' }} />
      </div>
    );
  }

  return (
    <div className="graph-view" ref={containerRef} style={{ width: '100%', height: '100%', position: 'relative' }}>
      {dimensions.width > 0 && nodes.length > 0 && (
        <ForceGraph2D
          width={dimensions.width}
          height={dimensions.height}
          graphData={graphData}
          nodeLabel="name"
          nodeColor={() => colors.accent}
          linkColor={() => colors.link}
          backgroundColor={colors.bg}
          onNodeClick={(node) => {
            if (onNodeClick && typeof node.id === 'string') {
              onNodeClick(node.id);
            }
          }}
          nodeCanvasObject={(node, ctx, globalScale) => {
            const label = node.name as string;
            const fontSize = 12 / globalScale;
            ctx.font = `${fontSize}px Sans-Serif`;
            const textWidth = ctx.measureText(label).width;
            const bckgDimensions = [textWidth, fontSize].map(n => n + fontSize * 0.2);

            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            
            // Draw node circle
            ctx.fillStyle = colors.accent;
            ctx.beginPath();
            ctx.arc(node.x!, node.y!, 5 / globalScale, 0, 2 * Math.PI, false);
            ctx.fill();

            // Draw text background for readability
            ctx.fillStyle = colors.bg;
            ctx.globalAlpha = 0.8;
            ctx.fillRect(node.x! - bckgDimensions[0] / 2, node.y! + (8 / globalScale) - bckgDimensions[1] / 2, bckgDimensions[0], bckgDimensions[1]);
            ctx.globalAlpha = 1;

            // Draw text
            ctx.fillStyle = colors.text;
            ctx.fillText(label, node.x!, node.y! + 8 / globalScale);
          }}
        />
      )}
      {!loading && nodes.length === 0 && (
        <div style={{
          position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)',
          color: 'var(--text-muted)', textAlign: 'center'
        }}>
          <p>No graph data available</p>
          <p style={{ fontSize: '12px', marginTop: '8px' }}>Create some files with links to see the graph</p>
        </div>
      )}
    </div>
  );
}
