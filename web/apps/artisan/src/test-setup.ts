// apps/artisan/src/test-setup.ts
//
// listing-draft.ts and outbox-send.ts open @kalakriti/offline's db at import
// time; jsdom has no IndexedDB, so this polyfill has to be wired in before
// any test file runs -- same reason packages/offline/src/test-setup.ts
// exists.
import 'fake-indexeddb/auto';
