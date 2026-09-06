<!--
  apps/buyer/src/routes/case-studies/+page.svelte

  Craft Guild Impact Case Studies:
  Detailed anthropological, economic, and provenance documentation
  demonstrating how Kalakriti's public infrastructure elevates artisan communities.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'Guild Case Studies' },
  ]);

  interface CaseStudy {
    id: string;
    title: string;
    subtitle: string;
    cluster: string;
    region: string;
    giTag: string;
    imageUrl: string;
    metrics: { label: string; value: string }[];
    problem: string;
    intervention: string;
    impact: string;
    quote: { text: string; author: string; role: string };
  }

  const CASE_STUDIES: CaseStudy[] = [
    {
      id: 'dhamadka-ajrakh',
      title: 'The Alchemists of Dhamadka',
      subtitle: 'Revitalizing 16-stage natural indigo mud-resist and securing living wages for 42 master families.',
      cluster: 'Kutch Artisans Collective',
      region: 'Dhamadka, Gujarat',
      giTag: 'GI-72 (Kutch Ajrakh)',
      imageUrl: 'https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=900&q=80',
      metrics: [
        { label: 'Artisan Income Surge', value: '+320%' },
        { label: 'Active Pit Vats', value: '18 Vats' },
        { label: 'Middleman Deduction', value: '0%' },
        { label: 'Sealed Ledger Proofs', value: '1,240' },
      ],
      problem: 'For decades, synthetic chemical block-prints imported by commercial fast-fashion houses severely undercut authentic hand block-printers in Dhamadka. Chemical substitutes flooded markets at 20% of the price, forcing traditional masters to dilute ancient 16-stage botanical resist techniques.',
      intervention: 'Through Kalakriti, the Dhamadka Guild implemented spectroscopic verification of real indigofera tinctoria vat fermentation and acacia arabica gum resists. Every stole and yardage length receives an immutable Ed25519 cryptographic seal and NFC QR tag before leaving the cluster.',
      impact: 'Authenticity verification allowed the guild to sell directly to luxury design houses and foreign cultural missions. Household incomes rose by 3.2x, enabling the creation of a collective natural dye seed bank and apprenticeship fund for 14 young printers.',
      quote: {
        text: 'When buyers scan our cloth and see our hands in the indigo vat, they understand why 16 stages cannot be hurried. Kalakriti gave our honor back.',
        author: 'Ismail Mohammad Khatri',
        role: 'Master Craftsman & National Awardee, Dhamadka',
      },
    },
    {
      id: 'varanasi-kadwa',
      title: 'Varanasi Kadwa Pit-Loom Revival',
      subtitle: 'Eliminating predatory 38% broker fees and automating direct DBT payouts for heritage silk weavers.',
      cluster: 'Varanasi Silk Weaver Facility Centre',
      region: 'Varanasi, Uttar Pradesh',
      giTag: 'GI-99 (Banarasi Kadwa)',
      imageUrl: 'https://images.unsplash.com/photo-1610030469983-98e550d6193c?auto=format&fit=crop&w=900&q=80',
      metrics: [
        { label: 'Direct Disbursements', value: '₹1.48 Cr' },
        { label: 'Broker Fee Cut', value: '0%' },
        { label: 'Lead Time Reduction', value: '68%' },
        { label: 'Weaver Looms Active', value: '86' },
      ],
      problem: 'Master zari weavers in Varanasi were dependent on multi-tier bazaar intermediaries who retained 35-40% of retail margins and delayed payments for up to 9 months, forcing weaving families into cyclical monsoon indebtedness.',
      intervention: 'Kalakriti’s institutional procurement hub allowed state summit hospitality groups and corporate gifting consortia to place collective bulk orders directly with the Pariwar Bunkar SHG. Payment escrow is automated with direct DBT transfers upon loom milestone confirmation.',
      impact: 'Disbursements totaling ₹1.48 Cr reached weavers within 48 hours of dispatch. 86 pit-looms are now permanently booked across 12-month advance cycles, completely eliminating seasonal poverty.',
      quote: {
        text: 'For three generations, we never saw the face of the final buyer. Now our name and loom coordinates travel with every saree we interlock.',
        author: 'Mohammad Kabir Ansari',
        role: 'Lead Pit-Loom Weaver, Varanasi Silk Guild',
      },
    },
    {
      id: 'bastar-lost-wax',
      title: 'Preserving Bastar Lost-Wax Metallurgy',
      subtitle: 'Sustaining 4,000-year-old Harappan cire-perdue casting through collective metal financing.',
      cluster: 'Bastar Bell Metal Guild',
      region: 'Kondagaon, Chhattisgarh',
      giTag: 'GI-117 (Bastar Dhokra)',
      imageUrl: 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=900&q=80',
      metrics: [
        { label: 'Raw Metal Financed', value: '14.2 Tons' },
        { label: 'Apprentices Trained', value: '28' },
        { label: 'Institutional Kits', value: '640 Lots' },
        { label: 'Pehchan Verified', value: '100%' },
      ],
      problem: 'Volatile copper and zinc scrap pricing made it nearly impossible for tribal Dhokra sculptors to afford raw beeswax, clay, and casting metals upfront, threatening to wipe out one of humanity’s oldest non-ferrous casting lineages.',
      intervention: 'The platform introduced raw material inventory credits backed by institutional summit procurement kits. Sculptors record the molten pour on their artisan app, proving unadulterated bell metal alloys.',
      impact: '28 new tribal apprentices have joined the Kondagaon furnaces. Bastar lost-wax sculptures now grace prominent institutional lobbies and diplomatic gifts worldwide.',
      quote: {
        text: 'Our ancestors used beeswax from the sal forests. Kalakriti ensures our furnaces burn bright for the next generation without fear of debt.',
        author: 'Budheshwar Ghadwa',
        role: 'Senior Cire-Perdue Metalworker, Kondagaon',
      },
    },
  ];
