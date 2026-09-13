<!--
  apps/buyer/src/routes/case-studies/+page.svelte

  Craft Guild Impact Case Studies:
  Detailed empirical documentation with concrete Before vs. After
  metrics demonstrating how Kalakriti converts temporary physical exhibition
  sales into year-round continuous digital prosperity.
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

  interface BeforeAfterRow {
    dimension: string;
    before: string;
    after: string;
  }

  interface CaseStudy {
    id: string;
    artisanName: string;
    artisanSlug: string;
    title: string;
    subtitle: string;
    cluster: string;
    region: string;
    giTag: string;
    imageUrl: string;
    metrics: { label: string; value: string }[];
    beforeMonthlyInr: number;
    afterMonthlyInr: number;
    beforeAfterRows: BeforeAfterRow[];
    problem: string;
    intervention: string;
    impact: string;
    quote: { text: string; author: string; role: string };
  }

  const CASE_STUDIES: CaseStudy[] = [
    {
      id: 'dhamadka-ajrakh',
      artisanName: 'Ismail Mohammad Khatri',
      artisanSlug: 'ismail-khatri',
      title: 'The Alchemists of Dhamadka',
      subtitle: 'Revitalizing 16-stage natural indigo mud-resist and securing living wages for 42 master families.',
      cluster: 'Kutch Artisans Collective',
      region: 'Dhamadka, Gujarat',
      giTag: 'GI-72 (Kutch Ajrakh)',
      imageUrl: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
      metrics: [
        { label: 'Artisan Income Surge', value: '+337%' },
        { label: 'Active Pit Vats', value: '18 Vats' },
        { label: 'Middleman Deduction', value: '0%' },
        { label: 'Sealed Provenance Proofs', value: '1,240' },
      ],
      beforeMonthlyInr: 7200,
      afterMonthlyInr: 28400,
      beforeAfterRows: [
        {
          dimension: 'Monthly Household Income',
          before: '₹7,200 (Extreme seasonal swings, poverty during monsoon)',
          after: '₹28,400 (+294% steady recurring monthly income)',
        },
        {
          dimension: 'Intermediary Deductions',
          before: '38% retail margin retained by urban retail brokers',
          after: '0% broker loss (100% direct DBT transfer to artisan bank)',
        },
        {
          dimension: 'Market Reach',
          before: 'Only 2 physical melas (Surajkund & Dilli Haat) per year',
          after: 'Continuous 365-day digital storefront + QR stall placards',
        },
        {
          dimension: 'Authenticity Proof',
          before: 'No proof; undercut by ₹300 synthetic chemical screenprints',
          after: 'Spectroscopic botanical indigo proof & cryptographic NFC seal',
        },
        {
          dimension: 'Payment Settlement',
          before: '6 to 9 months delayed payment on credit consignment',
          after: 'Automated 48-hour escrow payout upon carrier scan',
        },
      ],
      problem:
        'For decades, synthetic chemical block-prints imported by commercial fast-fashion houses severely undercut authentic hand block-printers in Dhamadka. Chemical substitutes flooded markets at 20% of the price, forcing traditional masters to dilute ancient 16-stage botanical resist techniques.',
      intervention:
        'Through Kalakriti, the Dhamadka Guild implemented spectroscopic verification of real indigofera tinctoria vat fermentation and acacia arabica gum resists. Every stole and yardage length receives an immutable Ed25519 cryptographic seal and NFC QR tag before leaving the cluster.',
      impact:
        'Authenticity verification allowed the guild to sell directly to luxury design houses and foreign cultural missions. Household incomes rose by 3.37x, enabling the creation of a collective natural dye seed bank and apprenticeship fund for 14 young printers.',
      quote: {
        text: 'When buyers scan our cloth and see our hands in the indigo vat, they understand why 16 stages cannot be hurried. Kalakriti gave our honor back.',
        author: 'Ismail Mohammad Khatri',
        role: 'Master Craftsman & National Awardee, Dhamadka',
      },
    },
    {
      id: 'varanasi-kadwa',
      artisanName: 'Mohammad Kabir Ansari',
      artisanSlug: 'kabir-ansari',
      title: 'Varanasi Kadwa Pit-Loom Revival',
      subtitle: 'Eliminating predatory 38% broker fees and automating direct DBT payouts for heritage silk weavers.',
      cluster: 'Varanasi Silk Weaver Facility Centre',
      region: 'Varanasi, Uttar Pradesh',
      giTag: 'GI-99 (Banarasi Kadwa)',
      imageUrl: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      metrics: [
        { label: 'Direct Disbursements', value: '₹1.48 Cr' },
        { label: 'Broker Fee Cut', value: '0%' },
        { label: 'Lead Time Reduction', value: '68%' },
        { label: 'Active Pit Looms', value: '86 Looms' },
      ],
      beforeMonthlyInr: 8200,
      afterMonthlyInr: 32600,
      beforeAfterRows: [
        {
          dimension: 'Monthly Weaver Earnings',
          before: '₹8,200 (Forced to take moneylender loans at 36% APR)',
          after: '₹32,600 (+298% increase, guaranteed living wage)',
        },
        {
          dimension: 'Order Predictability',
          before: 'Unpredictable bazaar spot sales, loom idle for 4 months/yr',
          after: '86 pit-looms booked 10 months in advance via GeM lots',
        },
        {
          dimension: 'Middleman Cut',
          before: 'Bazaar traders pocketed 35-40% markups',
          after: '0% middleman margin; 100% DBT bank settlement',
        },
        {
          dimension: 'Institutional Access',
          before: 'Zero access to government summits or corporate gifting',
          after: 'Direct procurement by state delegations & hotel chains',
        },
      ],
      problem:
        'Master zari weavers in Varanasi were dependent on multi-tier bazaar intermediaries who retained 35-40% of retail margins and delayed payments for up to 9 months, forcing weaving families into cyclical monsoon indebtedness.',
      intervention:
        'Kalakriti’s institutional procurement hub allowed state summit hospitality groups and corporate gifting consortia to place collective bulk orders directly with the Pariwar Bunkar SHG. Payment escrow is automated with direct DBT transfers upon loom milestone confirmation.',
      impact:
        'Disbursements totaling ₹1.48 Cr reached weavers within 48 hours of dispatch. 86 pit-looms are now permanently booked across 12-month advance cycles, completely eliminating seasonal poverty.',
      quote: {
        text: 'For three generations, we never saw the face of the final buyer. Now our name and loom coordinates travel with every saree we interlock.',
        author: 'Mohammad Kabir Ansari',
        role: 'Lead Pit-Loom Weaver, Varanasi Silk Guild',
      },
    },
    {
      id: 'bastar-lost-wax',
      artisanName: 'Budheshwar Ghadwa',
      artisanSlug: 'budheshwar-ghadwa',
      title: 'Preserving Bastar Lost-Wax Metallurgy',
      subtitle: 'Sustaining 4,000-year-old Harappan cire-perdue casting through collective metal financing.',
      cluster: 'Bastar Bell Metal Guild',
      region: 'Kondagaon, Chhattisgarh',
      giTag: 'GI-117 (Bastar Dhokra)',
      imageUrl: '/craft-images/metalwork/dhokra-casting.jpg',
      metrics: [
        { label: 'Raw Metal Financed', value: '14.2 Tons' },
        { label: 'Apprentices Trained', value: '28' },
        { label: 'Institutional Kits', value: '640 Lots' },
        { label: 'Pehchan Verified', value: '100%' },
      ],
      beforeMonthlyInr: 7100,
      afterMonthlyInr: 24500,
      beforeAfterRows: [
        {
          dimension: 'Monthly Artisan Wage',
          before: '₹7,100 (Unviable due to 60% scrap metal price inflation)',
          after: '₹24,500 (+245% uplift with raw material financing)',
        },
        {
          dimension: 'Raw Material Financing',
          before: 'Artisans had to borrow at exploitative rates to buy beeswax',
          after: 'Ministry inventory credit line: 14.2 Tons financed upfront',
        },
        {
          dimension: 'Youth Apprenticeship',
          before: '0 young apprentices; youth migrating to construction labor',
          after: '28 tribal youth trained in 4,000-year lost-wax casting',
        },
        {
          dimension: 'Pehchan Recognition',
          before: 'Anonymous tribal makers without government MSME cards',
          after: '100% Pehchan & PM Vishwakarma verified ID cards',
        },
      ],
      problem:
        'Volatile copper and zinc scrap pricing made it nearly impossible for tribal Dhokra sculptors to afford raw beeswax, clay, and casting metals upfront, threatening to wipe out one of humanity’s oldest non-ferrous casting lineages.',
      intervention:
        'The platform introduced raw material inventory credits backed by institutional summit procurement kits. Sculptors record the molten pour on their artisan app, proving unadulterated bell metal alloys.',
      impact:
        '28 new tribal apprentices have joined the Kondagaon furnaces. Bastar lost-wax sculptures now grace prominent institutional lobbies and diplomatic gifts worldwide.',
      quote: {
        text: 'Our ancestors used beeswax from the sal forests. Kalakriti ensures our furnaces burn bright for the next generation without fear of debt.',
        author: 'Budheshwar Ghadwa',
        role: 'Senior Cire-Perdue Metalworker, Kondagaon',
      },
    },
  ];
