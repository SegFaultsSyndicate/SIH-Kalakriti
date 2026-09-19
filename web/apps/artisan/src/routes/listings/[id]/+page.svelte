<!--
  apps/artisan/src/routes/listings/[id]/+page.svelte

  Detail and edit. Translations (title/description) are real --
  ListingTranslation carries machine_generated, and PATCH /listings/{id}
  replaces the array wholesale. Editing one here sets machine_generated:false
  before saving, which is exactly core-svc's own artisan-wins rule
  (UpsertListingTranslation only ever stamps EditedBy when !MachineGenerated
  -- this screen mirrors that server-side precedence, it doesn't invent it).

  Attributes are real too now -- GET /listings/{id}/attributes, written by
  the cataloguing pipeline (see listing-draft.ts's ensureListingMediaAttachQueued
  and the processing step that triggers it). Fetched fresh from the server
  on every load, not read from a local draft, so this screen works from any
  device, not just the one the listing was created on. Read-only here: there
  is no UpsertListingAttributes write-back route wired to the BFF yet, so an
  editable input would silently discard whatever the artisan typed. No
  "regenerate" action either -- there is no real regenerate endpoint, only
  the one-shot pipeline run that already happened at creation time.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, FieldGroup, Input, Textarea, NumberStepper, Money, SpeakButton, showToast } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { ApiError, advisePricing, getListingAttributes, updateListing, type components } from '@kalakriti/api';
  import StateBadge from '$lib/StateBadge.svelte';
  import PriceAdvisory from '$lib/PriceAdvisory.svelte';
  import GemExportPreview from '$lib/GemExportPreview.svelte';
  import { fetchListing, groupFor, network, type Listing } from '$lib/listings';
  import { getDraft as getRegistrationDraft } from '$lib/registration';

  interface Attribute {
    name: string;
    value: string;
    confidence: number;
    source: 'MODEL' | 'ARTISAN' | 'CURATOR';
  }

  const t = $derived(locale.t);
  const listingId = $derived(page.params.id ?? '');

  let listing = $state<Listing | undefined>(undefined);
  let loading = $state(true);
  let loadError = $state<string | undefined>(undefined);

  let translations = $state<NonNullable<Listing['translations']>>([]);
  let savingTranslations = $state(false);

  let attributes = $state<Attribute[]>([]);

  /** "material" -> "Material"; attribute names are free-form from ml-svc, not an i18n-keyed set. */
  function humanize(name: string): string {
    const spaced = name.replace(/_/g, ' ');
    return spaced.charAt(0).toUpperCase() + spaced.slice(1);
  }

  let priceRupees = $state('');
  let materialCostRupees = $state('');
  let hours = $state(0);
  let advice = $state<components['schemas']['PriceAdvisory'] | undefined>(undefined);
  let advising = $state(false);
  let savingPrice = $state(false);

  let gemOpen = $state(false);
  let gemSellerName = $state('');
  let gemSellerDistrict = $state('');
  let gemSellerId = $state('');

  async function load(): Promise<void> {
    if (!listingId) return;
    loading = true;
    loadError = undefined;
    try {
      listing = await fetchListing(listingId);
      translations = structuredClone(listing.translations ?? []);
      priceRupees = listing.price?.amount_paise ? String(listing.price.amount_paise / 100) : '';
    } catch (cause) {
      loadError = cause instanceof ApiError ? cause.message : t('api.error.unknown');
    } finally {
      loading = false;
    }

    const attrResponse = await getListingAttributes(listingId).catch(() => undefined);
    attributes = (attrResponse?.attributes ?? []).map((a) => ({
      name: a.name ?? '',
      value: a.value ?? '',
      confidence: a.confidence ?? 0,
      source: (a.source ?? 'MODEL') as Attribute['source'],
    }));

    // Load registration details for the GeM export card (seller info).
    const regDraft = await getRegistrationDraft();
    gemSellerName = regDraft.name ?? '';
    gemSellerDistrict = regDraft.districtFreeText ?? regDraft.districtId ?? '';
    gemSellerId = regDraft.pehchanId ?? '';
  }

  $effect(() => {
    void load();
  });

  function markEdited(index: number): void {
    translations[index] = { ...translations[index], machine_generated: false };
  }

  async function saveTranslations(): Promise<void> {
    if (!listingId) return;
    savingTranslations = true;
    try {
      await updateListing(listingId, { translations });
      showToast({ variant: 'success', message: t('listings.detail.saved') });
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? cause.message : t('api.error.unknown') });
    } finally {
      savingTranslations = false;
    }
  }

  function toPaise(rupees: string): number | undefined {
    const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(rupees.trim());
    if (!match) return undefined;
    const [, whole, fraction = ''] = match;
    const amount = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
    return amount > 0 ? amount : undefined;
  }
  const priceAmountPaise = $derived(toPaise(priceRupees));

  async function savePrice(): Promise<void> {
    if (!listingId || priceAmountPaise === undefined) return;
    savingPrice = true;
    try {
      await updateListing(listingId, { price: { amount_paise: priceAmountPaise } });
      if (listing) listing = { ...listing, price: { amount_paise: priceAmountPaise, currency_code: 'INR' } };
      showToast({ variant: 'success', message: t('listings.detail.saved') });
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? cause.message : t('api.error.unknown') });
    } finally {
      savingPrice = false;
    }
  }

  async function getAdvice(): Promise<void> {
    if (!listingId) return;
    const materialCost = toPaise(materialCostRupees);
    if (materialCost === undefined || hours <= 0) return;
    advising = true;
    try {
      advice = await advisePricing({
        listing_id: listingId,
        material_cost: { amount_paise: materialCost },
        hours,
        ...(priceAmountPaise !== undefined ? { chosen_price: { amount_paise: priceAmountPaise } } : {}),
      });
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? cause.message : t('api.error.unknown') });
    } finally {
      advising = false;
    }
  }
