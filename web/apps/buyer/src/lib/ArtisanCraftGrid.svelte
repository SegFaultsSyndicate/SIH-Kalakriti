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
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { ARTISAN_CRAFT_CATEGORIES } from './craft-categories';

  const t = $derived(locale.t);
</script>

<section class="craft-categories-section" aria-labelledby="craft-categories-heading">
  <div class="craft-section-header">
    <div class="header-text-group">
      <span class="section-kicker">{t('craftGrid.kicker')}</span>
      <h2 id="craft-categories-heading" class="section-title">
        {t('craftGrid.title')}
      </h2>
      <p class="section-subtitle">
        {t('craftGrid.subtitle')}
      </p>
    </div>

    <a href="/catalog" class="catalog-all-link">
      <span>{t('craftGrid.viewDirectory')}</span>
      <Icon name="arrow-right" size="0.85rem" />
    </a>
  </div>

  <div class="craft-grid-12">
    {#each ARTISAN_CRAFT_CATEGORIES as craft (craft.id)}
      <a
        href={`/search?category=${encodeURIComponent(craft.name)}`}
        class="craft-card"
      >
        <div class="craft-card__header">
          <div class="craft-card__icon-badge">
            <Icon name={craft.icon} size="1.4rem" />
          </div>
          <span class="craft-card__gi-pill">
            {t('craftGrid.giHubs', { count: String(craft.giCount) })}
          </span>
        </div>

        <div class="craft-card__body">
          <div class="craft-card__name-row">
            <h3 class="craft-card__name">{t(craft.nameKey)}</h3>
            <span class="craft-card__hindi">{t(craft.nativeNameKey)}</span>
          </div>

          <p class="craft-card__subtitle">{t(craft.subtitleKey)}</p>
          <p class="craft-card__tagline">{t(craft.taglineKey)}</p>
        </div>

        <div class="craft-card__footer">
          <div class="craft-card__regions">
            <Icon name="location" size="0.75rem" />
            <span>{craft.regions.slice(0, 2).join(' · ')}</span>
          </div>
          <span class="craft-card__cta">
            {t('craftGrid.explore')}
          </span>
        </div>
      </a>
    {/each}
  </div>
</section>

<style>
  .craft-categories-section {
    padding-block: 2.5rem;
    border-block-start: 1px solid var(--k-border-subtle);
    border-block-end: 1px solid var(--k-border-subtle);
    background-color: var(--k-surface-base);
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
    color: var(--k-accent-danger-muted);
    margin-block-end: 0.4rem;
  }

  .section-title {
    font-size: clamp(1.4rem, 2.8vw, 2rem);
    font-weight: 800;
    color: var(--k-text-primary);
    line-height: 1.2;
    margin: 0 0 0.5rem 0;
    letter-spacing: -0.01em;
  }

  .section-subtitle {
    font-size: 0.9rem;
    color: var(--k-text-tertiary);
    line-height: 1.5;
    margin: 0;
  }

  .catalog-all-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    font-weight: 700;
    color: var(--k-accent-danger-muted);
    text-decoration: none;
    padding: 0.5rem 0.85rem;
    border-radius: 6px;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-muted);
    transition: all 0.15s ease;
  }

  .catalog-all-link:hover {
    background-color: var(--k-surface-raised);
    border-color: var(--k-border-hairline);
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
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
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
    border-color: var(--k-border-danger);
  }

  .craft-card:hover::after {
    background-color: var(--k-accent-danger-bg);
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
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-border-subtle);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--k-terracotta-600);
    transition: background-color 0.15s ease, color 0.15s ease;
  }

  .craft-card:hover .craft-card__icon-badge {
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-border-danger);
  }

  .craft-card__gi-pill {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--k-stone-600);
    background-color: var(--k-surface-raised);
    padding: 0.2rem 0.5rem;
    border-radius: 999px;
    border: 1px solid var(--k-border-muted);
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
    color: var(--k-text-primary);
    margin: 0;
  }

  .craft-card__hindi {
    font-size: 0.85rem;
    color: var(--k-stone-400);
    font-weight: 500;
  }

  .craft-card__subtitle {
    font-size: 0.775rem;
    font-weight: 600;
    color: var(--k-terracotta-600);
    margin: 0 0 0.45rem 0;
  }

  .craft-card__tagline {
    font-size: 0.775rem;
    color: var(--k-stone-600);
    line-height: 1.45;
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .craft-card__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-block-start: 0.75rem;
    border-block-start: 1px solid var(--k-khadi-100);
    font-size: 0.725rem;
  }

  .craft-card__regions {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--k-text-tertiary);
  }

  .craft-card__cta {
    font-weight: 700;
    color: var(--k-accent-danger-muted);
    transition: transform 0.15s ease;
  }

  .craft-card:hover .craft-card__cta {
    transform: translateX(3px);
  }
</style>
