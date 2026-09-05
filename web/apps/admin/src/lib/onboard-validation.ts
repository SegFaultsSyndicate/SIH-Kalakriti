import type { OnboardRow } from './bulk-onboard';

/** A sheet is committable only after every parsed row has passed validation. */
export function canCommitOnboard(rows: OnboardRow[], headerError = ''): boolean {
  return headerError.trim() === '' && rows.length > 0 && rows.every((row) => row.errors.length === 0);
}