</script>

<svelte:head>
  <title>{listing ? (translations[0]?.title ?? t('listings.untitled')) : t('listings.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="listing-detail">
  <a href="/listings" class="listing-detail__back">
    <Icon name="arrow-left" />
    {t('listings.detail.back')}
  </a>

  {#if loading}
    <Skeleton shape="card" height="8rem" />
  {:else if loadError}
    <p class="listing-detail__error" role="alert">{loadError}</p>
  {:else if listing}
    <header class="listing-detail__header">
      <StateBadge group={groupFor(listing)} />
      {#if listing.price?.amount_paise != null}
        <Money paise={listing.price.amount_paise} />
      {/if}
      {#if listing.state === 'PUBLISHED'}
        <a href="/listings/{listing.id}/provenance">{t('listings.detail.provenanceLink')}</a>
        <Button size="sm" variant="secondary" onclick={() => (gemOpen = true)} tooltip={tooltip('tooltip.viewGem')}>
          <Icon name="external-link" />
          {t('gem.exportButton')}
        </Button>
      {/if}
    </header>

    <section class="listing-detail__section">
      <div class="listing-detail__section-header">
        <h2>{t('listings.detail.copyHeading')}</h2>
      </div>

      {#each translations as translation, i (translation.language)}
        <Card variant="hairline" element="div" class="listing-detail__translation">
          <div class="listing-detail__translation-header">
            <p class="listing-detail__language">{translation.language}</p>
            <span
              class="listing-detail__provenance-pill"
              class:listing-detail__provenance-pill--ai={translation.machine_generated}
            >
              {translation.machine_generated ? t('listings.detail.aiGenerated') : t('listings.detail.artisanAuthored')}
            </span>
            <SpeakButton
              text={`${translation.title}. ${translation.description ?? ''}`}
              tag={translation.language}
              label={t('listings.detail.listen')}
              iconOnly
            />
          </div>
          <FieldGroup label={t('listings.detail.titleLabel')}>
            {#snippet children({ id })}
              <Input
                {id}
                bind:value={translation.title}
                oninput={() => markEdited(i)}
              />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('listings.detail.descriptionLabel')}>
            {#snippet children({ id })}
              <Textarea
                {id}
                bind:value={translation.description}
                oninput={() => markEdited(i)}
              />
            {/snippet}
          </FieldGroup>
        </Card>
      {/each}

      <Button size="sm" loading={savingTranslations} onclick={saveTranslations} tooltip={tooltip('tooltip.saveTranslations')}>
        {t('listings.detail.saveCopy')}
      </Button>
    </section>

    {#if attributes.length > 0}
      <section class="listing-detail__section">
        <h2>{t('listings.detail.attributesHeading')}</h2>
        <p class="listing-detail__attributes-note">{t('listings.detail.attributesNote')}</p>
        <ul class="listing-detail__attributes" role="list">
          {#each attributes as attribute (attribute.name)}
            <li class="listing-detail__attribute">
              <div class="listing-detail__attribute-label">
                <p>{humanize(attribute.name)}</p>
                <span
                  class="listing-detail__provenance-pill"
                  class:listing-detail__provenance-pill--ai={attribute.source !== 'ARTISAN'}
                >
                  {attribute.source === 'ARTISAN' ? t('listings.detail.artisanAuthored') : t('listings.detail.aiGuessed')}
                </span>
              </div>
              <p class="listing-detail__attribute-value">{attribute.value}</p>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <section class="listing-detail__section">
      <h2>{t('listings.detail.priceHeading')}</h2>
      <FieldGroup label={t('listing.pricing.priceLabel')}>
        {#snippet children({ id })}
          <Input {id} type="tel" bind:value={priceRupees} placeholder="0" />
        {/snippet}
      </FieldGroup>
      <Button size="sm" loading={savingPrice} disabled={priceAmountPaise === undefined} onclick={savePrice} tooltip={tooltip('tooltip.savePrice')}>
        {t('listings.detail.savePrice')}
      </Button>

      {#if !network.online}
        <p class="listing-detail__offline-note">{t('listing.pricing.adviceNeedsOnline')}</p>
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
        <Button size="sm" variant="secondary" loading={advising} onclick={getAdvice} tooltip={tooltip('tooltip.getPriceAdvice')}>
          {t('listing.pricing.adviceButton')}
        </Button>
        {#if advice}
          <PriceAdvisory {advice} />
        {/if}
      {/if}
    </section>

    {#if listing.state === 'PUBLISHED'}
      <GemExportPreview
        bind:open={gemOpen}
        {listing}
        sellerName={gemSellerName}
        sellerDistrict={gemSellerDistrict}
        sellerId={gemSellerId}
      />
    {/if}
  {/if}
</main>

<style>
  .listing-detail {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .listing-detail__back {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .listing-detail__error {
    color: var(--k-accent-danger);
  }

  .listing-detail__header {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    flex-wrap: wrap;
  }

  .listing-detail__section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .listing-detail__section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .listing-detail__section h2 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  :global(.listing-detail__translation) {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
  }

  .listing-detail__translation-header {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .listing-detail__language {
    text-transform: uppercase;
    font-size: var(--k-text-sm);
    font-weight: 700;
    color: var(--k-text-secondary);
  }

  /*
   * The one visual rule this whole screen exists to make legible: model
   * output looks provisional (dashed, muted); an artisan's own words look
   * settled (solid, full-contrast). Same two states, translations or
   * attributes -- one pill style for both.
   */
  .listing-detail__provenance-pill {
    padding-block: 2px;
    padding-inline: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-text-primary);
    border-radius: var(--k-radius-pill);
    font-size: var(--k-text-xs, 0.75rem);
    font-weight: 600;
  }

  .listing-detail__provenance-pill--ai {
    border-style: dashed;
    border-color: var(--k-text-secondary);
    color: var(--k-text-secondary);
  }

  .listing-detail__attributes-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .listing-detail__attributes {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .listing-detail__attribute {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .listing-detail__attribute-label {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .listing-detail__attribute-value {
    color: var(--k-text-secondary);
  }

  .listing-detail__offline-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
