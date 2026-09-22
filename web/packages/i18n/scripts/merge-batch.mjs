#!/usr/bin/env node
// packages/i18n/scripts/merge-batch.mjs
//
// Merge a translation batch into every catalogue at once:
//
//   node scripts/merge-batch.mjs path/to/batch.json [...more.json]
//
// batch.json is { "<locale>": { "<key>": "<value>" } } for en plus all 20
// other locales. An existing key's line is replaced in place; a new key is
// appended before the catalogue's closing `} as const;`. A new key must be
// present for every locale, or nothing is written -- a key in en.ts that a
// locale lacks is a compile error under `Messages`, so a half-merged batch
// would break every app's type-check.
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const dir = join(dirname(fileURLToPath(import.meta.url)), '../src/messages');
const LOCALES = ['en', 'hi', 'as', 'bn', 'brx', 'doi', 'gu', 'kn', 'ks', 'kok', 'mai', 'ml', 'mr', 'ne', 'or', 'pa', 'sa', 'sd', 'ta', 'te', 'ur'];

const batch = {};
for (const path of process.argv.slice(2)) {
  const part = JSON.parse(readFileSync(path, 'utf8'));
  for (const [loc, entries] of Object.entries(part)) Object.assign((batch[loc] ??= {}), entries);
}

const escapeRe = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const keyLine = (key) => new RegExp(`^  (['"])${escapeRe(key)}\\1: .*$`, 'm');
const sources = Object.fromEntries(LOCALES.map((l) => [l, readFileSync(join(dir, `${l}.ts`), 'utf8')]));

// Every key that is new to en.ts must arrive in every locale.
const errors = [];
for (const key of Object.keys(batch.en ?? {})) {
  if (keyLine(key).test(sources.en)) continue;
  for (const l of LOCALES) if (typeof batch[l]?.[key] !== 'string') errors.push(`${l}: missing new key ${key}`);
}
for (const [l, entries] of Object.entries(batch)) {
  if (!LOCALES.includes(l)) errors.push(`unknown locale ${l}`);
  for (const key of Object.keys(entries)) {
    if (!keyLine(key).test(sources.en) && !(key in (batch.en ?? {}))) errors.push(`${l}: ${key} is in neither en.ts nor the batch's en`);
    // Placeholders must survive translation.
    const want = (batch.en?.[key] ?? '').match(/\{\w+\}/g)?.sort().join() ?? null;
    const got = entries[key].match(/\{\w+\}/g)?.sort().join() ?? '';
    if (want !== null && batch.en?.[key] !== undefined && want !== got) errors.push(`${l}: ${key} placeholders ${got} != ${want}`);
  }
}
if (errors.length) {
  console.error(errors.join('\n'));
  process.exit(1);
}

let added = 0, replaced = 0;
for (const l of LOCALES) {
  const entries = batch[l];
  if (!entries) continue;
  let src = sources[l];
  const single = (src.match(/^  '/gm) ?? []).length > (src.match(/^  "/gm) ?? []).length;
  const q = (s) => (single ? `'${s.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/\n/g, '\\n')}'` : JSON.stringify(s));
  const fresh = [];
  for (const [key, value] of Object.entries(entries)) {
    const line = `  ${q(key)}: ${q(value)},`;
    if (keyLine(key).test(src)) {
      src = src.replace(keyLine(key), () => line);
      replaced++;
    } else {
      fresh.push(line);
      added++;
    }
  }
  if (fresh.length) {
    const end = src.lastIndexOf('\n} as const;');
    if (end < 0) throw new Error(`${l}.ts: no closing "} as const;"`);
    src = src.slice(0, end) + '\n' + fresh.join('\n') + src.slice(end);
  }
  writeFileSync(join(dir, `${l}.ts`), src);
}
console.log(`merged: ${added} added, ${replaced} replaced`);
