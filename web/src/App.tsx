import React, { useState, useCallback, useRef, useEffect } from 'react';
import { FileTree } from './components/FileTree';
import { CodeView } from './components/CodeView';
import { FlowTimeline } from './components/FlowTimeline';
import { StatusBar } from './components/StatusBar';
import { useEventStream } from './hooks/useEventStream';
import { EventStore } from './stores/eventStore';
import { CoverEvent, FileState } from './types';

const store = new EventStore();

export const App: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [files, setFiles] = useState<FileState[]>([]);
  const [goroutines, setGoroutines] = useState<number[]>([]);
  const [recentEvents, setRecentEvents] = useState<CoverEvent[]>([]);
  const [totalEvents, setTotalEvents] = useState(0);
  const [coverage, setCoverage] = useState({ totalLines: 0, hitLines: 0, pct: 0 });
  const [connected, setConnected] = useState(false);
  const [_, setTick] = useState(0);

  const batchRef = useRef<number>(0);

  // Debounced UI update
  const scheduleUpdate = useCallback(() => {
    if (batchRef.current) return;
    batchRef.current = requestAnimationFrame(() => {
      batchRef.current = 0;
      setFiles(store.getFiles());
      setGoroutines(store.getGoroutines());
      setRecentEvents(store.getRecentEvents(200));
      setTotalEvents(store.getTotalEvents());
      setCoverage(store.getOverallCoverage());
    });
  }, []);

  const handleEvent = useCallback(
    (event: CoverEvent) => {
      if (!connected) setConnected(true);
      store.push(event);
      scheduleUpdate();
    },
    [connected, scheduleUpdate]
  );

  useEventStream(handleEvent);

  // Periodic refresh for "recently hit" glow decay
  useEffect(() => {
    const interval = setInterval(() => {
      setTick((t) => t + 1);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  const selectedFileState = selectedFile ? store.getFile(selectedFile) : null;

  return (
    <div className="app">
      <header className="app-header">
        <h1 className="app-title">gococo</h1>
        <span className="app-subtitle">Live Coverage Viewer</span>
      </header>

      <StatusBar
        totalEvents={totalEvents}
        goroutineCount={goroutines.length}
        fileCount={files.length}
        coveragePct={coverage.pct}
        connected={connected}
      />

      <div className="app-body">
        <aside className="app-sidebar">
          <FileTree
            files={files}
            selected={selectedFile}
            onSelect={setSelectedFile}
          />
        </aside>

        <main className="app-main">
          <CodeView
            fileState={selectedFileState ?? null}
            isRecentlyHit={(ts) => store.isRecentlyHit(ts)}
          />
        </main>
      </div>

      <div className="app-bottom">
        <FlowTimeline
          events={recentEvents}
          goroutines={goroutines}
          onSelectFile={setSelectedFile}
        />
      </div>
    </div>
  );
};
