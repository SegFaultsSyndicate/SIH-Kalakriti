// packages/voice/src/commands.test.ts
import { describe, expect, it } from 'vitest';
import { matchCommand, hasCommandGrammar } from './commands';

describe('hasCommandGrammar', () => {
  it('reports en and hi as built, everything else as not', () => {
    expect(hasCommandGrammar('en')).toBe(true);
    expect(hasCommandGrammar('hi')).toBe(true);
    expect(hasCommandGrammar('ta')).toBe(false);
  });
});

describe('matchCommand', () => {
  it('matches an English phrase to its command', () => {
    expect(matchCommand('go back', 'en')).toBe('goBack');
    expect(matchCommand('show my listings', 'en')).toBe('myListings');
  });

  it('matches inside a longer sentence', () => {
    expect(matchCommand('okay go back please', 'en')).toBe('goBack');
  });

  it('matches a Hindi phrase to its command', () => {
    expect(matchCommand('वापस जाएँ', 'hi')).toBe('goBack');
  });

  it('is case-insensitive', () => {
    expect(matchCommand('GO BACK', 'en')).toBe('goBack');
  });

  it('returns null for an unrecognized phrase', () => {
    expect(matchCommand('what is the weather', 'en')).toBeNull();
  });

  it('falls back to the English grammar for a locale with none built', () => {
    expect(matchCommand('go back', 'ta')).toBe('goBack');
  });
});
