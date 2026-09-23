<!--
  apps/artisan/src/lib/BracketPicker.svelte

  The F13 income-baseline question: monthly income from craft before
  Kalakriti, as a bracket (income_bracket enum, pkg/impact). Brackets rather
  than an exact figure on purpose -- nobody remembers an exact month, and a
  bracket is honest about that. Used by /register/income and the home
  income card's "tell us" sheet.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';

  interface Props {
    value?: string;
    onselect?: (bracket: string) => void;
  }

  let { value = $bindable(''), onselect }: Props = $props();

  const t = $derived(locale.t);

  const BRACKETS: readonly { code: string; key: MessageKey }[] = [
    { code: 'LT_3K', key: 'income.bracket.lt3k' },
    { code: 'B3K_6K', key: 'income.bracket.3to6k' },
    { code: 'B6K_10K', key: 'income.bracket.6to10k' },
    { code: 'B10K_15K', key: 'income.bracket.10to15k' },
    { code: 'GT_15K', key: 'income.bracket.gt15k' },
    { code: 'PREFER_NOT_TO_SAY', key: 'registration.socialCategory.preferNotToSay' },
  ];

  function pick(code: string): void {
    value = code;
    onselect?.(code);
  }
</script>

<div class="brackets" role="radiogroup" aria-label={t('income.baseline.question')}>
  {#each BRACKETS as b (b.code)}
    <button
      type="button"
      role="radio"
      aria-checked={value === b.code}
      class="bracket"
      class:bracket--on={value === b.code}
      onclick={() => pick(b.code)}
    >
      {t(b.key)}
    </button>
  {/each}
</div>

<style>
  .brackets {
    display: grid;
    gap: var(--k-space-2);
  }

  .bracket {
    min-block-size: calc(var(--k-touch-min) * 1.25);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
    font-weight: 500;
    text-align: start;
    cursor: pointer;
  }

  .bracket--on {
    border-color: var(--k-accent-primary-bg);
    border-width: var(--k-rule);
    background-color: var(--k-surface-sunken);
    font-weight: 700;
  }
</style>
