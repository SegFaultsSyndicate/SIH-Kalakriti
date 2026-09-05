import { afterEach, describe, expect, it, vi } from 'vitest';
import { reportError } from './report';

describe('reportError', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('logs without throwing when reporting is disabled', () => {
    const error = new Error('boom');
    const log = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    expect(() => reportError(error, { source: 'test' })).not.toThrow();
    expect(log).toHaveBeenCalledWith('[test]', error);
  });

  it('posts a Sentry envelope when a valid DSN is configured', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response());
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    reportError(new Error('boom'), { source: 'test', route: '/demo' }, 'https://public@example.test/42');
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/api/42/envelope/');
    expect(init?.method).toBe('POST');
    expect(init?.headers).toEqual({ 'Content-Type': 'application/x-sentry-envelope' });
    expect(String(init?.body)).toContain('"source":"test"');
  });
});
