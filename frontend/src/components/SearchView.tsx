import React, { useState, useCallback, useEffect } from 'react';
import { api } from '../lib/api';

interface SearchResult {
  path: string;
  title: string;
  content: string;
}

interface SearchViewProps {
  onSelectFile: (path: string) => void;
}

export default function SearchView({ onSelectFile }: SearchViewProps) {
  const [query, setQuery] = useState('');
  const [searchType, setSearchType] = useState<'lexical' | 'semantic'>('lexical');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [debouncedQuery, setDebouncedQuery] = useState('');

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 300);
    return () => clearTimeout(timer);
  }, [query]);

  // Perform search when debounced query changes
  useEffect(() => {
    if (!debouncedQuery.trim()) {
      setResults([]);
      return;
    }

    const performSearch = async () => {
      setIsSearching(true);
      try {
        const data = await api.search(debouncedQuery, searchType);
        setResults(data.results || []);
      } catch (err) {
        console.error('Search failed:', err);
        setResults([]);
      } finally {
        setIsSearching(false);
      }
    };

    performSearch();
  }, [debouncedQuery, searchType]);

  const handleSelectResult = useCallback((path: string) => {
    onSelectFile(path);
  }, [onSelectFile]);

  return (
    <div className="search-results">
      <div style={{ marginBottom: '24px' }}>
        <input
          type="text"
          placeholder="Search markdown files..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          style={{
            width: '100%',
            padding: '12px 16px',
            background: 'var(--bg-primary)',
            border: '1px solid var(--border-color)',
            borderRadius: '6px',
            color: 'var(--text-primary)',
            fontSize: '14px',
            marginBottom: '12px'
          }}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px', color: 'var(--text-secondary)' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '4px', cursor: 'pointer' }}>
            <input 
              type="radio" 
              checked={searchType === 'lexical'} 
              onChange={() => setSearchType('lexical')} 
              name="searchType"
            />
            Lexical Match
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: '4px', cursor: 'pointer' }}>
            <input 
              type="radio" 
              checked={searchType === 'semantic'} 
              onChange={() => setSearchType('semantic')} 
              name="searchType"
            />
            Semantic / RAG
          </label>
        </div>
      </div>

      {isSearching && (
        <div style={{ marginBottom: '16px', textAlign: 'center' }}>
          <span className="shimmer" style={{ width: '100px', height: '16px', display: 'inline-block' }} />
        </div>
      )}

      {!isSearching && query && results.length === 0 && (
        <div style={{
          textAlign: 'center',
          padding: '48px 0',
          color: 'var(--text-muted)'
        }}>
          <p style={{ fontSize: '48px', opacity: 0.3 }}>🔍</p>
          <p>No results found for "{query}"</p>
        </div>
      )}

      <div>
        {results.map((result, index) => (
          <div
            key={index}
            className="search-result-item"
            onClick={() => handleSelectResult(result.path)}
            style={{ cursor: 'pointer' }}
          >
            <h3>{result.title || result.path}</h3>
            <p>{result.content.substring(0, 150)}...</p>
            <p style={{ fontSize: '11px', marginTop: '4px' }}>{result.path}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
