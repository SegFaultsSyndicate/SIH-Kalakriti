// packages/api/src/errors.ts
//
// Maps a failed request to the i18n key its screen should show. A user must
// never see components.schemas.Error's raw `message` or an HTTP status
// number -- both are for logs, not for the artisan.
//
// The envelope here is {error, message} (flat), matching
// components.schemas.Error in services/bff/openapi.json today. A nested
// {error: {code, message, details}} shape does not exist in the spec --
// isErrorCode below is the seam that keeps this working either way once it
// does, without hand-typing the nested form now.

import type { MessageKey } from '@kalakriti/i18n';
import { ApiError } from './transport';
import type { components } from './generated/schema';

type ErrorCode = components['schemas']['Error']['error'];

const CODE_TO_KEY: Record<ErrorCode, MessageKey> = {
  not_found: 'api.error.not_found',
  conflict: 'api.error.conflict',
  invalid_input: 'api.error.invalid_input',
  forbidden: 'api.error.forbidden',
  unavailable: 'api.error.unavailable',
  internal_error: 'api.error.internal_error',
};

function isErrorCode(value: unknown): value is ErrorCode {
  return typeof value === 'string' && value in CODE_TO_KEY;
}

export function messageKeyFor(error: ApiError): MessageKey {
  if (error.status === 0) return 'api.error.network';

  const body = error.body;
  const code = typeof body === 'object' && body !== null ? (body as { error?: unknown }).error : undefined;

  return isErrorCode(code) ? CODE_TO_KEY[code] : 'api.error.unknown';
}
