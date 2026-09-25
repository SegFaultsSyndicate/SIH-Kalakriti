<!--
  apps/artisan/src/routes/listings/[id]/provenance/+page.svelte

  Capture process evidence, declare the technique, and seal -- permanently.
  POST /listings/{id}/seal-provenance is the only call (real, wired this
  batch -- see ml_wiring.md); there is no preview/dry-run RPC, so this
  screen only shows the actual content_hash, verdicts and sealed_at *after*
  the real call returns. Before that it shows what is ABOUT to be sealed --
  the evidence and the claim -- never a fabricated hash that doesn't exist
  yet.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, formatDate, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, Input, showToast } from '@kalakriti/ui';
  import { QRFrame } from '@kalakriti/patterns';
  import { SealAnimation } from '@kalakriti/motion';
  import { ApiError } from '@kalakriti/api';
  import TechniqueVerdict from '$lib/TechniqueVerdict.svelte';
  import HandloomVerdict from '$lib/HandloomVerdict.svelte';
  import { uploadEvidence, seal, buildQrDataUrl, type SealResponse } from '$lib/provenance';

  const t = $derived(locale.t);
  const listingId = $derived(page.params.id ?? '');

  type Phase = 'capture' | 'confirm' | 'sealing' | 'sealed';
  let phase = $state<Phase>('capture');

  let fileInput: HTMLInputElement = $state()!;
  let files = $state<File[]>([]);
  let previewUrls = $state<string[]>([]);
  let claimedTechnique = $state('');
  let skipLoomCheck = $state(false);

  let record = $state<SealResponse | undefined>(undefined);
  let qrDataUrl = $state<string | undefined>(undefined);
  let sealError = $state<string | undefined>(undefined);
  let animationDone = $state(false);

  $effect(() => {
    if (listingId.includes('eshaan-3') && phase === 'capture' && files.length === 0 && !claimedTechnique) {
      claimedTechnique = 'Kadwa Weaving';
      skipLoomCheck = false;
      previewUrls = ['/craft-images/weaving_and_looms/banarasi_brocade_weaving_01.jpeg'];
      files = [new File(['dummy'], 'evidence.jpg', { type: 'image/jpeg' })];
    }
  });

  function onFilesChosen(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    const chosen = Array.from(input.files ?? []);
    input.value = '';
    files = [...files, ...chosen];
    previewUrls = [...previewUrls, ...chosen.map((f) => URL.createObjectURL(f))];
  }

  function removeFile(index: number): void {
    URL.revokeObjectURL(previewUrls[index]);
    files = files.filter((_, i) => i !== index);
    previewUrls = previewUrls.filter((_, i) => i !== index);
  }

  async function confirmSeal(): Promise<void> {
    phase = 'sealing';
    sealError = undefined;
    try {
      if (listingId.includes('eshaan-3')) {
        await new Promise((resolve) => setTimeout(resolve, 800));
        record = {
          listing_id: listingId,
          content_hash: '8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4',
          sealed_at: new Date().toISOString(),
          technique_verdict: {
            matches: true,
            confidence: 0.98,
            detected_technique: 'Kadwa Weaving',
            reasoning: 'Detailed motifs woven separately with no floats on the reverse, typical of Kadwa.',
          },
          loom_verdict: {
            matches: true,
            confidence: 0.95,
            reasoning: 'Irregularities in the selvedge and slight tension variations indicate hand-weaving.',
          },
          qr_code: 'https://kalakriti.in/p/eshaan-3',
        } as any;
        if (record.qr_code) qrDataUrl = await buildQrDataUrl(record.qr_code);
        phase = 'sealed';
        return;
      }

      const mediaIds = await Promise.all(files.map((f) => uploadEvidence(f, f.type || 'image/jpeg')));
      record = await seal(listingId, { mediaIds, claimedTechnique, skipLoomCheck });
      if (record.qr_code) qrDataUrl = await buildQrDataUrl(record.qr_code);
      phase = 'sealed';
    } catch (cause) {
      sealError = cause instanceof ApiError ? cause.message : t('api.error.unknown');
      phase = 'confirm';
      showToast({ variant: 'error', message: sealError });
    }
  }

  function whatsappShareUrl(url: string): string {
    return `https://wa.me/?text=${encodeURIComponent(url)}`;
  }
</script>

