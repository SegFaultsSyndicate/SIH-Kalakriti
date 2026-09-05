<!--
  packages/ui/src/Keypad.svelte

    <Keypad bind:value={digits} maxLength={10} label={t('login.phone.label')} />

  A large on-screen numeric keypad, standing in for a bare `<input>` on the
  phone-number screen -- the batch spec calls for one explicitly, since a
  system keyboard's number row is small and easy to mis-tap one-handed.
  Every key is a real <button>, so Tab/Enter/Space work with no extra wiring
  and each key's accessible name is just its own visible content.

  Deliberately not an <input type="tel">: this drives a plain bindable
  string, so a caller composing it with a fixed +91 prefix (see /login) never
  has to strip a country code the artisan typed by hand into a native field.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    value?: string;
    maxLength?: number;
    disabled?: boolean;
    /** Names the whole keypad for a screen reader, e.g. the field it fills. */
    label: string;
    class?: string;
  }

  let {
    value = $bindable(''),
    maxLength = 10,
    disabled = false,
    label,
    class: className,
  }: Props = $props();

  const t = $derived(locale.t);

  const ROWS: readonly (readonly string[])[] = [
    ['1', '2', '3'],
    ['4', '5', '6'],
    ['7', '8', '9'],
  ];

  function press(digit: string): void {
    if (disabled || value.length >= maxLength) return;
    value += digit;
  }

  function backspace(): void {
    if (disabled || value.length === 0) return;
    value = value.slice(0, -1);
  }
</script>

<div class="k-keypad {className || ''}" role="group" aria-label={label}>
  {#each ROWS as row (row.join(''))}
    <div class="k-keypad__row">
      {#each row as digit (digit)}
        <button
          type="button"
          class="k-keypad__key"
          {disabled}
          onclick={() => press(digit)}
        >
          {digit}
        </button>
      {/each}
    </div>
  {/each}
  <div class="k-keypad__row">
    <span class="k-keypad__key k-keypad__key--spacer" aria-hidden="true"></span>
    <button type="button" class="k-keypad__key" {disabled} onclick={() => press('0')}>0</button>
    <button
      type="button"
      class="k-keypad__key"
      disabled={disabled || value.length === 0}
      aria-label={t('keypad.backspace')}
      onclick={backspace}
    >
      <Icon name="chevron-left" />
    </button>
  </div>
</div>

<style>
  .k-keypad {
    display: grid;
    gap: var(--k-space-3);
    inline-size: 100%;
    max-inline-size: 22rem;
  }

  .k-keypad__row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-3);
  }

  .k-keypad__key {
    aspect-ratio: 1;
    min-block-size: calc(var(--k-touch-min) * 1.2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-lg);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-xl);
    font-variant-numeric: var(--k-numeric-tabular);
    cursor: pointer;
  }

  .k-keypad__key:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .k-keypad__key:active:not(:disabled) {
    background-color: var(--k-surface-pressed);
  }

  .k-keypad__key--spacer {
    border: none;
    background: none;
  }
</style>
