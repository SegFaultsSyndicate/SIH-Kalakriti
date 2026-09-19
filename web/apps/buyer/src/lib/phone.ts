// apps/buyer/src/lib/phone.ts
//
// POST /auth/otp/request and /verify both take `phone` as "E.164 phone
// number" (services/bff/openapi.json) -- a plain string, not a
// {countryCode, number} pair. See apps/artisan/src/lib/phone.ts for the
// same helpers; kept per-app rather than shared since neither app imports
// across the other's $lib.

/** Indian mobile numbers are 10 digits, first digit 6-9. */
export function isValidIndianMobile(digits: string): boolean {
  return /^[6-9]\d{9}$/.test(digits);
}

export function toE164(digits: string): string {
  return `+91${digits}`;
}
