<!--
  apps/artisan/src/lib/TipOfTheDayModal.svelte

  A non-intrusive, once-a-day business/financial tip -- mounted globally
  in +layout.svelte alongside PwaUpdatePrompt/InstallPrompt, but shown as a
  dismissible @kalakriti/ui Dialog (title bar + close button + Escape +
  backdrop click, all free) rather than a floating banner, since this is a
  short read rather than an action prompt.

  `active` (passed from +layout.svelte as showBottomNav) gates the eligibility
  check so this never appears mid-onboarding (/language, /register/*, etc.) --
  only once the artisan has a real home screen to land on.

  Also waits on literacy.tutorial_completed (liveQuery, same pref
  DigitalLiteracyTutorial.svelte's handleFinish/handleDismiss write) so this
  never pops up stacked on top of the Home screen's first-run tutorial --
  the two used to fire independently and could both be open at once for a
  brand-new artisan.

  Persistence follows self-badges.ts's convention: getPref/setPref from
  @kalakriti/offline, not localStorage, so it survives the same way other
  per-device artisan prefs do.
-->
<script lang="ts">
  import { liveQuery } from 'dexie';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Dialog, Button } from '@kalakriti/ui';
  import { getPref, setPref, db } from '@kalakriti/offline';
  import { BUSINESS_TIPS } from './business-tips';

  interface Props {
    active: boolean;
  }

  let { active }: Props = $props();

  const t = $derived(locale.t);

  interface TipShownState {
    date: string;
    index: number;
  }

  const PREF_KEY = 'artisan.tips.lastShown';

  let open = $state(false);
  let tip = $state(BUSINESS_TIPS[0]);
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
      const nextIndex = last ? (last.index + 1) % BUSINESS_TIPS.length : 0;
      tip = BUSINESS_TIPS[nextIndex];
      open = true;
      await setPref(PREF_KEY, { date: today, index: nextIndex });
    })();
  });
</script>

<Dialog bind:open title={t('tips.modal.title')} class="tip-dialog">
  <div class="tip-dialog__body">
    <span class="tip-dialog__icon"><Icon name={tip.icon} size="1.5rem" /></span>
    <p class="tip-dialog__text">{t(tip.bodyKey)}</p>
  </div>
  <div class="tip-dialog__actions">
    <Button variant="primary" onclick={() => (open = false)}>{t('tips.modal.gotIt')}</Button>
  </div>
</Dialog>

<style>
  .tip-dialog__body {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-3);
  }

  .tip-dialog__icon {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.5rem;
    block-size: 2.5rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-sunken);
    color: var(--k-accent-primary-text);
  }

  .tip-dialog__text {
    margin: 0;
    font-size: var(--k-text-sm);
    line-height: 1.55;
    color: var(--k-text-primary);
  }

  .tip-dialog__actions {
    display: flex;
    justify-content: flex-end;
    margin-block-start: var(--k-space-4);
  }
</style>
