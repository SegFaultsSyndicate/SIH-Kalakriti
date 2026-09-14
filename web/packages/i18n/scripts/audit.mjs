#!/usr/bin/env node
// packages/i18n/scripts/audit.mjs
//
// Bootstrap: register the extensionless-import resolve hook (see
// ts-extension-hook.mjs), then load the real CLI. No build step -- Node 24
// strips TS types natively, so this runs the .ts sources directly.

import { register } from 'node:module';

register('./ts-extension-hook.mjs', import.meta.url);

await import('./audit-main.mjs');
