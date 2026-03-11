import React, { useMemo, useState } from 'react';
import { FileState } from '../types';
import { EventStore } from '../stores/eventStore';

interface Props {
  files: FileState[];
  selected: string | null;
  onSelect: (path: string) => void;
  store: EventStore;
}

interface TreeNode {
  name: string;
  fullPath: string;
  isDir: boolean;
  children: TreeNode[];
  fileState?: FileState;
}

function buildTree(files: FileState[]): TreeNode[] {
  const root: TreeNode = { name: '', fullPath: '', isDir: true, children: [] };

  for (const f of files) {
    const parts = f.path.split('/');
    let node = root;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      let child = node.children.find((c) => c.name === part);
      if (!child) {
        child = {
          name: part,
          fullPath: parts.slice(0, i + 1).join('/'),
          isDir: !isLast,
          children: [],
        };
        if (isLast) child.fileState = f;
        node.children.push(child);
      }
      node = child;
    }
  }

  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
    for (const n of nodes) {
      if (n.children.length > 0) sortNodes(n.children);
    }
  };
  sortNodes(root.children);
  return root.children;
}

const TreeItem: React.FC<{
  node: TreeNode;
  depth: number;
  selected: string | null;
  onSelect: (path: string) => void;
  store: EventStore;
}> = ({ node, depth, selected, onSelect, store }) => {
  const [expanded, setExpanded] = useState(true);

  if (node.isDir) {
    return (
      <>
        <div
          className="file-tree-dir"
          style={{ paddingLeft: depth * 16 + 8 }}
          onClick={() => setExpanded(!expanded)}
        >
          <span className="dir-arrow">{expanded ? '\u25BE' : '\u25B8'}</span>
          <span className="dir-name">{node.name}/</span>
        </div>
        {expanded &&
          node.children.map((child) => (
            <TreeItem
              key={child.fullPath}
              node={child}
              depth={depth + 1}
              selected={selected}
              onSelect={onSelect}
              store={store}
            />
          ))}
      </>
    );
  }

  const isSelected = node.fullPath === selected;
  // Use server-side per-file coverage
  const fc = store.getFileCoverage(node.fullPath);
  const pct = fc ? fc.pct.toFixed(0) : '0';

  return (
    <div
      className={`file-tree-item ${isSelected ? 'selected' : ''}`}
      style={{ paddingLeft: depth * 16 + 8 }}
      onClick={() => onSelect(node.fullPath)}
      title={node.fullPath}
    >
      <span className="file-name">{node.name}</span>
      <span className="file-coverage">{pct}%</span>
    </div>
  );
};

export const FileTree: React.FC<Props> = ({ files, selected, onSelect, store }) => {
  const tree = useMemo(() => buildTree(files), [files]);

  return (
    <div className="file-tree">
      <div className="file-tree-header">Files</div>
      <div className="file-tree-list">
        {tree.length === 0 && (
          <div className="file-tree-empty">Waiting for events...</div>
        )}
        {tree.map((node) => (
          <TreeItem
            key={node.fullPath}
            node={node}
            depth={0}
            selected={selected}
            onSelect={onSelect}
            store={store}
          />
        ))}
      </div>
    </div>
  );
};
