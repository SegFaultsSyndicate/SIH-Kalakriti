// packages/api/src/config.ts
//
// App-supplied seams the transport needs but must not own: locale is UI
// state, and what happens on a 401 is routing -- neither belongs hard-coded
// into a shared fetch wrapper. The app sets these once at startup.

let acceptLanguage = 'en';

export function setAcceptLanguage(code: string): void {
  acceptLanguage = code;
}

export function getAcceptLanguage(): string {
  return acceptLanguage;
}

/**
 * Called when a request cannot be recovered by the shared refresh flow. This
 * is how the app's router finds out it needs to send the user to login.
 */
export type UnauthorizedHandler = (path: string) => void;

let unauthorizedHandler: UnauthorizedHandler | undefined;

export function setUnauthorizedHandler(fn: UnauthorizedHandler | undefined): void {
  unauthorizedHandler = fn;
}

export function getUnauthorizedHandler(): UnauthorizedHandler | undefined {
  return unauthorizedHandler;
}

export type SessionRefreshHandler = (accessToken: string) => void;

let sessionRefreshHandler: SessionRefreshHandler | undefined;

export function setSessionRefreshHandler(fn: SessionRefreshHandler | undefined): void {
  sessionRefreshHandler = fn;
}

export function getSessionRefreshHandler(): SessionRefreshHandler | undefined {
  return sessionRefreshHandler;
}

/** Per-request AbortController timeout when the caller doesn't specify one. */
export const DEFAULT_TIMEOUT_MS = 10_000;
