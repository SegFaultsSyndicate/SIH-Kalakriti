<!--
  packages/observability/src/ErrorBoundary.svelte

  The global error boundary for a whole app: wrap `{@render children()}` in
  the root +layout.svelte with this. Any render or effect error thrown
  beneath it is caught by Svelte 5's <svelte:boundary>, reported, and
  replaced with a designed recovery screen instead of a blank tab.

  `dsn` is passed in from the app (PUBLIC_SENTRY_DSN via $env/static/public)
  rather than read here, so this package stays free of a SvelteKit
  dependency and stays usable from a plain Vite app too.
-->
<script lang="ts">
  import { reportError } from './report';

  interface Props {
    source: string;
    dsn?: string;
    title: string;
    body: string;
    retryLabel: string;
    children: import('svelte').Snippet;
  }

  let { source, dsn, title, body, retryLabel, children }: Props = $props();
</script>

<svelte:boundary
  onerror={(error) => reportError(error, { source }, dsn)}
>
  {@render children()}

  {#snippet failed(error, reset)}
    <div class="k-error-boundary" role="alert">
      <h1>{title}</h1>
      <p>{body}</p>
      {#if import.meta.env.DEV && error}
        <details class="k-error-boundary__details">
          <summary>Development Error Details</summary>
          <pre>{error instanceof Error ? (error.stack || error.message) : String(error)}</pre>
        </details>
      {/if}
      <button type="button" onclick={reset}>{retryLabel}</button>
    </div>
  {/snippet}
</svelte:boundary>

<style>
  .k-error-boundary {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--k-space-3);
    padding: var(--k-space-6) var(--k-gutter);
    text-align: start;
  }

  .k-error-boundary__details {
    width: 100%;
    margin-block: var(--k-space-2);
    padding: var(--k-space-3);
    background: #fdf2f2;
    border: 1px solid #f87171;
    border-radius: var(--k-radius-sm);
    color: #991b1b;
    font-size: 0.85rem;
  }

  .k-error-boundary__details summary {
    font-weight: 600;
    cursor: pointer;
    margin-bottom: var(--k-space-2);
  }

  .k-error-boundary__details pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
    font-family: monospace;
    font-size: 0.8rem;
  }

  .k-error-boundary button {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: var(--k-radius-sm);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
  }
</style>
