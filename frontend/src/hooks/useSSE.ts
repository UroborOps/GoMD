import { useState, useEffect, useRef, useCallback } from 'react';

interface SSEEvent {
  type: string;
  path: string;
  message?: string;
}

export function useSSE(onEvent?: (event: SSEEvent) => void) {
  const [connected, setConnected] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (esRef.current) return;
    const es = new EventSource('/events');
    es.onopen = () => setConnected(true);
    es.onmessage = (evt) => {
      try {
        const event: SSEEvent = JSON.parse(evt.data);
        onEvent?.(event);
      } catch {
        // ignore
      }
    };
    es.onerror = () => {
      setConnected(false);
      es.close();
      esRef.current = null;
    };
    esRef.current = es;
  }, [onEvent]);

  useEffect(() => {
    connect();
    return () => {
      esRef.current?.close();
      esRef.current = null;
    };
  }, [connect]);

  return { connected };
}
