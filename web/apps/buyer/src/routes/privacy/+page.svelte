<!--
  apps/buyer/src/routes/privacy/+page.svelte

  Official Privacy Policy for Kalakriti Platform.
  Compliant with the Digital Personal Data Protection (DPDP) Act 2023.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Marketplace', href: '/' },
    { label: 'Privacy Policy' },
  ]);
</script>

<svelte:head>
  <title>Privacy Policy — {t('app.name')}</title>
  <meta name="description" content="Official Privacy Policy compliant with India's Digital Personal Data Protection (DPDP) Act 2023, outlining artisan digital identity security and buyer data rights." />
</svelte:head>

<div class="legal-page">
  <div class="legal-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <header class="legal-header">
      <span class="legal-kicker">DPDP Act 2023 Statutory Compliance</span>
      <h1 class="legal-title">Privacy Policy</h1>
      <p class="legal-date">Last Updated: September 2026 • Government of India National Digital Infrastructure</p>
    </header>

    <div class="legal-content">
      <section class="legal-section">
        <h2>1. Statutory Commitment & Scope</h2>
        <p>
          Kalakriti is dedicated to the highest standards of data fiduciary responsibility in compliance with the Digital Personal Data Protection Act, 2023 (DPDP Act). This policy explains how we collect, handle, safeguard, and process personal data across our Artisan PWA, Buyer Marketplace, and Admin portals.
        </p>
      </section>

      <section class="legal-section">
        <h2>2. Data We Collect (Principle of Data Minimisation)</h2>
        <p>
          We adhere strictly to the statutory principle of data minimisation. For buyers, we collect only the contact information and shipping address strictly necessary to deliver artisanal orders. For artisans, we collect verified mobile credentials, Ministry Pehchan card numbers, and cluster geographical coordinates necessary to certify Geographical Indication (GI) authenticity.
        </p>
      </section>

      <section class="legal-section">
        <h2>3. Purpose of Processing & Direct Payouts</h2>
        <p>
          Data collected is processed solely for the following explicit purposes:
        </p>
        <ul class="legal-bullets">
          <li>Facilitating direct DBT bank payments to artisan guilds with 0% platform deductions.</li>
          <li>Minting tamper-proof cryptographic Ed25519 digital provenance certificates.</li>
          <li>Scheduling optional 10-minute live loom video consultations between collectors and weavers.</li>
          <li>Ensuring fulfillment and delivery status notifications via SMS and email.</li>
        </ul>
      </section>

      <section class="legal-section">
        <h2>4. Absolute Ban on Third-Party Data Selling</h2>
        <p>
          As a public national initiative, Kalakriti will never sell, rent, monetize, or disclose your personal or commercial transaction data to third-party advertisers, data brokers, or commercial tracking syndicates.
        </p>
      </section>

      <section class="legal-section">
        <h2>5. Cryptographic Security & Cookie Policy</h2>
        <p>
          All network transmissions utilize TLS 1.3 enforced by HTTP Strict Transport Security (HSTS). Authentication tokens are maintained with strict HttpOnly, Secure, SameSite=Strict cookies. We do not deploy invasive tracking pixels or cross-site tracking beacons.
        </p>
      </section>

      <section class="legal-section">
        <h2>6. Data Subject Rights & Grievance Redressal</h2>
        <p>
          You possess statutory rights to access, review, correct, or request the erasure of your personal data under the DPDP Act 2023. For inquiries or data deletion requests, contact the designated Data Protection Grievance Officer at <code>dpo-grievance@kalakriti.gov.in</code>.
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

  .legal-bullets {
    margin: var(--k-space-2) 0 0 var(--k-space-4);
    padding: 0;
    font-size: var(--k-text-sm);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }
</style>
