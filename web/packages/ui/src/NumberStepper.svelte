<!--
  packages/ui/src/NumberStepper.svelte

  Large +/- buttons flank a directly-editable number, for quantity entry
  where the artisan may not read the field's own label reliably -- the
  buttons alone communicate "more" and "fewer" by size and position.
  56px touch targets regardless of app, since this is always a quantity a
  transaction depends on getting right.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';

  interface Props {
    value?: number;
    min?: number;
    max?: number;
    step?: number;
    id?: string;
    class?: string;
    [key: string]: unknown;
  }

  let {
    value = $bindable(0),
    min = 0,
    max = Number.MAX_SAFE_INTEGER,
    step = 1,
    id,
    class: className,
    ...rest
  }: Props = $props();

  const t = $derived(locale.t);

  function clamp(next: number): number {
    return Math.min(max, Math.max(min, next));
  }

  function decrease(): void {
    value = clamp(value - step);
  }

  function increase(): void {
    value = clamp(value + step);
  }

  function onInput(event: Event): void {
    const raw = (event.currentTarget as HTMLInputElement).valueAsNumber;
    if (!Number.isNaN(raw)) value = clamp(raw);
  }
</script>

<div class="k-number-stepper {className || ''}">
  <button
    type="button"
    class="k-number-stepper__button"
    onclick={decrease}
    disabled={value <= min}
    aria-label={t('ui.numberStepper.decrease')}
  >
    &minus;
  </button>
  <input
    class="k-number-stepper__value"
    type="number"
    inputmode="numeric"
    {id}
    {min}
    {max}
    {step}
    {value}
    oninput={onInput}
    {...rest}
  />
  <button
    type="button"
    class="k-number-stepper__button"
    onclick={increase}
    disabled={value >= max}
    aria-label={t('ui.numberStepper.increase')}
  >
    +
  </button>
</div>
