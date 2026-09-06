<!--
  apps/buyer/src/routes/terms/+page.svelte

  Official Terms of Service for the Kalakriti Cultural Marketplace.
  Governed under Ministry of Social Justice & Empowerment standards.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Marketplace', href: '/' },
    { label: 'Terms of Service' },
  ]);
</script>

<svelte:head>
  <title>Terms of Service — {t('app.name')}</title>
  <meta name="description" content="Official terms of service governing artisan protection, direct payouts, GI cryptographic certification, and institutional collective fulfillment on Kalakriti." />
</svelte:head>

<div class="legal-page">
  <div class="legal-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <header class="legal-header">
      <span class="legal-kicker">Government Governance & Guild Mandate</span>
      <h1 class="legal-title">Terms of Service</h1>
      <p class="legal-date">Last Updated: September 2026 • Ministry of Social Justice & Empowerment, Govt of India</p>
    </header>

    <div class="legal-content">
      <section class="legal-section">
        <h2>1. Platform Mandate & Public Infrastructure</h2>
        <p>
          Kalakriti is a public digital goods infrastructure established to safeguard, preserve, and commercially empower traditional Indian craft communities, scheduled caste, tribal, and marginalized artisans across all recognized 28 States and Union Territories. Access to this platform is governed by the rules and guidelines issued by the Ministry of Social Justice & Empowerment.
        </p>
      </section>

      <section class="legal-section">
        <h2>2. Geographical Indications (GI) & Authenticity Standard</h2>
        <p>
          All listings carrying the "GI Certified" distinction must correspond to authentic clusters registered under the Geographical Indications of Goods (Registration and Protection) Act, 1999. Every approved product is sealed with an immutable Ed25519 digital signature and public ledger entry verifiable via our cryptographic terminal.
        </p>
      </section>

      <section class="legal-section">
        <h2>3. 100% Direct Payouts & 0% Platform Deductions</h2>
        <p>
          Unlike commercial brokerages, Kalakriti operates with zero commission. 100% of the item purchase price (excluding mandatory statutory GST and logistics tariffs) is transferred directly to the verified DBT bank account of the registered artisan or Self-Help Group (SHG) upon dispatch confirmation.
        </p>
      </section>

      <section class="legal-section">
        <h2>4. Made-to-Order & Institutional Fulfillment</h2>
        <p>
          Purchases designated as Made-to-Order involve custom weaving, hand block-printing, or lost-wax casting. Buyers acknowledge that production lead times (typically 2 to 6 weeks) represent authentic handcrafted labor cycles. Cancellations requested after warp tensioning or raw alloy casting has commenced are subject to reimbursement for artisan material inputs.
        </p>
      </section>

      <section class="legal-section">
        <h2>5. Intellectual Property of Traditional Cultural Expressions</h2>
        <p>
          Traditional motifs, weaving geometries, and regional craft heritage remain the inalienable intellectual property of the respective artisan guilds and traditional practitioner communities. Commercial replication, unauthorized digital cloning, or industrial imitation of Kalakriti-cataloged motifs is strictly prohibited under Indian law.
        </p>
      </section>

      <section class="legal-section">
        <h2>6. Dispute Resolution & Jurisdiction</h2>
        <p>
          Any dispute arising under these terms shall be referred to the Cluster Grievance Redressal Committee under the Ministry of Social Justice & Empowerment, New Delhi, and shall be governed by the laws of India.
        </p>
      </section>
    </div>
  </div>
</div>

<style>
  .legal-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .legal-container {
    max-inline-size: 52rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .legal-header {
    margin-block-end: var(--k-space-6);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-4);
  }

  .legal-kicker {
    display: inline-block;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-secondary);
    margin-block-end: var(--k-space-1);
  }

  .legal-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-3xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .legal-date {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    margin: 0;
  }

  .legal-content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-6);
    line-height: 1.7;
    color: var(--k-text-secondary);
  }

  .legal-section h2 {
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .legal-section p {
    font-size: var(--k-text-sm);
    margin: 0;
  }
</style>
