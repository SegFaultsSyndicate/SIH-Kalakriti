<!--
  apps/artisan/src/routes/listing/new/terms/+page.svelte

  Step 7 of 7, the last one. Publish is disabled until both reviewApproved
  (set by the explicit Approve on the review screen) and termsAccepted (this
  screen's own checkbox) are true -- two separate deliberate actions, not
  one checkbox standing in for both. Publish queues the real submit+approve
  pair (see listing-draft.ts's publishListing) and returns home; nothing
  here publishes synchronously, so the wizard's very last screen still works
  with no connection -- the outbox just carries the intent until one exists.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Button, FieldGroup, NumberStepper, Switch, Checkbox, showToast } from '@kalakriti/ui';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields, queueListingUpdate, publishListing, type ListingDraftFields } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let type = $state<'READY_STOCK' | 'MADE_TO_ORDER'>('MADE_TO_ORDER');
  let reviewApproved = $state(false);
  let leadTimeDays = $state(7);
  let capacityPerMonth = $state(10);
  let acceptingOrders = $state(true);
  let advancePct = $state(0);
  let fragile = $state(false);
  let oversized = $state(false);
  let requiresCustomCrating = $state(false);
  let termsAccepted = $state(false);
  let publishing = $state(false);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      const f = draft.fields as ListingDraftFields;
      type = f.type ?? 'MADE_TO_ORDER';
      reviewApproved = f.reviewApproved ?? false;
      leadTimeDays = f.madeToOrderTerms?.lead_time_days ?? 7;
      capacityPerMonth = f.madeToOrderTerms?.capacity_per_month ?? 10;
      acceptingOrders = f.madeToOrderTerms?.accepting_orders ?? true;
      advancePct = f.madeToOrderTerms?.advance_pct ?? 0;
      fragile = f.packaging?.fragile ?? false;
      oversized = f.packaging?.oversized ?? false;
      requiresCustomCrating = f.packaging?.requires_custom_crating ?? false;
    });
  });

  async function publish(): Promise<void> {
    if (!draftId || !reviewApproved || !termsAccepted || publishing) return;
    publishing = true;
    try {
      const madeToOrderTerms =
        type === 'MADE_TO_ORDER'
          ? {
              lead_time_days: leadTimeDays,
              capacity_per_month: capacityPerMonth,
              accepting_orders: acceptingOrders,
              advance_pct: advancePct,
            }
          : undefined;
      const packaging = { fragile, oversized, requires_custom_crating: requiresCustomCrating };

      await patchFields(draftId, { madeToOrderTerms, packaging, termsAccepted: true });
      await queueListingUpdate(draftId, { made_to_order_terms: madeToOrderTerms, packaging });
      await publishListing(draftId);
      showToast({ message: t('listing.terms.queued'), variant: 'success' });
      await goto('/');
    } finally {
      publishing = false;
    }
  }
</script>

<svelte:head>
  <title>{t('listing.terms.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={6} heading={t('listing.terms.heading')} backHref="/listing/new/pricing?d={draftId}">
  {#snippet children()}
    {#if type === 'MADE_TO_ORDER'}
      <FieldGroup label={t('listing.terms.leadTimeLabel')}>
        {#snippet children({ id })}
          <NumberStepper {id} bind:value={leadTimeDays} min={1} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('listing.terms.capacityLabel')}>
        {#snippet children({ id })}
          <NumberStepper {id} bind:value={capacityPerMonth} min={1} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('listing.terms.advanceLabel')}>
        {#snippet children({ id })}
          <NumberStepper {id} bind:value={advancePct} min={0} max={100} />
        {/snippet}
      </FieldGroup>
      <Switch bind:checked={acceptingOrders}>{t('listing.terms.acceptingOrders')}</Switch>
    {/if}

    <fieldset class="packaging">
      <legend>{t('listing.terms.packagingHeading')}</legend>
      <Checkbox bind:checked={fragile}>{t('listing.terms.fragile')}</Checkbox>
      <Checkbox bind:checked={oversized}>{t('listing.terms.oversized')}</Checkbox>
      <Checkbox bind:checked={requiresCustomCrating}>{t('listing.terms.customCrating')}</Checkbox>
    </fieldset>

    {#if !reviewApproved}
      <p class="terms-warning" role="alert">{t('listing.terms.needsApproval')}</p>
    {/if}

    <Checkbox bind:checked={termsAccepted}>{t('listing.terms.confirm')}</Checkbox>
  {/snippet}
  {#snippet actions()}
    <Button
      size="xl"
      loading={publishing}
      disabled={!reviewApproved || !termsAccepted}
      onclick={publish}
    >
      {t('listing.terms.publish')}
    </Button>
  {/snippet}
</ListingStep>

<style>
  .packaging {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    border: none;
    padding: 0;
  }

  .packaging legend {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    padding: 0;
    margin-block-end: var(--k-space-2);
  }

  .terms-warning {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }
</style>
