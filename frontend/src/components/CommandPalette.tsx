import React, { useState, useEffect, useRef } from 'react';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  commands: { id: string; label: string; action: () => void }[];
}

export default function CommandPalette({ isOpen, onClose, commands }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const filteredCommands = commands.filter(c => c.label.toLowerCase().includes(query.toLowerCase()));

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;
      if (e.key === 'Escape') {
        onClose();
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex(prev => Math.min(prev + 1, filteredCommands.length - 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex(prev => Math.max(prev - 1, 0));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        if (filteredCommands[selectedIndex]) {
          filteredCommands[selectedIndex].action();
          onClose();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, filteredCommands, selectedIndex, onClose]);

  if (!isOpen) return null;

  return (
    <div 
      style={{
        position: 'fixed', inset: 0, zIndex: 10000, 
        display: 'flex', alignItems: 'flex-start', justifyContent: 'center', 
        paddingTop: '10vh', background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)'
      }}
      onClick={onClose}
    >
      <div 
        style={{
          background: 'var(--bg-secondary)', width: '600px', maxWidth: '90%', 
          borderRadius: '8px', boxShadow: '0 16px 40px rgba(0,0,0,0.2)', 
          overflow: 'hidden', border: '1px solid var(--border-color)',
          display: 'flex', flexDirection: 'column'
        }}
        onClick={e => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          value={query}
          onChange={e => { setQuery(e.target.value); setSelectedIndex(0); }}
          placeholder="Type a command or search..."
          style={{
            width: '100%', padding: '16px 20px', fontSize: '18px',
            background: 'transparent', border: 'none', borderBottom: '1px solid var(--border-color)',
            color: 'var(--text-primary)', outline: 'none'
          }}
        />
        <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
          {filteredCommands.map((cmd, i) => (
            <div
              key={cmd.id}
              onClick={() => { cmd.action(); onClose(); }}
              onMouseEnter={() => setSelectedIndex(i)}
              style={{
                padding: '12px 20px',
                cursor: 'pointer',
                display: 'flex', alignItems: 'center',
                background: selectedIndex === i ? 'var(--accent)' : 'transparent',
                color: selectedIndex === i ? '#fff' : 'var(--text-primary)',
                transition: 'background 0.1s'
              }}
            >
              {cmd.label}
            </div>
          ))}
          {filteredCommands.length === 0 && (
            <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-secondary)' }}>
              No commands found.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
