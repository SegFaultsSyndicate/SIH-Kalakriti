<!--
  apps/artisan/src/routes/language/+page.svelte

  The very first screen, per the batch spec's do-not: no English gate in
  front of it. Every label on this page is either a language's own endonym
  (LOCALES' `endonym`, never translated -- it IS the content) or an icon; the
  one caption ("Choose your language") is rendered in a neutral mixed script
  so an artisan can recognise SOME of it regardless of which language they
  end up wanting, rather than in whichever locale happened to resolve from
  the browser.

  Tapping a tile both chooses the language and moves on -- there is no
  separate "confirm" step, since the artisan can always come back and change
  it from the accessibility control once past this screen. The small speaker
  icon inside each tile previews the name aloud without selecting; it stops
  propagation so a mis-aimed tap on it does not also navigate.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, LOCALES, LOCALE_CODES, type LocaleCode } from '@kalakriti/i18n';
  import { SpeakButton } from '@kalakriti/ui';

  async function choose(code: LocaleCode): Promise<void> {
    await locale.set(code, { persist: true });
    await goto('/welcome');
  }
</script>

<svelte:head>
  <title>Kalakriti — भाषा चुनें / Choose your language</title>
</svelte:head>

<div class="language">
  <p class="language__heading">भाषा चुनें · Choose your language</p>

  <ul role="list" class="language__grid">
    {#each LOCALE_CODES as code (code)}
      {@const meta = LOCALES[code]}
      <li class="language__tile">
        <!--
          Two sibling controls, not a button nested in a button (invalid
          HTML, and it breaks keyboard/AT navigation): the select button
          fills the tile, the speak button sits above it via z-index so a
          tap there never reaches the select button underneath.
        -->
        <button
          type="button"
          class="language__select"
          onclick={() => choose(code)}
          lang={meta.tag}
        >
          <span class="language__endonym">{meta.endonym}</span>
        </button>
        <span class="language__speak">
          <SpeakButton text={meta.endonym} tag={meta.tag} label={meta.endonym} iconOnly />
        </span>
      </li>
    {/each}
  </ul>
</div>

<style>
  .language {
    padding-block: var(--k-space-6);
  }

  .language__heading {
    margin-block-end: var(--k-space-6);
    text-align: center;
    font-size: var(--k-text-lg);
    color: var(--k-text-secondary);
  }

  .language__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--k-space-3);
  }

  /* At phone widths two cramped columns choke the endonyms and the 1fr
     auto-minimum blows the grid past the viewport; a single full-width
     column reads and scrolls better. */
  @media (max-width: 30rem) {
    .language__grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .language__tile {
    position: relative;
    min-block-size: calc(var(--k-touch-min) * 1.6);
  }

  .language__select {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 100%;
    block-size: 100%;
    min-block-size: calc(var(--k-touch-min) * 1.6);
    padding: var(--k-space-4) var(--k-space-8) var(--k-space-4) var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-lg);
    background-color: var(--k-surface-raised);
    cursor: pointer;
  }

  .language__select:active {
    background-color: var(--k-surface-pressed);
  }

  .language__endonym {
    font-size: var(--k-text-lg);
    color: var(--k-text-primary);
    text-align: center;
  }

  .language__speak {
    position: absolute;
    inset-inline-end: var(--k-space-2);
    inset-block-start: 50%;
    translate: 0 -50%;
    z-index: 1;
  }
</style>
