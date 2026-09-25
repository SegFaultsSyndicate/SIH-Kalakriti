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

  const breadcrumbs = $derived([{ label: t('caseStudies.breadcrumbLabel') }]);

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

  const CASE_STUDIES: CaseStudy[] = $derived([
    {
      id: 'dhamadka-ajrakh',
      artisanName: 'Ismail Mohammad Khatri',
      artisanSlug: 'ismail-khatri',
      title: t('caseStudies.dhamadka-ajrakh.title'),
      subtitle: t('caseStudies.dhamadka-ajrakh.subtitle'),
      cluster: t('caseStudies.dhamadka-ajrakh.cluster'),
      region: t('caseStudies.dhamadka-ajrakh.region'),
      giTag: t('caseStudies.dhamadka-ajrakh.giTag'),
      imageUrl: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
      metrics: [
        { label: t('caseStudies.dhamadka-ajrakh.metrics.0.label'), value: t('caseStudies.dhamadka-ajrakh.metrics.0.value') },
        { label: t('caseStudies.dhamadka-ajrakh.metrics.1.label'), value: t('caseStudies.dhamadka-ajrakh.metrics.1.value') },
        { label: t('caseStudies.dhamadka-ajrakh.metrics.2.label'), value: t('caseStudies.dhamadka-ajrakh.metrics.2.value') },
        { label: t('caseStudies.dhamadka-ajrakh.metrics.3.label'), value: t('caseStudies.dhamadka-ajrakh.metrics.3.value') },
      ],
      beforeMonthlyInr: 7200,
      afterMonthlyInr: 28400,
      beforeAfterRows: [
        {
          dimension: t('caseStudies.dhamadka-ajrakh.rows.0.dimension'),
          before: t('caseStudies.dhamadka-ajrakh.rows.0.before'),
          after: t('caseStudies.dhamadka-ajrakh.rows.0.after'),
        },
        {
          dimension: t('caseStudies.dhamadka-ajrakh.rows.1.dimension'),
          before: t('caseStudies.dhamadka-ajrakh.rows.1.before'),
          after: t('caseStudies.dhamadka-ajrakh.rows.1.after'),
        },
        {
          dimension: t('caseStudies.dhamadka-ajrakh.rows.2.dimension'),
          before: t('caseStudies.dhamadka-ajrakh.rows.2.before'),
          after: t('caseStudies.dhamadka-ajrakh.rows.2.after'),
        },
        {
          dimension: t('caseStudies.dhamadka-ajrakh.rows.3.dimension'),
          before: t('caseStudies.dhamadka-ajrakh.rows.3.before'),
          after: t('caseStudies.dhamadka-ajrakh.rows.3.after'),
        },
        {
          dimension: t('caseStudies.dhamadka-ajrakh.rows.4.dimension'),
          before: t('caseStudies.dhamadka-ajrakh.rows.4.before'),
          after: t('caseStudies.dhamadka-ajrakh.rows.4.after'),
        },
      ],
      problem: t('caseStudies.dhamadka-ajrakh.problem'),
      intervention: t('caseStudies.dhamadka-ajrakh.intervention'),
      impact: t('caseStudies.dhamadka-ajrakh.impact'),
      quote: {
        text: t('caseStudies.dhamadka-ajrakh.quote.text'),
        author: t('caseStudies.dhamadka-ajrakh.quote.author'),
        role: t('caseStudies.dhamadka-ajrakh.quote.role'),
      },
    },
    {
      id: 'varanasi-kadwa',
      artisanName: 'Eshaan',
      artisanSlug: 'kabir-ansari',
      title: t('caseStudies.varanasi-kadwa.title'),
      subtitle: t('caseStudies.varanasi-kadwa.subtitle'),
      cluster: t('caseStudies.varanasi-kadwa.cluster'),
      region: t('caseStudies.varanasi-kadwa.region'),
      giTag: t('caseStudies.varanasi-kadwa.giTag'),
      imageUrl: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      metrics: [
        { label: t('caseStudies.varanasi-kadwa.metrics.0.label'), value: t('caseStudies.varanasi-kadwa.metrics.0.value') },
        { label: t('caseStudies.varanasi-kadwa.metrics.1.label'), value: t('caseStudies.varanasi-kadwa.metrics.1.value') },
        { label: t('caseStudies.varanasi-kadwa.metrics.2.label'), value: t('caseStudies.varanasi-kadwa.metrics.2.value') },
        { label: t('caseStudies.varanasi-kadwa.metrics.3.label'), value: t('caseStudies.varanasi-kadwa.metrics.3.value') },
      ],
      beforeMonthlyInr: 8200,
      afterMonthlyInr: 32600,
      beforeAfterRows: [
        {
          dimension: t('caseStudies.varanasi-kadwa.rows.0.dimension'),
          before: t('caseStudies.varanasi-kadwa.rows.0.before'),
          after: t('caseStudies.varanasi-kadwa.rows.0.after'),
        },
        {
          dimension: t('caseStudies.varanasi-kadwa.rows.1.dimension'),
          before: t('caseStudies.varanasi-kadwa.rows.1.before'),
          after: t('caseStudies.varanasi-kadwa.rows.1.after'),
        },
        {
          dimension: t('caseStudies.varanasi-kadwa.rows.2.dimension'),
          before: t('caseStudies.varanasi-kadwa.rows.2.before'),
          after: t('caseStudies.varanasi-kadwa.rows.2.after'),
        },
        {
          dimension: t('caseStudies.varanasi-kadwa.rows.3.dimension'),
          before: t('caseStudies.varanasi-kadwa.rows.3.before'),
          after: t('caseStudies.varanasi-kadwa.rows.3.after'),
        },
      ],
      problem: t('caseStudies.varanasi-kadwa.problem'),
      intervention: t('caseStudies.varanasi-kadwa.intervention'),
      impact: t('caseStudies.varanasi-kadwa.impact'),
      quote: {
        text: t('caseStudies.varanasi-kadwa.quote.text'),
        author: t('caseStudies.varanasi-kadwa.quote.author'),
        role: t('caseStudies.varanasi-kadwa.quote.role'),
      },
    },
    {
      id: 'bastar-lost-wax',
      artisanName: 'Budheshwar Ghadwa',
      artisanSlug: 'budheshwar-ghadwa',
      title: t('caseStudies.bastar-lost-wax.title'),
      subtitle: t('caseStudies.bastar-lost-wax.subtitle'),
      cluster: t('caseStudies.bastar-lost-wax.cluster'),
      region: t('caseStudies.bastar-lost-wax.region'),
      giTag: t('caseStudies.bastar-lost-wax.giTag'),
      imageUrl: '/craft-images/metalwork/dhokra-casting.jpg',
      metrics: [
        { label: t('caseStudies.bastar-lost-wax.metrics.0.label'), value: t('caseStudies.bastar-lost-wax.metrics.0.value') },
        { label: t('caseStudies.bastar-lost-wax.metrics.1.label'), value: t('caseStudies.bastar-lost-wax.metrics.1.value') },
        { label: t('caseStudies.bastar-lost-wax.metrics.2.label'), value: t('caseStudies.bastar-lost-wax.metrics.2.value') },
        { label: t('caseStudies.bastar-lost-wax.metrics.3.label'), value: t('caseStudies.bastar-lost-wax.metrics.3.value') },
      ],
      beforeMonthlyInr: 7100,
      afterMonthlyInr: 24500,
      beforeAfterRows: [
        {
          dimension: t('caseStudies.bastar-lost-wax.rows.0.dimension'),
          before: t('caseStudies.bastar-lost-wax.rows.0.before'),
          after: t('caseStudies.bastar-lost-wax.rows.0.after'),
        },
        {
          dimension: t('caseStudies.bastar-lost-wax.rows.1.dimension'),
          before: t('caseStudies.bastar-lost-wax.rows.1.before'),
          after: t('caseStudies.bastar-lost-wax.rows.1.after'),
        },
        {
          dimension: t('caseStudies.bastar-lost-wax.rows.2.dimension'),
          before: t('caseStudies.bastar-lost-wax.rows.2.before'),
          after: t('caseStudies.bastar-lost-wax.rows.2.after'),
        },
        {
          dimension: t('caseStudies.bastar-lost-wax.rows.3.dimension'),
          before: t('caseStudies.bastar-lost-wax.rows.3.before'),
          after: t('caseStudies.bastar-lost-wax.rows.3.after'),
        },
      ],
      problem: t('caseStudies.bastar-lost-wax.problem'),
      intervention: t('caseStudies.bastar-lost-wax.intervention'),
      impact: t('caseStudies.bastar-lost-wax.impact'),
      quote: {
        text: t('caseStudies.bastar-lost-wax.quote.text'),
        author: t('caseStudies.bastar-lost-wax.quote.author'),
        role: t('caseStudies.bastar-lost-wax.quote.role'),
      },
    },
  ]);
