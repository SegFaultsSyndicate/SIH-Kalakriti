import { describe, expect, it } from 'vitest';
import { parseOnboardSheet } from './bulk-onboard';
import { canCommitOnboard } from './onboard-validation';

describe('canCommitOnboard', () => {
  it('blocks a sheet when any row is invalid, even if another row is valid', () => {
    const { rows } = parseOnboardSheet(
      'display_name,phone_e164,craft_ids,languages,state_code\n' +
        'Asha,+919876543210,craft-1,HINDI,IN-GJ\n' +
        ',not-phone,,,',
    );
    expect(canCommitOnboard(rows)).toBe(false);
  });

  it('allows a completely valid sheet and rejects header errors', () => {
    const { rows } = parseOnboardSheet(
      'display_name,phone_e164,craft_ids,languages,state_code\nAsha,+919876543210,craft-1,HINDI,IN-GJ',
    );
    expect(canCommitOnboard(rows)).toBe(true);
    expect(canCommitOnboard([], 'missing column(s): craft_ids')).toBe(false);
  });
});
