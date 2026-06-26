import React from 'react';

interface BacklinkItem {
  file: string;
  heading?: string;
  alias?: string;
}

interface BacklinksProps {
  backlinks: BacklinkItem[];
  onSelectFile: (path: string) => void;
}

export default function Backlinks({ backlinks, onSelectFile }: BacklinksProps) {
  if (backlinks.length === 0) {
    return (
      <div className="backlinks">
        <h4>Backlinks</h4>
        <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>No backlinks found</span>
      </div>
    );
  }

  return (
    <div className="backlinks">
      <h4>Backlinks ({backlinks.length})</h4>
      {backlinks.map((bl, index) => (
        <div
          key={index}
          className="backlink-item"
          onClick={() => onSelectFile(bl.file)}
        >
          {bl.alias || bl.file}
          {bl.heading && <span style={{ color: 'var(--text-secondary)', fontSize: '10px' }}> → {bl.heading}</span>}
        </div>
      ))}
    </div>
  );
}
