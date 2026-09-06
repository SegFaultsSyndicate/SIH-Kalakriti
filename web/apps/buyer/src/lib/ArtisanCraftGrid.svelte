<!--
  apps/buyer/src/lib/ArtisanCraftGrid.svelte

  12 National Artisan Craft Disciplines Grid for Buyer Marketplace.
  Exposes all 12 artisan onboarding craft disciplines as first-class buyer categories.
  Adheres to the Kalakriti design system:
  - Khadi/paper textured cards with hairline rules
  - Authentic craft iconography from @kalakriti/icons
  - Direct navigation to /search?category=...
  - Svelte 5 runes
-->
<script lang="ts">
  import { Icon } from '@kalakriti/icons';
  import { ARTISAN_CRAFT_CATEGORIES, type CraftCategory } from './craft-categories';

  let hoveredCraft = $state<string | null>(null);
</script>

<section class="craft-categories-section" aria-labelledby="craft-categories-heading">
  <div class="craft-section-header">
    <div class="header-text-group">
      <span class="section-kicker">Indigenous Craft Taxonomy</span>
      <h2 id="craft-categories-heading" class="section-title">
        Browse by Artisan Craft Discipline
      </h2>
      <p class="section-subtitle">
        Direct from verified guild looms, foundry forges, and carving studios across 74 registered GI clusters.
      </p>
    </div>

    <a href="/catalog" class="catalog-all-link">
      <span>View Cluster Directory</span>
      <Icon name="arrow-right" size="0.85rem" />
    </a>
  </div>

  <div class="craft-grid-12">
    {#each ARTISAN_CRAFT_CATEGORIES as craft (craft.id)}
      <a
        href={`/search?category=${encodeURIComponent(craft.name)}`}
        class="craft-card"
        onmouseenter={() => (hoveredCraft = craft.id)}
        onmouseleave={() => (hoveredCraft = null)}
      >
        <div class="craft-card__header">
          <div class="craft-card__icon-badge">
            <Icon name={craft.icon} size="1.4rem" />
          </div>
          <span class="craft-card__gi-pill">
            {craft.giCount} GI Hubs
          </span>
        </div>

        <div class="craft-card__body">
          <div class="craft-card__name-row">
            <h3 class="craft-card__name">{craft.name}</h3>
            <span class="craft-card__hindi">{craft.hindiName}</span>
          </div>

          <p class="craft-card__subtitle">{craft.subtitle}</p>
          <p class="craft-card__tagline">{craft.tagline}</p>
        </div>

        <div class="craft-card__footer">
          <div class="craft-card__regions">
            <Icon name="location" size="0.75rem" />
            <span>{craft.regions.slice(0, 2).join(' · ')}</span>
          </div>
          <span class="craft-card__cta">
            Explore ➔
          </span>
        </div>
      </a>
    {/each}
  </div>
</section>

<style>
  .craft-categories-section {
    padding-block: 2.5rem;
    border-block-start: 1px solid #e8e2d8;
    border-block-end: 1px solid #e8e2d8;
    background-color: #faf7f2;
    margin-block: 1rem;
  }

  .craft-section-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 1.5rem;
    margin-block-end: 2rem;
    flex-wrap: wrap;
  }

  .header-text-group {
    max-inline-size: 42rem;
  }

  .section-kicker {
    display: inline-block;
    font-size: 0.75rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: #b84a39;
    margin-block-end: 0.4rem;
  }

  .section-title {
    font-size: clamp(1.4rem, 2.8vw, 2rem);
    font-weight: 800;
    color: #1e1915;
    line-height: 1.2;
    margin: 0 0 0.5rem 0;
    letter-spacing: -0.01em;
  }

  .section-subtitle {
    font-size: 0.9rem;
    color: #6b635a;
    line-height: 1.5;
    margin: 0;
  }

  .catalog-all-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    font-weight: 700;
    color: #b84a39;
    text-decoration: none;
    padding: 0.5rem 0.85rem;
    border-radius: 6px;
    background-color: #ffffff;
    border: 1px solid #ded7cb;
    transition: all 0.15s ease;
  }

  .catalog-all-link:hover {
    background-color: #f0eae1;
    border-color: #c9bea9;
  }

  .craft-grid-12 {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1rem;
  }

  @media (max-width: 1100px) {
    .craft-grid-12 {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  @media (max-width: 768px) {
    .craft-grid-12 {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .craft-grid-12 {
      grid-template-columns: 1fr;
    }
  }

  .craft-card {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    background-color: #ffffff;
    border: 1px solid #e5dfd5;
    border-radius: 10px;
    padding: 1.25rem;
    text-decoration: none;
    color: inherit;
    transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
    position: relative;
    overflow: hidden;
  }

  .craft-card::after {
    content: '';
    position: absolute;
    inset-inline: 0;
    inset-block-start: 0;
    block-size: 3px;
    background: transparent;
    transition: background-color 0.18s ease;
  }

  .craft-card:hover {
    transform: translateY(-3px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    border-color: #b84a39;
  }

  .craft-card:hover::after {
    background-color: #b84a39;
  }

  .craft-card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-block-end: 0.85rem;
  }

  .craft-card__icon-badge {
    inline-size: 2.6rem;
    block-size: 2.6rem;
    border-radius: 8px;
    background-color: #f7f4ed;
    border: 1px solid #e5dfd3;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #8c3b2d;
    transition: background-color 0.15s ease, color 0.15s ease;
  }

  .craft-card:hover .craft-card__icon-badge {
    background-color: #b84a39;
    color: #ffffff;
    border-color: #b84a39;
  }

  .craft-card__gi-pill {
    font-size: 0.68rem;
    font-weight: 700;
    color: #635b52;
    background-color: #f2eee8;
    padding: 0.2rem 0.5rem;
    border-radius: 999px;
    border: 1px solid #e0d8cd;
  }

  .craft-card__body {
    flex: 1;
    margin-block-end: 1rem;
  }

  .craft-card__name-row {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    margin-block-end: 0.25rem;
  }

  .craft-card__name {
    font-size: 1.05rem;
    font-weight: 700;
    color: #1e1915;
    margin: 0;
  }

  .craft-card__hindi {
    font-size: 0.85rem;
    color: #8c8278;
    font-weight: 500;
  }

  .craft-card__subtitle {
    font-size: 0.775rem;
    font-weight: 600;
    color: #8c3b2d;
    margin: 0 0 0.45rem 0;
  }

  .craft-card__tagline {
    font-size: 0.775rem;
    color: #5c544d;
    line-height: 1.45;
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .craft-card__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-block-start: 0.75rem;
    border-block-start: 1px solid #f0ebe2;
    font-size: 0.725rem;
  }

  .craft-card__regions {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    color: #7a7269;
  }

  .craft-card__cta {
    font-weight: 700;
    color: #b84a39;
    transition: transform 0.15s ease;
  }

  .craft-card:hover .craft-card__cta {
    transform: translateX(3px);
  }
</style>
