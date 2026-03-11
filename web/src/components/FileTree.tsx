import React from 'react';
import { FileState } from '../types';

interface Props {
  files: FileState[];
  selected: string | null;
  onSelect: (path: string) => void;
}

export const FileTree: React.FC<Props> = ({ files, selected, onSelect }) => {
  const sorted = [...files].sort((a, b) => a.path.localeCompare(b.path));

  return (
    <div className="file-tree">
      <div className="file-tree-header">Files</div>
      <div className="file-tree-list">
        {sorted.length === 0 && (
          <div className="file-tree-empty">Waiting for events...</div>
        )}
        {sorted.map((f) => {
          const hitLines = Array.from(f.lines.values()).filter(
            (l) => l.hitCount > 0
          ).length;
          const totalLines = f.lines.size;
          const pct =
            totalLines > 0 ? ((hitLines / totalLines) * 100).toFixed(0) : '0';
          const shortName = f.path.split('/').slice(-2).join('/');
          const isSelected = f.path === selected;

          return (
            <div
              key={f.path}
              className={`file-tree-item ${isSelected ? 'selected' : ''}`}
              onClick={() => onSelect(f.path)}
              title={f.path}
            >
              <span className="file-name">{shortName}</span>
              <span className="file-coverage">{pct}%</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
