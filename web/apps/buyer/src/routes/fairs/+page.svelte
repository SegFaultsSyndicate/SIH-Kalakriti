<!--
  apps/buyer/src/routes/fairs/+page.svelte

  National Craft Fairs & Exhibitions Calendar:
  Bridges physical government trade exhibitions (Shilp Samagam, Surajkund Mela,
  Dilli Haat) with year-round digital storefronts and repeat ordering.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'Exhibition Calendar' },
  ]);

  interface Fair {
    id: string;
    title: string;
    edition: string;
    venue: string;
    city: string;
    state: string;
    dates: string;
    status: 'active' | 'upcoming';
    artisanCount: number;
    description: string;
    crafts: string[];
    partnerMinistry: string;
    bannerUrl: string;
  }

  const FAIRS: Fair[] = [
    {
      id: 'surajkund-2026',
      title: 'Surajkund International Crafts Mela',
      edition: '39th Annual Edition',
      venue: 'Surajkund Mela Grounds, Faridabad',
      city: 'Faridabad',
      state: 'Haryana',
      dates: 'Feb 2 – Feb 18, 2026',
      status: 'active',
      artisanCount: 420,
      description:
        "The world's largest open-air handicraft mela showcasing authentic rural craft lineages, living pit looms, and live woodcarving demonstrations by national awardees.",
      crafts: ['Kutch Ajrakh', 'Varanasi Kadwa Silk', 'Bastar Dhokra', 'Channapatna Lacquerware', 'Sohrai Painting'],
      partnerMinistry: 'Ministry of Tourism & Haryana Tourism Council',
      bannerUrl: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
    },
    {
      id: 'dilli-haat-ina',
      title: 'Dilli Haat Master Crafts Fortnight',
      edition: 'National Artisan Rotation',
      venue: 'INA Market Complex, New Delhi',
      city: 'New Delhi',
      state: 'Delhi NCR',
      dates: 'Year-Round Rotating Fortnights',
      status: 'active',
      artisanCount: 180,
      description:
        'A permanent cultural marketplace enabling empanelled weavers and rural craft cooperatives to bypass wholesale brokers and sell directly to metropolitan buyers.',
      crafts: ['Kalamkari Textiles', 'Pochampally Ikat', 'Bankura Terracotta', 'Madhubani Art', 'Jaipur Blue Pottery'],
      partnerMinistry: 'Delhi Tourism & DC (Handicrafts)',
      bannerUrl: '/craft-images/pottery/nizamabad-black-pottery.jpg',
    },
    {
      id: 'shilp-samagam-2026',
      title: 'Shilp Samagam Apex Handicraft Expo',
      edition: '2026 National Pavilion',
      venue: 'Major Dhyan Chand National Stadium, India Gate',
      city: 'New Delhi',
      state: 'Delhi NCR',
      dates: 'Nov 1 – Nov 15, 2026',
      status: 'upcoming',
      artisanCount: 280,
      description:
        'The apex central government exhibition uniting GI-certified craft clusters, tribal SHG micro-entrepreneurs, and PM Vishwakarma certified master lineages.',
      crafts: ['Kashmir Pashmina', 'Bidri Metalware', 'Tanjore Sacred Art', 'Dhokra Casting', 'Chamba Rumal'],
      partnerMinistry: 'Ministry of Social Justice & Ministry of Textiles',
      bannerUrl: '/craft-images/embroidery/kashmir_pashmina_sozni_01.jpeg',
    },
    {
      id: 'saras-mela-2026',
      title: 'Saras Mela Rural Livelihoods Festival',
      edition: 'Eastern India Pavilion',
      venue: 'Gandhi Maidan, Patna',
      city: 'Patna',
      state: 'Bihar',
      dates: 'Dec 10 – Dec 22, 2026',
      status: 'upcoming',
      artisanCount: 310,
      description:
        'Dedicated rural livelihoods mela fostering institutional B2B procurement and direct market linkages for women artisan self-help groups (SHGs).',
      crafts: ['Sikki Golden Grass', 'Bhagalpur Tussar Silk', 'Sujani Kantha Stitch', 'Tikuli Art'],
      partnerMinistry: 'Ministry of Rural Development (MoRD)',
      bannerUrl: '/craft-images/basketry/sikki-grass-basketry.jpg',
    },
    {
      id: 'hunar-haat-mumbai',
      title: 'Hunar Haat Heritage Pavilion',
      edition: 'Western Metropolitan Expo',
      venue: 'MMRDA Grounds, Bandra-Kurla Complex',
      city: 'Mumbai',
      state: 'Maharashtra',
      dates: 'Jan 14 – Jan 25, 2026',
      status: 'upcoming',
      artisanCount: 250,
      description:
        'High-volume urban expo linking master leather crafters, bell metal founders, and handloom cooperatives directly with interior design houses and exporters.',
      crafts: ['Kolhapuri Chappal', 'Paithani Zari Silk', 'Warli Tribal Murals', 'Ajanta Terracotta'],
      partnerMinistry: 'Ministry of Minority Affairs',
      bannerUrl: '/craft-images/leatherwork/kolhapuri_chappals_01.jpeg',
    },
  ];

  let selectedFilter = $state<'all' | 'active' | 'upcoming'>('all');

  const filteredFairs = $derived(
    selectedFilter === 'all' ? FAIRS : FAIRS.filter((f) => f.status === selectedFilter),
  );
