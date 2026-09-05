<!--
  apps/artisan/src/lib/TechniqueVerdict.svelte

    <TechniqueVerdict {verdict} />

  Renders inference.v1.TechniqueVerdict from a sealed ProvenanceRecord. The
  real model only ever returns matches:boolean + confidence:float -- there is
  no third state upstream (see $lib/provenance.ts's deriveVerdictOutcome and
  ml_wiring.md's batch 9 section). INSUFFICIENT_EVIDENCE is this component's
  most-tested state, deliberately: a low-confidence read is common (poor
  lighting, an unfamiliar technique variant) and must never look like a
  failure or an accusation -- it reads as "we couldn't tell, here's what to
  do about it," the same register as a blurry-photo retake prompt.
-->
<script lang="ts">
  import { locale, formatPercent } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { SpeakButton } from '@kalakriti/ui';
  import type { components } from '@kalakriti/api';
  import { deriveVerdictOutcome } from './provenance';

  interface Props {
    verdict: components['schemas']['TechniqueVerdict'];
  }

  let { verdict }: Props = $props();

  const t = $derived(locale.t);
  const outcome = $derived(deriveVerdictOutcome(verdict));
  const confidencePct = $derived(formatPercent(verdict.confidence ?? 0, locale.code));

  const ICON = { MATCH: 'success', MISMATCH: 'warning', INSUFFICIENT_EVIDENCE: 'info' } as const;
  const HEADLINE_KEY = {
    MATCH: 'provenance.technique.match',
    MISMATCH: 'provenance.technique.mismatch',
    INSUFFICIENT_EVIDENCE: 'provenance.technique.insufficientEvidence',
  } as const;
</script>

<div class="k-technique-verdict k-technique-verdict--{outcome.toLowerCase()}">
  <div class="k-technique-verdict__headline">
    <Icon name={ICON[outcome]} class="k-technique-verdict__icon" />
    <div>
      <p class="k-technique-verdict__result">{t(HEADLINE_KEY[outcome])}</p>
      <p class="k-technique-verdict__confidence">{t('provenance.technique.confidence', { pct: confidencePct })}</p>
    </div>
    <SpeakButton text={t(HEADLINE_KEY[outcome])} label={t('provenance.technique.listen')} iconOnly />
  </div>

  {#if outcome === 'INSUFFICIENT_EVIDENCE'}
    <p class="k-technique-verdict__detail">{t('provenance.technique.insufficientEvidenceDetail')}</p>
  {:else if verdict.explanation}
    <p class="k-technique-verdict__detail">{verdict.explanation}</p>
  {/if}

  {#if verdict.claimed || verdict.observed}
    <dl class="k-technique-verdict__compare">
      <div>
        <dt>{t('provenance.technique.claimed')}</dt>
        <dd>{verdict.claimed}</dd>
      </div>
      <div>
        <dt>{t('provenance.technique.observed')}</dt>
        <dd>{verdict.observed}</dd>
      </div>
    </dl>
  {/if}
</div>

<style>
  .k-technique-verdict {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
  }

  .k-technique-verdict--match {
    border-color: var(--k-accent-success);
  }

  .k-technique-verdict--insufficient_evidence {
    border-color: var(--k-border-hairline);
  }

  .k-technique-verdict--mismatch {
    border-color: var(--k-accent-danger);
  }

  .k-technique-verdict__headline {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
  }

  .k-technique-verdict__icon {
    flex-shrink: 0;
    inline-size: 1.75rem;
    block-size: 1.75rem;
  }

  .k-technique-verdict__result {
    font-weight: 700;
  }

  .k-technique-verdict__confidence,
  .k-technique-verdict__detail {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-technique-verdict__compare {
    display: flex;
    gap: var(--k-space-4);
    margin: 0;
    font-size: var(--k-text-sm);
  }

  .k-technique-verdict__compare dt {
    color: var(--k-text-secondary);
  }

  .k-technique-verdict__compare dd {
    margin: 0;
    font-weight: 600;
  }
</style>
