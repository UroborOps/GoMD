/// <reference types="vite/client" />

declare module 'markdown-it' {
  import MarkdownIt from 'markdown-it';
  export = MarkdownIt;
}

declare module 'react-dom/client' {
  import React from 'react';
  export function createRoot(container: HTMLElement): { render(children: React.ReactNode): void };
}

declare module '*.css' {
  const content: Record<string, string>;
  export default content;
}

declare module '*.md' {
  const content: string;
  export default content;
}
