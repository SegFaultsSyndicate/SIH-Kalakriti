// packages/voice/vitest.config.ts
//
// screen-reader.ts walks a real DOM (querySelector, closest, HTMLInputElement)
// -- Node has none of that, so this package's tests need jsdom, unlike the
// rest of the workspace which runs plain Node.
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
  },
});