</script>

<svelte:head>
  <title>{t('caseStudies.headTitle')}</title>
  <meta name="description" content={t('caseStudies.metaDescription')} />
</svelte:head>

<div class="studies-page">
  <div class="studies-container">
    <Breadcrumbs items={breadcrumbs} homeLabel={t('nav.marketplace')} />

    <!-- Header Section -->
    <header class="studies-header">
      <div class="kicker-wrap">
        <span class="studies-kicker">{t('caseStudies.kicker')}</span>
        <span class="kicker-sub">{t('caseStudies.kickerSub')}</span>
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
              <span class="portrait-role">{t('caseStudies.masterLineage')}</span>
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
                <span class="bar-title">{t('caseStudies.incomeChartTitle')}</span>
                <span class="bar-uplift-badge">
                  {t('caseStudies.upliftSurge', { percent: String(Math.round(((study.afterMonthlyInr - study.beforeMonthlyInr) / study.beforeMonthlyInr) * 100)) })}
                </span>
              </div>

              <div class="dual-progress-bar">
                <div class="bar-item bar-item--before">
                  <span class="bar-item__label">{t('caseStudies.beforeAmountLabel', { amount: study.beforeMonthlyInr.toLocaleString('en-IN') })}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill bar-fill--red"
                      style:inline-size="{(study.beforeMonthlyInr / study.afterMonthlyInr) * 100}%"
                    ></div>
                  </div>
                </div>

                <div class="bar-item bar-item--after">
                  <span class="bar-item__label">{t('caseStudies.afterAmountLabel', { amount: study.afterMonthlyInr.toLocaleString('en-IN') })}</span>
                  <div class="bar-track">
                    <div class="bar-fill bar-fill--green" style:inline-size="100%"></div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Before vs. After Structured Comparison Table -->
            <div class="before-after-section">
              <h3 class="comparison-heading">{t('caseStudies.comparisonHeading')}</h3>
              <div class="table-responsive">
                <table class="comparison-table">
                  <thead>
                    <tr>
                      <th scope="col">{t('caseStudies.tableDimensionHeader')}</th>
                      <th scope="col" class="th-before">{t('caseStudies.tableBeforeHeader')}</th>
                      <th scope="col" class="th-after">{t('caseStudies.tableAfterHeader')}</th>
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
                <h3>{t('caseStudies.narrative.problemHeading')}</h3>
                <p>{study.problem}</p>
              </div>

              <div class="narrative-block">
                <h3>{t('caseStudies.narrative.interventionHeading')}</h3>
                <p>{study.intervention}</p>
              </div>

              <div class="narrative-block">
                <h3>{t('caseStudies.narrative.impactHeading')}</h3>
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
                <span>{t('caseStudies.viewCardAction')}</span>
              </a>
              <a href="/artisan/{study.artisanSlug}" class="action-btn action-btn--store">
                <span>{t('caseStudies.exploreStorefrontAction')}</span>
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
