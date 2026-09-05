// packages/api/src/errors.test.ts
import { describe, expect, it } from 'vitest';
import { messageKeyFor } from './errors';
import { ApiError } from './transport';

describe('messageKeyFor', () => {
  it('maps every known error envelope code to its i18n key', () => {
    const cases: [string, string][] = [
      ['not_found', 'api.error.not_found'],
      ['conflict', 'api.error.conflict'],
      ['invalid_input', 'api.error.invalid_input'],
      ['forbidden', 'api.error.forbidden'],
      ['unavailable', 'api.error.unavailable'],
      ['internal_error', 'api.error.internal_error'],
    ];
    for (const [code, key] of cases) {
      const error = new ApiError(400, { error: code, message: 'raw backend text' });
      expect(messageKeyFor(error)).toBe(key);
    }
  });

  it('maps a transport failure (status 0) to the network key, not a code lookup', () => {
    const error = new ApiError(0, null, 'Failed to fetch');
    expect(messageKeyFor(error)).toBe('api.error.network');
  });

  it('falls back to the unknown key for an unrecognised or malformed body', () => {
    expect(messageKeyFor(new ApiError(500, { error: 'something_new' }))).toBe('api.error.unknown');
    expect(messageKeyFor(new ApiError(500, null))).toBe('api.error.unknown');
    expect(messageKeyFor(new ApiError(500, 'plain text body'))).toBe('api.error.unknown');
  });
});
