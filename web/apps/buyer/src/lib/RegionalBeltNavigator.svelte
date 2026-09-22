<!--
  apps/buyer/src/lib/RegionalBeltNavigator.svelte

  Geographical Heritage Corridors connecting buyers to India's 5 craft belts.
  Follows the Kalakriti design law: editorial rhythm, hairline separation,
  authentic typography, and first-class GI tag status.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { SectionHeader, Tooltip } from '@kalakriti/ui';
  import { Icon, type IconName } from '@kalakriti/icons';

  const t = $derived(locale.t);

  type Belt = {
    id: 'north' | 'west' | 'south' | 'east' | 'central';
    nameKey: string;
    craftsKey: string;
    descKey: string;
    icon: IconName;
    states: string;
    featuredCrafts: { nameKey: MessageKey; query: string; giYear: string }[];
  };

  const BELTS: Belt[] = [
    {
      id: 'north',
      nameKey: 'home.belts.north.name',
      craftsKey: 'home.belts.north.crafts',
      descKey: 'home.belts.north.desc',
      icon: 'weaving',
      states: 'Jammu & Kashmir, Himachal Pradesh, Uttar Pradesh, Punjab',
      featuredCrafts: [
        { nameKey: 'home.belts.north.craft.1.name', query: 'pashmina', giYear: 'GI-2008' },
        { nameKey: 'home.belts.north.craft.2.name', query: 'banarasi', giYear: 'GI-2009' },
        { nameKey: 'home.belts.north.craft.3.name', query: 'chikankari', giYear: 'GI-2008' },
        { nameKey: 'home.belts.north.craft.4.name', query: 'kullu', giYear: 'GI-2004' },
      ],
    },
    {
      id: 'west',
      nameKey: 'home.belts.west.name',
      craftsKey: 'home.belts.west.crafts',
      descKey: 'home.belts.west.desc',
      icon: 'block-printing',
      states: 'Gujarat, Rajasthan, Western Maharashtra',
      featuredCrafts: [
        { nameKey: 'home.belts.west.craft.1.name', query: 'ajrakh', giYear: 'GI-2024' },
        { nameKey: 'home.belts.west.craft.2.name', query: 'bagru', giYear: 'GI-2011' },
        { nameKey: 'home.belts.west.craft.3.name', query: 'patola', giYear: 'GI-2013' },
        { nameKey: 'home.belts.west.craft.4.name', query: 'rogan', giYear: 'GI-2023' },
      ],
    },
    {
      id: 'south',
      nameKey: 'home.belts.south.name',
      craftsKey: 'home.belts.south.crafts',
      descKey: 'home.belts.south.desc',
      icon: 'jewellery',
      states: 'Tamil Nadu, Telangana, Karnataka, Andhra Pradesh, Kerala',
      featuredCrafts: [
        { nameKey: 'home.belts.south.craft.1.name', query: 'kanchipuram', giYear: 'GI-2005' },
        { nameKey: 'home.belts.south.craft.2.name', query: 'pochampally', giYear: 'GI-2004' },
        { nameKey: 'home.belts.south.craft.3.name', query: 'bidriware', giYear: 'GI-2006' },
        { nameKey: 'home.belts.south.craft.4.name', query: 'kalamkari', giYear: 'GI-2006' },
      ],
    },
    {
      id: 'east',
      nameKey: 'home.belts.east.name',
      craftsKey: 'home.belts.east.crafts',
      descKey: 'home.belts.east.desc',
      icon: 'metalwork',
      states: 'West Bengal, Assam, Odisha, Nagaland, Manipur',
      featuredCrafts: [
        { nameKey: 'home.belts.east.craft.1.name', query: 'muga', giYear: 'GI-2007' },
        { nameKey: 'home.belts.east.craft.2.name', query: 'jamdani', giYear: 'GI-2016' },
        { nameKey: 'home.belts.east.craft.3.name', query: 'sambalpuri', giYear: 'GI-2010' },
        { nameKey: 'home.belts.east.craft.4.name', query: 'dokra', giYear: 'GI-2008' },
      ],
    },
    {
      id: 'central',
      nameKey: 'home.belts.central.name',
      craftsKey: 'home.belts.central.crafts',
      descKey: 'home.belts.central.desc',
      icon: 'charkha-spinner',
      states: 'Madhya Pradesh, Chhattisgarh, Eastern Maharashtra',
      featuredCrafts: [
        { nameKey: 'home.belts.central.craft.1.name', query: 'chanderi', giYear: 'GI-2005' },
        { nameKey: 'home.belts.central.craft.2.name', query: 'maheshwari', giYear: 'GI-2012' },
        { nameKey: 'home.belts.central.craft.3.name', query: 'bastar', giYear: 'GI-2008' },
        { nameKey: 'home.belts.central.craft.4.name', query: 'gond', giYear: 'GI-2023' },
      ],
    },
  ];

  let selectedBeltId = $state<'north' | 'west' | 'south' | 'east' | 'central'>('west');

  // Footer corridor links land on /?belt=<id>#belts.
  $effect(() => {
    const belt = BELTS.find((b) => b.id === page.url.searchParams.get('belt'));
    if (belt) selectedBeltId = belt.id;
  });

  const selectedBelt = $derived(
    BELTS.find((b) => b.id === selectedBeltId) ?? BELTS[1]
  );
