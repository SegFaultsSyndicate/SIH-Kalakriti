// packages/api/src/sse.svelte.ts
//
// GET /orders/{id}/events (services/bff/openapi.json) is text/event-stream.
// Native EventSource can't set a Last-Event-ID header or control reconnect
// timing, so this reads the stream by hand over fetch + ReadableStream --
// the only way to send a real Last-Event-ID and back off exponentially
// instead of the browser's fixed retry delay.

import { getAccessToken } from './auth';
import { getAcceptLanguage } from './config';
import { API_BASE } from './transport';
import { nextBackoffMs, parseSseEvent, type SseEvent } from './sse-parse';

export type SseStatus = 'connecting' | 'open' | 'closed';

interface OrderEventsState {
  status: SseStatus;
  lastEvent: SseEvent | null;
  error: string | null;
}

/** Connects to an order's event stream, reconnecting with backoff and resuming from the last event id seen. */
export function watchOrderEvents(orderId: string): { state: OrderEventsState; stop: () => void } {
  const state = $state<OrderEventsState>({ status: 'connecting', lastEvent: null, error: null });
  let stopped = false;
  let lastEventId: string | undefined;
  let attempt = 0;

  async function connectOnce(): Promise<void> {
    const token = getAccessToken();
    const response = await fetch(`${API_BASE}/orders/${encodeURIComponent(orderId)}/events`, {
      headers: {
        Accept: 'text/event-stream',
        'Accept-Language': getAcceptLanguage(),
        ...(token === undefined ? {} : { Authorization: `Bearer ${token}` }),
        ...(lastEventId === undefined ? {} : { 'Last-Event-ID': lastEventId }),
      },
    });
    if (!response.ok || response.body === null) {
      throw new Error(`SSE connect failed: ${response.status}`);
    }

    state.status = 'open';
    state.error = null;
    attempt = 0;

    const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
    let buffer = '';
    while (!stopped) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += value;
      const blocks = buffer.split('\n\n');
      buffer = blocks.pop() ?? '';
      for (const block of blocks) {
        const parsed = parseSseEvent(block);
        if (parsed === null) continue;
        if (parsed.id !== undefined) lastEventId = parsed.id;
        state.lastEvent = parsed;
      }
    }
  }

  async function loop(): Promise<void> {
    while (!stopped) {
      try {
        await connectOnce();
        if (stopped) return;
      } catch (cause) {
        state.error = cause instanceof Error ? cause.message : String(cause);
      }
      if (stopped) return;
      state.status = 'connecting';
      await new Promise((resolve) => setTimeout(resolve, nextBackoffMs(attempt)));
      attempt += 1;
    }
  }

  void loop();

  return {
    state,
    stop: () => {
      stopped = true;
      state.status = 'closed';
    },
  };
}
