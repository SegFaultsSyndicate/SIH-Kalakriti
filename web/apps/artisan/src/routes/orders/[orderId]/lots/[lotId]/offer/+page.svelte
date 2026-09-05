<!--
  apps/artisan/src/routes/orders/[orderId]/lots/[lotId]/offer/+page.svelte

  A time-boxed decision: what's asked, the deadline, how it fits what this
  artisan already has committed, and an explicit accept/decline -- each
  spoken aloud before it is confirmed, per the batch 10 brief. POST
  /orders/lots/{id}/respond is real; the response always queues through the
  outbox (queueLotResponse, $lib/orders.ts) so it works with no connection --
  the UI just says so plainly when offline, rather than pretending it went
  straight through.

  The countdown to responds_by is a plain, calm sentence and a re-rendered
  "time left" line, not a shrinking red bar or a ticking clock font --
  "prominent but not anxiety-inducing" per the brief, and explicitly no
  urgency dark patterns.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, formatDate } from '@kalakriti/i18n';
  import { Button, Dialog, Input, Label, Money, SpeakButton, showToast } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { Icon } from '@kalakriti/icons';
  import {
    fetchOrder,
    cachedOrder,
    myLots,
    queueLotResponse,
    network,
    type BulkOrder,
    type OrderLot,
  } from '$lib/orders';
  import { getArtisanId } from '$lib/registration';

  const t = $derived(locale.t);
  const orderId = $derived(page.params.orderId ?? '');
  const lotId = $derived(page.params.lotId ?? '');

  let loading = $state(true);
  let order = $state<BulkOrder | undefined>(undefined);
  let artisanId = $state<string | undefined>(undefined);
  let now = $state(Date.now());

  let confirmingAccept = $state(false);
  let confirmingDecline = $state(false);
  let shipDate = $state('');
  let declineReason = $state('');
  let submitting = $state(false);

  $effect(() => {
    void (async () => {
      loading = true;
      artisanId = await getArtisanId();
      order = (await cachedOrder(orderId)) ?? undefined;
      loading = false;
      if (network.online) {
        try {
          order = await fetchOrder(orderId);
        } catch {
          /* keep the cached copy; the offer's own numbers still show. */
        }
      }
    })();
  });

  $effect(() => {
    const timer = setInterval(() => (now = Date.now()), 30_000);
    return () => clearInterval(timer);
  });

  const lot = $derived<OrderLot | undefined>(order?.lots?.find((l) => l.id === lotId));
  const otherCommitments = $derived(
    artisanId && order
      ? myLots([order], artisanId).filter(
          (l) => l.id !== lotId && (l.state === 'ACCEPTED' || l.state === 'IN_PRODUCTION'),
        ).length
      : 0,
  );

  const respondsByMs = $derived(lot?.responds_by ? new Date(lot.responds_by).getTime() : undefined);
  const msLeft = $derived(respondsByMs ? respondsByMs - now : undefined);
  const expired = $derived(msLeft !== undefined && msLeft <= 0);

  function timeLeftLabel(ms: number): string {
    const hours = Math.max(0, Math.floor(ms / 3_600_000));
    if (hours < 1) return t('lotOffer.deadline.lessThanHour');
    if (hours < 24) return t('lotOffer.deadline.hours', { hours: String(hours) });
    return t('lotOffer.deadline.days', { days: String(Math.floor(hours / 24)) });
  }

  const speakSummary = $derived(
    lot
      ? t('lotOffer.speak.summary', {
          quantity: String(lot.quantity ?? ''),
          price: String(lot.unit_price?.amount_paise ?? 0),
        })
      : '',
  );

  async function confirmAccept(): Promise<void> {
    if (!shipDate) return;
    submitting = true;
    try {
      await queueLotResponse(lotId, true, { promised_ship_date: new Date(shipDate).toISOString() });
      showToast({
        variant: 'success',
        message: network.online ? t('lotOffer.accepted') : t('lotOffer.queuedOffline'),
      });
      await goto('/orders');
    } finally {
      submitting = false;
      confirmingAccept = false;
    }
  }

  async function confirmDecline(): Promise<void> {
    if (!declineReason) return;
    submitting = true;
    try {
      await queueLotResponse(lotId, false, { decline_reason: declineReason });
      showToast({
        variant: 'success',
        message: network.online ? t('lotOffer.declined') : t('lotOffer.queuedOffline'),
      });
      await goto('/orders');
    } finally {
      submitting = false;
      confirmingDecline = false;
    }
  }
</script>

