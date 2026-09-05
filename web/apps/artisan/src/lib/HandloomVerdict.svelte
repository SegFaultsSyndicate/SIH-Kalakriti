<!--
  apps/artisan/src/lib/HandloomVerdict.svelte

    <HandloomVerdict {verdict} />

  Renders inference.v1.LoomVerdict, present on a sealed ProvenanceRecord
  (only when skip_loom_check was not set -- see $lib/provenance.ts). is_handloom
  and confidence come straight off the model; fft_peak_ratio and explanation
  are the plain-language "why" the batch 9 brief asks for.

  The prototype-limitation note is fixed text, not conditional on anything --
  the ml-svc mock model (services/ml-svc/app/models/mock.py) is a single
  fixed threshold on one ratio, not a trained classifier, and that fact
  belongs on the same screen a buyer or artisan reads the verdict from, not
  only in the hackathon deck.
-->
<script lang="ts">
  import { locale, formatPercent } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { SpeakButton } from '@kalakriti/ui';
  import type { Component } from 'svelte';
  import type { SVGAttributes } from 'svelte/elements';
  import FftComparisonRaw from '@kalakriti/illustrations/src/fft-comparison.svg';
  import type { components } from '@kalakriti/api';

  interface Props {
    verdict: components['schemas']['LoomVerdict'];
  }

  let { verdict }: Props = $props();

  // Bare .svg import types as `string` under vite/client's ambient module --
  // see packages/ui/src/ToastRegion.svelte's comment for why this cast, not
  // a second ambient declaration, is the fix.
  const FftComparison = FftComparisonRaw as unknown as Component<SVGAttributes<SVGSVGElement>>;

  const t = $derived(locale.t);
  const confidencePct = $derived(formatPercent(verdict.confidence ?? 0, locale.code));

  let showFigure = $state(false);
</script>

<div class="k-handloom-verdict">
  <div class="k-handloom-verdict__headline">
    <Icon name="handloom-verified" class="k-handloom-verdict__icon" />
    <div>
      <p class="k-handloom-verdict__result">
        {verdict.is_handloom ? t('handloom.verdict.handloom') : t('handloom.verdict.powerloom')}
      </p>
      <p class="k-handloom-verdict__confidence">{t('handloom.confidence', { pct: confidencePct })}</p>
    </div>
    <SpeakButton
      text={`${verdict.is_handloom ? t('handloom.verdict.handloom') : t('handloom.verdict.powerloom')}. ${t('handloom.confidence', { pct: confidencePct })}. ${verdict.explanation ?? ''}`}
      label={t('handloom.listen')}
      iconOnly
    />
  </div>

  {#if verdict.explanation}
    <p class="k-handloom-verdict__explanation">{verdict.explanation}</p>
  {/if}

  {#if verdict.fft_peak_ratio != null}
    <p class="k-handloom-verdict__ratio">
      {t('handloom.fftRatio', { ratio: verdict.fft_peak_ratio.toFixed(1) })}
    </p>
  {/if}

  <button type="button" class="k-handloom-verdict__figure-toggle" onclick={() => (showFigure = !showFigure)}>
    <Icon name={showFigure ? 'chevron-up' : 'chevron-right'} />
    {t('handloom.figureLink')}
  </button>
  {#if showFigure}
    <FftComparison role="img" aria-label={t('handloom.figureAlt')} class="k-handloom-verdict__figure" />
  {/if}

  <p class="k-handloom-verdict__prototype-note">
    <Icon name="info" />
    {t('handloom.prototypeNote')}
  </p>
</div>

<style>
  .k-handloom-verdict {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .k-handloom-verdict__headline {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
  }

  :global(.k-handloom-verdict__icon) {
    flex-shrink: 0;
    inline-size: 2rem;
    block-size: 2rem;
  }

  .k-handloom-verdict__result {
    font-weight: 700;
  }

  .k-handloom-verdict__confidence {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-handloom-verdict__explanation,
  .k-handloom-verdict__ratio {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-handloom-verdict__figure-toggle {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    align-self: start;
    border: none;
    background: none;
    color: var(--k-accent-primary-text);
    font-size: var(--k-text-sm);
    cursor: pointer;
  }

  :global(.k-handloom-verdict__figure) {
    max-inline-size: 100%;
    border-radius: var(--k-radius-md);
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .k-handloom-verdict__prototype-note {
    display: flex;
    align-items: start;
    gap: var(--k-space-2);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
