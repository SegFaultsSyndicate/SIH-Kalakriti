// packages/api/src/jwt.test.ts
import { describe, expect, it } from 'vitest';
import { decodeJwtClaims } from './jwt';

function fakeJwt(claims: object): string {
  const b64url = (s: string) => btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  const header = b64url(JSON.stringify({ alg: 'RS256', typ: 'JWT' }));
  const body = b64url(JSON.stringify(claims));
  return `${header}.${body}.signature`;
}

describe('decodeJwtClaims', () => {
  it('reads the claim bag without needing to know field names', () => {
    expect(decodeJwtClaims(fakeJwt({ sub: 'artisan-1', role: 'artisan' }))).toEqual({
      sub: 'artisan-1',
      role: 'artisan',
    });
  });

  it('returns null for a malformed token rather than throwing', () => {
    expect(decodeJwtClaims('not-a-jwt')).toBeNull();
    expect(decodeJwtClaims('a.b.c')).toBeNull();
  });
});
