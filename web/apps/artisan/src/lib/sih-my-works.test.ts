import { describe, expect, it } from 'vitest';
import { getSihListingTitleKey } from './sih-my-works';

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