</script>

<svelte:head>
  <title>Guild Impact Case Studies - Kalakriti</title>
  <meta name="description" content="In-depth case studies documenting how Kalakriti's public digital infrastructure, cryptographic provenance, and direct payouts transform Indian artisan guilds." />
</svelte:head>

<div class="studies-page">
  <div class="studies-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <header class="studies-header">
      <span class="studies-kicker">Empirical Field Documentation</span>
      <h1 class="studies-title">Craft Guild Impact Case Studies</h1>
      <p class="studies-subhead">
        Documenting systemic transformation across India’s craft corridors: how cryptographic provenance, zero-broker direct payouts, and collective institutional fulfillment restore dignity and economic vitality to master lineages.
      </p>
    </header>

    <div class="studies-list">
      {#each CASE_STUDIES as study (study.id)}
        <article class="study-card">
          <div class="study-card__image-box">
            <img src={study.imageUrl} alt={study.title} loading="lazy" class="study-img" />
            <span class="study-gi-badge"><Icon name="gi-tagged" size="0.9rem" /> {study.giTag}</span>
          </div>

          <div class="study-card__body">
            <div class="study-meta-top">
              <span class="study-region">{study.region}</span>
              <span class="dot">•</span>
              <span class="study-cluster">{study.cluster}</span>
            </div>

            <h2 class="study-item-title">{study.title}</h2>
            <p class="study-item-subtitle">{study.subtitle}</p>

            <div class="study-metrics-grid">
              {#each study.metrics as metric}
                <div class="metric-box">
                  <span class="metric-val">{metric.value}</span>
                  <span class="metric-lbl">{metric.label}</span>
                </div>
              {/each}
            </div>

            <div class="study-narrative">
              <div class="narrative-block">
                <h3>The Challenge</h3>
                <p>{study.problem}</p>
              </div>

              <div class="narrative-block">
                <h3>The Kalakriti Intervention</h3>
                <p>{study.intervention}</p>
              </div>

              <div class="narrative-block">
                <h3>Economic & Guild Impact</h3>
                <p>{study.impact}</p>
              </div>
            </div>

            <blockquote class="study-quote">
              <p>"{study.quote.text}"</p>
              <footer>
                <strong>{study.quote.author}</strong>
                <span>{study.quote.role}</span>
              </footer>
            </blockquote>
          </div>
        </article>
      {/each}
    </div>
  </div>
</div>

<style>
  .studies-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .studies-container {
    max-inline-size: 68rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .studies-header {
    margin-block-end: var(--k-space-8);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-6);
  }

  .studies-kicker {
    display: inline-block;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-secondary);
    margin-block-end: var(--k-space-1);
  }

  .studies-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-3xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-3) 0;
  }

  .studies-subhead {
    font-size: var(--k-text-base);
    color: var(--k-text-secondary);
    line-height: 1.6;
    max-inline-size: 48rem;
    margin: 0;
  }

  .studies-list {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-10);
  }

  .study-card {
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-card, #ffffff);
    overflow: hidden;
    display: grid;
    grid-template-columns: 24rem 1fr;
    gap: 0;
  }

  @media (max-width: 54rem) {
    .study-card {
      grid-template-columns: 1fr;
    }
  }

  .study-card__image-box {
    position: relative;
    inline-size: 100%;
    min-block-size: 20rem;
    background-color: var(--k-surface-sunken, #f7f4ee);
  }

  .study-img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
    display: block;
  }

  .study-gi-badge {
    position: absolute;
    inset-block-start: var(--k-space-3);
    inset-inline-start: var(--k-space-3);
    background-color: rgba(26, 26, 26, 0.85);
    color: #ffffff;
    font-size: 0.72rem;
    font-weight: var(--k-weight-semibold);
    padding: 0.25rem 0.6rem;
    border-radius: var(--k-radius-sm);
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    backdrop-filter: blur(4px);
  }

  .study-card__body {
    padding: var(--k-space-6);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .study-meta-top {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-accent-secondary);
    font-weight: var(--k-weight-semibold);
  }

  .dot {
    color: var(--k-border-subtle);
  }

  .study-item-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-2xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0;
    line-height: 1.25;
  }

  .study-item-subtitle {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  .study-metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--k-space-2);
    padding-block: var(--k-space-3);
    border-block: 1px solid var(--k-border-subtle);
  }

  @media (max-width: 40rem) {
    .study-metrics-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  .metric-box {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .metric-val {
    font-size: var(--k-text-lg);
    font-weight: var(--k-weight-bold);
    color: var(--k-accent-secondary);
  }

  .metric-lbl {
    font-size: 0.68rem;
    color: var(--k-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .study-narrative {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .narrative-block h3 {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--k-text-primary);
    margin: 0 0 0.25rem 0;
  }

  .narrative-block p {
    font-size: var(--k-text-xs);
    line-height: 1.65;
    color: var(--k-text-secondary);
    margin: 0;
  }

  .study-quote {
    margin: 0;
    padding: var(--k-space-4);
    background-color: var(--k-surface-sunken, #fbf9f6);
    border-inline-start: 3px solid var(--k-accent-secondary);
    border-radius: var(--k-radius-sm);
  }

  .study-quote p {
    font-family: var(--k-font-display, Georgia, serif);
    font-style: italic;
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
    line-height: 1.5;
  }

  .study-quote footer {
    display: flex;
    flex-direction: column;
    font-size: 0.72rem;
  }

  .study-quote strong {
    color: var(--k-text-primary);
  }

  .study-quote span {
    color: var(--k-text-muted);
  }
</style>
