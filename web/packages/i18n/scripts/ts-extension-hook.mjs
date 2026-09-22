// packages/i18n/scripts/ts-extension-hook.mjs
//
// This package's TS sources omit extensions on relative imports (matching
// every file in the package, e.g. locale.svelte.ts's `from './locales'`) --
// correct under the bundler moduleResolution every app's tsconfig uses, but
// plain Node ESM requires an explicit extension. audit.mjs registers this
// hook before loading anything, so it never has to choose between "passes
// tsc" and "runs under plain node": try the specifier as given, and if
// Node's resolver can't find it, try it again with .ts appended.
export async function resolve(specifier, context, nextResolve) {
  try {
    return await nextResolve(specifier, context);
  } catch (err) {
    if (err.code === 'ERR_MODULE_NOT_FOUND' && !specifier.endsWith('.ts')) {
      return nextResolve(`${specifier}.ts`, context);
    }
    throw err;
  }
}
