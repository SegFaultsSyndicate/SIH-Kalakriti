<script lang="ts">
  import Logomark from './src/logomark-solid.svg';
  import Wordmark from './src/wordmark-horizontal.svg';
  import Crest from './src/heritage-crest-fine.svg';
  import Seal from './src/seal.svg';

  interface Props {
    mode?: 'header' | 'gallery';
    title?: string;
    class?: string;
  }

  let { mode = 'gallery', title = 'Kalakriti identity', class: className = '' }: Props = $props();
</script>

<section class="identity-showcase identity-showcase--{mode} {className}" aria-labelledby="identity-showcase-title">
  <div class="identity-showcase__crest" aria-hidden="true">
    <Crest />
  </div>
  <div class="identity-showcase__mark" aria-hidden="true">
    <Logomark />
  </div>
  <div class="identity-showcase__wordmark">
    <Wordmark title={title} />
  </div>
  {#if mode === 'gallery'}
    <div class="identity-showcase__seal">
      <Seal title="Kalakriti provenance seal" />
    </div>
    <h2 id="identity-showcase-title">{title}</h2>
  {:else}
    <span id="identity-showcase-title" class="visually-hidden">{title}</span>
  {/if}
</section>

<style>
  .identity-showcase {
    position: relative;
    display: grid;
    align-items: center;
    gap: var(--k-space-4);
    color: var(--k-text-primary);
  }

  .identity-showcase--header {
    grid-template-columns: auto minmax(0, 1fr);
    max-inline-size: 16rem;
  }

  .identity-showcase--gallery {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    padding: var(--k-space-6);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background: var(--k-surface-raised);
  }

  .identity-showcase__crest {
    position: absolute;
    inset: 0;
    color: var(--k-text-secondary);
    opacity: 0.08;
    pointer-events: none;
  }

  .identity-showcase__crest :global(svg) {
    inline-size: 100%;
    block-size: 100%;
  }

  .identity-showcase__mark,
  .identity-showcase__seal {
    inline-size: 4rem;
    block-size: 4rem;
  }

  .identity-showcase__mark :global(svg),
  .identity-showcase__seal :global(svg),
  .identity-showcase__wordmark :global(svg) {
    inline-size: 100%;
    block-size: 100%;
  }

  .identity-showcase h2 {
    grid-column: 1 / -1;
    font-size: var(--k-text-sm);
  }
</style>
