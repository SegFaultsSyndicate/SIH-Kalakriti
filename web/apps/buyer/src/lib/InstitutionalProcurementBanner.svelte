<!--
  apps/buyer/src/lib/InstitutionalProcurementBanner.svelte

  Institutional & Enterprise Procurement Hub for B2B buyers, Government Summits,
  and Corporate Gifting. Pure line-and-space architecture, hairline separation,
  transparent lead-time math, and direct handoff to /bulk-order.
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { SectionHeader, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  let quantity = $state(120);

  // Transparent lead-time calculation based on registered cluster capacity
  const leadWeeks = $derived.by(() => {
    if (quantity <= 50) return 2;
    if (quantity <= 150) return 3;
    if (quantity <= 350) return 5;
    if (quantity <= 700) return 7;
    return 10;
  });

  const activeTier = $derived.by(() => {
    if (quantity < 100) return 1;
    if (quantity <= 500) return 2;
    return 3;
  });

  function downloadCatalog() {
    const content = `KALAKRITI INSTITUTIONAL & ENTERPRISE PROCUREMENT CATALOGUE 2024-2025\n` +
      `Ministry of Social Justice and Empowerment, Government of India\n\n` +
      `DIRECT CLUSTER PROCUREMENT FOR CORPORATE & STATE BANQUET GIFTING\n` +
      `-----------------------------------------------------------------\n` +
      `Tier 1: Boutique Executive (25-100 units) - Custom Brass Seal & Artisan Certificate\n` +
      `Tier 2: Institutional & Summit (100-500 units) - GI Guild Allocation & Unified Billing\n` +
      `Tier 3: National Enterprise (500+ units) - Multi-Cluster Orchestration & Lead-Time Guarantee\n\n` +
      `All orders carry SHA-256 Cryptographic Provenance Seals.\n` +
      `GST-Compliant Invoicing | 100% Direct Weaver Bank Transfers.`;

    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'Kalakriti_Enterprise_Procurement_Guide.txt';
    a.click();
    URL.revokeObjectURL(url);
  }
</script>

<div class="procurement-container">
  <SectionHeader
    kicker={t('home.b2b.kicker')}
    heading={t('home.b2b.heading')}
    href="/bulk-order"
  />

  <p class="procurement-intro">{t('home.b2b.subheading')}</p>

  <div class="procurement-layout">
    <!-- Volume & Capacity Calculator -->
    <div class="calculator-panel">
      <div class="calc-header">
        <label for="procurement-qty-slider" class="calc-label">{t('home.b2b.calculator.label')}</label>
        <span class="calc-numeric">{quantity} <span class="calc-units">{t('home.b2b.calculator.units')}</span></span>
      </div>

      <input
        id="procurement-qty-slider"
        type="range"
        min="25"
        max="1000"
        step="25"
        bind:value={quantity}
        class="calc-slider"
      />

      <div class="slider-notches">
        <span>25</span>
        <span>250</span>
        <span>500</span>
        <span>1,000+</span>
      </div>

      <div class="capacity-signal">
        <Icon name="clock" size="1.1rem" />
        <div class="signal-copy">
          <span class="signal-title">{t('home.b2b.calculator.leadTime')}</span>
          <strong class="signal-duration">{leadWeeks} {t('home.b2b.calculator.weeks')}</strong>
        </div>
      </div>

      <div class="panel-ctas">
        <a
          href={`/bulk-order?quantity=${quantity}`}
          class="k-procure-btn"
        >
          <span>{t('home.b2b.cta')}</span>
          <Icon name="arrow-right" size="1rem" />
        </a>

        <Tooltip text={tooltip('tooltip.downloadCatalog')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="k-deck-btn"
              onclick={downloadCatalog}
              {...tp}
            >
              <Icon name="download" size="0.95rem" />
              <span>{t('home.b2b.downloadDeck')}</span>
            </button>
          {/snippet}
        </Tooltip>
      </div>
    </div>

    <!-- Institutional Allocation Tiers -->
    <div class="tiers-panel">
      <div class="tier-entry" class:active={activeTier === 1}>
        <div class="tier-heading-row">
          <span class="tier-tag">Tier 1</span>
          <h3 class="tier-name">{t('home.b2b.tier1')}</h3>
        </div>
        <p class="tier-explanation">{t('home.b2b.tier1.desc')}</p>
      </div>

      <div class="tier-entry" class:active={activeTier === 2}>
        <div class="tier-heading-row">
          <span class="tier-tag">Tier 2</span>
          <h3 class="tier-name">{t('home.b2b.tier2')}</h3>
        </div>
        <p class="tier-explanation">{t('home.b2b.tier2.desc')}</p>
      </div>

      <div class="tier-entry" class:active={activeTier === 3}>
        <div class="tier-heading-row">
          <span class="tier-tag">Tier 3</span>
          <h3 class="tier-name">{t('home.b2b.tier3')}</h3>
        </div>
        <p class="tier-explanation">{t('home.b2b.tier3.desc')}</p>
      </div>
    </div>
  </div>
</div>

<style>
  .procurement-container {
    margin-block: var(--k-space-5);
  }

  .procurement-intro {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0 0 var(--k-space-5) 0;
    max-inline-size: 70ch;
  }

  .procurement-layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-5);
  }

  @media (min-width: 52rem) {
    .procurement-layout {
      grid-template-columns: 1.25fr 1fr;
      align-items: stretch;
    }
  }

  .calculator-panel {
    padding: var(--k-space-5);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .calc-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .calc-label {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .calc-numeric {
    font-family: var(--k-font-display);
    font-size: var(--k-text-2xl);
    font-weight: bold;
    color: var(--k-accent-primary-text);
  }

  .calc-units {
    font-family: inherit;
    font-size: var(--k-text-sm);
    font-weight: normal;
    color: var(--k-text-secondary);
  }

  .calc-slider {
    inline-size: 100%;
    accent-color: var(--k-accent-primary-text);
    cursor: pointer;
  }

  .slider-notches {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--k-text-secondary);
  }

  .capacity-signal {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    background-color: var(--k-surface-sunken);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    color: var(--k-accent-primary-text);
  }

  .signal-copy {
    display: flex;
    flex-direction: column;
  }

  .signal-title {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .signal-duration {
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .panel-ctas {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-2);
  }

  @media (min-width: 32rem) {
    .panel-ctas {
      flex-direction: row;
    }
  }

  .k-procure-btn {
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-4);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    text-decoration: none;
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    border-radius: var(--k-radius-sm);
  }

  .k-deck-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-primary);
    font-family: inherit;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
  }

  .tiers-panel {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .tier-entry {
    padding: var(--k-space-3) var(--k-space-4);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    transition: border-color 0.15s ease;
  }

  .tier-entry.active {
    border-inline-start: 3px solid var(--k-accent-primary-bg);
    background-color: var(--k-surface-sunken);
  }

  .tier-heading-row {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-1);
  }

  .tier-tag {
    font-size: 0.65rem;
    font-weight: bold;
    text-transform: uppercase;
    padding: 1px 6px;
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: 2px;
    color: var(--k-text-secondary);
  }

  .tier-name {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    margin: 0;
  }

  .tier-explanation {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: 1.4;
    margin: 0;
  }
</style>
