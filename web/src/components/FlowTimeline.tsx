import React, { useMemo } from 'react';
import { CoverEvent } from '../types';

interface Props {
  events: CoverEvent[];
  goroutines: number[];
  onSelectFile: (file: string) => void;
  sourceCache: Map<string, string[]>;
}

export const FlowTimeline: React.FC<Props> = ({
  events,
  goroutines: _goroutines,
  onSelectFile,
  sourceCache,
}) => {
  // For each goroutine, find its most recent event
  const goroutineLatest = useMemo(() => {
    const map = new Map<number, CoverEvent>();
    for (const ev of events) {
      map.set(ev.gid, ev);
    }
    // Sort by goroutine ID
    return Array.from(map.entries()).sort((a, b) => a[0] - b[0]);
  }, [events]);

  if (goroutineLatest.length === 0) {
    return (
      <div className="flow-timeline-empty">No execution flows yet</div>
    );
  }

  return (
    <div className="flow-timeline">
      <div className="flow-timeline-header">
        Goroutines ({goroutineLatest.length})
      </div>
      <div className="flow-timeline-body">
        {goroutineLatest.map(([gid, ev]) => {
          const lines = sourceCache.get(ev.file);
          const firstLine = lines?.[ev.sl - 1]?.trim() ?? '';
          const blockLines = ev.el - ev.sl + 1;
          const shortFile = ev.file.split('/').pop() ?? '';
          const snippet =
            firstLine.length > 80
              ? firstLine.slice(0, 77) + '...'
              : firstLine;
          const age = (Date.now() - ev.ts / 1e6) / 1000;
          const isHot = age < 3;

          return (
            <div
              key={gid}
              className={`flow-event ${isHot ? 'flow-event-hot' : ''}`}
              onClick={() => onSelectFile(ev.file)}
            >
              <span className="flow-event-gid">g{gid}</span>
              <span className="flow-event-file">
                {shortFile}:{ev.sl}
              </span>
              <span className="flow-event-code" title={firstLine}>
                {snippet || `(line ${ev.sl})`}
              </span>
              <span className="flow-event-lines">
                {blockLines === 1 ? '1 line' : `${blockLines} lines`}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
