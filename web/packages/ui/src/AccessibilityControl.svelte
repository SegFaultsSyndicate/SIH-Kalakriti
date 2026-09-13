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
  import Button, { type ButtonSize } from './Button.svelte';
  import Switch from './Switch.svelte';
  import { a11y, TEXT_SCALE_STEPS } from './a11y.svelte';

  interface Props {
    /** Route to the accessibility statement page, per-app. */
    statementHref: string;
    /** xl (Button's own default) is the artisan app's minimum comfortable
     * tap target. A header that also has to fit a wordmark and two more
     * icon buttons on one row at phone width (the buyer shell) can pass a
     * smaller size -- lg is still above --k-touch-min (44px). */
    triggerSize?: ButtonSize;
  }

  let { statementHref, triggerSize }: Props = $props();

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

  function onReduceMotionChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setReduceMotionOverride(checked);
  }

  function onReadableFontChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setReadableFont(checked);
  }

  function onLineSpacingChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setLineSpacing(checked ? 'relaxed' : 'normal');
  }

  function onHighlightLinksChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setHighlightLinks(checked);
  }

  function onMonochromeChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setMonochrome(checked);
  }

  function onBigCursorChange(event: Event): void {
    const checked = (event.currentTarget as HTMLInputElement).checked;
    void a11y.setBigCursor(checked);
  }

  function speakPage(): void {
    if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;
    window.speechSynthesis.cancel();
    const main = document.getElementById('main-content') || document.querySelector('main') || document.body;
    const text = main.innerText.slice(0, 600);
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = locale.meta?.tag || 'en-IN';
    window.speechSynthesis.speak(utterance);
  }
</script>

<Popover align="end">
  {#snippet trigger(props)}
    <Button icon="accessibility" label={t('a11y.settings')} variant="ghost" size={triggerSize} {...props} />
  {/snippet}
  {#snippet children()}
    <div class="k-a11y-panel">
      <div class="k-a11y-panel__header">
        <span class="k-a11y-panel__title">
          <Icon name="accessibility" size="1.1rem" />
          {t('a11y.settings')}
        </span>
        <button
          type="button"
          class="k-a11y-panel__reset-btn"
          onclick={() => void a11y.resetAll()}
          title={t('a11y.reset')}
        >
          <Icon name="refresh" size="0.85rem" />
          {t('a11y.reset')}
        </button>
      </div>

      <div class="k-a11y-panel__row">
        <span class="k-a11y-panel__label">
          <Icon name="text-size" size="1rem" />
          {t('a11y.textSize')}
        </span>
        <div class="k-a11y-panel__stepper">
          <span class="k-a11y-panel__step-val" aria-live="polite">{t(`a11y.textSize.${a11y.textScale}`)}</span>
          <Button
            icon="chevron-down"
            label={t('ui.numberStepper.decrease')}
            variant="ghost"
            size="sm"
            disabled={!canDecrease}
            onclick={decrease}
          />
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
        <Icon name="contrast" size="1rem" />
        {t('a11y.contrast')}
      </Switch>

      <Switch checked={a11y.reduceMotion} onchange={onReduceMotionChange}>
        <Icon name="play" size="1rem" />
        {t('a11y.reduceMotion')}
      </Switch>

      <Switch checked={a11y.readableFont} onchange={onReadableFontChange}>
        <Icon name="edit" size="1rem" />
        {t('a11y.readableFont')}
      </Switch>

      <Switch checked={a11y.lineSpacing === 'relaxed'} onchange={onLineSpacingChange}>
        <Icon name="more-horizontal" size="1rem" />
        {t('a11y.lineSpacing')}
      </Switch>

      <Switch checked={a11y.highlightLinks} onchange={onHighlightLinksChange}>
        <Icon name="link" size="1rem" />
        {t('a11y.highlightLinks')}
      </Switch>

      <Switch checked={a11y.monochrome} onchange={onMonochromeChange}>
        <Icon name="eye" size="1rem" />
        {t('a11y.monochrome')}
      </Switch>

      <Switch checked={a11y.bigCursor} onchange={onBigCursorChange}>
        <Icon name="plus" size="1rem" />
        {t('a11y.bigCursor')}
      </Switch>

      <button type="button" class="k-a11y-panel__speech-btn" onclick={speakPage}>
        <Icon name="speaker" size="1rem" />
        {t('a11y.readScreen')}
      </button>

      <a class="k-a11y-panel__statement" href={statementHref}>
        {t('a11y.statement.linkLabel')} →
      </a>
    </div>
  {/snippet}
</Popover>

<style>
  .k-a11y-panel {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    min-inline-size: 16.5rem;
    max-inline-size: 20rem;
  }

  .k-a11y-panel__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
    padding-block-end: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .k-a11y-panel__title {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-weight: var(--k-weight-semibold, 600);
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .k-a11y-panel__reset-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    background: transparent;
    border: none;
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    cursor: pointer;
    padding: var(--k-space-1);
    border-radius: var(--k-radius-xs, 2px);
  }

  .k-a11y-panel__reset-btn:hover {
    color: var(--k-accent-primary-bg);
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
    color: var(--k-text-primary);
  }

  .k-a11y-panel__stepper {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
  }

  .k-a11y-panel__step-val {
    font-size: var(--k-text-xs);
    min-inline-size: 3.25rem;
    text-align: end;
    font-weight: var(--k-weight-medium, 500);
    margin-inline-end: var(--k-space-1);
    color: var(--k-text-primary);
  }

  .k-a11y-panel__speech-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    width: 100%;
    min-block-size: var(--k-touch-min);
    background: var(--k-surface-sunken, #f7f4ef);
    border: var(--k-hairline) solid var(--k-border-hairline, #e2ded5);
    border-radius: var(--k-radius-sm, 2px);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium, 500);
    color: var(--k-text-primary);
    cursor: pointer;
    transition: background 150ms ease;
  }

  .k-a11y-panel__speech-btn:hover {
    background: var(--k-surface-raised, #ffffff);
    border-color: var(--k-accent-primary-bg, #9b2c16);
  }

  .k-a11y-panel__statement {
    display: block;
    font-size: var(--k-text-xs);
    color: var(--k-accent-secondary, #9b2c16);
    text-decoration: none;
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    text-align: center;
    font-weight: var(--k-weight-medium, 500);
  }

  .k-a11y-panel__statement:hover {
    text-decoration: underline;
  }
</style>
