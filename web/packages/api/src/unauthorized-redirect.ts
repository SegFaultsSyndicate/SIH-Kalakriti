// packages/api/src/unauthorized-redirect.ts
//
// The actual "401 -> clean redirect to login, preserving the intended
// destination" behaviour, wired in as: setUnauthorizedHandler(
// createLoginRedirectHandler(goto)) once at app start, passing SvelteKit's
// own `goto` from '$app/navigation'.
//
// This package cannot import '$app/navigation' itself -- that alias only
// resolves inside a real SvelteKit app's module graph, not in a shared
// package built and tested standalone -- so `goto` is injected instead.

import { session } from './session.svelte';

export interface LoginRedirectOptions {
  /** Default '/login'. */
  loginPath?: string;
}

/** Clears the stale session and sends the browser to login?redirect=<path the 401 happened on>. */
export function createLoginRedirectHandler(
  goto: (url: string) => void,
  options: LoginRedirectOptions = {},
): (path: string) => void {
  const loginPath = options.loginPath ?? '/login';
  return (path: string) => {
    session.clear();
    goto(`${loginPath}?redirect=${encodeURIComponent(path)}`);
  };
}
