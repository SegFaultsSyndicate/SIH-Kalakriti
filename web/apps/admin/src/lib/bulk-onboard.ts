// apps/admin/src/lib/bulk-onboard.ts
//
// Parses and validates a bulk artisan-onboarding CSV before anything is
// committed to the backend -- POST /clusters/{id}/onboard is called once per
// valid row only after the whole sheet has been checked and shown to the
// officer, per the acceptance criterion ("CSV upload shows validation errors
// before any commit").

import { parseCsv } from './csv';

export const REQUIRED_HEADERS = ['display_name', 'phone_e164', 'craft_ids', 'languages', 'state_code'] as const;
export const OPTIONAL_HEADERS = ['district'] as const;

export interface OnboardRow {
  rowNumber: number;
  display_name: string;
  phone_e164: string;
  craft_ids: string[];
  languages: string[];
  state_code: string;
  district?: string;
  errors: string[];
}

const PHONE_RE = /^\+\d{10,15}$/;

function splitList(value: string): string[] {
  return value
    .split(/[;|]/)
    .map((v) => v.trim())
    .filter((v) => v.length > 0);
}

export interface ParsedSheet {
  rows: OnboardRow[];
  headerError?: string;
}

/** Parses raw CSV text into validated rows. Nothing here calls the network. */
export function parseOnboardSheet(text: string): ParsedSheet {
  const table = parseCsv(text);
  if (table.length === 0) {
    return { rows: [], headerError: 'The file is empty.' };
  }

  const header = table[0].map((h) => h.trim().toLowerCase());
  const missing = REQUIRED_HEADERS.filter((h) => !header.includes(h));
  if (missing.length > 0) {
    return { rows: [], headerError: `missing column(s): ${missing.join(', ')}` };
  }

  const colIndex = (name: string) => header.indexOf(name);
  const rows: OnboardRow[] = [];

  for (let i = 1; i < table.length; i++) {
    const cells = table[i];
    const get = (name: string) => (cells[colIndex(name)] ?? '').trim();

    const displayName = get('display_name');
    const phone = get('phone_e164');
    const craftIds = splitList(get('craft_ids'));
    const languages = splitList(get('languages'));
    const stateCode = get('state_code');
    const district = colIndex('district') >= 0 ? get('district') : undefined;

    const errors: string[] = [];
    if (displayName === '') errors.push('display_name is required');
    if (phone === '') errors.push('phone_e164 is required');
    else if (!PHONE_RE.test(phone)) errors.push('phone_e164 must be E.164, e.g. +919876543210');
    if (craftIds.length === 0) errors.push('craft_ids: at least one, separated by ; or |');
    if (languages.length === 0) errors.push('languages: at least one, separated by ; or |');
    if (stateCode === '') errors.push('state_code is required');

    rows.push({
      rowNumber: i + 1,
      display_name: displayName,
      phone_e164: phone,
      craft_ids: craftIds,
      languages,
      state_code: stateCode,
      district: district || undefined,
      errors,
    });
  }

  return { rows };
}