<svelte:head>
  <title>{t('lotOffer.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="lot-offer-page">
  <h1>{t('lotOffer.heading')}</h1>

  {#if !network.online}
    <p class="lot-offer-page__offline-note">{t('lotOffer.offlineNote')}</p>
  {/if}

  {#if loading && !lot}
    <Skeleton shape="card" height="10rem" />
  {:else if !lot}
    <p role="alert">{t('lotOffer.notFound')}</p>
  {:else if lot.state !== 'OFFERED'}
    <p role="status">{t('lotOffer.alreadyAnswered')}</p>
  {:else}
    <Card variant="hairline" element="div" class="lot-offer-page__summary">
      <div class="lot-offer-page__summary-row">
        <span>{t('lotOffer.quantity')}</span>
        <strong>{lot.quantity}</strong>
      </div>
      <div class="lot-offer-page__summary-row">
        <span>{t('lotOffer.unitPrice')}</span>
        {#if lot.unit_price?.amount_paise != null}
          <Money paise={lot.unit_price.amount_paise} />
        {/if}
      </div>
      <div class="lot-offer-page__summary-row lot-offer-page__summary-row--total">
        <span>{t('lotOffer.totalEarnings')}</span>
        {#if lot.lot_value?.amount_paise != null}
          <Money paise={lot.lot_value.amount_paise} />
        {/if}
      </div>
    </Card>

    <div class="lot-offer-page__deadline" class:lot-offer-page__deadline--expired={expired}>
      <Icon name="clock" />
      <div>
        {#if expired}
          <p>{t('lotOffer.deadline.expired')}</p>
        {:else if msLeft !== undefined}
          <p>{timeLeftLabel(msLeft)}</p>
        {/if}
        {#if lot.responds_by}
          <p class="lot-offer-page__deadline-date">
            {t('lotOffer.deadline.by', {
              date: formatDate(lot.responds_by, locale.code, { dateStyle: 'medium', timeStyle: 'short' }),
            })}
          </p>
        {/if}
      </div>
    </div>

    <p class="lot-offer-page__capacity">
      {otherCommitments > 0
        ? t('lotOffer.capacity.withOthers', { count: String(otherCommitments) })
        : t('lotOffer.capacity.none')}
    </p>

    <SpeakButton text={speakSummary} label={t('lotOffer.speak.label')} />

    {#if !expired}
      <div class="lot-offer-page__actions">
        <Button size="xl" onclick={() => (confirmingAccept = true)}>{t('lotOffer.accept')}</Button>
        <Button size="xl" variant="secondary" onclick={() => (confirmingDecline = true)}>
          {t('lotOffer.decline')}
        </Button>
      </div>
    {/if}
  {/if}
</main>

<Dialog bind:open={confirmingAccept} title={t('lotOffer.confirmAccept.title')}>
  <p>{t('lotOffer.confirmAccept.body')}</p>
  <SpeakButton text={t('lotOffer.confirmAccept.body')} label={t('lotOffer.speak.label')} />
  <Label for="ship-date">{t('lotOffer.shipDateLabel')}</Label>
  <Input id="ship-date" type="date" bind:value={shipDate} />
  <div class="lot-offer-page__dialog-actions">
    <Button onclick={confirmAccept} disabled={!shipDate || submitting} loading={submitting}>
      {t('lotOffer.confirmAccept.confirm')}
    </Button>
    <Button variant="ghost" onclick={() => (confirmingAccept = false)}>{t('action.cancel')}</Button>
  </div>
</Dialog>

<Dialog bind:open={confirmingDecline} title={t('lotOffer.confirmDecline.title')}>
  <p>{t('lotOffer.confirmDecline.body')}</p>
  <SpeakButton text={t('lotOffer.confirmDecline.body')} label={t('lotOffer.speak.label')} />
  <Label for="decline-reason">{t('lotOffer.declineReasonLabel')}</Label>
  <Input id="decline-reason" bind:value={declineReason} />
  <div class="lot-offer-page__dialog-actions">
    <Button onclick={confirmDecline} disabled={!declineReason || submitting} loading={submitting}>
      {t('lotOffer.confirmDecline.confirm')}
    </Button>
    <Button variant="ghost" onclick={() => (confirmingDecline = false)}>{t('action.cancel')}</Button>
  </div>
</Dialog>

<style>
  .lot-offer-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .lot-offer-page__offline-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  :global(.lot-offer-page__summary) {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
  }

  .lot-offer-page__summary-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .lot-offer-page__summary-row--total {
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    font-weight: 700;
  }

  .lot-offer-page__deadline {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
  }

  .lot-offer-page__deadline--expired {
    color: var(--k-accent-danger);
  }

  .lot-offer-page__deadline-date {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .lot-offer-page__capacity {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .lot-offer-page__actions {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .lot-offer-page__dialog-actions {
    display: flex;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
  }
</style>
