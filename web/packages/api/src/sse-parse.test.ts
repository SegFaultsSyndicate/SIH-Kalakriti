// packages/api/src/sse-parse.test.ts
import { describe, expect, it } from 'vitest';
import { nextBackoffMs, parseSseEvent } from './sse-parse';

describe('parseSseEvent', () => {
  it('defaults to a "message" event when none is given', () => {
    expect(parseSseEvent('data: hello')).toEqual({ event: 'message', data: 'hello', id: undefined });
  });

  it('reads a custom event name and id', () => {
    expect(parseSseEvent('event: lot.responded\ndata: {"lotId":"l1"}\nid: 42')).toEqual({
      event: 'lot.responded',
      data: '{"lotId":"l1"}',
      id: '42',
    });
  });

  it('joins multiple data lines with a newline, per the SSE spec', () => {
    expect(parseSseEvent('data: line one\ndata: line two')).toEqual({
      event: 'message',
      data: 'line one\nline two',
      id: undefined,
    });
  });

  it('returns null for a block with no data line (a bare comment or id ping)', () => {
    expect(parseSseEvent('id: 42')).toBeNull();
    expect(parseSseEvent(': keep-alive')).toBeNull();
  });
});

describe('nextBackoffMs', () => {
  it('doubles per attempt', () => {
    expect(nextBackoffMs(0, 300)).toBe(300);
    expect(nextBackoffMs(1, 300)).toBe(600);
    expect(nextBackoffMs(2, 300)).toBe(1200);
  });

  it('caps at maxMs', () => {
    expect(nextBackoffMs(10, 300, 30_000)).toBe(30_000);
  });
});
