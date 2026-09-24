<!--
  apps/buyer/src/routes/bulk-order/+page.svelte

  Craft -> listing -> quantity/deadline/delivery -> review -> submit, through
  the same POST /orders/bulk as a single-item purchase (see
  lib/PurchaseForm.svelte's header comment for why there is only one
  order-placing call in this backend).

  Feasibility feedback before submission: ProposeAllocation's dry_run only
  works against an order that already exists (fulfilment.proto's
  ProposeAllocationRequest takes a bulk_order_id), so there is no honest way
  to dry-run allocation before CreateBulkOrder is called. What IS real and
  available before submission is how many artisans practise the chosen
  craft at all (GET /crafts/{slug} -> artisans, already wired for the craft
  landing page) -- shown as a directional capacity signal, explicitly
  labelled as a headcount, not a promise the order will fully allocate.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Button, Select, NumberStepper, Textarea, Stepper, EmptyState, Skeleton, showToast, Tooltip } from '@kalakriti/ui';
  import { createBulkOrder, listCrafts, getCraft, listListings, getListingSummary, session, type components } from '@kalakriti/api';
  import { rememberOrder } from '$lib/order-store';

  type Craft = components['schemas']['Craft'];
  type CraftDetail = components['schemas']['CraftDetail'];
  type ListingSummary = components['schemas']['ListingSummary'];

  const t = $derived(locale.t);
  const steps = $derived([
    t('bulkOrder.step.craft'),
    t('bulkOrder.step.listing'),
    t('bulkOrder.step.details'),
    t('bulkOrder.step.review'),
  ]);

  let current = $state(0);
  let loadingCrafts = $state(true);
  let crafts = $state<Craft[]>([]);
  let craftsFailed = $state(false);
  let craftDetail = $state<CraftDetail | undefined>(undefined);

  let loadingListings = $state(false);
  let listings = $state<ListingSummary[]>([]);
  let selectedListing = $state<ListingSummary | undefined>(undefined);

  let quantity = $state(50);
  let deadline = $state(defaultDeadline());
  let budgetBand = $state('');
  // ?preset=hospitality|corporate (footer's "Handloom Hospitality Linen" /
  // "Eco-Festive Corporate Gifting" links): there's no backend concept of a
  // preset, no craft is unambiguously "for" one, so this only pre-fills the
  // delivery notes the buyer would otherwise type themselves -- still
  // editable, never silently submitted.
  const preset = page.url.searchParams.get('preset');
  let delivery = $state(preset === 'hospitality' ? t('bulkOrder.preset.hospitality.deliveryPrefill') : preset === 'corporate' ? t('bulkOrder.preset.corporate.deliveryPrefill') : '');
  let submitting = $state(false);

  function defaultDeadline(): string {
    const d = new Date();
    d.setDate(d.getDate() + 30);
    return d.toISOString().slice(0, 10);
  }

  $effect(() => {
    void (async () => {
      loadingCrafts = true;
      try {
        const res = await listCrafts();
        crafts = res.crafts ?? [];
      } catch {
        craftsFailed = true;
      } finally {
        loadingCrafts = false;
      }
    })();
  });

  async function chooseCraft(slug: string): Promise<void> {
    loadingListings = true;
    selectedListing = undefined;
    try {
      craftDetail = await getCraft(slug);
      const own = await listListings({ craft_id: craftDetail?.id, state: 'PUBLISHED' });
      const ids = (own.listings ?? []).map((l) => l.id!).filter(Boolean);
      const fetched = await Promise.allSettled(ids.map((id) => getListingSummary(id)));
      listings = fetched
        .filter((r): r is PromiseFulfilledResult<ListingSummary> => r.status === 'fulfilled')
        .map((r) => r.value);
    } catch {
      showToast({ variant: 'error', message: t('api.error.unavailable') });
      return;
    } finally {
      loadingListings = false;
    }
    current = 1;
  }

  function chooseListing(id: string): void {
    selectedListing = listings.find((l) => l.id === id);
    quantity = Math.max(quantity, selectedListing?.min_order_quantity ?? 1);
    current = 2;
  }

  async function submit(): Promise<void> {
    if (submitting || !selectedListing) return;
    submitting = true;
    try {
      const notesParts = [
        budgetBand ? `Budget band: ${budgetBand}` : '',
        delivery ? `Delivery: ${delivery}` : '',
      ].filter(Boolean);
      const res = await createBulkOrder({
        listing_id: selectedListing.id ?? '',
        quantity,
        required_by: new Date(`${deadline}T00:00:00Z`).toISOString(),
        notes: notesParts.length > 0 ? notesParts.join(' · ') : undefined,
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

  const artisanCount = $derived(craftDetail?.artisans?.length ?? 0);
</script>

<svelte:head>
  <title>{t('bulkOrder.heading')} — {t('app.name')}</title>
</svelte:head>

<h1>{t('bulkOrder.heading')}</h1>
<Stepper label={t('bulkOrder.heading')} {steps} {current} />

{#if session.status !== 'authenticated'}
  <EmptyState illustration="empty-error" heading={t('tooltip.signIn')}>
    {#snippet action()}
      <a class="k-button k-button--primary" href="/login?next={encodeURIComponent(page.url.pathname + page.url.search)}">{t('login.signIn')}</a>
    {/snippet}
  </EmptyState>
{:else}
  {#if current === 0}
    <section>
      {#if preset === 'hospitality' || preset === 'corporate'}
        <p class="bulk-order__preset-banner">
          {t(preset === 'hospitality' ? 'bulkOrder.preset.hospitality.banner' : 'bulkOrder.preset.corporate.banner')}
        </p>
      {/if}
      <h2>{t('bulkOrder.craft.label')}</h2>
      {#if loadingCrafts}
        <div class="bulk-order__craft-grid" aria-hidden="true">
          {#each Array(6) as _, i (i)}<Skeleton height="3rem" radius="var(--k-radius-md)" />{/each}
        </div>
      {:else if craftsFailed}
        <EmptyState illustration="empty-error" heading={t('api.error.unavailable')} />
      {:else}
        <div class="bulk-order__craft-grid">
          {#each crafts as craft (craft.id)}
            <Tooltip text={tooltip('tooltip.selectCraft')}>
            {#snippet trigger(tp)}
              <button type="button" class="bulk-order__craft" onclick={() => void chooseCraft(craft.slug ?? '')} {...tp}>
                {craft.display_name}
              </button>
            {/snippet}
          </Tooltip>
          {/each}
        </div>
      {/if}
    </section>
  {:else if current === 1}
    <section>
      <h2>{t('bulkOrder.listing.label')}</h2>
      {#if craftDetail}
        <p class="bulk-order__feasibility">
          {#if artisanCount === 0}
            {t('bulkOrder.feasibility.none')}
          {:else if artisanCount < 3}
            {t('bulkOrder.feasibility.low', { count: String(artisanCount) })}
          {:else}
            {t('bulkOrder.feasibility', { count: String(artisanCount) })}
          {/if}
        </p>
      {/if}
      {#if loadingListings}
        <div class="bulk-order__listing-grid" aria-hidden="true">
          {#each Array(6) as _, i (i)}<Skeleton height="4.5rem" radius="var(--k-radius-md)" />{/each}
        </div>
      {:else if listings.length === 0}
        <p>{t('bulkOrder.listing.empty')}</p>
      {:else}
        <div class="bulk-order__listing-grid">
          {#each listings as listing (listing.id)}
            <Tooltip text={tooltip('tooltip.viewListing')}>
            {#snippet trigger(tp)}
              <button type="button" class="bulk-order__listing" onclick={() => chooseListing(listing.id ?? '')} {...tp}>
                {listing.translations?.[0]?.title}
              </button>
            {/snippet}
          </Tooltip>
          {/each}
        </div>
      {/if}
      <Button variant="secondary" onclick={() => (current = 0)} tooltip={tooltip('tooltip.back')}>{t('bulkOrder.back')}</Button>
    </section>
  {:else if current === 2 && selectedListing}
    <section class="bulk-order__form">
      <label class="bulk-order__field">
        <span>{t('bulkOrder.quantity')}</span>
        <NumberStepper bind:value={quantity} min={selectedListing.min_order_quantity ?? 1} />
      </label>
      <label class="bulk-order__field">
        <span>{t('bulkOrder.deadline')}</span>
        <input type="date" bind:value={deadline} class="bulk-order__date" />
      </label>
      <label class="bulk-order__field">
        <span>{t('bulkOrder.budgetBand')}</span>
        <Select
          bind:value={budgetBand}
          options={[
            { value: '', label: '—' },
            { value: 'under-1l', label: 'Under ₹1,00,000' },
            { value: '1l-5l', label: '₹1,00,000–5,00,000' },
            { value: 'over-5l', label: 'Over ₹5,00,000' },
          ]}
        />
        <span class="bulk-order__hint">{t('bulkOrder.budgetBand.hint')}</span>
      </label>
      <label class="bulk-order__field">
        <span>{t('bulkOrder.delivery')}</span>
        <Textarea bind:value={delivery} rows={3} />
      </label>
      <div class="bulk-order__actions">
        <Button variant="secondary" onclick={() => (current = 1)} tooltip={tooltip('tooltip.back')}>{t('bulkOrder.back')}</Button>
        <Button onclick={() => (current = 3)} tooltip={tooltip('tooltip.next')}>{t('bulkOrder.next')}</Button>
      </div>
    </section>
  {:else if current === 3 && selectedListing}
    <section>
      <h2>{t('bulkOrder.review.heading')}</h2>
      <dl class="bulk-order__review">
        <div><dt>{t('bulkOrder.step.listing')}</dt><dd>{selectedListing.translations?.[0]?.title}</dd></div>
        <div><dt>{t('bulkOrder.quantity')}</dt><dd>{quantity}</dd></div>
        <div><dt>{t('bulkOrder.deadline')}</dt><dd>{deadline}</dd></div>
      </dl>
      <div class="bulk-order__actions">
        <Button variant="secondary" onclick={() => (current = 2)} tooltip={tooltip('tooltip.back')}>{t('bulkOrder.back')}</Button>
        <Button onclick={() => void submit()} loading={submitting} tooltip={tooltip('tooltip.submit')}>{t('bulkOrder.submit')}</Button>
      </div>
    </section>
  {/if}
{/if}

<style>
  .bulk-order__craft-grid,
  .bulk-order__listing-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    margin-block: var(--k-space-4);
  }

  .bulk-order__craft,
  .bulk-order__listing {
    text-align: start;
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    cursor: pointer;
    min-block-size: var(--k-touch-min);
  }

  .bulk-order__feasibility {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-block-end: var(--k-space-3);
  }

  .bulk-order__preset-banner {
    background: var(--k-surface-raised, var(--k-surface-base));
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-4);
  }

  .bulk-order__form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    max-inline-size: 32rem;
  }

  .bulk-order__field {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .bulk-order__date {
    font: inherit;
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
    min-block-size: var(--k-touch-min);
  }

  .bulk-order__hint {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .bulk-order__review {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-4);
  }

  .bulk-order__review div {
    display: flex;
    justify-content: space-between;
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-end: var(--k-space-2);
  }

  .bulk-order__actions {
    display: flex;
    gap: var(--k-space-3);
  }
</style>
