import React, { useRef, useEffect, useCallback, CSSProperties, useState } from 'react';
import MonacoEditor, { useMonaco } from '@monaco-editor/react';
import { BoldIcon, ItalicIcon, HeadingIcon, StrikethroughIcon, CodeIcon, LinkIcon, ListIcon, OrderedListIcon, QuoteIcon, UndoIcon, RedoIcon } from './Icons';

interface EditorProps {
  style?: CSSProperties;
  value: string;
  onChange?: (value: string) => void;
  currentFile: string;
  readOnly?: boolean;
}

export default function Editor({ style, value, onChange, currentFile, readOnly = false }: EditorProps) {
  const [localValue, setLocalValue] = useState(value);
  const editorRef = useRef<any>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const monaco = useMonaco();

  // Detect theme from body class
  const [theme, setTheme] = useState(document.body.className.replace('theme-', '') || 'dark');
  useEffect(() => {
    const observer = new MutationObserver(() => {
      setTheme(document.body.className.replace('theme-', '') || 'dark');
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ['class'] });
    return () => observer.disconnect();
  }, []);

  // Update Monaco theme based on app theme
  const getMonacoTheme = () => {
    if (theme.includes('light')) return 'vs';
    return 'vs-dark'; // dark, dracula, monokai, solarized-dark map well enough to vs-dark by default
  };

  // Reset local value only when currentFile changes
  useEffect(() => {
    setLocalValue(value);
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentFile]);

  // Handle monaco input with debounce
  const handleChange = useCallback((newValue: string | undefined) => {
    if (readOnly) return;
    const val = newValue || '';
    setLocalValue(val);

    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    typingTimeoutRef.current = setTimeout(() => {
      onChange?.(val);
    }, 500);
  }, [onChange, readOnly]);

  const triggerUndo = useCallback(() => {
    if (editorRef.current && !readOnly) {
      editorRef.current.trigger('toolbar', 'undo', null);
      editorRef.current.focus();
    }
  }, [readOnly]);

  const triggerRedo = useCallback(() => {
    if (editorRef.current && !readOnly) {
      editorRef.current.trigger('toolbar', 'redo', null);
      editorRef.current.focus();
    }
  }, [readOnly]);

  const insertFormatting = useCallback((prefix: string, suffix: string = '') => {
    const editor = editorRef.current;
    if (!editor || readOnly) return;

    const selection = editor.getSelection();
    if (!selection) return;

    const model = editor.getModel();
    const selectedText = model.getValueInRange(selection);
    
    const textToInsert = prefix + selectedText + suffix;

    editor.executeEdits('toolbar', [
      {
        range: selection,
        text: textToInsert,
        forceMoveMarkers: true
      }
    ]);
    editor.focus();
    
    // reposition cursor if there's a suffix and no text was selected
    if (suffix && !selectedText) {
      const position = editor.getPosition();
      editor.setPosition({
        lineNumber: position.lineNumber,
        column: position.column - suffix.length
      });
    }
  }, [readOnly]);

  return (
    <div style={{ ...style, display: 'flex', flexDirection: 'column', height: '100%', background: 'var(--bg-primary)' }}>
      {!readOnly && (
        <div className="editor-toolbar">
          <button className="toolbar-btn" onClick={triggerUndo} title="Undo"><UndoIcon /></button>
          <button className="toolbar-btn" onClick={triggerRedo} title="Redo"><RedoIcon /></button>
          <div className="toolbar-divider" />
          <button className="toolbar-btn" onClick={() => insertFormatting('**', '**')} title="Bold"><BoldIcon /></button>
          <button className="toolbar-btn" onClick={() => insertFormatting('*', '*')} title="Italic"><ItalicIcon /></button>
          <button className="toolbar-btn" onClick={() => insertFormatting('~~', '~~')} title="Strikethrough"><StrikethroughIcon /></button>
          <div className="toolbar-divider" />
          <button className="toolbar-btn" onClick={() => insertFormatting('# ')} title="Heading 1"><HeadingIcon /></button>
          <div className="toolbar-divider" />
          <button className="toolbar-btn" onClick={() => insertFormatting('- ')} title="Unordered List"><ListIcon /></button>
          <button className="toolbar-btn" onClick={() => insertFormatting('1. ')} title="Ordered List"><OrderedListIcon /></button>
          <div className="toolbar-divider" />
          <button className="toolbar-btn" onClick={() => insertFormatting('> ')} title="Quote"><QuoteIcon /></button>
          <button className="toolbar-btn" onClick={() => insertFormatting('`', '`')} title="Code"><CodeIcon /></button>
          <button className="toolbar-btn" onClick={() => insertFormatting('[', '](url)')} title="Link"><LinkIcon /></button>
        </div>
      )}
      <div style={{ flex: 1, position: 'relative' }}>
        <MonacoEditor
          height="100%"
          language="markdown"
          theme={getMonacoTheme()}
          value={localValue}
          onChange={handleChange}
          onMount={(editor) => { editorRef.current = editor; }}
          options={{
            readOnly,
            wordWrap: "on",
            minimap: { enabled: false },
            lineNumbers: "off",
            fontFamily: "'SF Mono', 'Fira Code', 'Consolas', monospace",
            fontSize: 14,
            padding: { top: 24, bottom: 24 },
            scrollBeyondLastLine: false,
            renderLineHighlight: "none",
            hideCursorInOverviewRuler: true,
            overviewRulerBorder: false,
            scrollbar: {
              useShadows: false,
              verticalHasArrows: false,
              horizontalHasArrows: false,
              verticalScrollbarSize: 8,
              horizontalScrollbarSize: 8,
            }
          }}
        />
      </div>
    </div>
  );
}