</script>

<svelte:head>
  <title>National Craft Fairs & Exhibitions — Kalakriti</title>
  <meta
    name="description"
    content="Official calendar of national craft exhibitions (Shilp Samagam, Surajkund, Dilli Haat) connecting physical fair stalls to continuous digital storefronts."
  />
</svelte:head>

<div class="fairs-page">
  <div class="fairs-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <!-- Header Section -->
    <header class="fairs-header">
      <div class="gov-badge-strip">
        <span class="gov-badge">GOVERNMENT OF INDIA • EXHIBITION CORRIDOR</span>
        <span class="gov-badge-sub">PHYSICAL-TO-DIGITAL FAIR BRIDGE</span>
      </div>
      <h1 class="fairs-title">{t('exhibition.fairs.title')}</h1>
      <p class="fairs-subhead">
        {t('exhibition.fairs.subtitle')}
      </p>

      <!-- Filter Tabs -->
      <div class="filter-strip" role="tablist">
        <button
          type="button"
          class="filter-btn"
          class:filter-btn--active={selectedFilter === 'all'}
          onclick={() => (selectedFilter = 'all')}
        >
          All Exhibitions ({FAIRS.length})
        </button>
        <button
          type="button"
          class="filter-btn"
          class:filter-btn--active={selectedFilter === 'active'}
          onclick={() => (selectedFilter = 'active')}
        >
          <Icon name="calendar" size="1rem" /> Happening Now (2)
        </button>
        <button
          type="button"
          class="filter-btn"
          class:filter-btn--active={selectedFilter === 'upcoming'}
          onclick={() => (selectedFilter = 'upcoming')}
        >
          <Icon name="clock" size="1rem" /> Upcoming Fairs (3)
        </button>
      </div>
    </header>

    <!-- Fairs List -->
    <div class="fairs-grid">
      {#each filteredFairs as fair (fair.id)}
        <article class="fair-card">
          <div class="fair-card__banner">
            <img src={fair.bannerUrl} alt={fair.title} loading="lazy" />
            <div class="fair-status-badge" class:fair-status-badge--live={fair.status === 'active'}>
              {#if fair.status === 'active'}
                <span class="pulse-dot"></span> HAPPENING NOW
              {:else}
                UPCOMING
              {/if}
            </div>
          </div>

          <div class="fair-card__body">
            <div class="fair-meta">
              <span class="fair-edition">{fair.edition}</span>
              <span class="dot">•</span>
              <span class="fair-ministry">{fair.partnerMinistry}</span>
            </div>

            <h2 class="fair-card__title">{fair.title}</h2>

            <div class="fair-info-row">
              <div class="info-item">
                <Icon name="calendar" size="0.95rem" />
                <span>{fair.dates}</span>
              </div>
              <div class="info-item">
                <Icon name="location" size="0.95rem" />
                <span>{fair.venue}</span>
              </div>
            </div>

            <p class="fair-card__desc">{fair.description}</p>

            <div class="fair-crafts-section">
              <span class="crafts-label">Featured GI Craft Traditions:</span>
              <div class="crafts-tags">
                {#each fair.crafts as craft}
                  <span class="craft-tag">{craft}</span>
                {/each}
              </div>
            </div>

            <div class="fair-card__footer">
              <div class="artisan-count-badge">
                <strong>{fair.artisanCount}+</strong> Master Artisans Exhibiting
              </div>
              <a
                href="/search?q={encodeURIComponent(fair.crafts[0])}"
                class="explore-btn"
              >
                <span>{t('exhibition.fairs.exploreArtisans')}</span>
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
  .fairs-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .fairs-container {
    max-inline-size: 72rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .fairs-header {
    margin-block-end: var(--k-space-8);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-6);
  }

  .gov-badge-strip {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-2);
  }

  .gov-badge {
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: 0.68rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    padding: 3px 8px;
    border-radius: 3px;
  }

  .gov-badge-sub {
    font-size: 0.68rem;
    color: var(--k-text-secondary);
    font-weight: 600;
    letter-spacing: 0.05em;
  }

  .fairs-title {
    font-size: var(--k-text-2xl);
    font-weight: 800;
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2);
  }

  .fairs-subhead {
    max-inline-size: 52rem;
    color: var(--k-text-secondary);
    font-size: var(--k-text-md);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-5);
  }

  .filter-strip {
    display: flex;
    gap: var(--k-space-2);
    overflow-x: auto;
  }

  .filter-btn {
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-raised);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    padding: var(--k-space-2) var(--k-space-4);
    border-radius: var(--k-radius-pill);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s ease;
  }

  .filter-btn--active {
    background: var(--k-accent-primary-bg, var(--k-accent-primary-bg));
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg, var(--k-border-accent));
  }

  .fairs-grid {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-6);
  }

  .fair-card {
    display: grid;
    grid-template-columns: 20rem 1fr;
    border-radius: var(--k-radius-lg);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background: var(--k-surface-base);
    overflow: hidden;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.05);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }

  .fair-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.09);
  }

  .fair-card__banner {
    position: relative;
    inline-size: 100%;
    block-size: 100%;
    min-block-size: 14rem;
  }

  .fair-card__banner img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .fair-status-badge {
    position: absolute;
    inset-block-start: var(--k-space-3);
    inset-inline-start: var(--k-space-3);
    background: rgba(15, 23, 42, 0.85);
    color: var(--k-text-on-accent);
    font-size: 0.7rem;
    font-weight: 800;
    padding: 4px 10px;
    border-radius: var(--k-radius-pill);
    display: flex;
    align-items: center;
    gap: 6px;
    letter-spacing: 0.06em;
  }

  .fair-status-badge--live {
    background: var(--k-accent-success-bg);
  }

  .pulse-dot {
    inline-size: 7px;
    block-size: 7px;
    border-radius: 50%;
    background: var(--k-neem-400);
    box-shadow: 0 0 0 2px rgba(74, 222, 128, 0.4);
    animation: pulse 1.5s infinite;
  }

  @keyframes pulse {
    0% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.7);
    }
    70% {
      transform: scale(1);
      box-shadow: 0 0 0 6px rgba(74, 222, 128, 0);
    }
    100% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(74, 222, 128, 0);
    }
  }

  .fair-card__body {
    padding: var(--k-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .fair-meta {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .fair-edition {
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  .fair-card__title {
    font-size: var(--k-text-xl);
    font-weight: 800;
    color: var(--k-text-primary);
    margin: 0;
  }

  .fair-info-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-4);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .info-item {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-weight: 600;
  }

  .fair-card__desc {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    line-height: var(--k-leading-normal);
    margin: 0;
  }

  .fair-crafts-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .crafts-label {
    font-size: var(--k-text-xs);
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .crafts-tags {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-1);
  }

  .craft-tag {
    background: var(--k-surface-raised);
    border: 1px solid var(--k-border-hairline);
    padding: 3px 8px;
    border-radius: var(--k-radius-xs);
    font-size: 0.72rem;
    color: var(--k-text-secondary);
  }

  .fair-card__footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-block-start: 1px solid var(--k-border-subtle);
    padding-block-start: var(--k-space-3);
    margin-block-start: auto;
  }

  .artisan-count-badge {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .artisan-count-badge strong {
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .explore-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    padding: var(--k-space-2) var(--k-space-4);
    border-radius: var(--k-radius-md);
    font-size: var(--k-text-xs);
    font-weight: 700;
    text-decoration: none;
    transition: background 0.15s ease;
  }

  .explore-btn:hover {
    background: var(--k-accent-primary-bg);
  }

  @media (max-width: 768px) {
    .fair-card {
      grid-template-columns: 1fr;
    }
    .fair-card__banner {
      min-block-size: 10rem;
    }
  }
</style>
