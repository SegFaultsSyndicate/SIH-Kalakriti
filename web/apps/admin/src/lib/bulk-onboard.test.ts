// apps/admin/src/lib/bulk-onboard.test.ts
import { describe, it, expect } from 'vitest';
import { parseOnboardSheet } from './bulk-onboard';

const HEADER = 'display_name,phone_e164,craft_ids,languages,state_code,district';

describe('parseOnboardSheet', () => {
  it('flags a missing required column', () => {
    const { rows, headerError } = parseOnboardSheet('display_name,phone_e164\nAsha,+919876543210\n');
    expect(rows).toEqual([]);
    expect(headerError).toMatch(/craft_ids/);
  });

  it('accepts a fully valid row', () => {
    const { rows } = parseOnboardSheet(
      `${HEADER}\nAsha Devi,+919876543210,craft-1;craft-2,ENGLISH;HINDI,IN-GJ,Kutch\n`,
    );
    expect(rows).toHaveLength(1);
    expect(rows[0].errors).toEqual([]);
    expect(rows[0].craft_ids).toEqual(['craft-1', 'craft-2']);
    expect(rows[0].languages).toEqual(['ENGLISH', 'HINDI']);
  });

  it('collects every validation error on a bad row, without touching the network', () => {
    const { rows } = parseOnboardSheet(`${HEADER}\n,not-e164,,,,\n`);
    expect(rows).toHaveLength(1);
    expect(rows[0].errors).toContain('display_name is required');
    expect(rows[0].errors).toContain('phone_e164 must be E.164, e.g. +919876543210');
    expect(rows[0].errors).toContain('craft_ids: at least one, separated by ; or |');
    expect(rows[0].errors).toContain('languages: at least one, separated by ; or |');
    expect(rows[0].errors).toContain('state_code is required');
  });
});
