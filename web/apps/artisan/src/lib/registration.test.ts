// apps/artisan/src/lib/registration.test.ts
import { describe, expect, it } from 'vitest';
import { buildRegisterBody } from './registration';

describe('buildRegisterBody', () => {
  it('sends only display_name and language -- the two fields POST /artisans accepts', () => {
    const body = buildRegisterBody(
      { name: 'Kamla Devi', craftId: 'weaving', districtId: 'varanasi', pehchanId: 'PMV123' },
      'hi',
    );
    expect(body).toEqual({ display_name: 'Kamla Devi', language: 'hi' });
    expect(Object.keys(body)).toEqual(['display_name', 'language']);
  });

  it('trims the name and falls back to an empty string when absent', () => {
    expect(buildRegisterBody({ name: '  Ravi  ' }, 'en').display_name).toBe('Ravi');
    expect(buildRegisterBody({}, 'en').display_name).toBe('');
  });
});
