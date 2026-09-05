// packages/voice/src/commands.ts
//
// Voice navigation: matching a transcript to a command, per locale. Only
// what's actually built maps to a grammar (en, hi) -- the same coverage
// split as @kalakriti/i18n's message catalogues, for the same reason. This
// module only recognizes a command; it does not act on one. Mapping a
// command to a route is app-specific (the artisan app's "my listings" route
// isn't necessarily the buyer app's), so the caller supplies that mapping.

import type { LocaleCode } from '@kalakriti/i18n';

export type VoiceCommand = 'goBack' | 'myListings' | 'newListing' | 'help';

const GRAMMAR: Partial<Record<LocaleCode, Record<VoiceCommand, readonly string[]>>> = {
  en: {
    goBack: ['go back', 'back'],
    myListings: ['my listings', 'my work'],
    newListing: ['new listing', 'add new listing', 'new photo'],
    help: ['help'],
  },
  hi: {
    goBack: ['वापस जाएँ', 'वापस'],
    myListings: ['मेरा काम', 'मेरी लिस्टिंग'],
    newListing: ['नया सामान', 'नई लिस्टिंग'],
    help: ['मदद'],
  },
};

/** True if `code` has a voice-navigation grammar built. */
export function hasCommandGrammar(code: LocaleCode): boolean {
  return code in GRAMMAR;
}

/**
 * Matches a transcript against `code`'s grammar (falling back to English's,
 * since English phrases are a reasonable bet even in an untranslated UI).
 * `includes` rather than exact match: "okay go back please" should still
 * match "go back".
 */
export function matchCommand(transcript: string, code: LocaleCode): VoiceCommand | null {
  const grammar = GRAMMAR[code] ?? GRAMMAR.en!;
  const normalized = transcript.trim().toLowerCase();
  for (const [command, phrases] of Object.entries(grammar) as [VoiceCommand, string[]][]) {
    if (phrases.some((phrase) => normalized.includes(phrase.toLowerCase()))) {
      return command;
    }
  }
  return null;
}
