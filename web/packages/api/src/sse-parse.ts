// packages/api/src/sse-parse.ts
//
// Pure parsing/backoff helpers for sse.svelte.ts, split out so they're
// testable without the svelte compiler runes need.

export interface SseEvent {
  event: string;
  data: string;
  id?: string;
}

/** Parses one blank-line-delimited "event:\ndata:\nid:\n" block. Returns null for a block with no data line. */
export function parseSseEvent(chunk: string): SseEvent | null {
  let event = 'message';
  let id: string | undefined;
  const dataLines: string[] = [];

  for (const line of chunk.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim();
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim());
    else if (line.startsWith('id:')) id = line.slice(3).trim();
  }

  return dataLines.length === 0 ? null : { event, data: dataLines.join('\n'), id };
}

/** Exponential backoff, capped, no jitter needed -- one reconnecting client, not a thundering herd. */
export function nextBackoffMs(attempt: number, baseMs = 300, maxMs = 30_000): number {
  return Math.min(baseMs * 2 ** attempt, maxMs);
}
