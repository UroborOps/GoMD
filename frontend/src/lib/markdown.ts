import MarkdownIt from 'markdown-it';
import highlight from 'markdown-it-highlightjs';

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
});

// Wiki link syntax: [[Page Name]] or [[Page Name|Alias]]
md.core.ruler.push('wiki-links', (state: any) => {
  state.tokens.forEach((token: any) => {
    if (token.type === 'inline' && token.content) {
      const wikiRegex = /\[\[([^\]]+)\]\]/g;
      let match;
      const parts: string[] = [];
      let lastIndex = 0;
      while ((match = wikiRegex.exec(token.content)) !== null) {
        // Text before match
        if (match.index > lastIndex) {
          parts.push(token.content.slice(lastIndex, match.index));
        }
        const content = match[1];
        const [link, alias] = content.split('|').map((s) => s.trim());
        const display = alias || link;
        // Escape the display text for safe HTML
        const safeDisplay = display.replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
        parts.push(`<a href="#wiki:${encodeURIComponent(link)}" class="wiki-link">${safeDisplay}</a>`);
        lastIndex = match.index + match[0].length;
      }
      if (parts.length > 0) {
        parts.push(token.content.slice(lastIndex));
        token.content = parts.join('');
      }
    }
  });
});

// Code blocks with language
md.use(highlight);

export default md;
