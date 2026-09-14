<!--
  apps/buyer/src/lib/CurrencySelector.svelte

  Mounted in the buyer shell header, once: <CurrencySelector />. Every currency
  the catalogue can be displayed in (CURRENCY_RATES) is listed in its own
  panel -- a mirror of LanguageSelector's -- each option labelled by its code
  with its symbol alongside. The chosen currency re-renders every listing
  card's price line via the shared `currency` store.
-->
<script lang="ts">
  import { locale, CURRENCY_META, type CurrencyCode } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { currency, CURRENCY_RATES } from '../../../apps/buyer/src/lib/currency.svelte';
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
  const tt = $derived(locale.tooltip);
  let open = $state(false);

  const options = Object.keys(CURRENCY_RATES) as CurrencyCode[];

  function select(code: CurrencyCode): void {
    open = false;
    currency.set(code);
  }
</script>

<Popover align="end" bind:open>
  {#snippet trigger(props)}
    <Button icon="dollar-sign" label={t('currency.selector.label')} variant="ghost" tooltip={tt('tooltip.currency')} size={triggerSize} {...props} />
  {/snippet}
  {#snippet children()}
    <ul class="k-currency-panel" role="list">
      {#each options as code (code)}
        <li>
          <button
            type="button"
            class="k-currency-panel__option"
            class:k-currency-panel__option--active={currency.code === code}
            aria-current={currency.code === code ? 'true' : undefined}
            onclick={() => select(code)}
          >
            <span class="k-currency-panel__code">{code}</span>
            <span class="k-currency-panel__symbol">{CURRENCY_META[code].symbol}</span>
            {#if currency.code === code}
              <Icon name="check" />
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {/snippet}
</Popover>

<style>
  .k-currency-panel {
    display: flex;
    flex-direction: column;
    max-block-size: 20rem;
    overflow-y: auto;
    min-inline-size: 14rem;
  }

  .k-currency-panel__option {
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

  .k-currency-panel__option:hover,
  .k-currency-panel__option--active {
    background-color: var(--k-surface-sunken);
  }

  .k-currency-panel__symbol {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-inline-start: auto;
  }
</style>