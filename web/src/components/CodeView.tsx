import React, { useMemo } from 'react';
import { FileState } from '../types';

interface Props {
  fileState: FileState | null;
  isRecentlyHit: (lastHitAt: number) => boolean;
}

export const CodeView: React.FC<Props> = ({ fileState, isRecentlyHit }) => {
  if (!fileState) {
    return (
      <div className="code-view-empty">
        <div className="code-view-empty-icon">{'</>'}</div>
        <div>Select a file to view coverage</div>
      </div>
    );
  }

  // Build line display: we only know which lines are hit, not the actual source.
  // We show line numbers with hit indicators.
  const maxLine = useMemo(() => {
    let max = 0;
    for (const lh of fileState.lines.values()) {
      if (lh.lineNumber > max) max = lh.lineNumber;
    }
    return max;
  }, [fileState]);

  const lines = useMemo(() => {
    const result = [];
    for (let i = 1; i <= maxLine; i++) {
      const lh = fileState.lines.get(i);
      result.push({ lineNumber: i, highlight: lh || null });
    }
    return result;
  }, [fileState, maxLine]);

  const shortPath = fileState.path.split('/').slice(-3).join('/');

  return (
    <div className="code-view">
      <div className="code-view-header">
        <span className="code-view-path" title={fileState.path}>
          {shortPath}
        </span>
      </div>
      <div className="code-view-content">
        {lines.map(({ lineNumber, highlight }) => {
          const hit = highlight && highlight.hitCount > 0;
          const recent = highlight && isRecentlyHit(highlight.lastHitAt);
          const goroutines = highlight
            ? Array.from(highlight.goroutineIds)
            : [];

          let className = 'code-line';
          if (hit && recent) className += ' code-line-hot';
          else if (hit) className += ' code-line-hit';

          return (
            <div key={lineNumber} className={className}>
              <span className="line-number">{lineNumber}</span>
              <span className="line-gutter">
                {hit && (
                  <span className="hit-indicator" title={`hit ${highlight!.hitCount}x`}>
                    {recent ? '\u25CF' : '\u25CB'}
                  </span>
                )}
              </span>
              <span className="line-content">
                {hit && (
                  <>
                    <span className="hit-count">x{highlight!.hitCount}</span>
                    {goroutines.length > 0 && (
                      <span className="goroutine-tags">
                        {goroutines.slice(0, 3).map((g) => (
                          <span key={g} className="goroutine-tag">
                            g{g}
                          </span>
                        ))}
                        {goroutines.length > 3 && (
                          <span className="goroutine-tag">
                            +{goroutines.length - 3}
                          </span>
                        )}
                      </span>
                    )}
                  </>
                )}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
