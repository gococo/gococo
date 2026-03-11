import { CoverEvent, FileState, CoverageSummary } from '../types';

const RECENT_HIT_WINDOW_MS = 3000; // lines hit within 3s glow brighter

// Block data from /api/coverage/blocks
interface BlockData {
  block_idx: number;
  sl: number;
  sc: number;
  el: number;
  ec: number;
  stmts: number;
  hit_count: number;
  last_hit_ts: number;
}

export class EventStore {
  private files = new Map<string, FileState>();
  private goroutines = new Set<number>();
  private totalEvents = 0;
  private recentEvents: CoverEvent[] = [];
  private maxRecent = 5000;
  private listeners: Array<() => void> = [];

  // Server-side coverage (ground truth)
  private serverCoverage: CoverageSummary | null = null;

  subscribe(fn: () => void) {
    this.listeners.push(fn);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== fn);
    };
  }

  private notify() {
    for (const fn of this.listeners) fn();
  }

  push(event: CoverEvent) {
    this.totalEvents++;
    this.goroutines.add(event.gid);

    let fileState = this.files.get(event.file);
    if (!fileState) {
      fileState = {
        path: event.file,
        lines: new Map(),
        totalBlocks: 0,
        hitBlocks: 0,
      };
      this.files.set(event.file, fileState);
    }

    const now = Date.now();
    for (let line = event.sl; line <= event.el; line++) {
      let lh = fileState.lines.get(line);
      if (!lh) {
        lh = {
          lineNumber: line,
          hitCount: 0,
          lastHitAt: 0,
          goroutineIds: new Set(),
        };
        fileState.lines.set(line, lh);
      }
      lh.hitCount++;
      lh.lastHitAt = now;
      lh.goroutineIds.add(event.gid);
    }

    if (this.recentEvents.length >= this.maxRecent) {
      this.recentEvents.shift();
    }
    this.recentEvents.push(event);
    this.notify();
  }

  // Hydrate from server block-level data (for blocks hit before page load)
  hydrateBlocks(filePath: string, blocks: BlockData[]) {
    let fileState = this.files.get(filePath);
    if (!fileState) {
      fileState = {
        path: filePath,
        lines: new Map(),
        totalBlocks: 0,
        hitBlocks: 0,
      };
      this.files.set(filePath, fileState);
    }

    for (const b of blocks) {
      if (b.hit_count === 0) continue;
      for (let line = b.sl; line <= b.el; line++) {
        let lh = fileState.lines.get(line);
        if (!lh) {
          lh = {
            lineNumber: line,
            hitCount: 0,
            lastHitAt: 0,
            goroutineIds: new Set(),
          };
          fileState.lines.set(line, lh);
        }
        // Only set if not already updated by live events (don't overwrite fresher data)
        if (lh.hitCount === 0) {
          lh.hitCount = b.hit_count;
          lh.lastHitAt = b.last_hit_ts;
        }
      }
    }
  }

  setServerCoverage(cs: CoverageSummary) {
    this.serverCoverage = cs;
    this.totalEvents = cs.total_events;
  }

  getFiles(): FileState[] {
    // Merge: include files from server coverage that we haven't seen events for
    if (this.serverCoverage?.files) {
      for (const f of this.serverCoverage.files) {
        if (!this.files.has(f.file)) {
          this.files.set(f.file, {
            path: f.file,
            lines: new Map(),
            totalBlocks: f.total_blocks,
            hitBlocks: f.hit_blocks,
          });
        }
      }
    }
    return Array.from(this.files.values());
  }

  getFile(path: string): FileState | undefined {
    return this.files.get(path);
  }

  getGoroutines(): number[] {
    return Array.from(this.goroutines).sort((a, b) => a - b);
  }

  getTotalEvents(): number {
    return this.totalEvents;
  }

  getRecentEvents(n: number = 100): CoverEvent[] {
    return this.recentEvents.slice(-n);
  }

  isRecentlyHit(lastHitAt: number): boolean {
    return Date.now() - lastHitAt < RECENT_HIT_WINDOW_MS;
  }

  // Use server-side coverage as ground truth
  getOverallCoverage(): { totalStmts: number; hitStmts: number; pct: number } {
    if (this.serverCoverage) {
      return {
        totalStmts: this.serverCoverage.total_stmts,
        hitStmts: this.serverCoverage.hit_stmts,
        pct: this.serverCoverage.overall_pct,
      };
    }
    return { totalStmts: 0, hitStmts: 0, pct: 0 };
  }

  // Per-file coverage from server data
  getFileCoverage(filePath: string): { totalStmts: number; hitStmts: number; pct: number } | null {
    if (!this.serverCoverage?.files) return null;
    const f = this.serverCoverage.files.find((e) => e.file === filePath);
    if (!f) return null;
    return { totalStmts: f.total_stmts, hitStmts: f.hit_stmts, pct: f.percentage };
  }

  clear() {
    this.files.clear();
    this.goroutines.clear();
    this.totalEvents = 0;
    this.recentEvents = [];
    this.serverCoverage = null;
    this.notify();
  }
}
