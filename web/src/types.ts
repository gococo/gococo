export interface CoverEvent {
  seq: number;
  ts: number;
  gid: number;
  file: string;
  block: number;
  sl: number;
  sc: number;
  el: number;
  ec: number;
  stmts: number;
}

export interface AgentInfo {
  Info: {
    id: string;
    hostname: string;
    pid: number;
    cmdline: string;
    remote_ip: string;
  };
  Connected: boolean;
  Since: string;
}

export interface CoverageSummaryEntry {
  file: string;
  total_blocks: number;
  hit_blocks: number;
  total_stmts: number;
  hit_stmts: number;
  percentage: number;
}

export interface CoverageSummary {
  files: CoverageSummaryEntry[] | null;
  total_stmts: number;
  hit_stmts: number;
  overall_pct: number;
  total_events: number;
}

export interface LineHighlight {
  lineNumber: number;
  hitCount: number;
  lastHitAt: number; // timestamp ms
  goroutineIds: Set<number>;
}

export interface FileState {
  path: string;
  lines: Map<number, LineHighlight>;
  totalBlocks: number;
  hitBlocks: number;
}
