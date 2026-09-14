#!/usr/bin/env node
import { register } from 'node:module';
register('./ts-extension-hook.mjs', import.meta.url);
await import('./generate-catalogue.mjs');
