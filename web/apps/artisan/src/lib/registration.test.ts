// apps/artisan/src/lib/registration.test.ts
import { describe, expect, it } from 'vitest';
import { buildRegisterBody } from './registration';

describe('buildRegisterBody', () => {
  // POST /artisans actually requires display_name, a non-empty craft_ids
  // (real craft ontology UUIDs, not slugs -- callers get those from
  // loadCrafts(), this pure function just carries whatever craftId it's
  // given), and region.state_code (ISO 3166-2:IN). See CLAUDE.md's
  // "Artisan registration's real request contract" section for why this
  // isn't just { display_name, language }.
  it('sends the real POST /artisans shape: craft_ids, languages, and region.state_code', () => {
    const body = buildRegisterBody(
      { name: 'Kamla Devi', craftId: 'weaving', districtId: 'varanasi', pehchanId: 'PMV123' },
      'hi',
    );
    expect(body).toEqual({
      display_name: 'Kamla Devi',
      craft_ids: ['weaving'],
      languages: ['HINDI'],
      region: { state_code: 'IN-UP', district: 'Varanasi' },
      pehchan_id: 'PMV123',
    });
  });

  it('trims the name and falls back to an empty string when absent', () => {
    expect(buildRegisterBody({ name: '  Ravi  ' }, 'en').display_name).toBe('Ravi');
    expect(buildRegisterBody({}, 'en').display_name).toBe('');
  });
});
