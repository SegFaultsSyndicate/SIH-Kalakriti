// packages/api/src/jwt.ts
//
// Decodes a JWT's claims without verifying the signature -- verification is
// the BFF's job; the client only needs to read what it was handed. The claim
// names themselves are not documented anywhere in
// services/bff/openapi.json (bearerFormat: JWT is all it says), so this
// returns the raw claim bag instead of guessing field names like `role`.

export function decodeJwtClaims(token: string): Record<string, unknown> | null {
  const payload = token.split('.')[1];
  if (payload === undefined) return null;

  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const json = atob(base64);
    const claims: unknown = JSON.parse(json);
    return typeof claims === 'object' && claims !== null ? (claims as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}
