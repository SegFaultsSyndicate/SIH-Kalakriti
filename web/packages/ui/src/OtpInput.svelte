<!--
  packages/ui/src/OtpInput.svelte

    <OtpInput bind:value={code} label={t('verify.code.label')} oncomplete={submit} />

  Six single-digit boxes rather than one text field: each box is large enough
  to read at a glance and to hit reliably, and per-digit `aria-label`s (Digit
  1 of 6, ...) make sense read aloud one at a time, which a single 6-digit
  field's value would not.

  `autocomplete="one-time-code"` on every box, not just the first: iOS/Android
  SMS autofill and a manual paste both hand the whole code to whichever box
  currently has focus, so every box has to be able to receive it and hand off
  the extra digits to its neighbours -- see distribute() below.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';

  interface Props {
    value?: string;
    length?: number;
    disabled?: boolean;
    invalid?: boolean;
    label: string;
    /** Fires once, the moment the code reaches `length` digits. */
    oncomplete?: (code: string) => void;
    /** Focus the first box on mount, and again whenever the code is cleared (e.g. after a wrong code), so the user can type without tapping. */
    autofocus?: boolean;
  }

  let {
    value = $bindable(''),
    length = 6,
    disabled = false,
    invalid = false,
    label,
    oncomplete,
    autofocus = false,
  }: Props = $props();

  const t = $derived(locale.t);

  let boxes = $state<HTMLInputElement[]>([]);
  let firedFor = '';

  const digits = $derived(
    Array.from({ length }, (_, i) => value[i] ?? ''),
  );

  $effect(() => {
    if (value.length === length && value !== firedFor) {
      firedFor = value;
      oncomplete?.(value);
    }
  });

  $effect(() => {
    if (autofocus && value === '' && !disabled) boxes[0]?.focus();
  });

  function onlyDigits(raw: string): string {
    return raw.replace(/\D/g, '');
  }

  /** Merges `incoming` (one box's new content, possibly several pasted/autofilled digits) into `value` starting at `at`, and returns where focus should land next. */
  function distribute(incoming: string, at: number): number {
    const clean = onlyDigits(incoming).slice(0, length - at);
    if (clean === '') return at;
    const chars = value.split('');
    for (let i = 0; i < clean.length; i++) chars[at + i] = clean[i];
    value = chars.join('').slice(0, length);
    return Math.min(at + clean.length, length - 1);
  }

  function oninput(index: number, event: Event): void {
    const target = event.currentTarget as HTMLInputElement;
    const next = distribute(target.value, index);
    target.value = digits[index] ?? '';
    if (next !== index) boxes[next]?.focus();
  }

  function onkeydown(index: number, event: KeyboardEvent): void {
    if (event.key === 'Backspace' && !digits[index] && index > 0) {
      event.preventDefault();
      const chars = value.split('');
      chars[index - 1] = '';
      value = chars.join('');
      boxes[index - 1]?.focus();
    } else if (event.key === 'ArrowLeft' && index > 0) {
      event.preventDefault();
      boxes[index - 1]?.focus();
    } else if (event.key === 'ArrowRight' && index < length - 1) {
      event.preventDefault();
      boxes[index + 1]?.focus();
    }
  }

  function onpaste(index: number, event: ClipboardEvent): void {
    const text = event.clipboardData?.getData('text');
    if (!text) return;
    event.preventDefault();
    const next = distribute(text, index);
    boxes[next]?.focus();
  }
</script>

<div class="k-otp" role="group" aria-label={label} class:k-otp--invalid={invalid}>
  {#each digits as digit, index (index)}
    <input
      bind:this={boxes[index]}
      class="k-otp__box"
      type="text"
      inputmode="numeric"
      pattern="[0-9]*"
      autocomplete="one-time-code"
      maxlength={length}
      value={digit}
      {disabled}
      aria-invalid={invalid || undefined}
      aria-label={t('verify.digit.label', { position: index + 1, total: length })}
      oninput={(event) => oninput(index, event)}
      onkeydown={(event) => onkeydown(index, event)}
      onpaste={(event) => onpaste(index, event)}
    />
  {/each}
</div>

<style>
  .k-otp {
    display: flex;
    justify-content: center;
    gap: var(--k-space-2);
  }

  .k-otp__box {
    inline-size: calc(var(--k-touch-min) * 1.1);
    block-size: calc(var(--k-touch-min) * 1.3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-xl);
    text-align: center;
    font-variant-numeric: var(--k-numeric-tabular);
  }

  .k-otp--invalid .k-otp__box {
    border-color: var(--k-accent-danger);
  }

  .k-otp__box:disabled {
    opacity: 0.55;
  }
</style>
