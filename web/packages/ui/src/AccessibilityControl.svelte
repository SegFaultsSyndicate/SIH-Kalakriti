<!--
  packages/ui/src/AccessibilityControl.svelte

    Mounted in a layout header, once:
    <AccessibilityControl statementHref="/accessibility" />

  The GIGW-convention toolbar: a text-size stepper, a contrast toggle, and a
  link to the accessibility statement, behind one header button so it costs
  one tap target rather than three on a screen that is already tight on the
  artisan app's phone width. Reads and writes @kalakriti/ui's `a11y` store
  directly -- callers mount this once and never touch the store themselves.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import Popover from './Popover.svelte';
  import Button from './Button.svelte';
  import Switch from './Switch.svelte';
  import { a11y, TEXT_SCALE_STEPS } from './a11y.svelte';

  interface Props {
    /** Route to the accessibility statement page, per-app. */
    statementHref: string;
  }

  let { statementHref }: Props = $props();

  const t = $derived(locale.t);

  const stepIndex = $derived(TEXT_SCALE_STEPS.indexOf(a11y.textScale));
  const canDecrease = $derived(stepIndex > 0);
  const canIncrease = $derived(stepIndex < TEXT_SCALE_STEPS.length - 1);

  function decrease(): void {
    if (canDecrease) void a11y.setTextScale(TEXT_SCALE_STEPS[stepIndex - 1]);
  }

  function increase(): void {
    if (canIncrease) void a11y.setTextScale(TEXT_SCALE_STEPS[stepIndex + 1]);
  }

  function onContrastChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setContrast(checked ? 'high' : 'normal');
  }
</script>

<Popover align="end">
  {#snippet trigger(props)}
    <Button icon="accessibility" label={t('a11y.settings')} variant="ghost" {...props} />
  {/snippet}
  {#snippet children()}
    <div class="k-a11y-panel">
      <div class="k-a11y-panel__row">
        <span class="k-a11y-panel__label">
          <Icon name="text-size" />
          {t('a11y.textSize')}
        </span>
        <div class="k-a11y-panel__stepper">
          <Button
            icon="chevron-down"
            label={t('ui.numberStepper.decrease')}
            variant="ghost"
            size="sm"
            disabled={!canDecrease}
            onclick={decrease}
          />
          <span aria-live="polite">{t(`a11y.textSize.${a11y.textScale}`)}</span>
          <Button
            icon="chevron-up"
            label={t('ui.numberStepper.increase')}
            variant="ghost"
            size="sm"
            disabled={!canIncrease}
            onclick={increase}
          />
        </div>
      </div>

      <Switch checked={a11y.contrast === 'high'} onchange={onContrastChange}>
        <Icon name="contrast" />
        {t('a11y.contrast')}
      </Switch>

      <a class="k-a11y-panel__statement" href={statementHref}>
        {t('a11y.statement.linkLabel')}
      </a>
    </div>
  {/snippet}
</Popover>

<style>
  .k-a11y-panel {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-3);
    min-inline-size: 14rem;
  }

  .k-a11y-panel__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .k-a11y-panel__label {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
  }

  .k-a11y-panel__stepper {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .k-a11y-panel__statement {
    font-size: var(--k-text-sm);
    color: var(--k-accent-secondary);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }
</style>
