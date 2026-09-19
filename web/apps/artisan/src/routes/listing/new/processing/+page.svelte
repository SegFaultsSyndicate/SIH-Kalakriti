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
-->
<script lang="ts">
  import { liveQuery } from 'dexie';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { PipelineProgress } from '@kalakriti/motion';
  import { db } from '@kalakriti/offline';
  import { getListing, getListingAttributes } from '@kalakriti/api';
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields, ensureListingMediaAttachQueued } from '$lib/listing-draft';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  // 'enhance'/'describe' double as PipelineProgress's own icon keys, not just
  // an i18n choice -- see @kalakriti/motion's PipelineProgress.svelte ICONS
  // map. Reusing two of its four existing stage keys/labels rather than
  // adding new ones -- close enough in meaning, and every new i18n key needs
  // all 21 locale catalogues populated to keep the audit at zero (CLAUDE.md).
  type Stage = 'enhance' | 'describe';
  const STAGE_LABEL_KEY: Record<Stage, MessageKey> = {
    enhance: 'listing.processing.stage.enhance',
    describe: 'listing.processing.stage.describe',
  };

  let uploadRemaining = $state(0);
  let stage = $state<Stage>('enhance');
  let stageStatus = $state<'active' | 'done'>('active');
  let done = $state(false);
  let started = false;

  // Bounded so a slow or unreachable ml-svc never traps the artisan on this
  // screen -- see the header note.
  const POLL_INTERVAL_MS = 1500;
  const POLL_MAX_TRIES = 10;

  $effect(() => {
    if (!draftId) return;
    const sub = liveQuery(() =>
      db.outbox
        .where('draftId')
        .equals(draftId)
        .and((e) => e.kind === 'media.upload')
        .count(),
    ).subscribe((n) => (uploadRemaining = n));
    return () => sub.unsubscribe();
  });

  $effect(() => {
    if (!draftId || started) return;
    started = true;
    void run();
  });

  async function waitForOutboxKind(kind: string): Promise<void> {
    // eslint-disable-next-line no-constant-condition
    while (true) {
      const remaining = await db.outbox
        .where('draftId')
        .equals(draftId)
        .and((e) => e.kind === kind)
        .count();
      if (remaining === 0) return;
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
  }

  async function run(): Promise<void> {
    await waitForOutboxKind('media.upload');

    await ensureListingMediaAttachQueued(draftId);
    await waitForOutboxKind('listing.media.attach');
    stage = 'describe';
    stageStatus = 'active';

    const remoteId = (await getDraft(draftId))?.remoteId;
    if (remoteId) {
      let attributes: Awaited<ReturnType<typeof getListingAttributes>>['attributes'] = [];
      for (let i = 0; i < POLL_MAX_TRIES; i++) {
        const response = await getListingAttributes(remoteId).catch(() => undefined);
        attributes = response?.attributes ?? [];
        if (attributes.length > 0) break;
        await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
      }

      const listing = await getListing(remoteId).catch(() => undefined);
      await patchFields(draftId, {
        translations: listing?.translations ?? [],
        attributes: (attributes ?? []).map((a) => ({
          name: a.name ?? '',
          value: a.value ?? '',
          confidence: a.confidence ?? 0,
          source: (a.source ?? 'MODEL') as 'MODEL' | 'ARTISAN' | 'CURATOR',
        })),
      });
    }

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
    {#if uploadRemaining > 0}
      <p class="processing-status" role="status">
        <Icon name="sync" class="processing-status__icon" />
        {t('listing.processing.uploading', { count: uploadRemaining })}
      </p>
    {:else}
      <PipelineProgress stages={pipelineStages} label={t('listing.processing.heading')} />
    {/if}

    {#if done}
      <p class="processing-done" role="status">
        <Icon name="success" />
        {t('listing.processing.done')}
      </p>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={!done} onclick={next} tooltip={tooltip('tooltip.next')}>{t('action.next')}</Button>
  {/snippet}
</ListingStep>

<style>
  .processing-status {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-text-secondary);
  }

  .processing-status :global(.processing-status__icon) {
    animation: spin 1.2s linear infinite;
  }

  .processing-done {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-accent-success, var(--k-text-primary));
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
