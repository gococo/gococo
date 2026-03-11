import { CoverEvent, FileState } from '../types';

const RECENT_HIT_WINDOW_MS = 3000; // lines hit within 3s glow brighter

export class EventStore {
  private files = new Map<string, FileState>();
  private goroutines = new Set<number>();
  private totalEvents = 0;
  private recentEvents: CoverEvent[] = [];
  private maxRecent = 5000;
  private listeners: Array<() => void> = [];

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

    // Update file state
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

    // Mark all lines in the block range
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

    // Track recent events (circular)
    if (this.recentEvents.length >= this.maxRecent) {
      this.recentEvents.shift();
    }
    this.recentEvents.push(event);

    // Batch notifications to avoid excessive re-renders
    // (caller should debounce or use requestAnimationFrame)
    this.notify();
  }

  getFiles(): FileState[] {
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

  getOverallCoverage(): { totalLines: number; hitLines: number; pct: number } {
    let totalLines = 0;
    let hitLines = 0;
    for (const file of this.files.values()) {
      totalLines += file.lines.size;
      for (const lh of file.lines.values()) {
        if (lh.hitCount > 0) hitLines++;
      }
    }
    return {
      totalLines,
      hitLines,
      pct: totalLines > 0 ? (hitLines / totalLines) * 100 : 0,
    };
  }

  clear() {
    this.files.clear();
    this.goroutines.clear();
    this.totalEvents = 0;
    this.recentEvents = [];
    this.notify();
  }
}
