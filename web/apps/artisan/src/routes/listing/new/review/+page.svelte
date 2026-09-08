<!--
  apps/artisan/src/routes/listing/new/review/+page.svelte

  Step 5 of 7: the artisan checks the generated copy before anything about
  it reaches pricing or publication. Tapping a sentence highlights the
  attribute it claims to be based on (claims/attributes are MOCK -- see
  ml_wiring.md); the Hindi SpeakButton reads the generated description
  aloud on request, never automatically. Approve only sets a local flag
  that unlocks the rest of the wizard -- the real submit/approve calls to
  the server happen once at the very end, in the terms step, per the
  "nothing publishes without an explicit approve" rule applying to the
  actual publish action, not this local checkpoint.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Button, Input, SpeakButton } from '@kalakriti/ui';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields } from '$lib/listing-draft';

  interface Translation {
    language: string;
    title: string;
    description?: string;
  }
  interface Attribute {
    key: string;
    labelKey: MessageKey;
    value: string;
    source: 'MODEL' | 'ARTISAN';
    confidence: number;
    needs_artisan_input: boolean;
  }
  interface Claim {
    sentenceIndex: number;
    attributeKey: string;
  }

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let translations = $state<Translation[]>([]);
  let attributes = $state<Attribute[]>([]);
  let claims = $state<Claim[]>([]);
  let highlighted = $state<string | undefined>(undefined);
  let loaded = $state(false);

  $effect(() => {
    if (!draftId) return;
    void getDraft(draftId).then((draft) => {
      if (!draft) return;
      translations = (draft.fields.translations as Translation[] | undefined) ?? [];
      attributes = (draft.fields.attributes as Attribute[] | undefined) ?? [];
      claims = (draft.fields.claims as Claim[] | undefined) ?? [];
      loaded = true;
    });
  });

  const enTranslation = $derived(translations.find((tr) => tr.language === 'en'));
  const hiTranslation = $derived(translations.find((tr) => tr.language === 'hi'));
  const sentences = $derived(
    (enTranslation?.description ?? '').split(/(?<=[.!?])\s+/).filter((s) => s.length > 0),
  );

  function attributeFor(index: number): string | undefined {
    return claims.find((c) => c.sentenceIndex === index)?.attributeKey;
  }

  function tapSentence(index: number): void {
    const attr = attributeFor(index);
    if (!attr) return;
    highlighted = highlighted === attr ? undefined : attr;
  }

  async function approve(): Promise<void> {
    if (!draftId) return;
    await patchFields(draftId, { reviewApproved: true });
    await goto(`/listing/new/pricing?d=${draftId}`);
  }

  async function editAttribute(attribute: Attribute, value: string): Promise<void> {
    attribute.value = value;
    attribute.needs_artisan_input = false;
    await patchFields(draftId, { attributes });
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

      <p class="review-description">
        {#each sentences as sentence, index (index)}
          <button
            type="button"
            class="review-sentence"
            class:review-sentence--highlighted={attributeFor(index) === highlighted && highlighted !== undefined}
            class:review-sentence--tappable={attributeFor(index) !== undefined}
            onclick={() => tapSentence(index)}
          >
            {sentence}
          </button>{' '}
        {/each}
      </p>

      {#if hiTranslation?.description}
        <SpeakButton text={hiTranslation.description} tag="hi-IN" label={t('listing.review.listenHindi')} />
      {/if}

      <div class="review-attributes">
        <h3>{t('listing.review.attributesHeading')}</h3>
        <ul role="list">
          {#each attributes as attribute (attribute.key)}
            <li class="review-attribute" class:review-attribute--highlighted={attribute.key === highlighted}>
              <span class="review-attribute__label">
                {t(attribute.labelKey)}
                <span class="review-attribute__confidence">{Math.round(attribute.confidence * 100)}%</span>
                {#if attribute.needs_artisan_input}
                  <span class="review-attribute__needs-input">{t('listing.review.note')}</span>
                {/if}
              </span>
              {#if attribute.needs_artisan_input}
                <Input
                  value={attribute.value}
                  oninput={(event: Event) =>
                    editAttribute(attribute, (event.currentTarget as HTMLInputElement).value)}
                />
              {:else}
                <span class="review-attribute__value">{attribute.value}</span>
              {/if}
            </li>
          {/each}
        </ul>
      </div>

      <p class="review-note">{t('listing.review.note')}</p>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={!loaded} onclick={approve}>{t('listing.review.approve')}</Button>
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

  .review-sentence {
    border: none;
    background: none;
    padding: 0;
    margin: 0;
    font: inherit;
    color: inherit;
    cursor: default;
  }

  .review-sentence--tappable {
    cursor: pointer;
    text-decoration: underline dotted;
  }

  .review-sentence--highlighted {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-radius: var(--k-radius-sm);
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

  .review-attribute--highlighted {
    background-color: var(--k-surface-sunken);
  }

  .review-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
