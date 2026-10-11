// packages/api/src/retry.ts
//
// One logical call = one idempotency key, reused unchanged across every
// retry attempt -- a key minted per attempt would let a retried write apply
// twice, which defeats the reason the key exists. Every POST/PATCH/DELETE
// gets one automatically (UUIDv7) unless the caller supplies its own, e.g.
// the outbox's own key minted once at enqueue time.

import { ApiError, request, API_BASE, type RequestOptions } from './transport';
import { getAccessToken } from './auth';
import { DEFAULT_TIMEOUT_MS, getUnauthorizedHandler } from './config';
import { uuid7 } from './uuid7';
import { onBehalfHeader } from './acting';
import { getRefreshToken, refreshSession } from './session-refresh';
import { isLocalMockToken } from './jwt';

const IDEMPOTENT_METHODS = new Set(['POST', 'PATCH', 'PUT', 'DELETE']);

// --- Backend reachability gate -------------------------------------------
// When the backend host is gone (e.g. the VM is deallocated) packets are
// silently dropped, so every call would wait out its own timeout, times
// 1 + retries attempts. One shared, short probe decides instead: if nothing
// answers, every call fails at once (status 0, the marker the callers'
// mock fallbacks already key off) until the cooldown ends.
const RECENTLY_OK_MS = 30_000;
const DOWN_COOLDOWN_MS = 20_000;
const PROBE_TIMEOUT_MS = 3_000;

let lastOkAt = 0;
let downUntil = 0;
let probe: Promise<boolean> | null = null;

function probeUrl(): string | null {
  if (!/^https?:\/\//i.test(API_BASE)) return null; // relative base (dev proxy, tests): nothing to probe
  try {
    return `${new URL(API_BASE).origin}/healthz`;
  } catch {
    return null;
  }
}

async function runProbe(url: string): Promise<boolean> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
  try {
    // no-cors: resolves on ANY answer (even 404/502) and rejects only when
    // nothing answered, so /healthz needs no CORS headers.
    await fetch(url, { mode: 'no-cors', cache: 'no-store', signal: controller.signal });
    lastOkAt = Date.now();
    return true;
  } catch {
    downUntil = Date.now() + DOWN_COOLDOWN_MS;
    return false;
  } finally {
    clearTimeout(timer);
  }
}

async function backendReachable(): Promise<boolean> {
  const url = probeUrl();
  if (url === null) return true;
  const now = Date.now();
  if (now < downUntil) return false;
  if (now - lastOkAt < RECENTLY_OK_MS) return true;
  if (probe === null) probe = runProbe(url).finally(() => { probe = null; });
  return probe;
}


export interface CallOptions extends RequestOptions {
  /** Attempts after the first, for retryable failures. Default 2. */
  retries?: number;
  /** Base backoff in ms, doubled and jittered per attempt. Default 300. */
  backoffMs?: number;
  refreshToken?: string;
  /**
   * Assisted mode: the artisan this call is made for, frozen by the caller
   * (an outbox entry). null = explicitly nobody; undefined = the live
   * acting state (see acting.ts). Ignored on routes the bff does not
   * honour X-On-Behalf-Of on.
   */
  onBehalfOf?: string | null;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Full jitter: spreads retries from concurrent callers instead of them all firing on the same clock tick. */
function jittered(baseMs: number, attempt: number): number {
  return Math.random() * baseMs * 2 ** attempt;
}

/**
 * Attaches the bearer token and Idempotency-Key, retries ApiError.retryable
 * failures (a network failure, or 502/503/504) with jittered backoff, and on
 * a 401 attempts one refresh when a refresh token is available, then calls the
 * injected unauthorized handler if the request remains unauthorized.
 */
export async function call(path: string, options: CallOptions = {}): Promise<unknown> {
  const { retries = 2, backoffMs = 150, idempotencyKey, timeoutMs, refreshToken, onBehalfOf, ...rest } = options;
  const method = (rest.method ?? 'GET').toUpperCase();
  const key = idempotencyKey ?? (IDEMPOTENT_METHODS.has(method) ? uuid7() : undefined);

  for (let attempt = 0; ; attempt++) {
    if (!(await backendReachable())) throw new ApiError(0, null, 'Backend unreachable');

    const token = getAccessToken();
    const localMockToken = isLocalMockToken(token);
    try {
      const result = await request(path, {
        ...rest,
        idempotencyKey: key,
        timeoutMs: timeoutMs ?? DEFAULT_TIMEOUT_MS,
        headers: {
          ...(token === undefined || localMockToken ? {} : { Authorization: `Bearer ${token}` }),
          ...onBehalfHeader(method, path, onBehalfOf),
          ...rest.headers,
        },
      });
      lastOkAt = Date.now();
      return result;
    } catch (cause) {
      if (!(cause instanceof ApiError)) throw cause;

      // Status 0 = nothing answered mid-request: make the next attempt/call re-probe.
      // Any other status proves the server is alive.
      lastOkAt = cause.status === 0 ? 0 : Date.now();

      if (cause.status === 401 && localMockToken) throw cause;

      if (cause.status === 401 && attempt === 0 && (refreshToken ?? getRefreshToken())) {
        if (await refreshSession(refreshToken)) continue;
      }

      if (cause.status === 401) {
        if (!((import.meta.env as any)?.DEV && (token?.includes('devsignature') || token?.includes('dev-')))) {
          getUnauthorizedHandler()?.(path);
        }
        throw cause;
      }

      if (cause.retryable && attempt < retries) {
        await delay(jittered(backoffMs, attempt));
        continue;
      }

      throw cause;
    }
  }
}
