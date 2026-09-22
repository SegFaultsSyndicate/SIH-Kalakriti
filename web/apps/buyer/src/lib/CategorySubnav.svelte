<!--
  apps/buyer/src/lib/CategorySubnav.svelte

  Subtle, high-design secondary craft taxonomy navigation bar.
  Incorporates the authentic IndiaHandmade categories (Home & Living, Furniture, Paintings, GI Tagged)
  with Kalakriti's luxury minimalist aesthetic, eliminating clunky government styling.

  Features:
  - Clean horizontal taxonomy strip with subtle hover micro-interactions
  - Refined dropdown menus for Home & Living, Furniture, and Paintings
  - Direct links to GI Tagged Directory (/gi-tagged) and ODOP corridors
  - Accessible keyboard navigation & click-away dismissal
  - Svelte 5 runes ($state, $derived, $effect)
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
import { Tooltip } from '@kalakriti/ui';
  import { ARTISAN_CRAFT_CATEGORIES } from './craft-categories';

  const t = $derived(locale.t);

  let activeMenu = $state<string | null>(null);
  let navContainer: HTMLElement | null = $state(null);
  let menuPos = $state<{ top: number; left: number } | null>(null);
  let allCraftsWrap: HTMLElement | null = $state(null);
  let homeWrap: HTMLElement | null = $state(null);
  let furnitureWrap: HTMLElement | null = $state(null);
  let paintingsWrap: HTMLElement | null = $state(null);

  // .subnav-container scrolls horizontally, which clips an absolutely
  // positioned dropdown to that scrollbox (the CSS overflow spec forces
  // overflow-y to clip too once overflow-x isn't visible) -- it renders
  // hidden behind later page content instead of floating above it. Fixed
  // positioning computed from the trigger's own rect escapes that clip.
  function positionMenu(el: HTMLElement): void {
    if (window.innerWidth <= 768) {
      menuPos = null;
      return;
    }
    const r = el.getBoundingClientRect();
    // Dropdowns run up to ~880px wide (all-crafts mega-menu) -- clamp so a
    // trigger near the right edge doesn't push the panel off-screen.
    const left = Math.min(r.left, window.innerWidth - 900);
    menuPos = { top: r.bottom + 6, left: Math.max(left, 8) };
  }

  function handleMenuEnter(menu: string, el: HTMLElement): void {
    activeMenu = menu;
    positionMenu(el);
  }

  function handleMenuLeave(): void {
    activeMenu = null;
  }

  function toggleMenu(menu: string, el: HTMLElement): void {
    if (activeMenu === menu) {
      activeMenu = null;
      return;
    }
    activeMenu = menu;
    positionMenu(el);
  }

  function closeMenu(): void {
    activeMenu = null;
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape' && activeMenu) {
      activeMenu = null;
    }
  }

  // Click outside listener
  $effect(() => {
    if (!activeMenu) return;

    function handleClickOutside(event: MouseEvent): void {
      if (navContainer && !navContainer.contains(event.target as Node)) {
        activeMenu = null;
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<nav
  class="category-subnav"
  aria-label={t('subnav.ariaLabel')}
  bind:this={navContainer}
  onmouseleave={handleMenuLeave}
>
  <div class="subnav-container">
    <!-- 0. All Crafts Mega-Menu featuring all 12 Artisan Crafts -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="menu-item-wrap"
      bind:this={allCraftsWrap}
      onmouseenter={() => allCraftsWrap && handleMenuEnter('all-crafts', allCraftsWrap)}
    >
      <div class="subnav-split-pill {activeMenu === 'all-crafts' ? 'is-active' : ''}">
        <a href="/catalog" class="subnav-pill-link all-btn" onclick={closeMenu}>
          <Icon name="cluster" size="0.85rem" />
          <span>{t('subnav.allCrafts')}</span>
        </a>
        <Tooltip text={tooltip('tooltip.menu')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="subnav-chevron-btn"
              onclick={(e) => { e.preventDefault(); e.stopPropagation(); if (allCraftsWrap) toggleMenu('all-crafts', allCraftsWrap); }}
              aria-expanded={activeMenu === 'all-crafts'}
              aria-haspopup="true"
              aria-label={t('subnav.allCraftsToggle')}
              {...tp}
            >
              <Icon name="chevron-down" size="0.7rem" />
            </button>
          {/snippet}
        </Tooltip>
      </div>

      {#if activeMenu === 'all-crafts'}
        <div
          class="subnav-dropdown all-crafts-dropdown"
          role="menu"
          style={menuPos ? `position: fixed; top: ${menuPos.top}px; left: ${menuPos.left}px;` : ''}
        >
          <div class="all-crafts-header">
            <div class="all-crafts-title-group">
              <h4 class="col-title">{t('subnav.allCraftsHeading')}</h4>
              <p class="all-crafts-desc">{t('subnav.allCraftsDesc')}</p>
            </div>
            <a href="/catalog" class="all-crafts-all-link" onclick={closeMenu}>
              {t('subnav.allCraftsLink')}
            </a>
          </div>

          <div class="crafts-12-grid">
            {#each ARTISAN_CRAFT_CATEGORIES as craft}
              <a
                href={`/search?category=${encodeURIComponent(craft.name)}`}
                class="craft-mega-card"
                onclick={closeMenu}
              >
                <span class="craft-mega-card__icon">
                  <Icon name={craft.icon} size="1.25rem" />
                </span>
                <span class="craft-mega-card__info">
                  <span class="craft-mega-card__name">
                    <strong>{t(craft.nameKey)}</strong>
                    <span class="craft-mega-card__hindi">{t(craft.nativeNameKey)}</span>
                  </span>
                  <small>{t(craft.subtitleKey)}</small>
                </span>
              </a>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <span class="subnav-divider">|</span>

    <!-- 1. Weaving & Handlooms -->
    <a href="/search?category=Weaving" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.weaving')}</span>
    </a>

    <!-- 2. Block Printing -->
    <a href="/search?category=Block+printing" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.blockPrinting')}</span>
    </a>

    <!-- 3. Pottery -->
    <a href="/search?category=Pottery" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.pottery')}</span>
    </a>

    <!-- 4. Metalwork -->
    <a href="/search?category=Metalwork" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.metalwork')}</span>
    </a>

    <!-- 5. Woodwork -->
    <a href="/search?category=Woodwork" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.woodwork')}</span>
    </a>

    <!-- 6. Embroidery -->
    <a href="/search?category=Embroidery" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.embroidery')}</span>
    </a>

    <!-- 1. Home & Living (Image 1 reference) -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="menu-item-wrap"
      bind:this={homeWrap}
      onmouseenter={() => homeWrap && handleMenuEnter('home', homeWrap)}
    >
      <div class="subnav-split-pill {activeMenu === 'home' ? 'is-active' : ''}">
        <a
          href="/search?category=Home+and+living"
          class="subnav-pill-link"
          onclick={closeMenu}
        >
          <span>{t('subnav.home')}</span>
        </a>
        <Tooltip text={tooltip('tooltip.menu')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="subnav-chevron-btn"
              onclick={(e) => { e.preventDefault(); e.stopPropagation(); if (homeWrap) toggleMenu('home', homeWrap); }}
              aria-expanded={activeMenu === 'home'}
              aria-haspopup="true"
              aria-label={t('subnav.homeToggle')}
              {...tp}
            >
              <Icon name="chevron-down" size="0.7rem" />
            </button>
          {/snippet}
        </Tooltip>
      </div>

      {#if activeMenu === 'home'}
        <div
          class="subnav-dropdown home-dropdown"
          role="menu"
          style={menuPos ? `position: fixed; top: ${menuPos.top}px; left: ${menuPos.left}px;` : ''}
        >
          <div class="dropdown-grid-4">
            <!-- Col 1: Décor & Utility -->
            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.home.decor')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=candle" onclick={closeMenu}>{t('subnav.home.decor.candles')}</a></li>
                <li><a href="/search?q=clock" onclick={closeMenu}>{t('subnav.home.decor.clocks')}</a></li>
                <li><a href="/search?q=metalware" onclick={closeMenu}>{t('subnav.home.decor.metalware')}</a></li>
                <li><a href="/search?q=mirror" onclick={closeMenu}>{t('subnav.home.decor.mirrors')}</a></li>
                <li><a href="/search?q=papier+mache" onclick={closeMenu}>{t('subnav.home.decor.papierMache')}</a></li>
                <li><a href="/search?q=stoneware" onclick={closeMenu}>{t('subnav.home.decor.stoneware')}</a></li>
                <li><a href="/search?q=tapestry" onclick={closeMenu}>{t('subnav.home.decor.tapestry')}</a></li>
              </ul>
            </div>

            <!-- Col 2: Kitchen & Furnishings -->
            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.home.kitchen')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=placemat" onclick={closeMenu}>{t('subnav.home.kitchen.placemats')}</a></li>
                <li><a href="/search?q=copper" onclick={closeMenu}>{t('subnav.home.kitchen.copper')}</a></li>
                <li><a href="/search?q=table+mat" onclick={closeMenu}>{t('subnav.home.kitchen.tableMats')}</a></li>
                <li><a href="/search?q=kitchen" onclick={closeMenu}>{t('subnav.home.kitchen.cookware')}</a></li>
              </ul>

              <h4 class="col-title sub-title-margin">{t('subnav.home.furnishings')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=bedsheet" onclick={closeMenu}>{t('subnav.home.furnishings.bedsheets')}</a></li>
                <li><a href="/search?q=quilt" onclick={closeMenu}>{t('subnav.home.furnishings.quilts')}</a></li>
                <li><a href="/search?q=cushion" onclick={closeMenu}>{t('subnav.home.furnishings.cushions')}</a></li>
              </ul>
            </div>

            <!-- Col 3: Floor Coverings & Musical -->
            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.home.floorCoverings')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=carpet" onclick={closeMenu}>{t('subnav.home.floorCoverings.carpets')}</a></li>
                <li><a href="/search?q=durrie" onclick={closeMenu}>{t('subnav.home.floorCoverings.durries')}</a></li>
                <li><a href="/search?q=rug" onclick={closeMenu}>{t('subnav.home.floorCoverings.rugs')}</a></li>
                <li><a href="/search?q=yoga+mat" onclick={closeMenu}>{t('subnav.home.floorCoverings.yogaMats')}</a></li>
              </ul>

              <h4 class="col-title sub-title-margin">{t('subnav.home.musical')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=flute" onclick={closeMenu}>{t('subnav.home.musical.flutes')}</a></li>
                <li><a href="/search?q=tabla" onclick={closeMenu}>{t('subnav.home.musical.tabla')}</a></li>
                <li><a href="/search?q=sitar" onclick={closeMenu}>{t('subnav.home.musical.sitars')}</a></li>
                <li><a href="/search?q=dholak" onclick={closeMenu}>{t('subnav.home.musical.dholaks')}</a></li>
              </ul>
            </div>

            <!-- Col 4: Wellness & Temple Items -->
            <div class="dropdown-col highlight-col">
              <h4 class="col-title">{t('subnav.home.templeWellness')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=pooja" onclick={closeMenu}>{t('subnav.home.templeWellness.kalash')}</a></li>
                <li><a href="/search?q=incense" onclick={closeMenu}>{t('subnav.home.templeWellness.incense')}</a></li>
                <li><a href="/search?q=tulsi" onclick={closeMenu}>{t('subnav.home.templeWellness.tulsiKanthi')}</a></li>
                <li><a href="/search?q=towel" onclick={closeMenu}>{t('subnav.home.templeWellness.towels')}</a></li>
                <li><a href="/search?q=meditation" onclick={closeMenu}>{t('subnav.home.templeWellness.meditationAsanas')}</a></li>
              </ul>

              <div class="direct-gi-callout">
                <span class="callout-tag">{t('subnav.home.giCalloutTag')}</span>
                <p>{t('subnav.home.giCalloutDesc')}</p>
                <a href="/gi-tagged" class="callout-link" onclick={closeMenu}>
                  {t('subnav.home.giCalloutLink')}
                </a>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- 2. Furniture (Image 5 reference) -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="menu-item-wrap"
      bind:this={furnitureWrap}
      onmouseenter={() => furnitureWrap && handleMenuEnter('furniture', furnitureWrap)}
    >
      <div class="subnav-split-pill {activeMenu === 'furniture' ? 'is-active' : ''}">
        <a
          href="/search?category=Furniture"
          class="subnav-pill-link"
          onclick={closeMenu}
        >
          <span>{t('subnav.furniture')}</span>
        </a>
        <Tooltip text={tooltip('tooltip.menu')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="subnav-chevron-btn"
              onclick={(e) => { e.preventDefault(); e.stopPropagation(); if (furnitureWrap) toggleMenu('furniture', furnitureWrap); }}
              aria-expanded={activeMenu === 'furniture'}
              aria-haspopup="true"
              aria-label={t('subnav.furnitureToggle')}
              {...tp}
            >
              <Icon name="chevron-down" size="0.7rem" />
            </button>
          {/snippet}
        </Tooltip>
      </div>

      {#if activeMenu === 'furniture'}
        <div
          class="subnav-dropdown furniture-dropdown"
          role="menu"
          style={menuPos ? `position: fixed; top: ${menuPos.top}px; left: ${menuPos.left}px;` : ''}
        >
          <div class="dropdown-grid-3">
            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.furniture.outdoors')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=patio+chair" onclick={closeMenu}>{t('subnav.furniture.outdoors.patioChairs')}</a></li>
                <li><a href="/search?q=patio+sofa" onclick={closeMenu}>{t('subnav.furniture.outdoors.loungers')}</a></li>
                <li><a href="/search?q=swing" onclick={closeMenu}>{t('subnav.furniture.outdoors.swings')}</a></li>
              </ul>
            </div>

            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.furniture.indoor')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=table" onclick={closeMenu}>{t('subnav.furniture.indoor.bedsideTables')}</a></li>
                <li><a href="/search?q=dining" onclick={closeMenu}>{t('subnav.furniture.indoor.diningTables')}</a></li>
                <li><a href="/search?q=stool" onclick={closeMenu}>{t('subnav.furniture.indoor.stools')}</a></li>
              </ul>
            </div>

            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.furniture.office')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=cabinet" onclick={closeMenu}>{t('subnav.furniture.office.cabinets')}</a></li>
                <li><a href="/search?q=chair" onclick={closeMenu}>{t('subnav.furniture.office.accentChairs')}</a></li>
                <li><a href="/search?q=rack" onclick={closeMenu}>{t('subnav.furniture.office.magazineRacks')}</a></li>
                <li><a href="/search?q=sofa" onclick={closeMenu}>{t('subnav.furniture.office.sofaSets')}</a></li>
              </ul>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- 3. Paintings (Image 2 reference) -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="menu-item-wrap"
      bind:this={paintingsWrap}
      onmouseenter={() => paintingsWrap && handleMenuEnter('paintings', paintingsWrap)}
    >
      <div class="subnav-split-pill {activeMenu === 'paintings' ? 'is-active' : ''}">
        <a
          href="/search?category=Paintings"
          class="subnav-pill-link"
          onclick={closeMenu}
        >
          <span>{t('subnav.paintings')}</span>
        </a>
        <Tooltip text={tooltip('tooltip.menu')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="subnav-chevron-btn"
              onclick={(e) => { e.preventDefault(); e.stopPropagation(); if (paintingsWrap) toggleMenu('paintings', paintingsWrap); }}
              aria-expanded={activeMenu === 'paintings'}
              aria-haspopup="true"
              aria-label={t('subnav.paintingsToggle')}
              {...tp}
            >
              <Icon name="chevron-down" size="0.7rem" />
            </button>
          {/snippet}
        </Tooltip>
      </div>

      {#if activeMenu === 'paintings'}
        <div
          class="subnav-dropdown paintings-dropdown"
          role="menu"
          style={menuPos ? `position: fixed; top: ${menuPos.top}px; left: ${menuPos.left}px;` : ''}
        >
          <div class="dropdown-grid-2">
            <div class="dropdown-col">
              <h4 class="col-title">{t('subnav.paintings.traditional')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=madhubani" onclick={closeMenu}>{t('subnav.paintings.traditional.madhubani')}</a></li>
                <li><a href="/search?q=pattachitra" onclick={closeMenu}>{t('subnav.paintings.traditional.pattachitra')}</a></li>
                <li><a href="/search?q=warli" onclick={closeMenu}>{t('subnav.paintings.traditional.warli')}</a></li>
                <li><a href="/search?q=aipan" onclick={closeMenu}>{t('subnav.paintings.traditional.aipan')}</a></li>
                <li><a href="/search?q=pichwai" onclick={closeMenu}>{t('subnav.paintings.traditional.pichwai')}</a></li>
                <li><a href="/search?q=thangka" onclick={closeMenu}>{t('subnav.paintings.traditional.thangka')}</a></li>
              </ul>
            </div>

            <div class="dropdown-col highlight-col">
              <h4 class="col-title">{t('subnav.paintings.modern')}</h4>
              <ul class="col-links">
                <li><a href="/search?q=modern+folk" onclick={closeMenu}>{t('subnav.paintings.modern.abstracts')}</a></li>
                <li><a href="/search?q=canvas" onclick={closeMenu}>{t('subnav.paintings.modern.canvases')}</a></li>
                <li><a href="/search?q=framed" onclick={closeMenu}>{t('subnav.paintings.modern.framedEditions')}</a></li>
              </ul>

              <div class="direct-gi-callout">
                <span class="callout-tag">{t('subnav.paintings.calloutTag')}</span>
                <p>{t('subnav.paintings.calloutDesc')}</p>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- 7. Jewellery -->
    <a href="/search?category=Jewellery" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.jewellery')}</span>
    </a>

    <!-- 8. Bamboo & Basketry -->
    <a href="/search?category=Bamboo+craft" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.bambooCane')}</span>
    </a>

    <!-- 9. Stone Carving & Leather -->
    <a href="/search?category=Stone+carving" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.stoneLeather')}</span>
    </a>

    <!-- 10. GI Tagged Direct Link (Image 3 reference) -->
    <a href="/gi-tagged" class="subnav-item gi-tagged-link" onclick={closeMenu}>
      <span class="gi-tricolor-dot"></span>
      <span>{t('subnav.giTaggedProducts')}</span>
    </a>

    <!-- 11. ODOP Corridors -->
    <a href="/catalog" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.odop')}</span>
    </a>

    <!-- 12. Bulk Institutional Orders -->
    <a href="/bulk-order" class="subnav-item" onclick={closeMenu}>
      <span>{t('subnav.institutionalRfqs')}</span>
    </a>
  </div>
</nav>

<style>
  .category-subnav {
    position: relative;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-border-subtle);
    z-index: 80;
  }

  .subnav-container {
    inline-size: min(100% - (2 * var(--k-gutter, 1rem)), var(--k-container-buyer, 74rem));
    margin-inline: auto;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    overflow-x: auto;
    scrollbar-width: none;
    padding-block: 0.35rem;
  }

  .subnav-container::-webkit-scrollbar {
    display: none;
  }

  .subnav-item {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.35rem 0.65rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--k-text-secondary);
    text-decoration: none;
    white-space: nowrap;
    background: transparent;
    border: none;
    cursor: pointer;
    font-family: inherit;
    transition: color 0.12s ease, background-color 0.12s ease;
  }

  .subnav-item:hover {
    color: var(--k-accent-danger-muted);
    background-color: var(--k-surface-raised);
  }

  /* Split pill navigation: Left side navigates, Right side toggles */
  .subnav-split-pill {
    display: inline-flex;
    align-items: stretch;
    border-radius: 6px;
    background: transparent;
    transition: background-color 0.12s ease;
  }

  .subnav-split-pill:hover,
  .subnav-split-pill.is-active {
    background-color: var(--k-surface-raised);
  }

  .subnav-pill-link {
    display: inline-flex;
    align-items: center;
    padding: 0.35rem 0.3rem 0.35rem 0.65rem;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--k-text-secondary);
    text-decoration: none;
    white-space: nowrap;
    transition: color 0.12s ease;
  }

  .subnav-split-pill:hover .subnav-pill-link,
  .subnav-split-pill.is-active .subnav-pill-link {
    color: var(--k-accent-danger-muted);
  }

  .subnav-chevron-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.35rem 0.45rem 0.35rem 0.15rem;
    background: transparent;
    border: none;
    cursor: pointer;
    color: var(--k-text-tertiary);
    border-radius: 0 6px 6px 0;
    transition: color 0.12s ease;
  }

  .subnav-split-pill:hover .subnav-chevron-btn,
  .subnav-split-pill.is-active .subnav-chevron-btn {
    color: var(--k-accent-danger-muted);
  }

  .all-btn {
    color: var(--k-text-primary);
    font-weight: 700;
  }

  .subnav-divider {
    color: var(--k-khadi-200);
    font-size: 0.75rem;
    margin-inline: 0.15rem;
  }

  .gi-tagged-link {
    color: var(--k-accent-primary-text);
    font-weight: 700;
  }

  .gi-tricolor-dot {
    inline-size: 0.55rem;
    block-size: 0.55rem;
    border-radius: 50%;
    background: linear-gradient(180deg, #ff9933 33%, var(--k-surface-base) 33%, var(--k-surface-base) 66%, #138808 66%);
    border: 1px solid var(--k-border-hairline);
    flex: none;
  }

  /* Dropdown Positioning & Layout */
  .menu-item-wrap {
    position: relative;
  }

  .subnav-dropdown {
    position: absolute;
    inset-inline-start: 0;
    inset-block-start: calc(100% + 0.35rem);
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-muted);
    border-radius: 12px;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12), 0 2px 6px rgba(0, 0, 0, 0.04);
    padding: 1.5rem;
    z-index: 1000;
    animation: subnavFade 0.15s ease-out;
  }

  @keyframes subnavFade {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .all-crafts-dropdown {
    inline-size: 52rem;
    max-inline-size: 94vw;
  }

  .all-crafts-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-block-end: 0.85rem;
    margin-block-end: 1rem;
    border-block-end: 1px solid var(--k-border-subtle);
  }

  .all-crafts-desc {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
    margin: 0.2rem 0 0 0;
  }

  .all-crafts-all-link {
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--k-accent-danger-muted);
    text-decoration: none;
    white-space: nowrap;
  }

  .all-crafts-all-link:hover {
    text-decoration: underline;
  }

  @media (max-width: 768px) {
    .subnav-dropdown {
      position: fixed;
      inset-inline-start: 0.75rem;
      inset-inline-end: 0.75rem;
      inset-block-start: 5.25rem;
      inline-size: auto !important;
      max-inline-size: calc(100vw - 1.5rem) !important;
      max-block-size: 75vh;
      overflow-y: auto;
      padding: 1rem;
    }
  }

  .crafts-12-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.65rem;
  }

  @media (max-width: 800px) {
    .crafts-12-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .crafts-12-grid {
      grid-template-columns: 1fr;
    }
  }

  .craft-mega-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-radius: 8px;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    text-decoration: none;
    color: inherit;
    transition: all 0.15s ease;
  }

  .craft-mega-card:hover {
    background-color: var(--k-surface-base);
    border-color: var(--k-border-danger);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
    transform: translateY(-1px);
  }

  .craft-mega-card__icon {
    inline-size: 2.2rem;
    block-size: 2.2rem;
    border-radius: 6px;
    background-color: var(--k-surface-raised);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--k-terracotta-600);
    flex: none;
    transition: background-color 0.15s ease, color 0.15s ease;
  }

  .craft-mega-card:hover .craft-mega-card__icon {
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
  }

  .craft-mega-card__info {
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
  }

  .craft-mega-card__name {
    display: flex;
    align-items: baseline;
    gap: 0.4rem;
  }

  .craft-mega-card__name strong {
    font-size: 0.825rem;
    font-weight: 700;
    color: var(--k-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .craft-mega-card__hindi {
    font-size: 0.725rem;
    color: var(--k-stone-400);
  }

  .craft-mega-card__info small {
    font-size: 0.68rem;
    color: var(--k-text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .home-dropdown {
    inline-size: 48rem;
    max-inline-size: 92vw;
  }

  .furniture-dropdown {
    inline-size: 40rem;
    max-inline-size: 92vw;
  }

  .paintings-dropdown {
    inline-size: 36rem;
    max-inline-size: 92vw;
  }

  .dropdown-grid-4 {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.5rem;
  }

  .dropdown-grid-3 {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
  }

  .dropdown-grid-2 {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 800px) {
    .dropdown-grid-4,
    .dropdown-grid-3,
    .dropdown-grid-2 {
      grid-template-columns: 1fr;
    }
  }

  .dropdown-col {
    display: flex;
    flex-direction: column;
  }

  .highlight-col {
    background-color: var(--k-surface-base);
    padding: 0.85rem;
    border-radius: 8px;
    border: 1px solid var(--k-border-subtle);
  }

  .col-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 0.85rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0 0 0.65rem;
    padding-block-end: 0.35rem;
    border-block-end: 1px solid var(--k-border-subtle);
  }

  .sub-title-margin {
    margin-block-start: 1rem;
  }

  .col-links {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .col-links li a {
    font-size: 0.775rem;
    color: var(--k-stone-600);
    text-decoration: none;
    transition: color 0.12s ease;
    line-height: 1.3;
  }

  .col-links li a:hover {
    color: var(--k-accent-danger-muted);
    text-decoration: underline;
  }

  .direct-gi-callout {
    margin-block-start: 1rem;
    padding-block-start: 0.75rem;
    border-block-start: 1px dashed var(--k-border-hairline);
  }

  .callout-tag {
    font-size: 0.65rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-accent-success-muted);
    background-color: var(--k-surface-raised);
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
  }

  .direct-gi-callout p {
    font-size: 0.725rem;
    color: var(--k-text-tertiary);
    margin: 0.35rem 0 0.5rem;
    line-height: 1.35;
  }

  .callout-link {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--k-accent-danger-muted);
    text-decoration: none;
  }

  .callout-link:hover {
    text-decoration: underline;
  }
</style>
