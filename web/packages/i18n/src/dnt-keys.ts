// packages/i18n/src/dnt-keys.ts
//
// Catalogue keys whose value must render identically in every locale --
// proper nouns with no target-language equivalent. Keep this list short: an
// English loanword or acronym *inside* an otherwise-translated sentence
// (hi.ts's "बड़ा कर्सर (Large Cursor)") is not a reason to add a key here --
// catalogue-audit's script/sameness checks already tolerate that. Add a key
// here only when the *whole value* must stay untranslated.
export const DNT_KEYS: ReadonlySet<string> = new Set<string>([
  'app.name', // "Kalakriti" -- the product's brand name
  'a11y.statement.contact.email', // email address
  'profile.email.placeholder', // example email address
  'login.phone.countryCode', // "+91" -- a dialling code, not prose
]);
