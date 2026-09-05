// packages/api/src/auth-flow.ts
//
// verifyOtp on its own is a pure typed fetch -- it hands back
// {access_token, refresh_token} and does nothing else, same as every other
// function in operations.ts. Every OTP flow needs the same next two steps
// (store the token, establish the session), so this composes them instead of
// leaving every call site to remember both.

import { verifyOtp } from './operations';
import type { CallOptions } from './retry';
import { setAccessToken } from './auth';
import { session } from './session.svelte';
import { setRefreshToken } from './auth';

/** Verifies the OTP, stores the access token, and establishes the session. Returns whether a token came back. */
export async function completeOtpVerification(
  input: Parameters<typeof verifyOtp>[0],
  options?: CallOptions,
): Promise<boolean> {
  const response = await verifyOtp(input, options);
  if (response.access_token === undefined) return false;

  setAccessToken(response.access_token);
  setRefreshToken(response.refresh_token);
  session.establish(response.access_token);
  return true;
}
