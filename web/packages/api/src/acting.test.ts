import { afterEach, describe, expect, it } from 'vitest';
import { onBehalfHeader, setActingFor } from './acting';

describe('X-On-Behalf-Of', () => {
  afterEach(() => setActingFor(undefined));

  it('is only sent on routes the bff honours it on', () => {
    setActingFor('artisan-1');
    expect(onBehalfHeader('POST', '/listings')).toEqual({ 'X-On-Behalf-Of': 'artisan-1' });
    expect(onBehalfHeader('GET', '/income/summary')).toEqual({ 'X-On-Behalf-Of': 'artisan-1' });
    // The agent's own calls and consent withdrawal never carry it.
    expect(onBehalfHeader('GET', '/assisted/artisans')).toEqual({});
    expect(onBehalfHeader('GET', '/staff/me')).toEqual({});
    expect(onBehalfHeader('DELETE', '/finance/links/abc')).toEqual({});
    expect(onBehalfHeader('DELETE', '/helpers/abc')).toEqual({});
  });

  it('prefers a frozen target over the live one', () => {
    setActingFor('artisan-live');
    expect(onBehalfHeader('POST', '/income/sales', 'artisan-frozen')).toEqual({ 'X-On-Behalf-Of': 'artisan-frozen' });
    expect(onBehalfHeader('POST', '/income/sales', null)).toEqual({});
  });

  it('sends nothing when nobody is being helped', () => {
    expect(onBehalfHeader('POST', '/listings')).toEqual({});
  });
});
