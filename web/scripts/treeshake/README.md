# Tree-shaking proof

Two esbuild entry points, bundled and minified with `--loader:.svg=text`:

- `entry-direct.js` — imports one icon directly: `packages/icons/src/search.svg`
- `entry-barrel.js` — imports `ICON_COMPONENTS` from `packages/icons/icons.js`, the
  name-lookup map `<Icon name="...">` needs, which references all 89 icons

Run:

```bash
npx --yes esbuild scripts/treeshake/entry-direct.js --bundle --minify --loader:.svg=text --outfile=/tmp/out-direct.js
npx --yes esbuild scripts/treeshake/entry-barrel.js --bundle --minify --loader:.svg=text --outfile=/tmp/out-barrel.js
wc -c /tmp/out-direct.js /tmp/out-barrel.js
```

## Result (recorded 2026-09-03)

| Entry | Bytes |
|---|---|
| direct import of one icon | 584 B |
| barrel import (`ICON_COMPONENTS`, all 89 icons) | 69,253 B |

Direct import is **under the 2KB icon budget** — 584 B for the SVG payload plus a
small esbuild wrapper. The barrel entry is a live object reference to every icon
(`ICON_COMPONENTS['search'] = Icon_search`, ...), so a bundler cannot drop any
entry from it — the ~778 B/icon average there is the expected cost of the
convenience path, not a shaking failure.

**Scope of this measurement**: this bundles the raw `.svg` files through esbuild's
text loader, i.e. the ES module import graph and dead-code elimination a real app
bundler performs. It does **not** run Svelte's compiler — `vite-plugin-svelte-svg`
turns each `.svg` into a `.svelte` component before Vite/Rollup ever sees it, and
that compile step is not exercised here. The <2KB claim is verified for the
direct-import path's module-graph cost; it is not a claim about final
Svelte-compiled bundle output.
