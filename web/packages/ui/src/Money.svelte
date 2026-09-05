<!--
  packages/ui/src/Money.svelte

    <Money paise={listing.priceMin} />
    <Money paise={statement.total} paiseMode="always" />
    <Money paise={lineItem.amount} symbol={false} />

  The only place a paise integer becomes a string in this codebase's UI
  layer -- everywhere else imports this component rather than calling
  formatMoney directly, so there is exactly one render path to audit for
  the "never a raw paise integer, never float math" rule. `paiseMode` is
  named apart from the `paise` amount prop on purpose: formatMoney's own
  option is also called `paise` (the display mode, not the value), and
  giving both the same name here would shadow the amount with the mode.
-->
<script lang="ts">
  import { locale, formatMoney, type MoneyOptions, type Paise } from '@kalakriti/i18n';

  interface Props {
    paise: Paise;
    paiseMode?: MoneyOptions['paise'];
    symbol?: MoneyOptions['symbol'];
    class?: string;
  }

  let { paise, paiseMode, symbol, class: className }: Props = $props();

  const formatted = $derived(formatMoney(paise, locale.code, { paise: paiseMode, symbol }));
</script>

<span class="k-money k-tabular {className || ''}">{formatted}</span>
