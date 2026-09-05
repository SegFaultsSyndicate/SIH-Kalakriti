// web/scripts/vite/size-report.js
//
// Bundle-size reporter.
//
// The number that matters for the artisan app is not "total JS emitted", it is
// "JS the browser must download and execute before the first screen is
// usable". That is the entry chunk plus everything it statically imports,
// transitively -- a dynamically imported route chunk is not part of it, and
// counting it would make the budget both wrong and un-actionable.
//
// Sizes are gzip, because the Go BFF serves these assets compressed and the
// raw byte count is not what goes over a 2G link.

import { gzipSync } from 'node:zlib';
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';

function gzipBytes(source) {
  return gzipSync(typeof source === 'string' ? Buffer.from(source, 'utf8') : source, {
    level: 9,
  }).length;
}

function formatKb(bytes) {
  return `${(bytes / 1024).toFixed(1)} KB`;
}

/**
 * Entry chunk plus its transitive static imports: the critical path.
 * Dynamic imports are excluded by construction -- rollup keeps them in
 * `dynamicImports`, which this never walks.
 *
 * SvelteKit's client build is multi-entry: every route's `+page`/`+layout`
 * node is ALSO `isEntry: true`, because that is how it gets a stable,
 * independently code-split chunk -- not because the browser loads all of
 * them up front. At runtime the router reaches every one of them through a
 * genuine `() => import(...)` (verified against the built output), so
 * treating every node as a root here would count every route in the app as
 * part of the first screen, growing without bound as routes are added
 * regardless of whether anyone ever navigates to them. Only
 * `_app/immutable/entry/*` (the two scripts the built `index.html` actually
 * references) are roots; `_app/immutable/nodes/*` are excluded from the
 * root set on purpose, though a node CAN still end up "critical" for a
 * legitimate reason -- being statically imported from one of those two
 * entries, or from another critical chunk -- since the walk below still
 * follows real static imports whichever chunk they originate from.
 *
 * This still slightly UNDER-counts in absolute terms: the root layout node
 * and the matched page node for whichever URL is opened first are also
 * genuinely downloaded before first paint (via their own dynamic import,
 * chained immediately by the router), and this walk does not add them back
 * in because "which page is opened first" is a question this generic,
 * three-app script has no way to answer. In this app that gap is on the
 * order of ~2KB gzip, not the tens of KB the unbounded over-count this
 * replaces was capable of -- accept the small known gap rather than guess
 * at a specific route.
 */
function criticalChunks(bundle) {
  const entries = Object.values(bundle).filter(
    (chunk) => chunk.type === 'chunk' && chunk.isEntry && !chunk.fileName.includes('/nodes/'),
  );
  const seen = new Set();
  const walk = (fileName) => {
    if (seen.has(fileName)) return;
    const chunk = bundle[fileName];
    if (!chunk || chunk.type !== 'chunk') return;
    seen.add(fileName);
    for (const imported of chunk.imports) walk(imported);
  };
  for (const entry of entries) walk(entry.fileName);
  return [...seen];
}

/**
 * @param {object} options
 * @param {string} options.app            label used in the printed report
 * @param {number} [options.budgetKb]     gzip budget for the critical JS path
 * @param {boolean} [options.enforce]     fail the build when over budget
 * @param {string} [options.outFile]      where to write the machine-readable report
 * @returns {import('vite').Plugin}
 */
export function sizeReport(options) {
  const { app, budgetKb, enforce = false, outFile = 'size-report.json' } = options;

  // SvelteKit runs two builds. The server pass exists only so the adapter can
  // render the fallback; its chunks never reach a browser, and letting it
  // report would overwrite the client numbers with figures for code nobody
  // downloads.
  let isServerBuild = false;

  return {
    name: 'kalakriti:size-report',
    apply: /** @type {const} */ ('build'),

    configResolved(config) {
      isServerBuild = Boolean(config.build.ssr);
    },

    writeBundle(_outputOptions, bundle) {
      if (isServerBuild) return;

      const critical = new Set(criticalChunks(bundle));

      const js = [];
      const css = [];
      const other = [];

      for (const [fileName, output] of Object.entries(bundle)) {
        const source = output.type === 'chunk' ? output.code : output.source;
        if (source === undefined || source === null) continue;
        const gzip = gzipBytes(source);
        const record = { fileName, gzip, critical: critical.has(fileName) };
        if (fileName.endsWith('.js')) js.push(record);
        else if (fileName.endsWith('.css')) css.push(record);
        else other.push({ ...record, critical: false });
      }

      const criticalJs = js.filter((f) => f.critical);
      const criticalBytes = criticalJs.reduce((sum, f) => sum + f.gzip, 0);
      const totalJsBytes = js.reduce((sum, f) => sum + f.gzip, 0);
      const totalCssBytes = css.reduce((sum, f) => sum + f.gzip, 0);

      const report = {
        app,
        generatedAt: new Date().toISOString(),
        budgetKb: budgetKb ?? null,
        criticalJsGzip: criticalBytes,
        totalJsGzip: totalJsBytes,
        totalCssGzip: totalCssBytes,
        chunks: [...js, ...css].sort((a, b) => b.gzip - a.gzip),
      };

      writeFileSync(join(process.cwd(), outFile), `${JSON.stringify(report, null, 2)}\n`);

      const lines = [
        '',
        `  ${app} — bundle size (gzip)`,
        `  ${'-'.repeat(58)}`,
        `  initial JS (entry + static imports)   ${formatKb(criticalBytes).padStart(10)}`,
        `  all JS (including lazy routes)        ${formatKb(totalJsBytes).padStart(10)}`,
        `  all CSS                               ${formatKb(totalCssBytes).padStart(10)}`,
      ];

      const biggest = report.chunks.slice(0, 5);
      if (biggest.length > 0) {
        lines.push(`  ${'-'.repeat(58)}`);
        for (const chunk of biggest) {
          const mark = chunk.critical ? '*' : ' ';
          lines.push(
            `  ${mark} ${chunk.fileName.slice(-44).padEnd(44)}${formatKb(chunk.gzip).padStart(10)}`,
          );
        }
        lines.push('  * counted against the initial payload');
      }

      if (budgetKb !== undefined) {
        const budgetBytes = budgetKb * 1024;
        const used = ((criticalBytes / budgetBytes) * 100).toFixed(0);
        const over = criticalBytes > budgetBytes;
        lines.push(`  ${'-'.repeat(58)}`);
        lines.push(
          `  budget ${String(budgetKb).padStart(4)} KB   used ${used.padStart(3)}%   ` +
            (over ? 'OVER BUDGET' : 'within budget'),
        );
        if (over && enforce) {
          lines.push('');
          console.error(lines.join('\n'));
          throw new Error(
            `${app}: initial JS is ${formatKb(criticalBytes)} gzip, over the ${budgetKb} KB budget.`,
          );
        }
      }

      lines.push('');
      console.log(lines.join('\n'));
    },
  };
}
