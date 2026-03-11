import React, { useMemo } from 'react';
import { CoverEvent } from '../types';

interface Props {
  events: CoverEvent[];
  goroutines: number[];
  onSelectFile: (file: string) => void;
}

export const FlowTimeline: React.FC<Props> = ({
  events,
  goroutines,
  onSelectFile,
}) => {
  const recentByGoroutine = useMemo(() => {
    const map = new Map<number, CoverEvent[]>();
    for (const gid of goroutines) {
      map.set(gid, []);
    }
    // Show last 50 events per goroutine
    for (const ev of events) {
      const arr = map.get(ev.gid);
      if (arr && arr.length < 50) {
        arr.push(ev);
      }
    }
    return map;
  }, [events, goroutines]);

  if (goroutines.length === 0) {
    return (
      <div className="flow-timeline-empty">
        No execution flows yet
      </div>
    );
  }

  return (
    <div className="flow-timeline">
      <div className="flow-timeline-header">
        Execution Flows ({goroutines.length} goroutines)
      </div>
      <div className="flow-timeline-body">
        {goroutines.slice(0, 20).map((gid) => {
          const gEvents = recentByGoroutine.get(gid) || [];
          const lastEvent = gEvents[gEvents.length - 1];
          const isActive =
            lastEvent && Date.now() - lastEvent.ts / 1e6 < 5000;

          return (
            <div key={gid} className="flow-row">
              <span
                className={`flow-label ${isActive ? 'flow-active' : 'flow-idle'}`}
              >
                g{gid}
              </span>
              <div className="flow-track">
                {gEvents.map((ev, i) => {
                  const shortFile = ev.file.split('/').pop() || '';
                  return (
                    <span
                      key={i}
                      className="flow-dot"
                      title={`${shortFile}:${ev.sl} (seq ${ev.seq})`}
                      onClick={() => onSelectFile(ev.file)}
                    />
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
