import React from 'react';

interface Props {
  totalEvents: number;
  goroutineCount: number;
  fileCount: number;
  coveragePct: number;
  connected: boolean;
}

export const StatusBar: React.FC<Props> = ({
  totalEvents,
  goroutineCount,
  fileCount,
  coveragePct,
  connected,
}) => {
  return (
    <div className="status-bar">
      <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`}>
        {connected ? '\u25CF Connected' : '\u25CB Disconnected'}
      </span>
      <span className="status-item">
        Events: <strong>{totalEvents.toLocaleString()}</strong>
      </span>
      <span className="status-item">
        Goroutines: <strong>{goroutineCount}</strong>
      </span>
      <span className="status-item">
        Files: <strong>{fileCount}</strong>
      </span>
      <span className="status-item coverage-pct">
        Coverage: <strong>{coveragePct.toFixed(1)}%</strong>
      </span>
    </div>
  );
};
