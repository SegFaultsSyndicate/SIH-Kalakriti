// apps/artisan/src/lib/phone.ts
//
// POST /auth/otp/request and /verify both take `phone` as "E.164 phone
// number" (services/bff/openapi.json) -- a plain string, not a
// {countryCode, number} pair. The +91 the login screen shows is fixed UI
// chrome, not a field the spec has room for, so this is where it gets glued
// back onto the 10 digits the keypad or the ASR transcript collected.

/** Indian mobile numbers are 10 digits, first digit 6-9 (landlines and other ranges are out of scope: this product logs in artisans, not businesses). */
export function isValidIndianMobile(digits: string): boolean {
  return /^[6-9]\d{9}$/.test(digits);
}

export function toE164(digits: string): string {
  return `+91${digits}`;
}

/** Strips everything but digits from an ASR transcript ("nine eight seven..." never happens -- Indic and English speech recognisers transcribe spoken digits as numerals), and caps at 10. */
export function digitsFromTranscript(transcript: string): string {
  return transcript.replace(/\D/g, '').slice(0, 10);
}
