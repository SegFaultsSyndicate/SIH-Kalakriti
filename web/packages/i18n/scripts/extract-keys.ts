// Extract all keys and values from en.ts as JSON
// Run: node --experimental-strip-types extract-keys.ts

import { en } from '../src/messages/en.ts';

const keys = Object.keys(en);
console.log(JSON.stringify({ keyCount: keys.length, keys: en }, null, 2));
