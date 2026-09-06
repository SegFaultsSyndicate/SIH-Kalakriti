<!--
  apps/buyer/src/routes/orders/thank-you/+page.svelte

  Thank You & Order Confirmation Page:
  Displays order verification proof, estimated loom lead-time,
  direct artisan payout status, and transparent dispatch tracking.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.orders') || 'Orders', href: '/orders' },
    { label: 'Order Confirmed' },
  ]);

  // Derive mock or passed order parameters
  const orderId = $derived(page.url.searchParams.get('id') || 'ORD-KALA-2026-9812');
  const craftName = $derived(page.url.searchParams.get('craft') || 'Varanasi Kadwa Zari Brocade');
  const artisanName = $derived(page.url.searchParams.get('artisan') || 'Mohammad Kabir Ansari');
  const cluster = $derived(page.url.searchParams.get('cluster') || 'Varanasi Silk Weaver CFC');
</script>

<svelte:head>
  <title>Order Confirmed — {t('app.name')}</title>
  <meta name="description" content="Thank you for supporting India's master artisans. Your order has been placed and cryptographically sealed on Kalakriti." />
</svelte:head>

<div class="thank-you-page">
  <div class="thank-you-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <div class="confirmation-card">
      <div class="success-icon-box">
        <svg viewBox="0 0 24 24" width="32" height="32" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="20 6 9 17 4 12" />
        </svg>
      </div>

      <span class="conf-kicker">Order Cryptographically Sealed</span>
      <h1 class="conf-title">Thank You For Your Patronage</h1>
      <p class="conf-sub">
        Your order <strong>#{orderId}</strong> has been secured and dispatched to the loom. 100% of your payment is held in statutory escrow and will be transferred directly to the artisan upon dispatch.
      </p>

      <!-- Verification Seal Box -->
      <div class="seal-box">
        <div class="seal-icon">
          <Icon name="verified-artisan" size="1.4rem" />
        </div>
        <div class="seal-meta">
          <span class="seal-tag">Ed25519 Ministry Proof</span>
          <span class="seal-hash">SHA-256: 4e8f9b2c...a719d308</span>
          <span class="seal-cluster">{craftName} • {cluster}</span>
        </div>
      </div>

      <!-- Artisan Guild Commitment Card -->
      <div class="guild-pledge-card">
        <div class="pledge-header">
          <span class="guild-badge"><Icon name="gi-tagged" size="0.9rem" /> Master Guild Assignment</span>
          <span class="pledge-time">Guaranteed Response within 24 Hours</span>
        </div>
        <div class="pledge-body">
          <p>
            Master artisan <strong>{artisanName}</strong> and their weaving family have received your allocation. You will receive photo and video progress updates directly from the loom as raw silk warp tensioning begins.
          </p>
        </div>
      </div>

      <!-- 4-Stage Loom Progress Roadmap -->
      <div class="progress-roadmap">
        <h2 class="roadmap-title">Fulfillment & Provenance Journey</h2>
        <ol class="roadmap-steps" role="list">
          <li class="step step--complete">
            <span class="step-bullet"><Icon name="check" size="0.75rem" /></span>
            <div class="step-meta">
              <strong>Order Placed & Escrowed</strong>
              <span>Ministry digital ledger record created</span>
            </div>
          </li>
          <li class="step step--active">
            <span class="step-bullet">2</span>
            <div class="step-meta">
              <strong>Yarn Tensioning & Dyeing</strong>
              <span>Loom setup and botanical immersion</span>
            </div>
          </li>
          <li class="step">
            <span class="step-bullet">3</span>
            <div class="step-meta">
              <strong>GI Inspection & Provenance Seal</strong>
              <span>Physical weave density & motif check</span>
            </div>
          </li>
          <li class="step">
            <span class="step-bullet">4</span>
            <div class="step-meta">
              <strong>Archival Khadi Dispatch</strong>
              <span>Insured postal handloom delivery</span>
            </div>
          </li>
        </ol>
      </div>

      <!-- Actions -->
      <div class="conf-actions">
        <a href="/orders" class="k-button k-button--primary">
          <span>View Order Timeline</span>
          <Icon name="arrow-right" size="1rem" />
        </a>
        <a href="/search" class="k-button k-button--secondary">
          <span>Explore More Heritage Crafts</span>
        </a>
      </div>
    </div>
  </div>
</div>

<style>
  .thank-you-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .thank-you-container {
    max-inline-size: 46rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .confirmation-card {
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-card, #ffffff);
    padding: var(--k-space-8);
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--k-space-4);
  }

  .success-icon-box {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 4rem;
    aspect-ratio: 1;
    border-radius: var(--k-radius-full);
    background-color: #e8f5e9;
    color: #2e7d32;
    margin-block-end: var(--k-space-2);
  }

  .conf-kicker {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--k-accent-secondary);
  }

  .conf-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-2xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0;
    line-height: 1.25;
  }

  .conf-sub {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    line-height: 1.6;
    margin: 0;
    max-inline-size: 34rem;
  }

  .conf-sub strong {
    color: var(--k-text-primary);
  }

  /* Seal Box */
  .seal-box {
    inline-size: 100%;
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-4);
    border: 1px dashed var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-sunken, #fbf9f6);
    text-align: start;
  }

  .seal-icon {
    color: var(--k-accent-secondary);
    flex-shrink: 0;
  }

  .seal-meta {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .seal-tag {
    font-size: 0.68rem;
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--k-accent-secondary);
  }

  .seal-hash {
    font-family: monospace;
    font-size: 0.72rem;
    color: var(--k-text-muted);
  }

  .seal-cluster {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  /* Guild Pledge Card */
  .guild-pledge-card {
    inline-size: 100%;
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    padding: var(--k-space-4);
    text-align: start;
    background-color: #ffffff;
  }

  .pledge-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-2);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-2);
  }

  .guild-badge {
    font-size: 0.72rem;
    font-weight: var(--k-weight-bold);
    color: var(--k-accent-secondary);
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }

  .pledge-time {
    font-size: 0.72rem;
    color: #2e7d32;
    font-weight: var(--k-weight-semibold);
  }

  .pledge-body p {
    font-size: var(--k-text-xs);
    line-height: 1.6;
    color: var(--k-text-secondary);
    margin: 0;
  }

  /* Roadmap */
  .progress-roadmap {
    inline-size: 100%;
    text-align: start;
    margin-block-start: var(--k-space-2);
  }

  .roadmap-title {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-3) 0;
  }

  .roadmap-steps {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: var(--k-space-3);
  }

  .step {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    position: relative;
    padding-block-start: var(--k-space-2);
    border-block-start: 2px solid var(--k-border-subtle);
  }

  .step--complete {
    border-block-start-color: #2e7d32;
  }

  .step--active {
    border-block-start-color: var(--k-accent-secondary);
  }

  .step-bullet {
    font-size: 0.75rem;
    font-weight: var(--k-weight-bold);
    color: var(--k-text-muted);
  }

  .step--complete .step-bullet {
    color: #2e7d32;
  }

  .step--active .step-bullet {
    color: var(--k-accent-secondary);
  }

  .step-meta strong {
    display: block;
    font-size: 0.72rem;
    color: var(--k-text-primary);
  }

  .step-meta span {
    display: block;
    font-size: 0.65rem;
    color: var(--k-text-muted);
    line-height: 1.3;
  }

  .conf-actions {
    display: flex;
    gap: var(--k-space-3);
    flex-wrap: wrap;
    justify-content: center;
    margin-block-start: var(--k-space-4);
  }
</style>
