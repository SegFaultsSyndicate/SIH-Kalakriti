// apps/artisan/src/lib/registration.test.ts
import { describe, expect, it } from 'vitest';
import { buildRegisterBody } from './registration';

describe('buildRegisterBody', () => {
  it('sends the chosen craft, region and language -- everything core-svc requires to persist', () => {
    const body = buildRegisterBody(
      { name: 'Kamla Devi', craftId: 'craft-uuid-1', districtId: 'varanasi', pehchanId: 'PMV123' },
      'hi',
    );
    expect(body).toEqual({
      display_name: 'Kamla Devi',
      craft_ids: ['craft-uuid-1'],
      languages: ['HINDI'],
      region: { state_code: 'IN-UP', district: 'Varanasi' },
    });
  });

  it('uses the state picked alongside a free-text district', () => {
    const body = buildRegisterBody(
      {
        name: 'Ravi',
        craftId: 'craft-uuid-2',
        districtFreeText: 'Some village',
        districtStateCode: 'IN-MH',
      },
      'hi',
    );
    expect(body.region).toEqual({ state_code: 'IN-MH', district: 'Some village' });
  });

  it('drops an unrecognised language code rather than sending it broken', () => {
    // Defensive only: the wizard only ever passes locale.code, which is
    // always a real @kalakriti/i18n locale, so this path is unreachable
    // through the UI -- see buildRegisterBody's doc comment.
    const body = buildRegisterBody({ name: 'Ravi', craftId: 'craft-uuid-2' }, 'zz');
    expect(body.languages).toEqual([]);
  });

  it('trims the name and falls back to an empty string when absent', () => {
    expect(buildRegisterBody({ name: '  Ravi  ' }, 'en').display_name).toBe('Ravi');
    expect(buildRegisterBody({}, 'en').display_name).toBe('');
  });

  it('sends no craft_ids when the draft has none', () => {
    expect(buildRegisterBody({}, 'en').craft_ids).toEqual([]);
  });
});
