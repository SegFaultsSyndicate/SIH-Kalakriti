// apps/admin/src/lib/csv.test.ts
import { describe, it, expect } from 'vitest';
import { toCsv, parseCsv } from './csv';

describe('toCsv', () => {
  it('quotes fields containing commas, quotes or newlines', () => {
    const csv = toCsv(['a', 'b'], [['plain', 'has,comma'], ['has"quote', 'line\nbreak']]);
    expect(csv).toBe('a,b\r\nplain,"has,comma"\r\n"has""quote","line\nbreak"');
  });
});

describe('parseCsv', () => {
  it('round-trips a simple sheet', () => {
    const rows = parseCsv('display_name,phone_e164\nAsha Devi,+919876543210\n');
    expect(rows).toEqual([
      ['display_name', 'phone_e164'],
      ['Asha Devi', '+919876543210'],
    ]);
  });

  it('handles quoted fields with embedded commas and escaped quotes', () => {
    const rows = parseCsv('name,note\n"Asha, Devi","She said ""hello"""\n');
    expect(rows).toEqual([
      ['name', 'note'],
      ['Asha, Devi', 'She said "hello"'],
    ]);
  });
});
