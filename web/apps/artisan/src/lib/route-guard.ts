// apps/artisan/src/lib/route-guard.ts
//
// One function decides where the artisan is allowed to be, given three
// booleans and the path they are trying to reach. Everything else (reading
// localStorage, the session store, the registered-artisan pref) is I/O and
// lives in +layout.svelte, which calls this with the values already read.
//
// The chain, forward and back:
//   no chosen language          -> /language (except /accessibility)
//   language chosen, no session -> /welcome, /login or /verify only
//   session, not registered     -> /register/* only
//   fully onboarded             -> anywhere except the screens above, which
//                                   redirect to home so a completed artisan
//                                   cannot re-submit registration by going back
//
// Registration's own step order is not enforced here: /register/craft with
// no name yet just shows an empty name on a later summary, and Dexie already
// has each answer the moment it is entered, so there is nothing to lose by
// letting the artisan jump between steps via the Stepper or the back button.

export const REGISTER_FIRST_STEP = '/register/name';
export const HOME_PATH = '/';

const PRE_SESSION_PATHS = new Set(['/welcome', '/login', '/verify']);
const ALWAYS_ALLOWED_PATHS = new Set(['/accessibility']);

export interface GuardState {
  hasExplicitLocale: boolean;
  authenticated: boolean;
  registered: boolean;
}

/** Returns the path to redirect to, or null if `path` is allowed as-is. */
export function resolveRedirect(state: GuardState, path: string): string | null {
  if (ALWAYS_ALLOWED_PATHS.has(path)) return null;

  if (!state.hasExplicitLocale) {
    return path === '/language' ? null : '/language';
  }

  if (!state.authenticated) {
    if (path === '/language') return '/welcome';
    return PRE_SESSION_PATHS.has(path) ? null : '/welcome';
  }

  if (!state.registered) {
    if (path === '/language' || PRE_SESSION_PATHS.has(path)) return REGISTER_FIRST_STEP;
    return path.startsWith('/register') ? null : REGISTER_FIRST_STEP;
  }

  const isOnboardingPath =
    path === '/language' || PRE_SESSION_PATHS.has(path) || path.startsWith('/register');
  return isOnboardingPath ? HOME_PATH : null;
}
