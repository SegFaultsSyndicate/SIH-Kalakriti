// packages/i18n/src/dnt-keys.ts
//
// Catalogue keys whose value must render identically in every locale --
// proper nouns with no target-language equivalent. Keep this list short: an
// English loanword or acronym *inside* an otherwise-translated sentence
// (hi.ts's "बड़ा कर्सर (Large Cursor)") is not a reason to add a key here --
// catalogue-audit's script/sameness checks already tolerate that. Add a key
// here only when the *whole value* must stay untranslated.
export const DNT_KEYS: ReadonlySet<string> = new Set<string>([
  'app.name', // "Kalakriti" brand name -- localized per script, but exempted from audit same-as-hi checks across Devanagari locales
  'a11y.statement.contact.email', // email address
  'profile.email.placeholder', // example email address
  'login.phone.countryCode', // "+91" -- a dialling code, not prose
]);

// Proper nouns (people, named crafts) that a Devanagari-script locale
// legitimately spells exactly as Hindi does. Exempt from the same-as-hi
// check ONLY -- they still have to be in the locale's script. Without this,
// earlier batches padded names with "(मराठी)"/"जी"/zero-width spaces just
// to look different from Hindi, and that padding shipped to the UI.
export const PROPER_NOUN_KEY_PATTERNS: readonly RegExp[] = [
  /^stub\.(artisanName|craftName)\./,
  /^giTagged\.(artisan|productCraft)\./,
  /^sellerShowcase\.testimonial\.\d+\.name$/,
  /\.(artisanName|craftName|personName)$/,
];
