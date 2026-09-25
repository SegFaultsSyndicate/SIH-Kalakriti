<!--
  apps/artisan/src/lib/TipOfTheDayModal.svelte

  A non-intrusive, once-a-day business, craft, revenue, and financial tip dialog.
  Randomizes informative tips across Business, Artistic Craft, Revenue Growth,
  and Financial Management. Includes an interactive "Another tip" button.
-->
<script lang="ts">
  import { liveQuery } from 'dexie';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Dialog, Button } from '@kalakriti/ui';
  import { getPref, setPref, db } from '@kalakriti/offline';
  import { BUSINESS_TIPS, getRandomTip, type BusinessTip } from './business-tips';

  interface Props {
    active: boolean;
  }

  let { active }: Props = $props();

  const t = $derived(locale.t);

  interface TipShownState {
    date: string;
    tipId: string;
  }

  const PREF_KEY = 'artisan.tips.lastShown';

  let open = $state(false);
  let tip = $state<BusinessTip>(BUSINESS_TIPS[0]);
  let tutorialSeen = $state(false);
  let checked = false;

  $effect(() => {
    const sub = liveQuery(() => db.prefs.get('literacy.tutorial_completed')).subscribe((row) => {
      tutorialSeen = row?.value === true;
    });
    return () => sub.unsubscribe();
  });

  function todayKey(): string {
    return new Date().toISOString().slice(0, 10);
  }

  $effect(() => {
    if (!active || !tutorialSeen || checked) return;
    checked = true;
    void (async () => {
      const last = await getPref<TipShownState>(PREF_KEY);
      const today = todayKey();
      if (last?.date === today) return;
      const nextTip = getRandomTip(last?.tipId);
      tip = nextTip;
      open = true;
      await setPref(PREF_KEY, { date: today, tipId: nextTip.id });
    })();
  });

  function handleNextTip(): void {
    tip = getRandomTip(tip.id);
  }
</script>

<Dialog bind:open title={t('tips.modal.title')} class="tip-dialog">
  <div class="tip-dialog__content">
    <div class="tip-dialog__category-row">
      <span class="tip-dialog__category-pill tip-dialog__category-pill--{tip.category}">
        {tip.categoryLabel}
      </span>
    </div>

    <div class="tip-dialog__body">
      <span class="tip-dialog__icon-wrap tip-dialog__icon-wrap--{tip.category}">
        <Icon name={tip.icon} size="1.4rem" />
      </span>
      <div class="tip-dialog__text-block">
        <h3 class="tip-dialog__title">{tip.title}</h3>
        <p class="tip-dialog__text">{tip.text}</p>
      </div>
    </div>
  </div>

  <div class="tip-dialog__actions">
    <button type="button" class="tip-dialog__cycle-btn" onclick={handleNextTip}>
      <Icon name="refresh" size="0.875rem" />
      <span>Another tip</span>
    </button>
    <Button variant="primary" onclick={() => (open = false)}>{t('tips.modal.gotIt')}</Button>
  </div>
</Dialog>

<style>
  .tip-dialog__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .tip-dialog__category-row {
    display: flex;
    align-items: center;
  }

  .tip-dialog__category-pill {
    display: inline-flex;
    align-items: center;
    padding: 0.15rem 0.55rem;
    border-radius: var(--k-radius-full);
    font-size: 0.6875rem;
    font-weight: var(--k-weight-semibold);
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .tip-dialog__category-pill--business {
    background-color: #f1f5f9;
    color: #334155;
    border: 1px solid #cbd5e1;
  }

  .tip-dialog__category-pill--craft {
    background-color: #fef2f2;
    color: #991b1b;
    border: 1px solid #fecaca;
  }

  .tip-dialog__category-pill--revenue {
    background-color: #f0fdf4;
    color: #166534;
    border: 1px solid #bbf7d0;
  }

  .tip-dialog__category-pill--finance {
    background-color: #fffbeb;
    color: #92400e;
    border: 1px solid #fde68a;
  }

  .tip-dialog__body {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-3);
  }

  .tip-dialog__icon-wrap {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.625rem;
    block-size: 2.625rem;
    border-radius: var(--k-radius-lg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .tip-dialog__icon-wrap--business {
    background-color: #f1f5f9;
    color: #1e293b;
  }

  .tip-dialog__icon-wrap--craft {
    background-color: #fee2e2;
    color: #991b1b;
  }

  .tip-dialog__icon-wrap--revenue {
    background-color: #dcfce7;
    color: #15803d;
  }

  .tip-dialog__icon-wrap--finance {
    background-color: #fef3c7;
    color: #b45309;
  }

  .tip-dialog__text-block {
    flex: 1;
    min-inline-size: 0;
  }

  .tip-dialog__title {
    margin: 0 0 0.35rem;
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    line-height: 1.35;
  }

  .tip-dialog__text {
    margin: 0;
    font-size: var(--k-text-sm);
    line-height: 1.55;
    color: var(--k-text-secondary);
  }

  .tip-dialog__actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-block-start: var(--k-space-4);
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    gap: var(--k-space-2);
  }

  .tip-dialog__cycle-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    background: none;
    border: none;
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
    padding: 0.4rem 0.6rem;
    border-radius: var(--k-radius-md);
    transition: background-color 0.15s ease, color 0.15s ease, transform 0.1s ease;
  }

  .tip-dialog__cycle-btn:hover {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  .tip-dialog__cycle-btn:active {
    transform: scale(0.96);
  }
</style>
