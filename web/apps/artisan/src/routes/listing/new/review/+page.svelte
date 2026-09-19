<!--
  apps/artisan/src/routes/listing/new/review/+page.svelte

  Step 5 of 7: the artisan checks the generated copy before anything about
  it reaches pricing or publication. Attributes and the description are both
  real now -- GET /listings/{id}/attributes and the listing's own
  translations, written by the cataloguing pipeline the processing step
  triggers (see listing-draft.ts's ensureListingMediaAttachQueued). There is
  no sentence-to-attribute mapping anywhere server-side (GeneratedCopy only
  carries title/description/highlights), so the old tap-a-sentence-to-
  highlight interaction is gone rather than kept fake. The Hindi SpeakButton
  reads the generated description aloud on request, never automatically.
  Approve only sets a local flag that unlocks the rest of the wizard -- the
  real submit/approve calls to the server happen once at the very end, in
  the terms step, per the "nothing publishes without an explicit approve"
  rule applying to the actual publish action, not this local checkpoint.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, matchesLocale, tooltip } from '@kalakriti/i18n';
  import { Button, SpeakButton } from '@kalakriti/ui';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields } from '$lib/listing-draft';

  interface Translation {
    language: string;
    title: string;
    description?: string;
  }
  interface Attribute {
    name: string;
    value: string;
    confidence: number;
    source: 'MODEL' | 'ARTISAN' | 'CURATOR';
  }

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let translations = $state<Translation[]>([]);
  let attributes = $state<Attribute[]>([]);
  let loaded = $state(false);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      translations = (draft.fields.translations as Translation[] | undefined) ?? [];
      attributes = (draft.fields.attributes as Attribute[] | undefined) ?? [];
      loaded = true;
    });
  });

  const enTranslation = $derived(translations.find((tr) => matchesLocale(tr.language, 'en')));
  const hiTranslation = $derived(translations.find((tr) => matchesLocale(tr.language, 'hi')));

  /** "material" -> "Material"; attribute names are free-form from ml-svc, not an i18n-keyed set. */
  function humanize(name: string): string {
    const spaced = name.replace(/_/g, ' ');
    return spaced.charAt(0).toUpperCase() + spaced.slice(1);
  }

  async function approve(): Promise<void> {
    if (!draftId) return;
    await patchFields(draftId, { reviewApproved: true });
    await goto(`/listing/new/pricing?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.review.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={5} heading={t('listing.review.heading')} backHref="/listing/new/processing?d={draftId}">
  {#snippet children()}
    {#if !loaded}
      <p class="review-status" role="status">{t('state.loading')}</p>
    {:else}
      <h2 class="review-title">{enTranslation?.title ?? ''}</h2>
      <p class="review-description">{enTranslation?.description ?? ''}</p>

      {#if hiTranslation?.description}
        <SpeakButton text={hiTranslation.description} tag="hi-IN" label={t('listing.review.listenHindi')} />
      {/if}

      {#if attributes.length > 0}
        <div class="review-attributes">
          <h3>{t('listing.review.attributesHeading')}</h3>
          <ul role="list">
            {#each attributes as attribute (attribute.name)}
              <li class="review-attribute">
                <span class="review-attribute__label">
                  {humanize(attribute.name)}
                  <span class="review-attribute__confidence">{Math.round(attribute.confidence * 100)}%</span>
                </span>
                <span class="review-attribute__value">{attribute.value}</span>
              </li>
            {/each}
          </ul>
        </div>
      {/if}

      <p class="review-note">{t('listing.review.note')}</p>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={!loaded} onclick={approve} tooltip={tooltip('tooltip.approve')}>{t('listing.review.approve')}</Button>
  {/snippet}
</ListingStep>

<style>
  .review-status {
    color: var(--k-text-secondary);
  }

  .review-title {
    font-size: var(--k-text-lg);
  }

  .review-description {
    line-height: 1.6;
  }

  .review-attributes h3 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .review-attribute {
    display: flex;
    justify-content: space-between;
    padding-block: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .review-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
