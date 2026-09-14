<!--
  packages/ui/src/LanguageSelector.svelte

  Mounted in a header, once: <LanguageSelector />. Every one of the 22
  Eighth Schedule locales is listed (LOCALE_CODES), each labelled in its own
  endonym per locales.ts's convention -- coverage: 'fallback' locales still
  select cleanly, they just render through the hi -> en message chain.
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { LOCALES, LOCALE_CODES, type LocaleCode } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import Popover from './Popover.svelte';
  import Button, { type ButtonSize } from './Button.svelte';

  interface Props {
    /** xl (Button's own default) is the artisan app's minimum comfortable
     * tap target. A header that also has to fit a wordmark and two more
     * icon buttons on one row at phone width (the buyer shell) can pass a
     * smaller size -- lg is still above --k-touch-min (44px). */
    triggerSize?: ButtonSize;
  }
  let { triggerSize }: Props = $props();

  const t = $derived(locale.t);
  let open = $state(false);

  function select(code: LocaleCode): void {
    open = false;
    void locale.set(code, { persist: true });
  }
</script>

<Popover align="end" bind:open>
  {#snippet trigger(props)}
    <Button icon="language" label={t('language.selector.label')} variant="ghost" tooltip={tooltip('tooltip.language')} size={triggerSize} {...props} />
  {/snippet}
  {#snippet children()}
    <ul class="k-lang-panel" role="list">
      {#each LOCALE_CODES as code (code)}
        <li>
          <button
            type="button"
            class="k-lang-panel__option"
            class:k-lang-panel__option--active={locale.code === code}
            aria-current={locale.code === code ? 'true' : undefined}
            onclick={() => select(code)}
          >
            <span class="k-lang-panel__endonym">{LOCALES[code].endonym}</span>
            <span class="k-lang-panel__english">{LOCALES[code].englishName}</span>
            {#if locale.code === code}
              <Icon name="check" />
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {/snippet}
</Popover>

<style>
  .k-lang-panel {
    display: flex;
    flex-direction: column;
    max-block-size: 20rem;
    overflow-y: auto;
    min-inline-size: 14rem;
  }

  .k-lang-panel__option {
    display: flex;
    align-items: baseline;
    gap: var(--k-space-2);
    inline-size: 100%;
    padding: var(--k-space-2) var(--k-space-3);
    border: none;
    background: none;
    text-align: start;
    cursor: pointer;
    font-size: var(--k-text-base);
  }

  .k-lang-panel__option:hover,
  .k-lang-panel__option--active {
    background-color: var(--k-surface-sunken);
  }

  .k-lang-panel__english {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-inline-start: auto;
  }
</style>
