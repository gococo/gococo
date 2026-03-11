import React, { useState, useCallback, useRef, useEffect } from 'react';
import { FileTree } from './components/FileTree';
import { CodeView } from './components/CodeView';
import { FlowTimeline } from './components/FlowTimeline';
import { StatusBar } from './components/StatusBar';
import { useEventStream } from './hooks/useEventStream';
import { EventStore } from './stores/eventStore';
import { CoverEvent, FileState, CoverageSummary } from './types';

const store = new EventStore();

export const App: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [files, setFiles] = useState<FileState[]>([]);
  const [goroutines, setGoroutines] = useState<number[]>([]);
  const [recentEvents, setRecentEvents] = useState<CoverEvent[]>([]);
  const [totalEvents, setTotalEvents] = useState(0);
  const [coverage, setCoverage] = useState({ totalStmts: 0, hitStmts: 0, pct: 0 });
  const [_, setTick] = useState(0);

  // Source code cache: file path -> lines
  const sourceCacheRef = useRef(new Map<string, string[]>());
  const fetchingRef = useRef(new Set<string>());
  const hydratedFilesRef = useRef(new Set<string>());

  const batchRef = useRef<number>(0);

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

  // Fetch source for a file if not cached
  const ensureSource = useCallback((filePath: string) => {
    if (sourceCacheRef.current.has(filePath) || fetchingRef.current.has(filePath)) {
      return;
    }
    fetchingRef.current.add(filePath);
    fetch(`/api/source?file=${encodeURIComponent(filePath)}`)
      .then((res) => (res.ok ? res.text() : ''))
      .then((text) => {
        if (text) {
          sourceCacheRef.current.set(filePath, text.split('\n'));
        }
      })
      .catch(() => {})
      .finally(() => fetchingRef.current.delete(filePath));
  }, []);

  // Hydrate block-level coverage for a file from server
  const hydrateFile = useCallback((filePath: string) => {
    if (hydratedFilesRef.current.has(filePath)) return;
    hydratedFilesRef.current.add(filePath);
    fetch(`/api/coverage/blocks?file=${encodeURIComponent(filePath)}`)
      .then((res) => res.json())
      .then((data) => {
        if (data.blocks) {
          store.hydrateBlocks(filePath, data.blocks);
          scheduleUpdate();
        }
      })
      .catch(() => {});
  }, [scheduleUpdate]);

  // Fetch server-side coverage summary (ground truth)
  const fetchServerCoverage = useCallback(() => {
    fetch('/api/coverage/summary')
      .then((res) => res.json())
      .then((data: CoverageSummary) => {
        store.setServerCoverage(data);
        // Ensure sources and hydrate blocks for all known files
        if (data.files) {
          for (const f of data.files) {
            ensureSource(f.file);
            hydrateFile(f.file);
          }
        }
        scheduleUpdate();
      })
      .catch(() => {});
  }, [scheduleUpdate, ensureSource, hydrateFile]);

  // On mount: fetch initial state from server
  useEffect(() => {
    fetchServerCoverage();
  }, [fetchServerCoverage]);

  const handleEvent = useCallback(
    (event: CoverEvent) => {
      store.push(event);
      ensureSource(event.file);
      scheduleUpdate();
    },
    [scheduleUpdate, ensureSource]
  );

  const { connected } = useEventStream(handleEvent);

  // Periodic: refresh glow decay, time-ago, and server coverage
  useEffect(() => {
    const interval = setInterval(() => {
      setTick((t) => t + 1);
      fetchServerCoverage();
    }, 2000);
    return () => clearInterval(interval);
  }, [fetchServerCoverage]);

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
        totalStmts={coverage.totalStmts}
        hitStmts={coverage.hitStmts}
        connected={connected}
      />

      <div className="app-body">
        <aside className="app-sidebar">
          <FileTree
            files={files}
            selected={selectedFile}
            onSelect={setSelectedFile}
            store={store}
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
          sourceCache={sourceCacheRef.current}
        />
      </div>
    </div>
  );
};
