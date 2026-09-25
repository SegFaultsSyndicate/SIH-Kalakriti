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
  import { db, type MediaRecord } from '@kalakriti/offline';
  import { liveQuery } from 'dexie';
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
  let thumbUrl = $state<string | undefined>(undefined);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      translations = (draft.fields.translations as Translation[] | undefined) ?? [];
      attributes = (draft.fields.attributes as Attribute[] | undefined) ?? [];
      loaded = true;
    });
  });

  // Load uploaded photo thumbnail to show in review
  $effect(() => {
    if (!draftId) return;
    let cancelled = false;
    const sub = liveQuery(async () => {
      const draft = await db.drafts.get(draftId);
      if (!draft) return undefined;
      const media = await db.media.bulkGet(draft.mediaIds);
      return media.find((m): m is MediaRecord => !!m && m.kind === 'photo');
    }).subscribe((photo) => {
      if (cancelled) return;
      if (photo?.blob) {
        thumbUrl = URL.createObjectURL(photo.blob);
      } else {
        thumbUrl = '/craft-images/weaving_and_looms/white-saree-maroon-border.jpeg';
      }
    });
    return () => {
      cancelled = true;
      sub.unsubscribe();
    };
  });

  const enTranslation = $derived(translations.find((tr) => matchesLocale(tr.language, 'en')));
  const hiTranslation = $derived(translations.find((tr) => matchesLocale(tr.language, 'hi')));

  /** Split newline-separated description into bullets for display. */
  const bullets = $derived(
    (enTranslation?.description ?? '')
      .split('\n')
      .map((b) => b.trim())
      .filter((b) => b.length > 0),
  );

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
      <!-- Photo preview -->
      {#if thumbUrl}
        <div class="review-photo">
          <img src={thumbUrl} alt="Uploaded photo" class="review-photo__img" />
          <span class="review-photo__badge">1 photo uploaded</span>
        </div>
      {/if}

      <h2 class="review-title">{enTranslation?.title ?? ''}</h2>

      <!-- About this item bullets (matching what buyer sees) -->
      {#if bullets.length > 0}
        <div class="review-bullets">
          <h3>About this item</h3>
          <ul>
            {#each bullets as bullet}
              <li>{bullet}</li>
            {/each}
          </ul>
        </div>
      {:else}
        <p class="review-description">{enTranslation?.description ?? ''}</p>
      {/if}

      {#if hiTranslation?.description}
        <SpeakButton text={hiTranslation.description} tag="hi-IN" label={t('listing.review.listenHindi')} />
      {/if}

      {#if attributes.length > 0}
        <div class="review-attributes">
          <h3>{t('listing.review.attributesHeading')}</h3>
          <ul role="list">
            <!--
              Keyed on name+value, not name alone: core-svc's
              InferredAttributes.ToListingAttributes flattens every detected
              colour/motif into its own row all sharing the same bare "colour"
              or "motif" name (ml-svc/inference.go) -- any listing with more
              than one of either (the normal case) crashed this whole screen
              with Svelte's each_key_duplicate, caught by the outer error
              boundary as an opaque "Something went wrong". Confirmed live.
            -->
            {#each attributes as attribute (`${attribute.name}:${attribute.value}`)}
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

  .review-photo {
    position: relative;
    margin-block-end: var(--k-space-3);
  }

  .review-photo__img {
    width: 100%;
    max-height: 240px;
    object-fit: cover;
    border-radius: var(--k-radius-md);
  }

  .review-photo__badge {
    position: absolute;
    top: var(--k-space-2);
    right: var(--k-space-2);
    background: rgba(0,0,0,0.55);
    color: #fff;
    font-size: 0.7rem;
    padding: 2px 8px;
    border-radius: 999px;
  }

  .review-title {
    font-size: var(--k-text-lg);
    font-weight: 700;
  }

  .review-bullets h3 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .review-bullets ul {
    padding-inline-start: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .review-bullets li {
    font-size: var(--k-text-sm);
    line-height: 1.5;
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

  .review-attribute__label {
    display: flex;
    gap: var(--k-space-1);
    align-items: center;
  }

  .review-attribute__confidence {
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  .review-attribute__value {
    font-weight: 600;
  }

  .review-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
