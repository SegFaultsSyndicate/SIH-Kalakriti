<!--
  apps/buyer/src/lib/PurchaseForm.svelte

    <PurchaseForm {listing} />

  Places an order through POST /orders/bulk (fulfilment.proto's
  CreateBulkOrder) -- there is no separate cart/checkout/payment RPC
  anywhere in the backend, so a single-item purchase is quantity: 1 through
  this same call, and a "bulk" order (see BulkOrderWizard.svelte) is the
  same call with a bigger quantity. Nothing here is a different code path.

  Radical transparency, honestly bounded: unit price, quantity, total and
  the advance/balance split all come from real fields (price,
  made_to_order_terms.advance_pct). The artisan's *actual* payout after
  commission has no pre-order source anywhere in the backend -- commission_pct
  is an internal settlement input (services/collab-svc/internal/collab/service/payment.go),
  never exposed pre-order -- so this form does not show a fabricated earnings
  number. The real, backend-sourced payment split is shown once it exists,
  on the allocation view (routes/orders/[id]), when payment_settled fires.

  There is no address field on CreateBulkOrderRequest at all, and no
  UPI/payment-initiation RPC anywhere in the proto tree -- collecting a
  polished address form or building a upi:// deep link would either drop
  the data silently or fabricate a payee VPA. Both are refused; delivery
  details go into the one real free-text channel the backend has (`notes`),
  labelled honestly, and "place order" creates the order with no payment
  collected, also labelled honestly (purchase.noPaymentGateway).
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Button, NumberStepper, Textarea, Select, Money, FieldGroup, showToast } from '@kalakriti/ui';
  import { createBulkOrder, session, type components } from '@kalakriti/api';
  import { rememberOrder } from './order-store';

  type ListingSummary = components['schemas']['ListingSummary'];

  interface Props {
    listing: ListingSummary;
  }

  let { listing }: Props = $props();

  const t = $derived(locale.t);
  const terms = $derived(listing.made_to_order_terms);
  const minOrder = $derived(listing.min_order_quantity ?? 1);

  let quantity = $state(minOrder);
  let notes = $state('');
  let customisations = $state<Record<string, string>>({});
  let neededBy = $state(defaultNeededBy(terms?.lead_time_days));
  let submitting = $state(false);

  $effect(() => {
    quantity = minOrder;
  });

  function defaultNeededBy(leadDays: number | undefined): string {
    const days = (leadDays ?? 0) + 7;
    const d = new Date();
    d.setDate(d.getDate() + days);
    return d.toISOString().slice(0, 10);
  }

  const unitPricePaise = $derived(listing.price?.amount_paise ?? 0);
  const totalPaise = $derived(unitPricePaise * quantity);
  const advancePct = $derived(terms?.advance_pct ?? 0);
  const advancePaise = $derived(Math.round((totalPaise * advancePct) / 100));

  async function submit(): Promise<void> {
    if (submitting) return;
    submitting = true;
    try {
      const combinedNotes = notes.trim();
      const res = await createBulkOrder({
        listing_id: listing.id ?? '',
        quantity,
        required_by: new Date(`${neededBy}T00:00:00Z`).toISOString(),
        customisations: Object.keys(customisations).length > 0 ? customisations : undefined,
        notes: combinedNotes || undefined,
      });
      if (res.order_id) {
        rememberOrder(res.order_id);
        showToast({ variant: 'success', message: t('purchase.success') });
        void goto(`/orders/${res.order_id}`);
      }
    } catch {
      showToast({ variant: 'error', message: t('purchase.error') });
    } finally {
      submitting = false;
    }
  }
</script>

{#if session.status === 'authenticated'}
  <form class="purchase-form" onsubmit={(e) => { e.preventDefault(); void submit(); }}>
    <FieldGroup label={t('purchase.quantity')} description={t('purchase.minOrder', { count: String(minOrder) })}>
      {#snippet children({ id })}
        <NumberStepper {id} bind:value={quantity} min={minOrder} />
      {/snippet}
    </FieldGroup>

    {#if terms?.customisation_options && terms.customisation_options.length > 0}
      {#each terms.customisation_options as option (option.name ?? '')}
        {@const name = option.name ?? ''}
        <FieldGroup label={name}>
          {#snippet children({ id })}
            <Select
              {id}
              bind:value={customisations[name]}
              options={(option.values ?? []).map((v) => ({ value: v, label: v }))}
            />
          {/snippet}
        </FieldGroup>
      {/each}
    {/if}

    <FieldGroup label={t('purchase.neededBy')}>
      {#snippet children({ id })}
        <input {id} type="date" bind:value={neededBy} class="purchase-form__date" />
      {/snippet}
    </FieldGroup>

    <FieldGroup label={t('purchase.deliveryNotes')} description={t('purchase.deliveryNotesHint')}>
      {#snippet children({ id })}
        <Textarea {id} bind:value={notes} rows={3} />
      {/snippet}
    </FieldGroup>

    <dl class="purchase-form__breakdown">
      <div><dt>{t('purchase.unitPrice')}</dt><dd><Money paise={unitPricePaise} /></dd></div>
      <div><dt>{t('purchase.total')}</dt><dd><Money paise={totalPaise} /></dd></div>
      {#if advancePct > 0}
        <div>
          <dt>{t('purchase.advanceDue', { pct: String(advancePct) })}</dt>
          <dd><Money paise={advancePaise} /></dd>
        </div>
        <div><dt>{t('purchase.balanceOnDispatch')}</dt><dd><Money paise={totalPaise - advancePaise} /></dd></div>
      {/if}
    </dl>

    <p class="purchase-form__disclosure">{t('purchase.noPaymentGateway')}</p>

    <Button type="submit" loading={submitting}>{submitting ? t('purchase.submitting') : t('purchase.submit')}</Button>
  </form>
{/if}

<style>
  .purchase-form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    margin-block: var(--k-space-5);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
  }

  .purchase-form__date {
    font: inherit;
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
    min-block-size: var(--k-touch-min);
  }

  .purchase-form__breakdown {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-3);
  }

  .purchase-form__breakdown div {
    display: flex;
    justify-content: space-between;
  }

  .purchase-form__disclosure {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }
</style>