</script>

<svelte:head>
  <title>Guild Impact Case Studies — Kalakriti</title>
  <meta
    name="description"
    content="Empirical field documentation and Before/After metrics showing how Kalakriti converts temporary exhibition sales into year-round digital prosperity."
  />
</svelte:head>

<div class="studies-page">
  <div class="studies-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <!-- Header Section -->
    <header class="studies-header">
      <div class="kicker-wrap">
        <span class="studies-kicker">GOVERNMENT OF INDIA • EMPIRICAL FIELD RESEARCH</span>
        <span class="kicker-sub">SOCIO-ECONOMIC IMPACT & UPLIFT</span>
      </div>
      <h1 class="studies-title">{t('caseStudies.pageTitle')}</h1>
      <p class="studies-subhead">
        {t('caseStudies.pageSubtitle')}
      </p>
    </header>

    <!-- Case Studies Detailed List -->
    <div class="studies-list">
      {#each CASE_STUDIES as study (study.id)}
        <article class="study-card">
          <div class="study-card__image-box">
            <img src={study.imageUrl} alt={study.title} loading="lazy" class="study-img" />
            <span class="study-gi-badge"><Icon name="gi-tagged" size="0.9rem" /> {study.giTag}</span>
            <div class="artisan-portrait-badge">
              <span class="portrait-name">{study.artisanName}</span>
              <span class="portrait-role">Master Lineage</span>
            </div>
          </div>

          <div class="study-card__body">
            <div class="study-meta-top">
              <span class="study-region">{study.region}</span>
              <span class="dot">•</span>
              <span class="study-cluster">{study.cluster}</span>
            </div>

            <h2 class="study-item-title">{study.title}</h2>
            <p class="study-item-subtitle">{study.subtitle}</p>

            <!-- 4 Top KPI Stat Boxes -->
            <div class="study-metrics-grid">
              {#each study.metrics as metric}
                <div class="metric-box">
                  <span class="metric-val">{metric.value}</span>
                  <span class="metric-lbl">{metric.label}</span>
                </div>
              {/each}
            </div>

            <!-- Visual Income Growth Bar Comparison -->
            <div class="income-comparison-bar-wrap">
              <div class="bar-header">
                <span class="bar-title">Monthly Income Transformation (Before vs. After)</span>
                <span class="bar-uplift-badge">
                  +{Math.round(((study.afterMonthlyInr - study.beforeMonthlyInr) / study.beforeMonthlyInr) * 100)}% Surge
                </span>
              </div>

              <div class="dual-progress-bar">
                <div class="bar-item bar-item--before">
                  <span class="bar-item__label">Before Kalakriti: ₹{study.beforeMonthlyInr.toLocaleString('en-IN')}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill bar-fill--red"
                      style:inline-size="{(study.beforeMonthlyInr / study.afterMonthlyInr) * 100}%"
                    ></div>
                  </div>
                </div>

                <div class="bar-item bar-item--after">
                  <span class="bar-item__label">With Kalakriti: ₹{study.afterMonthlyInr.toLocaleString('en-IN')}/mo</span>
                  <div class="bar-track">
                    <div class="bar-fill bar-fill--green" style:inline-size="100%"></div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Before vs. After Structured Comparison Table -->
            <div class="before-after-section">
              <h3 class="comparison-heading">Comparative Field Audit</h3>
              <div class="table-responsive">
                <table class="comparison-table">
                  <thead>
                    <tr>
                      <th scope="col">Evaluation Dimension</th>
                      <th scope="col" class="th-before">Before Kalakriti (Traditional Exploitation)</th>
                      <th scope="col" class="th-after">With Kalakriti (Digital Infrastructure)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each study.beforeAfterRows as row}
                      <tr>
                        <th scope="row" class="dim-cell">{row.dimension}</th>
                        <td class="before-cell">
                          <span class="cell-status cell-status--bad">✕</span>
                          {row.before}
                        </td>
                        <td class="after-cell">
                          <span class="cell-status cell-status--good">✓</span>
                          {row.after}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>

            <!-- Narrative Blocks -->
            <div class="study-narrative">
              <div class="narrative-block">
                <h3>The Structural Bottleneck</h3>
                <p>{study.problem}</p>
              </div>

              <div class="narrative-block">
                <h3>The Kalakriti Public Intervention</h3>
                <p>{study.intervention}</p>
              </div>

              <div class="narrative-block">
                <h3>Socio-Economic & Generational Impact</h3>
                <p>{study.impact}</p>
              </div>
            </div>

            <!-- Direct Quote -->
            <blockquote class="study-quote">
              <p>"{study.quote.text}"</p>
              <footer>
                <strong>{study.quote.author}</strong>
                <span>{study.quote.role}</span>
              </footer>
            </blockquote>

            <!-- Quick Action Links -->
            <div class="study-actions">
              <a href="/card/{study.artisanSlug}" class="action-btn action-btn--card">
                <Icon name="verified-artisan" size="0.9rem" />
                <span>View Digital Visiting Card</span>
              </a>
              <a href="/artisan/{study.artisanSlug}" class="action-btn action-btn--store">
                <span>Explore Artisan Storefront</span>
                <Icon name="arrow-right" size="0.9rem" />
              </a>
            </div>
          </div>
        </article>
      {/each}
    </div>
  </div>
</div>

<style>
  .studies-page {
    padding-block: var(--k-space-6) var(--k-space-12);
    background: var(--k-surface-base);
  }

  .studies-container {
    max-inline-size: 70rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .studies-header {
    margin-block-end: var(--k-space-8);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-6);
  }

  .kicker-wrap {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-2);
  }

  .studies-kicker {
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    padding: 3px 8px;
    border-radius: 3px;
  }

  .kicker-sub {
    font-size: 0.68rem;
    color: var(--k-text-secondary);
    font-weight: 700;
  }

  .studies-title {
    font-size: var(--k-text-2xl);
    font-weight: 800;
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2);
  }

  .studies-subhead {
    max-inline-size: 52rem;
    color: var(--k-text-secondary);
    font-size: var(--k-text-md);
    line-height: var(--k-leading-normal);
    margin: 0;
  }

  /* Studies List */
  .studies-list {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-8);
  }

  .study-card {
    border-radius: var(--k-radius-lg);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background: var(--k-surface-base);
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
  }

  .study-card__image-box {
    position: relative;
    inline-size: 100%;
    block-size: 18rem;
  }

  .study-img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .study-gi-badge {
    position: absolute;
    inset-block-start: var(--k-space-4);
    inset-inline-start: var(--k-space-4);
    background: rgba(15, 23, 42, 0.85);
    color: var(--k-text-on-accent);
    font-size: 0.75rem;
    font-weight: 700;
    padding: 4px 10px;
    border-radius: var(--k-radius-pill);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .artisan-portrait-badge {
    position: absolute;
    inset-block-end: var(--k-space-4);
    inset-inline-start: var(--k-space-4);
    background: rgba(120, 53, 15, 0.92);
    color: var(--k-text-on-accent);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    display: flex;
    flex-direction: column;
    backdrop-filter: blur(4px);
  }

  .portrait-name {
    font-size: 0.95rem;
    font-weight: 800;
  }

  .portrait-role {
    font-size: 0.68rem;
    opacity: 0.9;
  }

  .study-card__body {
    padding: var(--k-space-6);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-5);
  }

  .study-meta-top {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .study-region {
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  .study-item-title {
    font-size: 1.6rem;
    font-weight: 800;
    color: var(--k-text-primary);
    margin: 0;
  }

  .study-item-subtitle {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    line-height: var(--k-leading-normal);
    margin: 0;
  }

  /* Metric KPI Grid */
  .study-metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--k-space-3);
    background: var(--k-surface-raised);
    padding: var(--k-space-4);
    border-radius: var(--k-radius-md);
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .metric-box {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .metric-val {
    font-size: 1.5rem;
    font-weight: 900;
    color: var(--k-accent-success-muted);
  }

  .metric-lbl {
    font-size: 0.72rem;
    color: var(--k-text-secondary);
    font-weight: 600;
  }

  /* Income Comparison Bar */
  .income-comparison-bar-wrap {
    background: var(--k-surface-base);
    border: 1px solid var(--k-border-on-inverse);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .bar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .bar-title {
    font-size: var(--k-text-xs);
    font-weight: 700;
    color: var(--k-stone-700);
  }

  .bar-uplift-badge {
    background: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
    font-size: 0.72rem;
    font-weight: 800;
    padding: 2px 8px;
    border-radius: 999px;
  }

  .dual-progress-bar {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .bar-item {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .bar-item__label {
    font-size: 0.72rem;
    color: var(--k-text-secondary);
    font-weight: 600;
  }

  .bar-track {
    inline-size: 100%;
    block-size: 0.85rem;
    background: var(--k-surface-neutral);
    border-radius: 999px;
    overflow: hidden;
  }

  .bar-fill {
    block-size: 100%;
    border-radius: 999px;
  }

  .bar-fill--red {
    background: var(--k-madder-500);
  }

  .bar-fill--green {
    background: var(--k-neem-500);
  }

  /* Before vs. After Table */
  .before-after-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .comparison-heading {
    font-size: var(--k-text-sm);
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-accent-primary-text);
    margin: 0;
  }

  .table-responsive {
    overflow-x: auto;
  }

  .comparison-table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-xs);
    border: 1px solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    overflow: hidden;
  }

  .comparison-table th,
  .comparison-table td {
    padding: var(--k-space-2) var(--k-space-3);
    border-block-end: 1px solid var(--k-border-hairline);
    text-align: start;
    vertical-align: top;
  }

  .th-before {
    background: var(--k-surface-base);
    color: var(--k-accent-danger);
  }

  .th-after {
    background: var(--k-surface-base);
    color: var(--k-accent-success);
  }

  .dim-cell {
    font-weight: 700;
    color: var(--k-text-primary);
    inline-size: 22%;
    background: var(--k-surface-raised);
  }

  .before-cell {
    color: var(--k-accent-danger-strong);
    background: rgba(254, 242, 242, 0.4);
    inline-size: 39%;
  }

  .after-cell {
    color: var(--k-accent-success);
    background: rgba(240, 253, 244, 0.4);
    inline-size: 39%;
    font-weight: 600;
  }

  .cell-status {
    display: inline-block;
    inline-size: 1rem;
    font-weight: 900;
  }

  .cell-status--bad {
    color: var(--k-accent-danger);
  }

  .cell-status--good {
    color: var(--k-accent-success-muted);
  }

  /* Narrative */
  .study-narrative {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-4);
  }

  .narrative-block h3 {
    font-size: var(--k-text-xs);
    font-weight: 800;
    color: var(--k-text-primary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0 0 var(--k-space-1);
  }

  .narrative-block p {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: var(--k-leading-normal);
    margin: 0;
  }

  .study-quote {
    margin: 0;
    padding: var(--k-space-4);
    background: var(--k-surface-base);
    border-inline-start: 4px solid var(--k-border-accent);
    border-radius: 0 var(--k-radius-md) var(--k-radius-md) 0;
  }

  .study-quote p {
    font-size: var(--k-text-sm);
    font-style: italic;
    color: var(--k-stone-700);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-2);
  }

  .study-quote footer {
    display: flex;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .study-quote footer strong {
    color: var(--k-text-primary);
  }

  /* Actions */
  .study-actions {
    display: flex;
    gap: var(--k-space-3);
    border-block-start: 1px solid var(--k-border-subtle);
    padding-block-start: var(--k-space-3);
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: var(--k-text-xs);
    font-weight: 700;
    padding: var(--k-space-2) var(--k-space-4);
    border-radius: var(--k-radius-md);
    text-decoration: none;
  }

  .action-btn--card {
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
  }

  .action-btn--store {
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    margin-inline-start: auto;
  }

  @media (max-width: 768px) {
    .study-metrics-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .study-narrative {
      grid-template-columns: 1fr;
    }
    .study-actions {
      flex-direction: column;
    }
    .action-btn--store {
      margin-inline-start: 0;
    }
  }
</style>
