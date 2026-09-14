<!--
  apps/artisan/src/routes/listing/new/processing/+page.svelte

  Step 4 of 7. The upload phase is real state -- it reads this draft's
  media.upload outbox rows directly, so "uploading" only shows while
  something is actually queued. The four stages after that (enhance,
  attributes, describe, translate) are MOCK: ml_wiring.md explains why, and
  ml-mock.ts is the only place that fabricates them. Both are real state
  machines driving PipelineProgress -- see that component's own header on
  why it accepts no internal timer.
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
  import ListingStep from '$lib/ListingStep.svelte';
  import { getDraft, patchFields, queueListingUpdate } from '$lib/listing-draft';
  import { runMockPipeline, type PipelineStageState, type PipelineStageKey } from '$lib/ml-mock';

  const t = $derived(locale.t);
  const draftId = $derived(page.url.searchParams.get('d') ?? '');

  let uploadRemaining = $state(0);
  let stages = $state<PipelineStageState[]>([]);
  let done = $state(false);
  let started = false;

  const STAGE_LABEL_KEY: Record<Exclude<PipelineStageKey, 'upload'>, MessageKey> = {
    enhance: 'listing.processing.stage.enhance',
    attributes: 'listing.processing.stage.attributes',
    describe: 'listing.processing.stage.describe',
    translate: 'listing.processing.stage.translate',
  };
  const MOTION_STATE: Record<PipelineStageState['status'], 'pending' | 'active' | 'complete' | 'failed'> = {
    pending: 'pending',
    active: 'active',
    done: 'complete',
    error: 'failed',
  };

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

  async function run(): Promise<void> {
    const draft = await getDraft(draftId);
    const craftId = draft?.fields.craftId as string | undefined;
    const workingTitle = draft?.fields.workingTitle as string | undefined;

    const uploadDone = async (): Promise<boolean> => {
      const n = await db.outbox
        .where('draftId')
        .equals(draftId)
        .and((e) => e.kind === 'media.upload')
        .count();
      return n === 0;
    };

    const result = await runMockPipeline(craftId, workingTitle, uploadDone, (s) => (stages = s));

    await patchFields(draftId, {
      translations: result.translations,
      attributes: result.attributes,
      claims: result.claims,
    });
    await queueListingUpdate(draftId, { translations: result.translations });
    done = true;
  }

  const uploading = $derived(stages.find((s) => s.key === 'upload')?.status !== 'done');
  const pipelineStages = $derived(
    stages
      .filter((s): s is PipelineStageState & { key: Exclude<PipelineStageKey, 'upload'> } => s.key !== 'upload')
      .map((s) => ({ key: s.key, label: t(STAGE_LABEL_KEY[s.key]), state: MOTION_STATE[s.status] })),
  );

  async function next(): Promise<void> {
    await goto(`/listing/new/review?d=${draftId}`);
  }
</script>

<svelte:head>
  <title>{t('listing.processing.heading')} — {t('app.name')}</title>
</svelte:head>

<ListingStep index={4} heading={t('listing.processing.heading')} backHref="/listing/new/story?d={draftId}">
  {#snippet children()}
    {#if uploading}
      <p class="processing-status" role="status">
        <Icon name="sync" class="processing-status__icon" />
        {uploadRemaining > 0
          ? t('listing.processing.uploading', { count: uploadRemaining })
          : t('listing.processing.uploadingStart')}
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
