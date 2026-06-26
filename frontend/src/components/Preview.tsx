import React, { useEffect, useRef, useState, CSSProperties } from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize from 'rehype-sanitize';
import remarkGfm from 'remark-gfm';

interface PreviewProps {
  style?: CSSProperties;
  content: string;
  onNavigate?: (path: string) => void;
}

export default function Preview({ style, content, onNavigate }: PreviewProps) {
  const previewRef = useRef<HTMLDivElement>(null);
  const [scrollSync, setScrollSync] = useState(false);

  // Handle wiki link clicks
  const handleWikiLink = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    const href = e.currentTarget.getAttribute('href');
    if (href?.startsWith('#wiki:')) {
      const fileName = decodeURIComponent(href.substring(6));
      // Navigate to file
      if (onNavigate) {
        onNavigate(fileName);
      } else {
        console.log('Navigate to:', fileName);
      }
    }
  };

  // Scroll sync with editor
  useEffect(() => {
    if (scrollSync && previewRef.current) {
      const handleScroll = () => {
        // In a real app, sync scroll position with editor
      };
      previewRef.current.addEventListener('scroll', handleScroll);
      return () => previewRef.current?.removeEventListener('scroll', handleScroll);
    }
  }, [scrollSync]);

  return (
    <div
      ref={previewRef}
      style={{
        ...style,
        overflowY: 'auto',
        padding: '24px',
        background: 'var(--bg-secondary)',
        fontSize: '15px',
        lineHeight: '1.6',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
      }}
      className="preview"
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, rehypeSanitize]}
        components={{
          a: ({ href, children }) => {
            const isWikiLink = href?.startsWith('#wiki:');
            return (
              <a
                href={href}
                onClick={isWikiLink ? handleWikiLink : undefined}
                className={isWikiLink ? 'wiki-link' : ''}
                style={{ color: 'var(--accent)', textDecoration: 'none' }}
              >
                {children}
              </a>
            );
          },
          code: ({ children }) => (
            <code
              style={{
                background: 'var(--bg-primary)',
                padding: '2px 6px',
                borderRadius: '3px',
                fontFamily: "'SF Mono', 'Fira Code', monospace",
                fontSize: '0.9em'
              }}
            >
              {children}
            </code>
          ),
          pre: ({ children }) => (
            <pre
              style={{
                background: 'var(--bg-primary)',
                padding: '16px',
                borderRadius: '6px',
                overflowX: 'auto',
                margin: '1em 0'
              }}
            >
              {children}
            </pre>
          ),
          blockquote: ({ children }) => (
            <blockquote
              style={{
                borderLeft: '3px solid var(--accent)',
                paddingLeft: '16px',
                margin: '1em 0',
                color: 'var(--text-secondary)'
              }}
            >
              {children}
            </blockquote>
          ),
          h1: ({ children }) => (
            <h1
              style={{
                fontSize: '2em',
                borderBottom: '1px solid var(--border-color)',
                paddingBottom: '0.3em',
                marginBottom: '0.5em'
              }}
            >
              {children}
            </h1>
          ),
          h2: ({ children }) => (
            <h2
              style={{
                fontSize: '1.5em',
                borderBottom: '1px solid var(--border-color)',
                paddingBottom: '0.2em',
                marginBottom: '0.5em'
              }}
            >
              {children}
            </h2>
          ),
          h3: ({ children }) => (
            <h3 style={{ fontSize: '1.25em', marginBottom: '0.5em' }}>
              {children}
            </h3>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
