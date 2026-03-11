import React, { useEffect, useState } from 'react';
import { FileState } from '../types';

interface Props {
  fileState: FileState | null;
  isRecentlyHit: (lastHitAt: number) => boolean;
}

function formatTime(ts: number): string {
  const d = new Date(ts);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const ss = String(d.getSeconds()).padStart(2, '0');
  const time = `${hh}:${mm}:${ss}`;

  const diff = (Date.now() - ts) / 1000;
  let ago: string;
  if (diff < 1) ago = 'just now';
  else if (diff < 60) ago = `${Math.floor(diff)}s ago`;
  else if (diff < 3600) ago = `${Math.floor(diff / 60)}m ago`;
  else ago = `${Math.floor(diff / 3600)}h ago`;

  return `${time} (${ago})`;
}

export const CodeView: React.FC<Props> = ({ fileState, isRecentlyHit }) => {
  const [sourceLines, setSourceLines] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!fileState) {
      setSourceLines([]);
      return;
    }
    setLoading(true);
    fetch(`/api/source?file=${encodeURIComponent(fileState.path)}`)
      .then((res) => {
        if (!res.ok) throw new Error('not found');
        return res.text();
      })
      .then((text) => {
        setSourceLines(text.split('\n'));
        setLoading(false);
      })
      .catch(() => {
        setSourceLines([]);
        setLoading(false);
      });
  }, [fileState?.path]);

  if (!fileState) {
    return (
      <div className="code-view-empty">
        <div className="code-view-empty-icon">{'</>'}</div>
        <div>Select a file to view coverage</div>
      </div>
    );
  }

  const maxLine = Math.max(
    sourceLines.length,
    ...Array.from(fileState.lines.keys())
  );

  const shortPath = fileState.path.split('/').slice(-3).join('/');

  return (
    <div className="code-view">
      <div className="code-view-header">
        <span className="code-view-path" title={fileState.path}>
          {shortPath}
        </span>
      </div>
      <div className="code-view-content">
        {loading && <div className="code-view-loading">Loading source...</div>}
        {!loading &&
          Array.from({ length: maxLine }, (_, i) => i + 1).map((lineNumber) => {
            const lh = fileState.lines.get(lineNumber);
            const hit = lh && lh.hitCount > 0;
            const recent = lh && isRecentlyHit(lh.lastHitAt);
            const code = sourceLines[lineNumber - 1] ?? '';

            let className = 'code-line';
            if (hit && recent) className += ' code-line-hot';
            else if (hit) className += ' code-line-hit';

            return (
              <div key={lineNumber} className={className}>
                <span className="line-number">{lineNumber}</span>
                <span className="line-gutter">
                  {hit && (
                    <span
                      className="hit-indicator"
                      title={`hit ${lh!.hitCount}x`}
                    >
                      {recent ? '\u25CF' : '\u25CB'}
                    </span>
                  )}
                </span>
                <span className="line-code">
                  <code>{code}</code>
                </span>
                {hit && (
                  <span className="line-meta">
                    <span className="hit-count">x{lh!.hitCount}</span>
                    <span className="hit-time">{formatTime(lh!.lastHitAt)}</span>
                  </span>
                )}
              </div>
            );
          })}
      </div>
    </div>
  );
};
