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
 * Called once per request on a 401. There is no /auth/refresh in
 * services/bff/openapi.json yet, so a 401 cannot be silently repaired -- this
 * is how the app's router finds out it needs to send the artisan to login.
 */
export type UnauthorizedHandler = (path: string) => void;

let unauthorizedHandler: UnauthorizedHandler | undefined;

export function setUnauthorizedHandler(fn: UnauthorizedHandler | undefined): void {
  unauthorizedHandler = fn;
}

export function getUnauthorizedHandler(): UnauthorizedHandler | undefined {
  return unauthorizedHandler;
}

/** Per-request AbortController timeout when the caller doesn't specify one. */
export const DEFAULT_TIMEOUT_MS = 10_000;
