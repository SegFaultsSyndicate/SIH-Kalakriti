import { beforeEach, describe, expect, it } from 'vitest';
import { session } from '@kalakriti/api';
import { ESHAAN_BASE_LISTINGS, PAITHANI_LISTING } from './sih-demo-store';
import { getSihListingTitleKey, getSihMyWorks } from './sih-my-works';

function mockEshaanSession(): void {
  const payload = btoa(JSON.stringify({ phone: '+918779279060' }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
  session.establish(`eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.${payload}.mock`);
}

beforeEach(() => {
  session.clear();
  if (typeof localStorage !== 'undefined') {
    localStorage.removeItem('kalakriti.sih.demo.v4');
  }
});

describe('getSihListingTitleKey', () => {
  it('uses the translated catalogue title for each demo listing', () => {
    expect(getSihListingTitleKey('eshaan-1')).toBe('stub.listing.60.title');
    expect(getSihListingTitleKey('eshaan-2')).toBe('stub.listing.62.title');
    expect(getSihListingTitleKey('eshaan-3')).toBe('stub.listing.64.title');
    expect(getSihListingTitleKey('eshaan-paithani')).toBe('stub.listing.90.title');
  });

  it('does not invent a title key for unknown listings', () => {
    expect(getSihListingTitleKey('remote-listing')).toBeUndefined();
    expect(getSihListingTitleKey(undefined)).toBeUndefined();
  });
});

describe('getSihMyWorks', () => {
  it('hides the demo Paithani listing from the artisan work list', () => {
    mockEshaanSession();
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(
        'kalakriti.sih.demo.v4',
        JSON.stringify({
          artisan: {
            id: 'sih-artisan-eshaan',
            name: 'Eshaan',
            craftName: 'Weaving & Looms',
            location: 'Varanasi, Uttar Pradesh',
            pehchanId: 'UP-VNS-2024-0982',
            clusterName: 'Varanasi Silk Weaver Facility Centre',
          },
          listings: [...ESHAAN_BASE_LISTINGS, { ...PAITHANI_LISTING, state: 'PUBLISHED', publishedAt: new Date().toISOString() }],
          orders: [],
          paithaniPublished: true,
        }),
      );
    }

    const works = getSihMyWorks();
    expect(works.some((listing) => listing.id === 'eshaan-paithani')).toBe(false);
    expect(works.map((listing) => listing.id)).toEqual(['eshaan-1', 'eshaan-2', 'eshaan-3']);
  });
});