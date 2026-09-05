// apps/artisan/src/lib/route-guard.test.ts
import { describe, expect, it } from 'vitest';
import { resolveRedirect, type GuardState } from './route-guard';

const NONE: GuardState = { hasExplicitLocale: false, authenticated: false, registered: false };
const LOCALE_ONLY: GuardState = { ...NONE, hasExplicitLocale: true };
const SESSION: GuardState = { ...LOCALE_ONLY, authenticated: true };
const FULL: GuardState = { ...SESSION, registered: true };

describe('resolveRedirect', () => {
  it('sends every path to /language before a language is chosen', () => {
    expect(resolveRedirect(NONE, '/')).toBe('/language');
    expect(resolveRedirect(NONE, '/welcome')).toBe('/language');
    expect(resolveRedirect(NONE, '/language')).toBeNull();
  });

  it('allows /accessibility at any state, including before a language is chosen', () => {
    expect(resolveRedirect(NONE, '/accessibility')).toBeNull();
    expect(resolveRedirect(FULL, '/accessibility')).toBeNull();
  });

  it('confines a locale-only artisan to welcome/login/verify', () => {
    expect(resolveRedirect(LOCALE_ONLY, '/welcome')).toBeNull();
    expect(resolveRedirect(LOCALE_ONLY, '/login')).toBeNull();
    expect(resolveRedirect(LOCALE_ONLY, '/verify')).toBeNull();
    expect(resolveRedirect(LOCALE_ONLY, '/')).toBe('/welcome');
    expect(resolveRedirect(LOCALE_ONLY, '/register/name')).toBe('/welcome');
  });

  it('sends a chosen-language artisan forward past /language itself', () => {
    expect(resolveRedirect(LOCALE_ONLY, '/language')).toBe('/welcome');
  });

  it('confines an authenticated, unregistered artisan to /register/*', () => {
    expect(resolveRedirect(SESSION, '/register/name')).toBeNull();
    expect(resolveRedirect(SESSION, '/register/craft')).toBeNull();
    expect(resolveRedirect(SESSION, '/welcome')).toBe('/register/name');
    expect(resolveRedirect(SESSION, '/')).toBe('/register/name');
  });

  it('sends a fully onboarded artisan away from every onboarding path', () => {
    expect(resolveRedirect(FULL, '/language')).toBe('/');
    expect(resolveRedirect(FULL, '/welcome')).toBe('/');
    expect(resolveRedirect(FULL, '/login')).toBe('/');
    expect(resolveRedirect(FULL, '/register/craft')).toBe('/');
    expect(resolveRedirect(FULL, '/')).toBeNull();
    expect(resolveRedirect(FULL, '/work')).toBeNull();
  });
});
