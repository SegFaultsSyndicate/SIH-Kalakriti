// apps/buyer/src/lib/craft-icon.test.ts
import { describe, expect, it } from 'vitest';
import { craftIcon } from './craft-icon';

describe('craftIcon', () => {
  it('matches a slug that already names an icon family', () => {
    expect(craftIcon('ajrakh-block-printing')).toBe('block-printing');
  });

  it('matches via a craft-specific synonym not itself an icon name', () => {
    expect(craftIcon('banarasi-handloom-weaving')).toBe('weaving');
    expect(craftIcon('blue-pottery-jaipur')).toBe('pottery');
    expect(craftIcon('madhubani-painting')).toBe('painting');
  });

  it('falls back to the generic craft glyph for an unrecognised slug', () => {
    expect(craftIcon('something-entirely-new')).toBe('handmade-certified');
  });
});
