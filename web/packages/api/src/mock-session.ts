// packages/api/src/mock-session.ts
//
// Dev-only, VITE_USE_MOCKS-gated stand-in for a real OTP verify when there is
// no backend to hit at all. Earlier code forged this same shape of token and
// called session.establish unconditionally on any verify failure (removed --
// see git history on the login/verify pages) -- that was broken because it
// looked like a real login and then 401'd on the first real API call once a
// backend *was* up. This is the same trick, but only ever reached behind the
// explicit VITE_USE_MOCKS flag, so it never masks a real backend problem: a
// mistaken login still fails loudly the moment VITE_USE_MOCKS is off.
//
// decodeJwtClaims (jwt.ts) never checks the signature, so the third segment
// is just a filler string, never sent anywhere real.

import { setAccessToken, setRefreshToken } from './auth';
import { session } from './session.svelte';

function base64url(json: unknown): string {
  return btoa(JSON.stringify(json)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/** Fakes a signed-in session locally, for UI/nav work with no backend running. Never sent to a real API call's benefit. */
export function establishMockSession(role: string, phone: string): void {
  const header = base64url({ alg: 'none', typ: 'JWT' });
  // Mirror core-svc's VerifyOtp: an ARTISAN token only carries `sub` once a
  // profile exists, and the artisan app's route guard reads a `sub` as
  // "already registered". A fake sub here skipped the whole /register/*
  // wizard straight to home. Mock registration sets its own local artisan id
  // (outbox-send.ts), so the guard still lets a registered mock user through.
  const payload = base64url({
    ...(role === 'ARTISAN' ? {} : { sub: `mock-${phone}` }),
    role,
    phone,
    exp: Math.floor(Date.now() / 1000) + 60 * 60 * 24,
  });
  const token = `${header}.${payload}.mock`;

  setAccessToken(token);
  setRefreshToken('mock-refresh-token');
  session.establish(token);
}
