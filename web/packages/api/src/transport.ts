// packages/api/src/transport.ts
//
// Raw fetch transport. Nothing here is spec-shaped; typed operations in
// operations.ts sit on top via call() in retry.ts. App code should not call
// request() directly.

import { getAcceptLanguage } from './config';

declare global {
  interface ImportMetaEnv {
    readonly VITE_API_BASE?: string;
  }
  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

export const API_BASE: string = import.meta.env?.VITE_API_BASE ?? '/api/v1';

/** A non-2xx response, carrying enough to show the artisan a real reason. */
export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, body: unknown, message?: string) {
    super(message ?? `Request failed with status ${status}`);
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
  }

  /** Worth retrying unchanged: a network failure, or a gateway/upstream failure (502/503/504). Not 4xx, not a bare 500. */
  get retryable(): boolean {
    return this.status === 0 || this.status === 502 || this.status === 503 || this.status === 504;
  }
}

/** Bodies that must reach fetch as-is, never through JSON.stringify. */
type RawBody = Blob | ArrayBuffer;

function isRawBody(body: unknown): body is RawBody {
  return body instanceof Blob || body instanceof ArrayBuffer;
}

export interface RequestOptions extends Omit<RequestInit, 'body'> {
  /** A plain value is JSON-encoded; a Blob/ArrayBuffer is sent as-is (e.g. recorded audio) with no Content-Type forced -- set one via headers. */
  body?: unknown;
  /**
   * Sent as Idempotency-Key. Mandatory for any retried write, which on this
   * product means every write the outbox carries.
   */
  idempotencyKey?: string;
  /**
   * Abort after this many milliseconds. Defaults to none: the caller knows
   * whether it is a foreground tap (short) or a background drain (long).
   */
  timeoutMs?: number;
}

/**
 * Resolves against the page origin in a browser (unchanged prior behaviour).
 * Node's fetch has no document base URL and rejects a bare relative path, so
 * tests need an explicit fallback origin -- MSW matches on path regardless.
 */
function resolveUrl(path: string): string {
  const base = typeof location === 'undefined' ? 'http://localhost' : location.origin;
  return new URL(`${API_BASE}${path}`, base).toString();
}

/**
 * Transport only. Typed operation wrappers are generated on top of this once
 * the spec is available; callers in app code use those, not this.
 */
export async function request(path: string, options: RequestOptions = {}): Promise<unknown> {
  const { body, idempotencyKey, timeoutMs, headers, ...init } = options;

  const controller = timeoutMs === undefined ? null : new AbortController();
  const timer = controller === null ? null : setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(resolveUrl(path), {
      ...init,
      signal: controller?.signal ?? init.signal,
      headers: {
        Accept: 'application/json',
        'Accept-Language': getAcceptLanguage(),
        ...(body === undefined || isRawBody(body) ? {} : { 'Content-Type': 'application/json' }),
        ...(idempotencyKey === undefined ? {} : { 'Idempotency-Key': idempotencyKey }),
        ...headers,
      },
      body: body === undefined ? undefined : isRawBody(body) ? body : JSON.stringify(body),
    });

    const payload = response.status === 204 ? null : await response.json().catch(() => null);

    if (!response.ok) throw new ApiError(response.status, payload);
    return payload;
  } catch (cause) {
    if (cause instanceof ApiError) throw cause;
    // Status 0 is this codebase's marker for "never reached the server",
    // which is the case the outbox exists to absorb.
    throw new ApiError(0, null, cause instanceof Error ? cause.message : String(cause));
  } finally {
    if (timer !== null) clearTimeout(timer);
  }
}
