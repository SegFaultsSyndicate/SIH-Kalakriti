// packages/offline/src/connection.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { isSlowConnection, saveDataEnabled, shouldConserveData } from './connection';
import { NetworkStatus } from './network.svelte';

function setConnection(value: { saveData?: boolean; effectiveType?: string } | undefined) {
  Object.defineProperty(navigator, 'connection', { value, configurable: true });
}

afterEach(() => setConnection(undefined));

describe('connection', () => {
  it('defaults to false with no Network Information API', () => {
    expect(saveDataEnabled()).toBe(false);
    expect(isSlowConnection()).toBe(false);
    expect(shouldConserveData()).toBe(false);
  });

  it('reads saveData', () => {
    setConnection({ saveData: true });
    expect(saveDataEnabled()).toBe(true);
    expect(shouldConserveData()).toBe(true);
  });

  it('treats 2g and slow-2g as slow, 4g as not', () => {
    setConnection({ effectiveType: '2g' });
    expect(isSlowConnection()).toBe(true);
    setConnection({ effectiveType: 'slow-2g' });
    expect(isSlowConnection()).toBe(true);
    setConnection({ effectiveType: '4g' });
    expect(isSlowConnection()).toBe(false);
  });

  it('keeps the device online when the browser is online and internet is reachable even if the app health endpoint is missing', async () => {
    const prevOnLine = navigator.onLine;
    const prevFetch = globalThis.fetch;

    Object.defineProperty(window.navigator, 'onLine', { configurable: true, value: true });
    globalThis.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input) === '/healthz') {
        return Promise.reject(new Error('health endpoint missing'));
      }
      if (String(input) === 'https://connectivitycheck.gstatic.com/generate_204') {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      return Promise.reject(new Error(`unexpected fetch ${String(input)}`));
    }) as typeof fetch;

    const status = new NetworkStatus();
    status.start();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(status.online).toBe(true);

    status.start();
    Object.defineProperty(window.navigator, 'onLine', { configurable: true, value: prevOnLine });
    globalThis.fetch = prevFetch;
  });
});
