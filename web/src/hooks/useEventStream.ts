import { useEffect, useRef, useCallback } from 'react';
import { CoverEvent } from '../types';

export function useEventStream(
  onEvent: (event: CoverEvent) => void,
  enabled: boolean = true
) {
  const eventSourceRef = useRef<EventSource | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!enabled) return;

    const es = new EventSource('/api/events/stream');
    eventSourceRef.current = es;

    es.onmessage = (msg) => {
      try {
        const event: CoverEvent = JSON.parse(msg.data);
        onEventRef.current(event);
      } catch {
        // ignore malformed events
      }
    };

    es.onerror = () => {
      // EventSource auto-reconnects
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [enabled]);

  const disconnect = useCallback(() => {
    eventSourceRef.current?.close();
    eventSourceRef.current = null;
  }, []);

  return { disconnect };
}
