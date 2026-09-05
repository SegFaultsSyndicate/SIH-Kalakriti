// web/eslint.config.js
//
// Flat config, shared by every app and package. Deliberately small: the
// TypeScript compiler already rejects most of what a heavyweight rule set
// would catch, and svelte-check covers the component side. Rules here are
// the ones a type-checker cannot see.
import js from '@eslint/js';
import ts from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import svelteConfig from './apps/artisan/svelte.config.js';

export default ts.config(
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    languageOptions: {
      globals: { ...globals.browser, ...globals.node },
    },
    rules: {
      // The project bans `any` outright and non-null assertions without a
      // comment; the compiler cannot enforce either, so they live here.
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-non-null-assertion': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      // Core no-undef cannot see type-only lib.dom identifiers that have no
      // runtime global counterpart (BlobPart, RequestInit, ...) and flags
      // them as undefined -- exactly the false positive typescript-eslint's
      // own docs say to turn this rule off for. TypeScript strict mode is
      // the actual authority on an unresolved identifier here.
      'no-undef': 'off',
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts'],
    languageOptions: {
      parserOptions: { parser: ts.parser, svelteConfig },
    },
    rules: {
      // Svelte 5 runes only. These four catch the Svelte 4 idioms the
      // project bans; a plain type-check accepts all of them.
      'svelte/no-svelte-internal': 'error',
      // ignoreWarnings: this app never compiles a custom element, so the
      // compiler's "can't infer custom-element props from a ...rest" note
      // -- an informational warning, not a compile error -- is not a real
      // problem here. A genuine compile ERROR (the Svelte-4-syntax ban this
      // rule exists for) is a different diagnostic level and still fails.
      'svelte/valid-compile': ['error', { ignoreWarnings: true }],
    },
  },
  {
    ignores: [
      'node_modules/',
      '**/.svelte-kit/',
      '**/build/',
      '**/dist/',
      'docs/',
      'scripts/',
      'assets-temp/',
      // svelte-eslint-parser on the pinned eslint-plugin-svelte@2.x line
      // does not know the <svelte:boundary> element (added in Svelte
      // 5.3) and fails to parse it at all -- not a lint finding, a parser
      // gap. svelte-check (which uses the real Svelte compiler, not this
      // parser) still type-checks the file. Drop this once
      // eslint-plugin-svelte is upgraded past the 2.x line.
      'packages/observability/src/ErrorBoundary.svelte',
    ],
  },
);