</script>

<div class="navigator-container" id="belts">
  <SectionHeader
    kicker={t('home.belts.kicker')}
    heading={t('home.belts.heading')}
    href={`/search?q=${encodeURIComponent(selectedBelt.id)}`}
  />

  <!-- Belt Selector Navigation Tabs -->
  <div class="belt-nav-strip" role="tablist">
    {#each BELTS as belt (belt.id)}
      <Tooltip text={tooltip('tooltip.selectBelt')}>
        {#snippet trigger(tp)}
          <button
            type="button"
            role="tab"
            aria-selected={selectedBeltId === belt.id}
            class="belt-nav-item"
            class:active={selectedBeltId === belt.id}
            onclick={() => (selectedBeltId = belt.id)}
            {...tp}
          >
            <Icon name={belt.icon} size="1.1rem" />
            <span class="nav-text">
              {belt.id === 'north'
                ? t('home.belts.north.name')
                : belt.id === 'west'
                  ? t('home.belts.west.name')
                  : belt.id === 'south'
                    ? t('home.belts.south.name')
                    : belt.id === 'east'
                      ? t('home.belts.east.name')
                      : t('home.belts.central.name')}
            </span>
          </button>
        {/snippet}
      </Tooltip>
    {/each}
  </div>

  <!-- Editorial Corridor Showcase -->
  <div class="corridor-panel">
    <div class="corridor-summary">
      <div class="state-registry-line">
        <Icon name="location" size="1rem" />
        <span>{selectedBelt.states}</span>
      </div>

      <h3 class="corridor-title">
        {selectedBelt.id === 'north'
          ? t('home.belts.north.name')
          : selectedBelt.id === 'west'
            ? t('home.belts.west.name')
            : selectedBelt.id === 'south'
              ? t('home.belts.south.name')
              : selectedBelt.id === 'east'
                ? t('home.belts.east.name')
                : t('home.belts.central.name')}
      </h3>

      <p class="corridor-narrative">
        {selectedBelt.id === 'north'
          ? t('home.belts.north.desc')
          : selectedBelt.id === 'west'
            ? t('home.belts.west.desc')
            : selectedBelt.id === 'south'
              ? t('home.belts.south.desc')
              : selectedBelt.id === 'east'
                ? t('home.belts.east.desc')
                : t('home.belts.central.desc')}
      </p>

      <div class="gi-craft-register">
        <p class="register-label">{t('home.belts.registerLabel')}</p>
        <ul class="register-list" role="list">
          {#each selectedBelt.featuredCrafts as craft}
            <li class="register-item">
              <a href={`/search?q=${encodeURIComponent(craft.query)}`} class="craft-anchor">
                <span class="craft-name">{t(craft.nameKey)}</span>
                <span class="gi-badge">{craft.giYear}</span>
                <Icon name="arrow-right" size="0.85rem" />
              </a>
            </li>
          {/each}
        </ul>
      </div>
    </div>

    <div class="corridor-sidebar">
      <div class="sidebar-block">
        <p class="sidebar-header">{t('home.belts.sidebarHeader')}</p>
        <p class="sidebar-detail">
          {t('home.belts.sidebarDetail')}
        </p>
      </div>

      <div class="sidebar-metrics">
        <div class="metric-cell">
          <span class="metric-val">100%</span>
          <span class="metric-lbl">{t('home.belts.metric.directBankFloor')}</span>
        </div>
        <div class="metric-cell">
          <span class="metric-val">GI</span>
          <span class="metric-lbl">{t('home.belts.metric.pehchanVerified')}</span>
        </div>
      </div>

      <a
        href={`/search?q=${encodeURIComponent(selectedBelt.id)}`}
        class="k-corridor-link"
      >
        <span>{t('home.belts.viewBelt')}</span>
        <Icon name="arrow-right" size="1rem" />
      </a>
    </div>
  </div>
</div>

<style>
  .navigator-container {
    margin-block: var(--k-space-5);
  }

  .belt-nav-strip {
    display: flex;
    gap: var(--k-space-2);
    overflow-x: auto;
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    margin-block: var(--k-space-4);
    padding-block-end: var(--k-space-1);
    scrollbar-width: thin;
  }

  .belt-nav-item {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background: none;
    border: none;
    border-block-end: 2px solid transparent;
    color: var(--k-text-secondary);
    font-family: inherit;
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s ease;
  }

  .belt-nav-item:hover {
    color: var(--k-text-primary);
  }

  .belt-nav-item.active {
    color: var(--k-accent-primary-text);
    border-block-end-color: var(--k-accent-primary-bg);
    font-weight: var(--k-weight-semibold);
  }

  .corridor-panel {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-6);
    padding: var(--k-space-5);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
  }

  @media (min-width: 52rem) {
    .corridor-panel {
      grid-template-columns: 1.6fr 1fr;
      padding: var(--k-space-6);
    }
  }

  .state-registry-line {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .corridor-title {
    font-family: var(--k-font-display);
    font-size: var(--k-text-2xl);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .corridor-narrative {
    font-size: var(--k-text-base);
    line-height: 1.6;
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-4);
  }

  .register-label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .register-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr));
    gap: var(--k-space-2);
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .craft-anchor {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    transition: border-color 0.15s ease;
  }

  .craft-anchor:hover {
    border-color: var(--k-border-interactive);
    color: var(--k-accent-primary-text);
  }

  .gi-badge {
    font-family: monospace;
    font-size: 0.65rem;
    color: var(--k-text-secondary);
    background-color: var(--k-surface-sunken);
    padding: 2px 6px;
    border-radius: 2px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .corridor-sidebar {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--k-space-4);
    padding-inline-start: var(--k-space-5);
    border-inline-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  @media (max-width: 51.99rem) {
    .corridor-sidebar {
      padding-inline-start: 0;
      padding-block-start: var(--k-space-4);
      border-inline-start: none;
      border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    }
  }

  .sidebar-header {
    font-family: var(--k-font-display);
    font-size: var(--k-text-base);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-1) 0;
  }

  .sidebar-detail {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .sidebar-metrics {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--k-space-2);
  }

  .metric-cell {
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-sunken);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
  }

  .metric-val {
    display: block;
    font-family: var(--k-font-display);
    font-size: var(--k-text-lg);
    font-weight: bold;
    color: var(--k-accent-primary-text);
  }

  .metric-lbl {
    display: block;
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  .k-corridor-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-4);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    text-decoration: none;
    transition: background-color 0.15s ease;
  }

  .k-corridor-link:hover {
    background-color: var(--k-surface-sunken);
  }
</style>
