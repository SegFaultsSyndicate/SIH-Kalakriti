<!--
  apps/artisan/src/routes/listing/new/pricing/+page.svelte

  Step 6 of 7. Type, price, quantities. "Get fair price advice" calls the
  real POST /pricing/advise (not mocked -- see ml_wiring.md) directly, not
  through the outbox: it's a read, and it only works once the listing has a
  remote id and the device is online, so it degrades to "just enter a
  price" rather than blocking the step.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, FieldGroup, Input, NumberStepper, Money } from '@kalakriti/ui';
  import { network } from '@kalakriti/offline';
  import { advisePricing, ApiError, type components } from '@kalakriti/api';
  import ListingStep from '$lib/ListingStep.svelte';
  import PriceAdvisory from '$lib/PriceAdvisory.svelte';
  import SahayakTooltip from '$lib/SahayakTooltip.svelte';
  import { getDraft, patchFields, queueListingUpdate, type ListingDraftFields } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let type = $state<'READY_STOCK' | 'MADE_TO_ORDER'>('MADE_TO_ORDER');
  let priceRupees = $state('');
  let stockQuantity = $state(1);
  let minOrderQuantity = $state(1);
  let materialCostRupees = $state('');
  let hours = $state(0);
  let remoteId = $state<string | undefined>(undefined);

  let advisory = $state<components['schemas']['PriceAdvisory'] | undefined>(undefined);
  let advisoryError = $state<string | undefined>(undefined);
  let advising = $state(false);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      const f = draft.fields as ListingDraftFields;
      type = f.type ?? 'MADE_TO_ORDER';
      priceRupees = f.priceAmountPaise ? String(f.priceAmountPaise / 100) : '';
      stockQuantity = f.stockQuantity ?? 1;
      minOrderQuantity = f.minOrderQuantity ?? 1;
      remoteId = draft.remoteId;
    });
  });

  // String-digit parsing rather than Number(x) * 100: float multiplication
  // on rupee input (e.g. 449.35 * 100) can land a paise value off by one --
  // unacceptable on a money field. This is exact for any input the regex admits.
  function toPaise(rupees: string): number | undefined {
    const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(rupees.trim());
    if (!match) return undefined;
    const [, whole, fraction = ''] = match;
    const amount = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
    return amount > 0 ? amount : undefined;
  }

  const priceAmountPaise = $derived(toPaise(priceRupees));

  async function getAdvice(): Promise<void> {
    if (!remoteId) return;
    const materialCost = toPaise(materialCostRupees);
    if (materialCost === undefined || hours <= 0) return;
    advising = true;
    advisoryError = undefined;
    try {
      advisory = await advisePricing({
        listing_id: remoteId,
        material_cost: { amount_paise: materialCost },
        hours,
        // Lets the server run its own below-floor/above-ceiling check
        // (CheckAnomaly) against what the artisan has actually typed, rather
        // than this screen re-deriving that judgement client-side.
        ...(priceAmountPaise !== undefined ? { chosen_price: { amount_paise: priceAmountPaise } } : {}),
      });
    } catch (cause) {
      advisoryError = cause instanceof ApiError ? cause.message : t('api.error.unknown');
    } finally {
      advising = false;
    }
  }

  async function next(): Promise<void> {
    if (!draftId || priceAmountPaise === undefined) return;
    await patchFields(draftId, { type, priceAmountPaise, stockQuantity, minOrderQuantity });
    await queueListingUpdate(draftId, {
      type,
      price: { amount_paise: priceAmountPaise },
      stock_quantity: type === 'READY_STOCK' ? stockQuantity : undefined,
      min_order_quantity: minOrderQuantity,
    });
    await goto(`/listing/new/terms?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.pricing.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={6} heading={t('listing.pricing.heading')} backHref="/listing/new/review?d={draftId}">
  {#snippet children()}
    <div class="type-toggle" role="radiogroup" aria-label={t('listing.pricing.typeLabel')}>
      <button
        type="button"
        role="radio"
        aria-checked={type === 'MADE_TO_ORDER'}
        class="type-toggle__option"
        class:type-toggle__option--selected={type === 'MADE_TO_ORDER'}
        onclick={() => (type = 'MADE_TO_ORDER')}
      >
        <Icon name="made-to-order" />
        {t('listing.pricing.madeToOrder')}
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={type === 'READY_STOCK'}
        class="type-toggle__option"
        class:type-toggle__option--selected={type === 'READY_STOCK'}
        onclick={() => (type = 'READY_STOCK')}
      >
        <Icon name="ready-stock" />
        {t('listing.pricing.readyStock')}
      </button>
    </div>

    <FieldGroup label={t('listing.pricing.priceLabel')}>
      {#snippet children({ id })}
        <Input {id} type="tel" bind:value={priceRupees} placeholder="0" />
      {/snippet}
    </FieldGroup>
    {#if priceAmountPaise !== undefined}
      <Money paise={priceAmountPaise} />
    {/if}

    {#if type === 'READY_STOCK'}
      <FieldGroup label={t('listing.pricing.stockLabel')}>
        {#snippet children({ id })}
          <NumberStepper {id} bind:value={stockQuantity} min={1} />
        {/snippet}
      </FieldGroup>
    {/if}

    <FieldGroup label={t('listing.pricing.minOrderLabel')}>
      {#snippet children({ id })}
        <NumberStepper {id} bind:value={minOrderQuantity} min={1} />
      {/snippet}
    </FieldGroup>

    <SahayakTooltip text={t('literacy.sahayak.tooltip.pricing')} />

    <div class="advisory">
      <h3>{t('listing.pricing.adviceHeading')}</h3>
      {#if !remoteId}
        <p class="advisory__note">{t('listing.pricing.adviceNeedsSync')}</p>
      {:else if !network.online}
        <p class="advisory__note">{t('listing.pricing.adviceNeedsOnline')}</p>
      {:else}
        <FieldGroup label={t('listing.pricing.materialCostLabel')}>
          {#snippet children({ id })}
            <Input {id} type="tel" bind:value={materialCostRupees} placeholder="0" />
          {/snippet}
        </FieldGroup>
        <FieldGroup label={t('listing.pricing.hoursLabel')}>
          {#snippet children({ id })}
            <NumberStepper {id} bind:value={hours} min={0} />
          {/snippet}
        </FieldGroup>
        <Button size="sm" variant="secondary" loading={advising} onclick={getAdvice}>
          {t('listing.pricing.adviceButton')}
        </Button>
        {#if advisory}
          <PriceAdvisory advice={advisory} />
        {/if}
        {#if advisoryError}
          <p class="advisory__error" role="alert">{advisoryError}</p>
        {/if}
      {/if}
    </div>
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={priceAmountPaise === undefined} onclick={next}>{t('action.next')}</Button>
  {/snippet}
</ListingStep>

<style>
  .type-toggle {
    display: flex;
    gap: var(--k-space-2);
  }

  .type-toggle__option {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    min-block-size: calc(var(--k-touch-min) * 1.2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .type-toggle__option--selected {
    border-color: var(--k-accent-primary-bg);
    border-width: var(--k-rule);
    background-color: var(--k-surface-sunken);
  }

  .advisory {
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .advisory h3 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .advisory__note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .advisory__error {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }
</style>