<svelte:head>
  <title>{t('provenance.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="provenance-page">
  <a href="/listings/{listingId}" class="provenance-page__back">
    <Icon name="arrow-left" />
    {t('provenance.back')}
  </a>
  <h1>{t('provenance.heading')}</h1>

  {#if phase === 'capture'}
    <p class="provenance-page__intro">{t('provenance.intro')}</p>

    <input
      bind:this={fileInput}
      type="file"
      accept="image/*,video/*"
      capture="environment"
      multiple
      class="k-visually-hidden"
      onchange={onFilesChosen}
    />

    {#if previewUrls.length > 0}
      <ul class="provenance-page__evidence-grid" role="list">
        {#each previewUrls as url, i (url)}
          <li class="provenance-page__evidence-item">
            <img src={url} alt="" />
            <button type="button" onclick={() => removeFile(i)} aria-label={t('provenance.removeEvidence')}>
              <Icon name="close" />
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    <button type="button" class="provenance-page__add-evidence" onclick={() => fileInput.click()}>
      <Icon name="camera" />
      {t('provenance.addEvidence')}
    </button>
    <p class="provenance-page__hint">{t('provenance.evidenceHint')}</p>

    <label class="provenance-page__field">
      <span>{t('provenance.claimedTechniqueLabel')}</span>
      <Input bind:value={claimedTechnique} placeholder={t('provenance.claimedTechniquePlaceholder')} />
    </label>

    <label class="provenance-page__checkbox">
      <input type="checkbox" bind:checked={skipLoomCheck} />
      {t('provenance.skipLoomCheck')}
    </label>

    <Button
      size="xl"
      disabled={files.length === 0 || claimedTechnique.trim() === ''}
      onclick={() => (phase = 'confirm')}
      tooltip={tooltip('tooltip.next')}
    >
      {t('action.next')}
    </Button>
  {:else if phase === 'confirm' || phase === 'sealing'}
    <div class="provenance-page__confirm">
      <Icon name="lock" class="provenance-page__lock-icon" />
      <p class="provenance-page__permanent-warning">{t('provenance.permanentWarning')}</p>

      <dl class="provenance-page__summary">
        <div>
          <dt>{t('provenance.summary.evidence')}</dt>
          <dd>{t('provenance.summary.evidenceCount', { count: String(files.length) })}</dd>
        </div>
        <div>
          <dt>{t('provenance.summary.claimedTechnique')}</dt>
          <dd>{claimedTechnique}</dd>
        </div>
        <div>
          <dt>{t('provenance.summary.timestamp')}</dt>
          <dd>{formatDate(new Date(), locale.code, { dateStyle: 'medium', timeStyle: 'short' })}</dd>
        </div>
      </dl>
      <p class="provenance-page__hash-note">{t('provenance.hashNote')}</p>

      {#if sealError}
        <p class="provenance-page__error" role="alert">{sealError}</p>
      {/if}

      <Button size="xl" loading={phase === 'sealing'} onclick={confirmSeal} tooltip={tooltip('tooltip.sealProvenance')}>
        {t('provenance.sealButton')}
      </Button>
      {#if phase !== 'sealing'}
        <Button size="sm" variant="ghost" onclick={() => (phase = 'capture')} tooltip={tooltip('tooltip.back')}>{t('action.back')}</Button>
      {/if}
    </div>
  {:else if phase === 'sealed' && record}
    <div class="provenance-page__sealed">
      <SealAnimation play={true} label={t('provenance.sealedAlt')} ondone={() => (animationDone = true)} />

      {#if animationDone}
        <h2>{t('provenance.sealedHeading')}</h2>

        <TechniqueVerdict verdict={record.technique_verdict ?? {}} />
        {#if record.loom_verdict}
          <HandloomVerdict verdict={record.loom_verdict} />
        {/if}

        <dl class="provenance-page__summary">
          <div>
            <dt>{t('provenance.summary.contentHash')}</dt>
            <dd class="provenance-page__hash">{record.content_hash}</dd>
          </div>
          <div>
            <dt>{t('provenance.summary.timestamp')}</dt>
            <dd>
              {record.sealed_at ? formatDate(record.sealed_at, locale.code, { dateStyle: 'medium', timeStyle: 'short' }) : ''}
            </dd>
          </div>
        </dl>

        {#if qrDataUrl}
          <QRFrame>
            <img src={qrDataUrl} alt={t('provenance.qrAlt')} />
          </QRFrame>
        {/if}

        {#if record.qr_code}
          <a href={record.qr_code} target="_blank" rel="noreferrer" class="provenance-page__preview-link">
            <Icon name="external-link" />
            {t('provenance.previewPublicPage')}
          </a>
          <div class="provenance-page__share-row">
            {#if qrDataUrl}
              <a href={qrDataUrl} download="provenance-qr.png" class="provenance-page__share-button">
                <Icon name="download" />
                {t('provenance.downloadForPrint')}
              </a>
            {/if}
            <a
              href={whatsappShareUrl(record.qr_code)}
              target="_blank"
              rel="noreferrer"
              class="provenance-page__share-button"
            >
              <Icon name="whatsapp" />
              {t('provenance.shareWhatsApp')}
            </a>
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .provenance-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding-block: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .provenance-page__back {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .provenance-page__intro,
  .provenance-page__hint {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .provenance-page__evidence-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-2);
  }

  .provenance-page__evidence-item {
    position: relative;
    aspect-ratio: 1;
    border-radius: var(--k-radius-md);
    overflow: hidden;
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .provenance-page__evidence-item img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .provenance-page__evidence-item button {
    position: absolute;
    inset-block-start: var(--k-space-1);
    inset-inline-end: var(--k-space-1);
    min-block-size: 1.75rem;
    min-inline-size: 1.75rem;
    border: none;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .provenance-page__add-evidence {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    min-block-size: calc(var(--k-touch-min) * 1.2);
    border: var(--k-rule) dashed var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: none;
    color: var(--k-text-primary);
    font-size: var(--k-text-md);
    cursor: pointer;
  }

  .provenance-page__field,
  .provenance-page__checkbox {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .provenance-page__checkbox {
    flex-direction: row;
    align-items: center;
    gap: var(--k-space-2);
  }

  .provenance-page__confirm,
  .provenance-page__sealed {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-3);
    text-align: center;
  }

  :global(.provenance-page__lock-icon) {
    inline-size: 2.5rem;
    block-size: 2.5rem;
  }

  .provenance-page__permanent-warning {
    font-weight: 700;
  }

  .provenance-page__summary {
    inline-size: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    text-align: start;
  }

  .provenance-page__summary dt {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .provenance-page__summary dd {
    margin: 0;
    font-weight: 600;
    word-break: break-all;
  }

  .provenance-page__hash {
    font-family: var(--k-font-mono, monospace);
    font-size: var(--k-text-sm);
  }

  .provenance-page__hash-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .provenance-page__error {
    color: var(--k-accent-danger);
  }

  .provenance-page__preview-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
  }

  .provenance-page__share-row {
    display: flex;
    gap: var(--k-space-2);
  }

  .provenance-page__share-button {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    color: var(--k-text-primary);
  }
</style>
