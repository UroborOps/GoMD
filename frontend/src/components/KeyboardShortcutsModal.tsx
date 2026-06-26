import React from 'react';

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function KeyboardShortcutsModal({ isOpen, onClose }: KeyboardShortcutsModalProps) {
  if (!isOpen) return null;

  const shortcuts = [
    { keys: ['Ctrl', 'P'], description: 'Open Command Palette' },
    { keys: ['Ctrl', 'Z'], description: 'Undo' },
    { keys: ['Ctrl', 'Y'], description: 'Redo' },
    { keys: ['Ctrl', 'S'], description: 'Save File' },
    { keys: ['Esc'], description: 'Close Modals/Palette' }
  ];

  return (
    <div 
      style={{
        position: 'fixed', inset: 0, zIndex: 10000, 
        display: 'flex', alignItems: 'center', justifyContent: 'center', 
        background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)'
      }}
      onClick={onClose}
    >
      <div 
        style={{
          background: 'var(--bg-secondary)', width: '400px', maxWidth: '90%', 
          borderRadius: '8px', boxShadow: '0 16px 40px rgba(0,0,0,0.2)', 
          padding: '24px', border: '1px solid var(--border-color)',
          display: 'flex', flexDirection: 'column', gap: '16px'
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h3 style={{ margin: 0, fontSize: '18px', color: 'var(--text-primary)' }}>Keyboard Shortcuts</h3>
          <button 
            onClick={onClose}
            style={{ 
              background: 'transparent', border: 'none', color: 'var(--text-secondary)', 
              cursor: 'pointer', fontSize: '16px', padding: '4px'
            }}
          >
            ✕
          </button>
        </div>
        
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginTop: '8px' }}>
          {shortcuts.map((shortcut, index) => (
            <div key={index} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>{shortcut.description}</span>
              <div style={{ display: 'flex', gap: '4px' }}>
                {shortcut.keys.map((key, i) => (
                  <kbd 
                    key={i} 
                    style={{ 
                      background: 'var(--bg-primary)', border: '1px solid var(--border-color)', 
                      borderRadius: '4px', padding: '2px 6px', fontSize: '12px', 
                      color: 'var(--text-primary)', fontFamily: 'monospace'
                    }}
                  >
                    {key}
                  </kbd>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
