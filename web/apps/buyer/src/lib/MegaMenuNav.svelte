<!--
  apps/buyer/src/lib/MegaMenuNav.svelte

  IndiaHandmade-style Mega Navigation & Government Header for Kalakriti.
  Strictly adheres to GIGW 3.0 government accessibility standards,
  DPDP Act 2023 directives, and Svelte 5 runes ($state, $derived).

  Features:
  - Top Government of India & Ministry of Social Justice & Empowerment ribbon
  - Brand header with national emblem, search engine, and institutional action tools
  - Comprehensive category mega-menus:
    1. Home and Living (6 detailed columns: Décor, Kitchen, Furnishings, Floor, Music, Stationery, Temple)
    2. Women
    3. Kids
    4. Men
    5. Furniture (Home Outdoors, Home Indoor, Offices)
    6. Paintings (Traditional Folk, Modern/Contemporary)
    7. GI Tagged (highlighted corridor linking to /gi-tagged)
    8. ODOP (One District One Product directory)
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { goto } from '$app/navigation';
  import { showToast } from '@kalakriti/ui';

  const t = $derived(locale.t);

  let activeMenu = $state<string | null>(null);
  let searchQuery = $state('');
  let searchCategory = $state('all');
  let isMobileMenuOpen = $state(false);

  function handleSearchSubmit(e: Event): void {
    e.preventDefault();
    const q = searchQuery.trim();
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (searchCategory === 'gi') params.set('gi_tagged', 'true');
    else if (searchCategory !== 'all') params.set('category', searchCategory);
    
    activeMenu = null;
    isMobileMenuOpen = false;
    void goto(`/search?${params.toString()}`);
  }

  function handleMenuHover(menuName: string): void {
    activeMenu = menuName;
  }

  function handleMenuLeave(): void {
    activeMenu = null;
  }

  function toggleMenu(menuName: string): void {
    activeMenu = activeMenu === menuName ? null : menuName;
  }

  async function toggleLang(): Promise<void> {
    const nextLang = locale.code === 'hi' ? 'en' : 'hi';
    await locale.set(nextLang, { persist: true });
    showToast({
      message: nextLang === 'hi' ? 'भाषा हिन्दी में बदली गई' : 'Language switched to English',
      variant: 'info',
    });
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<header class="k-mega-header" onmouseleave={handleMenuLeave}>
  <!-- 1. Government of India Top Ribbon -->
  <div class="k-gov-ribbon">
    <div class="k-gov-ribbon-container">
      <div class="k-gov-left">
        <span class="flag-icon" aria-hidden="true">🇮🇳</span>
        <span class="gov-title">Government of India</span>
        <span class="gov-divider">|</span>
        <span class="gov-ministry">{t('app.ministry') || 'Ministry of Social Justice & Empowerment'}</span>
      </div>

      <div class="k-gov-right">
        <a href="/accessibility" class="gov-a11y-link" title="Accessibility & Screen Reader Access">
          Screen Reader Access
        </a>
        <span class="gov-divider">|</span>
        <button type="button" class="gov-lang-btn" onclick={toggleLang} aria-label="Toggle language">
          <span class="lang-symbol">Aअ</span>
          <span class="lang-name">{locale.code === 'hi' ? 'English' : 'हिन्दी'}</span>
          <Icon name="chevron-down" size="0.75rem" />
        </button>
      </div>
    </div>
  </div>

  <!-- 2. Main Branding & Search Header Bar -->
  <div class="k-main-header">
    <div class="k-main-header-container">
      <!-- Mobile hamburger -->
      <button
        type="button"
        class="mobile-menu-toggle"
        onclick={() => (isMobileMenuOpen = !isMobileMenuOpen)}
        aria-label="Toggle navigation menu"
        aria-expanded={isMobileMenuOpen}
      >
        <Icon name={isMobileMenuOpen ? 'close' : 'menu'} size="1.4rem" />
      </button>

      <!-- Brand Lockup -->
      <a class="k-brand-lockup" href="/">
        <img
          class="k-brand-emblem"
          src="/favicon.svg"
          alt="State Emblem of India"
          width="42"
          height="42"
        />
        <div class="k-brand-text">
          <span class="k-brand-name">Kalakriti</span>
          <span class="k-brand-tagline">Gateway to Indian GI Heritage</span>
        </div>
      </a>

      <!-- Centered Comprehensive Search Box -->
      <form class="k-search-bar" onsubmit={handleSearchSubmit} role="search">
        <div class="search-category-select">
          <select bind:value={searchCategory} aria-label="Filter search domain">
            <option value="all">All Heritage</option>
            <option value="gi">GI Tagged Only</option>
            <option value="home">Home & Living</option>
            <option value="apparel">Apparel & Silk</option>
            <option value="paintings">Paintings</option>
          </select>
        </div>
        <input
          type="search"
          class="k-search-input"
          placeholder="Search authentic GI crafts, handlooms, master artisans..."
          bind:value={searchQuery}
          aria-label="Search authentic GI crafts"
        />
        <button type="submit" class="k-search-submit" aria-label="Submit search">
          <Icon name="search" size="1.15rem" />
        </button>
      </form>

      <!-- Right Action Tools -->
      <div class="k-header-tools">
        <a href="/gi-tagged" class="k-tool-badge" title="Verified Geographical Indications">
          <span class="gi-pin-dot"></span>
          <span class="tool-text">GI Registry</span>
        </a>

        <a href="/orders" class="k-tool-link" aria-label="Artisan Direct Orders">
          <Icon name="collective-order" size="1.2rem" />
          <span class="tool-label">Orders</span>
        </a>

        <a href="/orders/thank-you" class="k-tool-link k-tool-link--highlight" aria-label="Sign In or Track Lots">
          <Icon name="user" size="1.2rem" />
          <span class="tool-label">Sign In</span>
        </a>
      </div>
    </div>
  </div>

  <!-- 3. Primary Horizontal Category Menu Bar -->
  <nav class="k-category-nav" aria-label="Main Merchandise Categories">
    <div class="k-category-container">
      <ul class="k-category-list" role="menubar">
        <!-- 1. HOME AND LIVING (MEGA MENU) -->
        <li
          class="k-nav-item"
          role="none"
          onmouseenter={() => handleMenuHover('home')}
        >
          <button
            type="button"
            class="k-nav-link"
            class:active={activeMenu === 'home'}
            onclick={() => toggleMenu('home')}
            aria-haspopup="true"
            aria-expanded={activeMenu === 'home'}
          >
            <span>HOME AND LIVING</span>
            <Icon name="chevron-down" size="0.7rem" />
          </button>

          {#if activeMenu === 'home'}
            <div class="k-mega-dropdown" role="menu">
              <div class="mega-grid-6">
                <!-- Col 1: Home Décor & Utility -->
                <div class="mega-col">
                  <h4 class="mega-heading">Home Décor and Utility</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=candle">Artistic Candles</a></li>
                    <li><a href="/search?q=painting">Artworks & Paintings</a></li>
                    <li><a href="/search?q=candlestick">Candle Sticks</a></li>
                    <li><a href="/search?q=clock">Ethnic Clock</a></li>
                    <li><a href="/search?q=incense">Incense Sticks</a></li>
                    <li><a href="/search?q=metal">Metal Wares & Dhokra</a></li>
                    <li><a href="/search?q=mirror">Mirrors & Inlay</a></li>
                    <li><a href="/search?q=papermache">Paper Works & Papier Mâché</a></li>
                    <li><a href="/search?q=stoneware">Stone Wares & Soapstone</a></li>
                    <li><a href="/search?q=tapestry">Tapestries & Wall Hangings</a></li>
                    <li><a href="/search?q=wood">Wooden Handicrafts</a></li>
                    <li><a href="/search?q=walkingstick">Walking Sticks</a></li>
                  </ul>
                </div>

                <!-- Col 2: Kitchen & Furnishings -->
                <div class="mega-col">
                  <h4 class="mega-heading">Kitchen and Dining</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=placemat">Place Mats</a></li>
                    <li><a href="/search?q=towel">Dish Cloths & Towels</a></li>
                    <li><a href="/search?q=napkin">Table Napkins</a></li>
                    <li><a href="/search?q=copper">Copper Bottles & Jugs</a></li>
                    <li><a href="/search?q=claypot">Vedic Clay Cookware</a></li>
                  </ul>

                  <h4 class="mega-heading" style="margin-top: 1rem;">Home Furnishings</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=bedsheet">Handloom Bedsheets</a></li>
                    <li><a href="/search?q=quilt">Kantha Throws & Quilts</a></li>
                    <li><a href="/search?q=pillow">Pillow Covers</a></li>
                    <li><a href="/search?q=tablerunner">Table Runners</a></li>
                    <li><a href="/search?q=cushion">Cushion Covers</a></li>
                  </ul>
                </div>

                <!-- Col 3: Floor Coverings & Sports -->
                <div class="mega-col">
                  <h4 class="mega-heading">Floor Coverings</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=carpet">Mirzapur Hand-Knotted Carpets</a></li>
                    <li><a href="/search?q=rug">Kashmir Silk Rugs</a></li>
                    <li><a href="/search?q=durrie">Bhavani Jamakkalam Durries</a></li>
                    <li><a href="/search?q=doormat">Coir & Jute Doormats</a></li>
                    <li><a href="/search?q=yogamat">Natural Kusha Grass Mats</a></li>
                  </ul>

                  <h4 class="mega-heading" style="margin-top: 1rem;">Sports & Fitness</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=bow">Traditional Bow & Arrow</a></li>
                    <li><a href="/search?q=yogamat">Organic Yoga Mats</a></li>
                    <li><a href="/search?q=massager">Wooden Acupressure Massager</a></li>
                  </ul>
                </div>

                <!-- Col 4: Musical Instruments & Accessories -->
                <div class="mega-col">
                  <h4 class="mega-heading">Musical Instruments</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=flute">Bamboo Flutes</a></li>
                    <li><a href="/search?q=tabla">Handcrafted Tabla Pair</a></li>
                    <li><a href="/search?q=sitar">Miraj Sitar & Tanpura</a></li>
                    <li><a href="/search?q=dholak">Wood Dholak</a></li>
                    <li><a href="/search?q=shehnai">Varanasi Shehnai</a></li>
                  </ul>

                  <h4 class="mega-heading" style="margin-top: 1rem;">Accessories</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=travelbag">Shantiniketan Leather Bags</a></li>
                    <li><a href="/search?q=bottlebag">Jute Bottle Bags</a></li>
                    <li><a href="/search?q=handfan">Palm Leaf Hand Fans</a></li>
                  </ul>
                </div>

                <!-- Col 5: Stationery & Lighting -->
                <div class="mega-col">
                  <h4 class="mega-heading">Stationery</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=diary">Handmade Paper Diaries</a></li>
                    <li><a href="/search?q=bookmark">Leather & Silk Bookmarks</a></li>
                    <li><a href="/search?q=folder">Kalamkari File Folders</a></li>
                    <li><a href="/search?q=penstand">Brass & Wood Pen Stands</a></li>
                  </ul>

                  <h4 class="mega-heading" style="margin-top: 1rem;">Lighting</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=floorlamp">Lacquered Wood Floor Lamps</a></li>
                    <li><a href="/search?q=pendant">Terracotta Pendant Lights</a></li>
                    <li><a href="/search?q=tealight">Brass Tea Light Holders</a></li>
                    <li><a href="/search?q=sconce">Moradabad Wall Sconces</a></li>
                  </ul>
                </div>

                <!-- Col 6: Bath, Wellness & Religious Items -->
                <div class="mega-col">
                  <h4 class="mega-heading">Bath & Wellness</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=gamcha">Assam Gamchas</a></li>
                    <li><a href="/search?q=towels">Khadi Bath Towels</a></li>
                    <li><a href="/search?q=wellness">Herbal & Meditation</a></li>
                  </ul>

                  <h4 class="mega-heading" style="margin-top: 1rem;">Religious Items</h4>
                  <ul class="mega-sublist">
                    <li><a href="/search?q=kalash">Pooja Brass Kalash</a></li>
                    <li><a href="/search?q=tulsi">Tulsi Kanthi Mala</a></li>
                    <li><a href="/search?q=idols">Swamimalai Bronze Idols</a></li>
                    <li><a href="/search?q=japamala">Sandalwood Japa Mala</a></li>
                  </ul>
                </div>
              </div>
            </div>
          {/if}
        </li>

        <!-- 2. WOMEN -->
        <li class="k-nav-item" role="none">
          <a href="/search?q=women" class="k-nav-link" role="menuitem">
            <span>WOMEN</span>
          </a>
        </li>

        <!-- 3. KIDS -->
        <li class="k-nav-item" role="none">
          <a href="/search?q=toys" class="k-nav-link" role="menuitem">
            <span>KIDS</span>
          </a>
        </li>

        <!-- 4. MEN -->
        <li class="k-nav-item" role="none">
          <a href="/search?q=men" class="k-nav-link" role="menuitem">
            <span>MEN</span>
          </a>
        </li>

        <!-- 5. FURNITURE (Image 5 Dropdown) -->
        <li
          class="k-nav-item"
          role="none"
          onmouseenter={() => handleMenuHover('furniture')}
        >
          <button
            type="button"
            class="k-nav-link"
            class:active={activeMenu === 'furniture'}
            onclick={() => toggleMenu('furniture')}
            aria-haspopup="true"
            aria-expanded={activeMenu === 'furniture'}
          >
            <span>FURNITURE</span>
            <Icon name="chevron-down" size="0.7rem" />
          </button>

          {#if activeMenu === 'furniture'}
            <div class="k-simple-dropdown" role="menu">
              <div class="dropdown-group">
                <h4 class="dropdown-heading">Home Outdoors</h4>
                <ul>
                  <li><a href="/search?q=patiochair">Patio Chairs</a></li>
                  <li><a href="/search?q=patiosofa">Patio Sofas</a></li>
                  <li><a href="/search?q=sunlounger">Sun Loungers</a></li>
                  <li><a href="/search?q=swing">Hand-carved Swings & Accessories</a></li>
                </ul>
              </div>

              <div class="dropdown-group">
                <h4 class="dropdown-heading">Home Indoor</h4>
                <ul>
                  <li><a href="/search?q=bedsidetable">Bedside Tables</a></li>
                  <li><a href="/search?q=diningtable">Dining Tables</a></li>
                </ul>
              </div>

              <div class="dropdown-group">
                <h4 class="dropdown-heading">Offices</h4>
                <ul>
                  <li><a href="/search?q=cabinet">Cabinets & Bookcases</a></li>
                  <li><a href="/search?q=chair">Solid Teak Chairs</a></li>
                  <li><a href="/search?q=rack">Magazines & Newspaper Racks</a></li>
                  <li><a href="/search?q=sofa">Sofa Sets & Couches</a></li>
                </ul>
              </div>
            </div>
          {/if}
        </li>

        <!-- 6. PAINTINGS (Image 2 Dropdown) -->
        <li
          class="k-nav-item"
          role="none"
          onmouseenter={() => handleMenuHover('paintings')}
        >
          <button
            type="button"
            class="k-nav-link"
            class:active={activeMenu === 'paintings'}
            onclick={() => toggleMenu('paintings')}
            aria-haspopup="true"
            aria-expanded={activeMenu === 'paintings'}
          >
            <span>PAINTINGS</span>
            <Icon name="chevron-down" size="0.7rem" />
          </button>

          {#if activeMenu === 'paintings'}
            <div class="k-simple-dropdown k-simple-dropdown--compact" role="menu">
              <div class="dropdown-group">
                <a href="/search?q=traditional+painting" class="dropdown-block-link">
                  <strong>TRADITIONAL PAINTINGS</strong>
                  <span>Madhubani, Pattachitra, Warli, Pichwai, Tanjore, Gond</span>
                </a>
              </div>
              <div class="dropdown-group">
                <a href="/search?q=modern+painting" class="dropdown-block-link">
                  <strong>MODERN PAINTINGS</strong>
                  <span>Contemporary Folk, Canvas Murals, Stylized Tribal Motifs</span>
                </a>
              </div>
            </div>
          {/if}
        </li>

        <!-- 7. GI TAGGED (Image 3 Direct Corridor) -->
        <li class="k-nav-item k-nav-item--gi" role="none">
          <a href="/gi-tagged" class="k-nav-link k-nav-link--gi" role="menuitem">
            <span class="gi-tag-star"><Icon name="gi-tagged" size="0.85rem" /></span>
            <span>GI TAGGED</span>
          </a>
        </li>

        <!-- 8. ODOP -->
        <li class="k-nav-item" role="none">
          <a href="/catalog" class="k-nav-link" role="menuitem">
            <span>ODOP</span>
          </a>
        </li>
      </ul>
    </div>
  </nav>
</header>

<!-- Mobile Dropdown Sidebar Drawer -->
{#if isMobileMenuOpen}
  <div class="mobile-drawer-overlay" onclick={() => (isMobileMenuOpen = false)} role="presentation">
    <div
      class="mobile-drawer"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && (isMobileMenuOpen = false)}
      tabindex="-1"
      role="dialog"
      aria-label="Mobile Navigation"
    >
      <div class="mobile-drawer-head">
        <div class="brand-mini">
          <img src="/favicon.svg" alt="" width="28" height="28" />
          <span>Kalakriti Categories</span>
        </div>
        <button type="button" class="drawer-close" onclick={() => (isMobileMenuOpen = false)}>
          &times;
        </button>
      </div>

      <div class="mobile-drawer-content">
        <a href="/gi-tagged" class="mobile-cat-link mobile-cat-link--gi" onclick={() => (isMobileMenuOpen = false)}>
          <Icon name="gi-tagged" size="0.9rem" /> GI TAGGED PRODUCTS
        </a>
        <a href="/search?q=home" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Home and Living
        </a>
        <a href="/search?q=women" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Women Apparel & Sarees
        </a>
        <a href="/search?q=men" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Men Khadi & Kurtas
        </a>
        <a href="/search?q=toys" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Kids Wooden Toys
        </a>
        <a href="/search?q=furniture" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Furniture & Atelier
        </a>
        <a href="/search?q=paintings" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Traditional & Folk Paintings
        </a>
        <a href="/catalog" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          ODOP - One District One Product
        </a>
        <a href="/case-studies" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          Guild Case Studies
        </a>
        <a href="/orders" class="mobile-cat-link" onclick={() => (isMobileMenuOpen = false)}>
          My Purchase Lots & Orders
        </a>
      </div>
    </div>
  </div>
{/if}

<style>
  .k-mega-header {
    position: relative;
    inline-size: 100%;
    background-color: var(--k-surface-base, var(--k-surface-base));
    border-block-end: 1px solid var(--k-border-hairline, var(--k-khadi-150));
    z-index: 40;
    font-family: inherit;
  }

  /* 1. Government Ribbon */
  .k-gov-ribbon {
    background-color: var(--k-madder-800); /* Official deep maroon ribbon */
    color: var(--k-text-on-accent);
    font-size: 0.75rem;
    padding: 0.35rem 1rem;
    border-block-end: 1px solid rgba(0, 0, 0, 0.1);
  }

  .k-gov-ribbon-container {
    max-inline-size: 80rem;
    margin-inline: auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .k-gov-left {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .gov-title {
    font-weight: 600;
    letter-spacing: 0.02em;
  }

  .gov-divider {
    color: rgba(255, 255, 255, 0.5);
    margin-inline: 0.2rem;
  }

  .gov-ministry {
    color: rgba(255, 255, 255, 0.9);
  }

  .k-gov-right {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .gov-a11y-link {
    color: var(--k-text-on-accent);
    text-decoration: none;
    font-size: 0.72rem;
  }

  .gov-a11y-link:hover {
    text-decoration: underline;
  }

  .gov-lang-btn {
    background: none;
    border: none;
    color: var(--k-text-on-accent);
    font-size: 0.72rem;
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    cursor: pointer;
    padding: 0;
  }

  .lang-symbol {
    font-weight: 700;
  }

  /* 2. Main Header Bar */
  .k-main-header {
    padding: 0.85rem 1rem;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-stone-200, var(--k-khadi-150));
  }

  .k-main-header-container {
    max-inline-size: 80rem;
    margin-inline: auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1.5rem;
  }

  .mobile-menu-toggle {
    display: none;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--k-text-primary);
    padding: 0.25rem;
  }

  @media (max-width: 64rem) {
    .mobile-menu-toggle {
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
  }

  .k-brand-lockup {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    text-decoration: none;
    color: inherit;
    flex-shrink: 0;
  }

  .k-brand-emblem {
    inline-size: 2.5rem;
    block-size: 2.5rem;
    object-fit: contain;
  }

  .k-brand-text {
    display: flex;
    flex-direction: column;
  }

  .k-brand-name {
    font-family: var(--k-font-display, serif);
    font-size: 1.45rem;
    font-weight: 700;
    line-height: 1.1;
    color: var(--k-madder-800); /* Heritage burgundy */
    letter-spacing: -0.01em;
  }

  .k-brand-tagline {
    font-size: 0.65rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--k-text-secondary, var(--k-stone-500));
    font-weight: 500;
  }

  /* Centered Search Engine */
  .k-search-bar {
    flex: 1;
    max-inline-size: 38rem;
    display: flex;
    align-items: center;
    background-color: var(--k-surface-base);
    border: 1.5px solid var(--k-madder-800);
    border-radius: 4px;
    overflow: hidden;
  }

  @media (max-width: 50rem) {
    .k-search-bar {
      display: none;
    }
  }

  .search-category-select select {
    background-color: var(--k-surface-raised);
    border: none;
    border-inline-end: 1px solid var(--k-stone-300, var(--k-border-hairline));
    font-size: 0.8rem;
    color: var(--k-text-primary);
    padding: 0.65rem 0.75rem;
    cursor: pointer;
    outline: none;
  }

  .k-search-input {
    flex: 1;
    border: none;
    padding: 0.65rem 0.85rem;
    font-size: 0.85rem;
    color: var(--k-text-primary);
    outline: none;
  }

  .k-search-submit {
    background-color: var(--k-madder-800);
    color: var(--k-text-on-accent);
    border: none;
    padding: 0.65rem 1.15rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 0.15s ease;
  }

  .k-search-submit:hover {
    background-color: var(--k-madder-800);
  }

  /* Header Right Tools */
  .k-header-tools {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    flex-shrink: 0;
  }

  .k-tool-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.75rem;
    border-radius: 999px;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-haldi-500);
    color: var(--k-haldi-700);
    text-decoration: none;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .gi-pin-dot {
    inline-size: 0.55rem;
    block-size: 0.55rem;
    border-radius: 50%;
    background-color: var(--k-accent-primary-bg);
    box-shadow: 0 0 0 2px var(--k-haldi-500);
  }

  .k-tool-link {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--k-text-primary, var(--k-text-primary));
    text-decoration: none;
    font-size: 0.85rem;
    font-weight: 500;
  }

  .k-tool-link:hover {
    color: var(--k-madder-800);
  }

  .k-tool-link--highlight {
    color: var(--k-madder-800);
    font-weight: 600;
  }

  /* 3. Horizontal Category Navigation Bar */
  .k-category-nav {
    background-color: var(--k-surface-base);
    border-block-end: 1.5px solid var(--k-madder-800);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
  }

  @media (max-width: 64rem) {
    .k-category-nav {
      display: none;
    }
  }

  .k-category-container {
    max-inline-size: 80rem;
    margin-inline: auto;
  }

  .k-category-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .k-nav-item {
    position: relative;
  }

  .k-nav-link {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.75rem 0.85rem;
    background: none;
    border: none;
    border-block-end: 3px solid transparent;
    color: var(--k-madder-800);
    text-decoration: none;
    font-size: 0.82rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    cursor: pointer;
    transition: all 0.15s ease;
    white-space: nowrap;
  }

  .k-nav-link:hover,
  .k-nav-link.active {
    background-color: var(--k-surface-base);
    border-block-end-color: var(--k-terracotta-700); /* Saffron underline */
    color: var(--k-accent-primary-text);
  }

  .k-nav-link--gi {
    color: var(--k-madder-600);
    font-weight: 800;
  }

  .gi-tag-star {
    color: var(--k-haldi-500);
    font-size: 0.9rem;
  }

  /* MEGA DROPDOWN (Image 1 Style) */
  .k-mega-dropdown {
    position: absolute;
    inset-block-start: 100%;
    inset-inline-start: 0;
    inline-size: 78rem;
    max-inline-size: 92vw;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-stone-300, var(--k-border-hairline));
    border-block-start: none;
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.16);
    padding: 1.5rem;
    z-index: 50;
    max-block-size: 80vh;
    overflow-y: auto;
  }

  .mega-grid-6 {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 1.25rem;
  }

  .mega-col {
    display: flex;
    flex-direction: column;
    border-inline-end: 1px solid var(--k-khadi-100);
    padding-inline-end: 0.85rem;
  }

  .mega-col:last-child {
    border-inline-end: none;
  }

  .mega-heading {
    font-size: 0.78rem;
    text-transform: uppercase;
    font-weight: 700;
    color: var(--k-madder-800);
    margin: 0 0 0.65rem 0;
    letter-spacing: 0.04em;
    line-height: 1.3;
    border-block-end: 1.5px solid var(--k-khadi-150);
    padding-block-end: 0.35rem;
  }

  .mega-sublist {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .mega-sublist a {
    color: var(--k-text-secondary);
    text-decoration: none;
    font-size: 0.78rem;
    line-height: 1.3;
    transition: color 0.15s ease;
  }

  .mega-sublist a:hover {
    color: var(--k-accent-primary-text);
    text-decoration: underline;
  }

  /* SIMPLE DROPDOWN (Image 2 & 5 Style) */
  .k-simple-dropdown {
    position: absolute;
    inset-block-start: 100%;
    inset-inline-start: 0;
    min-inline-size: 16rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-stone-300, var(--k-border-hairline));
    border-block-start: none;
    box-shadow: 0 12px 28px rgba(0, 0, 0, 0.12);
    padding: 1.25rem;
    z-index: 50;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .k-simple-dropdown--compact {
    min-inline-size: 20rem;
  }

  .dropdown-group h4 {
    font-size: 0.8rem;
    text-transform: uppercase;
    font-weight: 700;
    color: var(--k-madder-800);
    margin: 0 0 0.5rem 0;
    letter-spacing: 0.03em;
  }

  .dropdown-group ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .dropdown-group a {
    color: var(--k-text-secondary);
    text-decoration: none;
    font-size: 0.8rem;
    transition: color 0.15s ease;
  }

  .dropdown-group a:hover {
    color: var(--k-accent-primary-text);
    text-decoration: underline;
  }

  .dropdown-block-link {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    text-decoration: none;
    color: inherit;
    padding: 0.4rem;
    border-radius: 4px;
    transition: background-color 0.15s ease;
  }

  .dropdown-block-link:hover {
    background-color: var(--k-surface-raised);
  }

  .dropdown-block-link strong {
    color: var(--k-madder-800);
    font-size: 0.82rem;
  }

  .dropdown-block-link span {
    font-size: 0.72rem;
    color: var(--k-text-secondary, var(--k-stone-500));
  }

  /* Mobile Drawer Overlay */
  .mobile-drawer-overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.6);
    z-index: 100;
  }

  .mobile-drawer {
    inline-size: 20rem;
    max-inline-size: 85vw;
    block-size: 100%;
    background-color: var(--k-surface-base);
    box-shadow: 4px 0 24px rgba(0, 0, 0, 0.25);
    display: flex;
    flex-direction: column;
  }

  .mobile-drawer-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem;
    background-color: var(--k-madder-800);
    color: var(--k-text-on-accent);
  }

  .brand-mini {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 700;
  }

  .drawer-close {
    background: none;
    border: none;
    color: var(--k-text-on-accent);
    font-size: 1.5rem;
    cursor: pointer;
  }

  .mobile-drawer-content {
    display: flex;
    flex-direction: column;
    padding: 1rem;
    gap: 0.5rem;
    overflow-y: auto;
  }

  .mobile-cat-link {
    padding: 0.65rem 0.5rem;
    border-block-end: 1px solid var(--k-stone-200, var(--k-khadi-150));
    color: var(--k-text-primary, var(--k-text-primary));
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 600;
  }

  .mobile-cat-link--gi {
    color: var(--k-accent-primary-text);
    font-weight: 700;
  }
</style>
