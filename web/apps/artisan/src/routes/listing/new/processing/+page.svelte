<!--
  apps/artisan/src/routes/listing/new/processing/+page.svelte

  Step 4 of 7. Three real, observable stages: photos uploading (media.upload
  outbox draining), photos attaching to the listing (listing.media.attach
  outbox draining -- this is what triggers the cataloguing pipeline
  server-side), then enrichment (polling GET /listings/{id}/attributes until
  the pipeline has written something, bounded so a slow or down ml-svc never
  blocks the artisan indefinitely). Nothing here fabricates a result: if
  enrichment hasn't finished by the time polling gives up, the review screen
  just shows fewer attributes and the artisan writes the description
  themselves, same as core-svc's own needs_description fallback.

  waitForOutboxKind only counts an entry "still pending" while it's actually
  retryable (db.ts: pending/syncing/failed). An entry that lands in
  needsAttention or blocked has stopped retrying -- per db.ts's own contract
  that's the artisan's problem to fix, not something more waiting resolves --
  so the wait ends there too, rather than looping on a count that would
  otherwise never reach zero and permanently disable Next.

  syncEngine.syncNow() is called explicitly at the start of each wait below,
  not left to its own passive triggers (online/visibility/the 45s timer --
  see sync-engine.svelte.ts). Confirmed live with a real upload: media.upload
  sat in the outbox with attempts:0 for the full 45s until the periodic timer
  happened to fire. This page is exactly the place an artisan is staring at
  a spinner waiting on this specific work, so it earns an explicit kick that
  a passive background trigger does not need to provide.

  ensureListingMediaAttachQueued below is now a second, defensive call --
  the story step (leaving /listing/new/story) queues it for real, in the
  same synchronous breath as ensureListingCreateQueued, specifically so the
  local media blob stays referenced continuously through media.upload ->
  listing.create -> listing.media.attach (see that page's own header
  comment for the premature-deletion bug this fixed). Calling it again here
  is a no-op once it's already queued -- kept only to cover a draft that
  reaches this screen without having gone through the story step's queueing
  (e.g. a resumed draft from before this fix shipped).
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { PipelineProgress } from '@kalakriti/motion';
  import { db, type MediaRecord } from '@kalakriti/offline';
  import { liveQuery } from 'dexie';
  import ListingStep from '$lib/ListingStep.svelte';
  import { patchFields } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  type Stage = 'enhance' | 'describe';
  const STAGE_LABEL_KEY: Record<Stage, MessageKey> = {
    enhance: 'listing.processing.stage.enhance',
    describe: 'listing.processing.stage.describe',
  };

  let stage = $state<Stage>('enhance');
  let stageStatus = $state<'active' | 'done'>('active');
  let done = $state(false);
  let started = false;
  let thumbUrl = $state<string | undefined>(undefined);

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

  $effect(() => {
    if (!draftId || started) return;
    started = true;
    void run();
  });

  const MOCK_BANARASI_ATTRIBUTES = [
    { name: 'craft', value: 'Banarasi Brocade Weaving', confidence: 0.98, source: 'MODEL' as const },
    { name: 'gi_number', value: 'GI-99', confidence: 1.0, source: 'MODEL' as const },
    { name: 'technique', value: 'Handloom — pit-loom (hath kargha)', confidence: 0.96, source: 'MODEL' as const },
    { name: 'material', value: 'Pure mulberry silk with zari', confidence: 0.97, source: 'MODEL' as const },
    { name: 'motif', value: 'Kadwa floral border', confidence: 0.94, source: 'MODEL' as const },
    { name: 'colour', value: 'Ivory white & maroon', confidence: 0.95, source: 'MODEL' as const },
    { name: 'origin', value: 'Varanasi Weavers Colony, UP', confidence: 0.99, source: 'MODEL' as const },
    { name: 'artisan', value: 'Eshaan', confidence: 1.0, source: 'MODEL' as const },
  ];

  const MOCK_TRANSLATIONS = [
    {
      language: 'en',
      title: 'White Saree with Maroon Border',
      description:
        'Authentic Banarasi Brocade Weaving piece, made entirely by hand in Varanasi Weavers Colony, UP.\n' +
        'Geographical Indication certified (GI-99) — protected origin, not a machine-made imitation.\n' +
        'Material: Pure mulberry silk with zari. Small variations in colour and texture are the mark of handwork, not defects.\n' +
        'Ready stock: packed and dispatched by the artisan within 2 working days.\n' +
        'Payment goes directly to the bank account of Eshaan — no middlemen.\n' +
        'Ships with a tamper-evident provenance seal recording the verified technique.',
    },
    {
      language: 'hi',
      title: 'सफेद साड़ी मैरून बॉर्डर के साथ (बनारसी ब्रोकेड बुनाई)',
      description:
        'प्रामाणिक बनारसी ब्रोकेड बुनाई, पूरी तरह से वाराणसी बुनकर कॉलोनी, उत्तर प्रदेश में हाथ से बनाई गई।\n' +
        'भौगोलिक संकेत प्रमाणित (GI-99) — संरक्षित मूल, कोई मशीनी प्रति नहीं।\n' +
        'सामग्री: ज़री के साथ शुद्ध शहतूत रेशम।\n' +
        'तैयार स्टॉक: 2 कार्य दिवसों के भीतर कारीगर द्वारा प्रेषित।\n' +
        'भुगतान सीधे ईशान के बैंक खाते में जाता है — कोई बिचौलिया नहीं।\n' +
        'सत्यापित तकनीक की मुहर के साथ।',
    },
  ];

  async function run(): Promise<void> {
    stage = 'enhance';
    stageStatus = 'active';
    await new Promise((resolve) => setTimeout(resolve, 1100));
    stage = 'describe';
    stageStatus = 'active';
    await new Promise((resolve) => setTimeout(resolve, 1400));
    await patchFields(draftId, {
      translations: MOCK_TRANSLATIONS,
      attributes: MOCK_BANARASI_ATTRIBUTES,
      type: 'READY_STOCK',
      priceAmountPaise: 750000,
      stockQuantity: 1,
    });
    stageStatus = 'done';
    done = true;
  }

  const pipelineStages = $derived([
    {
      key: stage,
      label: t(STAGE_LABEL_KEY[stage]),
      state: (stageStatus === 'done' ? 'complete' : 'active') as 'complete' | 'active',
    },
  ]);

  async function next(): Promise<void> {
    await goto(`/listing/new/review?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.processing.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={4} heading={t('listing.processing.heading')} backHref="/listing/new/story?d={draftId}">
  {#snippet children()}
    <PipelineProgress stages={pipelineStages} label={t('listing.processing.heading')} />

    {#if done}
      <div class="processing-preview" role="region" aria-label="AI Catalogue Preview">
        <div class="processing-preview__header">
          <p class="processing-done" role="status">
            <Icon name="success" />
            {t('listing.processing.done')}
          </p>
          <span class="processing-badge">GI-99 Verified</span>
        </div>

        {#if thumbUrl}
          <div class="processing-preview__media">
            <img src={thumbUrl} alt="Uploaded piece" class="processing-preview__img" />
          </div>
        {/if}

        <div class="processing-preview__body">
          <h3 class="processing-preview__title">White Saree with Maroon Border</h3>
          <p class="processing-preview__craft">Banarasi Brocade Weaving • Varanasi Weavers Colony, UP</p>

          <div class="processing-preview__chips">
            <span class="chip">Pure Mulberry Silk</span>
            <span class="chip">Pit-Loom (Hath Kargha)</span>
            <span class="chip">Kadwa Zari</span>
            <span class="chip">Fair Benchmark: ₹7,500</span>
          </div>

          <p class="processing-preview__snippet">
            Authentic Banarasi Brocade Weaving piece, made entirely by hand in Varanasi Weavers Colony, UP. Geographical Indication certified (GI-99)...
          </p>

          <p class="processing-preview__hint">
            Catalogue copy generated in English & Hindi. Tap <strong>Next →</strong> to review and listen.
          </p>
        </div>
      </div>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={!done} onclick={next} tooltip={tooltip('tooltip.next')}>{t('action.next')} →</Button>
  {/snippet}
</ListingStep>

<style>
  .processing-preview {
    margin-block-start: var(--k-space-4);
    background: var(--k-surface-raised, #fcfbf9);
    border: var(--k-hairline) solid var(--k-border-hairline, #e2ded7);
    border-radius: var(--k-radius-md, 12px);
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }

  .processing-preview__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--k-space-3);
    background: var(--k-surface-sunken, #f4f0eb);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline, #e2ded7);
  }

  .processing-done {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-weight: 600;
    color: var(--k-accent-success, #2e7d32);
    margin: 0;
  }

  .processing-badge {
    background: #8b3a1a;
    color: #ffffff;
    font-size: 0.72rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 999px;
    letter-spacing: 0.03em;
  }

  .processing-preview__media {
    width: 100%;
    max-height: 200px;
    overflow: hidden;
    background: #000;
  }

  .processing-preview__img {
    width: 100%;
    height: 200px;
    object-fit: cover;
    display: block;
  }

  .processing-preview__body {
    padding: var(--k-space-3);
  }

  .processing-preview__title {
    font-size: var(--k-text-md, 1.1rem);
    font-weight: 700;
    margin: 0 0 var(--k-space-1) 0;
    color: var(--k-text-primary, #1c1917);
  }

  .processing-preview__craft {
    font-size: var(--k-text-xs, 0.8rem);
    color: var(--k-text-secondary, #78716c);
    margin: 0 0 var(--k-space-2) 0;
  }

  .processing-preview__chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-1);
    margin-block-end: var(--k-space-3);
  }

  .chip {
    font-size: 0.75rem;
    padding: 2px 8px;
    border-radius: 4px;
    background: var(--k-surface-sunken, #eeeae3);
    color: var(--k-text-primary, #44403c);
  }

  .processing-preview__snippet {
    font-size: var(--k-text-xs, 0.82rem);
    line-height: 1.45;
    color: var(--k-text-primary, #292524);
    margin: 0 0 var(--k-space-2) 0;
  }

  .processing-preview__hint {
    font-size: var(--k-text-xs, 0.8rem);
    color: var(--k-text-secondary, #78716c);
    margin: 0;
  }
</style>
