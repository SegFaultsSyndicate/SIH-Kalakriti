// packages/api/src/requireRole.ts
//
// A route guard for a +layout.ts load(). The role claim's key isn't in the
// spec (the JWT's claim names are undocumented -- see jwt.ts), so the caller
// states which claim to check instead of this package guessing `role`.

import { redirect } from '@sveltejs/kit';
import { session } from './session.svelte';

export interface RequireRoleOptions {
  /** Claim key holding the role, e.g. 'role'. */
  claimKey: string;
  /** Default '/login'. */
  loginPath?: string;
}

/** Throws a SvelteKit redirect to login, preserving currentPath, unless the session holds the expected role. */
export function requireRole(expected: string, currentPath: string, options: RequireRoleOptions): void {
  const role = session.claims?.[options.claimKey];
  if (session.status !== 'authenticated' || role !== expected) {
    const loginPath = options.loginPath ?? '/login';
    throw redirect(303, `${loginPath}?redirect=${encodeURIComponent(currentPath)}`);
  }
}
