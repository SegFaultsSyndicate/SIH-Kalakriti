<!--
  apps/buyer/src/routes/gi-tagged/+page.svelte

  Official GI Tagged Products Directory.
  Replicates the authentic IndiaHandmade (indiahandmade.com/gigoods/index) layout:
  - Breadcrumbs: Home > GI Tagged
  - Dynamic Item Counter & View switchers (Grid vs List)
  - Shopping Options Left Sidebar:
    1. Category Accordion (Home & living, Women, Men, Furniture, Paintings)
    2. Price Range Tiers (₹0 - ₹999, ₹1,000 - ₹1,999, etc.)
    3. Color Palette Swatches (12 botanical pigment swatches)
  - Product Cards with:
    - Official India GI Tricolor Map Pin Emblem
    - Strike-through MRP pricing with direct fair-trade discount %
    - Direct Order & Social Share copy actions
-->
<script lang="ts">
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { Breadcrumbs, type BreadcrumbItem, Money, showToast, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { wishlist } from '$lib/wishlist.svelte';
  import { GI_PRODUCTS, type GIProduct } from '$lib/demo-gi-products';

  const t = $derived(locale.t);

  const breadcrumbs = $derived<BreadcrumbItem[]>([{ label: t('giTagged.breadcrumb') }]);

  // View Layout
  let viewMode = $state<'grid' | 'list'>('grid');
  let selectedState = $state('all');
  let selectedCategory = $state('all');
  let selectedPriceRange = $state<string | null>(null);
  let selectedColor = $state<string | null>(null);
  let sortBy = $state('name_asc');

  // Accordion open states
  let isCategoryOpen = $state(true);
  let isPriceOpen = $state(true);
  let isColorOpen = $state(true);
  let isWeavingOpen = $state(true);
  let isPatternOpen = $state(true);
  let isFabricOpen = $state(true);
  let isDiscountOpen = $state(true);

  let selectedWeavingStyle = $state<string | null>(null);
  let selectedPattern = $state<string | null>(null);
  let selectedFabric = $state<string | null>(null);
  let selectedDiscount = $state<number | null>(null);


  // Colors from Image 4
  const COLOR_SWATCHES: { name: string; nameKey: MessageKey; hex: string }[] = [
    { name: 'Royal Blue', nameKey: 'giTagged.color.1', hex: '#0033cc' },
    { name: 'Raw Ivory', nameKey: 'giTagged.color.2', hex: '#fdfcf0' },
    { name: 'Kutch Ochre', nameKey: 'giTagged.color.3', hex: '#c68237' },
    { name: 'Madder Maroon', nameKey: 'giTagged.color.4', hex: '#8c2323' },
    { name: 'Pale Haldi', nameKey: 'giTagged.color.5', hex: '#fff9d2' },
    { name: 'Bright Haldi', nameKey: 'giTagged.color.6', hex: '#ffd200' },
    { name: 'Deep Crimson', nameKey: 'giTagged.color.7', hex: '#730000' },
    { name: 'Brass Gold', nameKey: 'giTagged.color.8', hex: '#f4c430' },
    { name: 'Marigold Orange', nameKey: 'giTagged.color.9', hex: '#ff9900' },
    { name: 'Gulabi Pink', nameKey: 'giTagged.color.10', hex: '#f8b9cb' },
    { name: 'Sindoor Red', nameKey: 'giTagged.color.11', hex: '#e60000' },
    { name: 'Forest Teal', nameKey: 'giTagged.color.12', hex: '#007a78' },
  ];

  // Price Tiers from Image 4
  const PRICE_TIERS: { label: string; labelKey: MessageKey; min: number; max: number; count: number }[] = [
    { label: '₹0 - ₹999', labelKey: 'giTagged.priceTier.1', min: 0, max: 99999, count: 20 },
    { label: '₹1,000 - ₹1,999', labelKey: 'giTagged.priceTier.2', min: 100000, max: 199999, count: 9 },
    { label: '₹2,000 - ₹2,999', labelKey: 'giTagged.priceTier.3', min: 200000, max: 299999, count: 9 },
    { label: '₹3,000 - ₹3,999', labelKey: 'giTagged.priceTier.4', min: 300000, max: 399999, count: 7 },
    { label: '₹4,000 - ₹4,999', labelKey: 'giTagged.priceTier.5', min: 400000, max: 499999, count: 10 },
    { label: '₹5,000 - ₹5,999', labelKey: 'giTagged.priceTier.6', min: 500000, max: 599999, count: 13 },
    { label: '₹6,000 - ₹6,999', labelKey: 'giTagged.priceTier.7', min: 600000, max: 699999, count: 5 },
    { label: '₹7,000 - ₹7,999', labelKey: 'giTagged.priceTier.8', min: 700000, max: 799999, count: 4 },
    { label: '₹9,000 - ₹9,999', labelKey: 'giTagged.priceTier.9', min: 900000, max: 999999, count: 2 },
    { label: '₹10,000 and above', labelKey: 'giTagged.priceTier.10', min: 1000000, max: 999999999, count: 27 },
  ];

  const CATEGORY_FILTERS: { label: string; labelKey: MessageKey; count: number }[] = [
    { label: 'Weaving', labelKey: 'craft.weaving.name', count: 38 },
    { label: 'Block printing', labelKey: 'craft.block-printing.name', count: 14 },
    { label: 'Pottery', labelKey: 'craft.pottery.name', count: 11 },
    { label: 'Metalwork', labelKey: 'craft.metalwork.name', count: 19 },
    { label: 'Woodwork', labelKey: 'craft.woodwork.name', count: 16 },
    { label: 'Embroidery', labelKey: 'craft.embroidery.name', count: 21 },
    { label: 'Painting', labelKey: 'craft.painting.name', count: 61 },
    { label: 'Basketry', labelKey: 'craft.basketry.name', count: 8 },
    { label: 'Jewellery', labelKey: 'craft.jewellery.name', count: 12 },
    { label: 'Leatherwork', labelKey: 'craft.leather.name', count: 7 },
    { label: 'Stone carving', labelKey: 'craft.stone.name', count: 9 },
    { label: 'Bamboo craft', labelKey: 'craft.bamboo.name', count: 15 },
    { label: 'Home and living', labelKey: 'giTagged.category.homeAndLiving', count: 22 },
    { label: 'Furniture', labelKey: 'giTagged.category.furniture', count: 6 },
    { label: 'Discounted Products', labelKey: 'giTagged.category.discountedProducts', count: 5 },
  ];

  // Weaving Style Filters (from IndiaHandmade reference)
  const WEAVING_STYLES: { label: string; labelKey: MessageKey; count: number }[] = [
    { label: 'Handloom with extra-weft thread/zari work', labelKey: 'giTagged.weavingStyle.1', count: 6 },
    { label: 'Kanjeevaram / Kanchipuram', labelKey: 'giTagged.weavingStyle.2', count: 1 },
  ];

  // Print or Pattern Type (from IndiaHandmade reference)
  const PATTERN_TYPES: { label: string; labelKey: MessageKey; count: number }[] = [
    { label: 'Checkered', labelKey: 'giTagged.patternType.1', count: 1 },
    { label: 'Ethnic Motif', labelKey: 'giTagged.patternType.2', count: 1 },
    { label: 'Others', labelKey: 'giTagged.patternType.3', count: 5 },
  ];

  // Fabric Filters (from IndiaHandmade reference)
  const FABRICS: { label: string; labelKey: MessageKey; count: number }[] = [
    { label: 'Cotton', labelKey: 'giTagged.fabricOption.1', count: 4 },
    { label: 'Silk', labelKey: 'giTagged.fabricOption.2', count: 7 },
    { label: 'Wool', labelKey: 'giTagged.fabricOption.3', count: 5 },
    { label: 'Khadi', labelKey: 'giTagged.fabricOption.4', count: 11 },
    { label: 'Other', labelKey: 'giTagged.fabricOption.5', count: 1 },
  ];

  // Discount Tiers (from IndiaHandmade reference)
  const DISCOUNT_TIERS: { label: string; labelKey: MessageKey; minPct: number; count: number }[] = [
    { label: '10% and above', labelKey: 'giTagged.discountTier.1', minPct: 10, count: 95 },
    { label: '20% and above', labelKey: 'giTagged.discountTier.2', minPct: 20, count: 51 },
    { label: '30% and above', labelKey: 'giTagged.discountTier.3', minPct: 30, count: 11 },
    { label: '40% and above', labelKey: 'giTagged.discountTier.4', minPct: 40, count: 5 },
  ];

  const filteredProducts = $derived(
    GI_PRODUCTS.filter((p) => {
      if (selectedState !== 'all' && p.state !== selectedState) return false;
      if (selectedCategory !== 'all') {
        const norm = selectedCategory.toLowerCase();
        const matchesCat =
          p.category.toLowerCase() === norm ||
          p.craftName.toLowerCase().includes(norm) ||
          (norm.includes('weaving') && (p.craftName.toLowerCase().includes('brocade') || p.craftName.toLowerCase().includes('pashmina') || p.craftName.toLowerCase().includes('patola') || p.category === 'Women')) ||
          (norm.includes('block') && (p.craftName.toLowerCase().includes('ajrakh') || p.craftName.toLowerCase().includes('block'))) ||
          (norm.includes('pottery') && (p.craftName.toLowerCase().includes('pottery') || p.craftName.toLowerCase().includes('clay'))) ||
          (norm.includes('metal') && (p.craftName.toLowerCase().includes('dhokra') || p.craftName.toLowerCase().includes('bell') || p.craftName.toLowerCase().includes('metal'))) ||
          (norm.includes('wood') && (p.craftName.toLowerCase().includes('wood') || p.craftName.toLowerCase().includes('carving'))) ||
          (norm.includes('paint') && (p.craftName.toLowerCase().includes('aipan') || p.craftName.toLowerCase().includes('painting') || p.category === 'Paintings'));
        if (!matchesCat) return false;
      }
      if (selectedColor && p.colorHex !== selectedColor) return false;
      if (selectedPriceRange) {
        const tier = PRICE_TIERS.find((t) => t.label === selectedPriceRange);
        if (tier && (p.price < tier.min || p.price > tier.max)) return false;
      }
      if (selectedWeavingStyle && p.weavingStyle !== selectedWeavingStyle) return false;
      if (selectedPattern && p.pattern !== selectedPattern) return false;
      if (selectedFabric && p.fabric !== selectedFabric) return false;
      if (selectedDiscount && p.discountPct < selectedDiscount) return false;
      return true;
    }).sort((a, b) => {
      if (sortBy === 'price_asc') return a.price - b.price;
      if (sortBy === 'price_desc') return b.price - a.price;
      if (sortBy === 'discount') return b.discountPct - a.discountPct;
      return t(a.titleKey).localeCompare(t(b.titleKey));
    }),
  );

  function copyProductShare(prod: GIProduct): void {
    const shareUrl = `${window.location.origin}/listing/${prod.id}`;
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(shareUrl).then(() => {
        showToast({
          message: t('giTagged.linkCopiedToast', { title: t(prod.titleKey) }),
          variant: 'success',
        });
      });
    }
  }

  function resetAllFilters(): void {
    selectedState = 'all';
    selectedCategory = 'all';
    selectedPriceRange = null;
    selectedColor = null;
    selectedWeavingStyle = null;
    selectedPattern = null;
    selectedFabric = null;
    selectedDiscount = null;
    sortBy = 'name_asc';
  }
</script>

<svelte:head>
  <title>{t('giTagged.headTitle')}</title>
  <meta
    name="description"
    content={t('giTagged.metaDescription')}
  />
</svelte:head>

<div class="gi-page-container">
  <!-- Top Breadcrumbs -->
  <div class="breadcrumbs-row">
    <Breadcrumbs items={breadcrumbs} />
  </div>

  <!-- Page Header -->
  <header class="gi-page-header">
    <h1 class="gi-page-title">{t('giTagged.pageTitle')}</h1>
    <p class="gi-page-subtitle">
      {t('giTagged.pageSubtitle')}
    </p>
  </header>

  <!-- Layout: Sidebar + Catalog Grid -->
  <div class="gi-layout">
    <!-- LEFT SIDEBAR: SHOPPING OPTIONS -->
    <aside class="gi-sidebar" aria-label={t('giTagged.sidebarAriaLabel')}>
      <div class="sidebar-header">
        <h2 class="sidebar-title">{t('giTagged.shoppingOptions')}</h2>
        {#if selectedCategory !== 'all' || selectedPriceRange || selectedColor || selectedState !== 'all'}
          <Tooltip text={tooltip('tooltip.clearFilters')}>
          {#snippet trigger(tp)}
            <button type="button" class="sidebar-clear-btn" onclick={resetAllFilters} {...tp}>
              {t('giTagged.clearAll')}
            </button>
          {/snippet}
        </Tooltip>
        {/if}
      </div>

      <!-- 1. CATEGORY ACCORDION -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isCategoryOpen = !isCategoryOpen)}
              aria-expanded={isCategoryOpen}
              {...props}
            >
              <span>{t('giTagged.filter.category')}</span>
              <Icon name={isCategoryOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isCategoryOpen}
          <ul class="filter-list" role="list">
            {#each CATEGORY_FILTERS as cat}
              <li>
                <Tooltip text={tooltip('tooltip.selectCategory')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedCategory === cat.label}
                      onclick={() => (selectedCategory = selectedCategory === cat.label ? 'all' : cat.label)}
                      {...tp}
                    >
                      <span class="option-name">{t(cat.labelKey)}</span>
                      <span class="option-count">({cat.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 2. PRICE ACCORDION (IMAGE 4) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isPriceOpen = !isPriceOpen)}
              aria-expanded={isPriceOpen}
              {...props}
            >
              <span>{t('giTagged.filter.price')}</span>
              <Icon name={isPriceOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isPriceOpen}
          <ul class="filter-list" role="list">
            {#each PRICE_TIERS as tier}
              <li>
                <Tooltip text={tooltip('tooltip.selectPrice')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedPriceRange === tier.label}
                      onclick={() => (selectedPriceRange = selectedPriceRange === tier.label ? null : tier.label)}
                      {...tp}
                    >
                      <span class="option-name">{t(tier.labelKey)}</span>
                      <span class="option-count">({tier.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 3. COLOR PALETTE SWATCHES (IMAGE 4) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isColorOpen = !isColorOpen)}
              aria-expanded={isColorOpen}
              {...props}
            >
              <span>{t('giTagged.filter.color')}</span>
              <Icon name={isColorOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isColorOpen}
          <div class="swatches-grid">
            {#each COLOR_SWATCHES as swatch}
              <Tooltip text={tooltip('tooltip.selectColor')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="swatch-circle"
                    class:active={selectedColor === swatch.hex}
                    style="background-color: {swatch.hex};"
                    title={t(swatch.nameKey)}
                    aria-label={t('giTagged.colorFilterAriaLabel', { color: t(swatch.nameKey) })}
                    {...tp}
                    onclick={() => (selectedColor = selectedColor === swatch.hex ? null : swatch.hex)}
                  >
                    {#if selectedColor === swatch.hex}
                      <span class="swatch-check">✓</span>
                    {/if}
                  </button>
                {/snippet}
              </Tooltip>
            {/each}
          </div>
        {/if}
      </div>

      <!-- 4. WEAVING STYLE ACCORDION (IMAGE 1) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isWeavingOpen = !isWeavingOpen)}
              aria-expanded={isWeavingOpen}
              {...props}
            >
              <span>{t('giTagged.filter.weavingStyle')}</span>
              <Icon name={isWeavingOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isWeavingOpen}
          <ul class="filter-list" role="list">
            {#each WEAVING_STYLES as style}
              <li>
                <Tooltip text={tooltip('tooltip.selectWeaving')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedWeavingStyle === style.label}
                      onclick={() => (selectedWeavingStyle = selectedWeavingStyle === style.label ? null : style.label)}
                      {...tp}
                    >
                      <span class="option-name">{t(style.labelKey)}</span>
                      <span class="option-count">({style.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 5. PRINT OR PATTERN TYPE ACCORDION (IMAGE 1) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isPatternOpen = !isPatternOpen)}
              aria-expanded={isPatternOpen}
              {...props}
            >
              <span>{t('giTagged.filter.patternType')}</span>
              <Icon name={isPatternOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isPatternOpen}
          <ul class="filter-list" role="list">
            {#each PATTERN_TYPES as pat}
              <li>
                <Tooltip text={tooltip('tooltip.selectPattern')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedPattern === pat.label}
                      onclick={() => (selectedPattern = selectedPattern === pat.label ? null : pat.label)}
                      {...tp}
                    >
                      <span class="option-name">{t(pat.labelKey)}</span>
                      <span class="option-count">({pat.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 6. FABRIC ACCORDION (IMAGE 2) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isFabricOpen = !isFabricOpen)}
              aria-expanded={isFabricOpen}
              {...props}
            >
              <span>{t('giTagged.filter.fabric')}</span>
              <Icon name={isFabricOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isFabricOpen}
          <ul class="filter-list" role="list">
            {#each FABRICS as fab}
              <li>
                <Tooltip text={tooltip('tooltip.selectFabric')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedFabric === fab.label}
                      onclick={() => (selectedFabric = selectedFabric === fab.label ? null : fab.label)}
                      {...tp}
                    >
                      <span class="option-name">{t(fab.labelKey)}</span>
                      <span class="option-count">({fab.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 7. DISCOUNT ACCORDION (IMAGE 2) -->
      <div class="filter-accordion">
        <Tooltip text={tooltip('tooltip.selectFilter')} fill>
          {#snippet trigger(props)}
            <button
              type="button"
              class="filter-accordion-toggle"
              onclick={() => (isDiscountOpen = !isDiscountOpen)}
              aria-expanded={isDiscountOpen}
              {...props}
            >
              <span>{t('giTagged.filter.discount')}</span>
              <Icon name={isDiscountOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>

        {#if isDiscountOpen}
          <ul class="filter-list" role="list">
            {#each DISCOUNT_TIERS as disc}
              <li>
                <Tooltip text={tooltip('tooltip.selectDiscount')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="filter-option-btn"
                      class:active={selectedDiscount === disc.minPct}
                      onclick={() => (selectedDiscount = selectedDiscount === disc.minPct ? null : disc.minPct)}
                      {...tp}
                    >
                      <span class="option-name">{t(disc.labelKey)}</span>
                      <span class="option-count">({disc.count})</span>
                    </button>
                  {/snippet}
                </Tooltip>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    </aside>

    <!-- RIGHT MAIN: TOOLBAR & PRODUCT GRID -->
    <main class="gi-main-content">
      <!-- Toolbar Bar (Image 3) -->
      <div class="gi-toolbar">
        <div class="toolbar-left">
          <!-- View switcher -->
          <div class="view-switchers" role="group" aria-label={t('giTagged.viewFormatAriaLabel')}>
            <Tooltip text={tooltip('tooltip.gridView')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="view-btn"
                class:active={viewMode === 'grid'}
                onclick={() => (viewMode = 'grid')}
                aria-label={t('giTagged.gridView')}
                {...tp}
              >
                <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor">
                  <rect x="1" y="1" width="6" height="6" rx="1" />
                  <rect x="9" y="1" width="6" height="6" rx="1" />
                  <rect x="1" y="9" width="6" height="6" rx="1" />
                  <rect x="9" y="9" width="6" height="6" rx="1" />
                </svg>
              </button>
            {/snippet}
          </Tooltip>
          <Tooltip text={tooltip('tooltip.listView')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="view-btn"
                class:active={viewMode === 'list'}
                onclick={() => (viewMode = 'list')}
                aria-label={t('giTagged.listView')}
                {...tp}
              >
                <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor">
                  <rect x="1" y="2" width="14" height="2" rx="0.5" />
                  <rect x="1" y="7" width="14" height="2" rx="0.5" />
                  <rect x="1" y="12" width="14" height="2" rx="0.5" />
                </svg>
              </button>
            {/snippet}
          </Tooltip>
          </div>

          <span class="items-counter">
            {t('giTagged.itemsCounter', { shown: String(filteredProducts.length), total: String(GI_PRODUCTS.length) })}
          </span>
        </div>

        <div class="toolbar-right">
          <!-- State Filter Dropdown -->
          <div class="toolbar-select-wrap">
            <select bind:value={selectedState} aria-label={t('giTagged.selectStateAriaLabel')}>
              <option value="all">{t('giTagged.selectStateOption')}</option>
              <option value="Uttarakhand">Uttarakhand</option>
              <option value="Gujarat">Gujarat</option>
              <option value="Uttar Pradesh">Uttar Pradesh</option>
              <option value="Jammu & Kashmir">Jammu & Kashmir</option>
              <option value="Chhattisgarh">Chhattisgarh</option>
              <option value="Rajasthan">Rajasthan</option>
              <option value="Bihar">Bihar</option>
              <option value="Karnataka">Karnataka</option>
            </select>
          </div>

          <!-- Active Filter Pill -->
          <span class="active-filter-pill">
            <span>{t('giTagged.giCraftPill')}</span>
            <Icon name="check" size="0.75rem" />
          </span>

          <!-- Sort By Dropdown -->
          <div class="toolbar-select-wrap">
            <label for="sort-select" class="sort-label">{t('giTagged.sortByLabel')}</label>
            <select id="sort-select" bind:value={sortBy} aria-label={t('giTagged.sortProductsAriaLabel')}>
              <option value="name_asc">{t('giTagged.sort.productName')}</option>
              <option value="price_asc">{t('giTagged.sort.priceLowHigh')}</option>
              <option value="price_desc">{t('giTagged.sort.priceHighLow')}</option>
              <option value="discount">{t('giTagged.sort.highestDiscount')}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Products Grid / List -->
      {#if filteredProducts.length === 0}
        <div class="no-results-panel">
          <Icon name="info" size="2rem" />
          <h3>{t('giTagged.noResults.heading')}</h3>
          <p>{t('giTagged.noResults.body')}</p>
          <Tooltip text={tooltip('tooltip.reset')}>
          {#snippet trigger(tp)}
            <button type="button" class="reset-btn" onclick={resetAllFilters} {...tp}>{t('giTagged.noResults.resetButton')}</button>
          {/snippet}
        </Tooltip>
        </div>
      {:else}
        <div class={viewMode === 'grid' ? 'gi-products-grid' : 'gi-products-list'}>
          {#each filteredProducts as prod (prod.id)}
            <article class="gi-product-card">
              <div class="gi-media-frame">
                <a href={`/listing/${prod.id}`} class="gi-media-link">
                  <img
                    src={prod.image}
                    alt={`${t(prod.titleKey)} - ${t(prod.craftNameKey)}`}
                    loading="lazy"
                    class="gi-product-img"
                  />
                </a>

                <!-- Official India GI Tricolor Map Pin Emblem (Image 3) -->
                <div class="gi-official-pin-emblem" title={t('giTagged.registeredTitle', { regNo: prod.giRegNo })}>
                  <svg viewBox="0 0 32 38" width="28" height="34" fill="none" class="gi-pin-svg">
                    <!-- Pin outer shape -->
                    <path
                      d="M16 0C7.163 0 0 7.163 0 16c0 10.5 16 22 16 22s16-11.5 16-22C32 7.163 24.837 0 16 0z"
                      fill="#ffffff"
                      stroke="#d5cec5"
                      stroke-width="1.5"
                    />
                    <!-- Saffron Stripe -->
                    <path
                      d="M6 10 C6 6 10 5 16 5 C22 5 26 6 26 10 L25 14 L7 14 Z"
                      fill="#FF9933"
                    />
                    <!-- White with Ashoka Chakra dot -->
                    <rect x="7" y="14" width="18" height="6" fill="#ffffff" />
                    <circle cx="16" cy="17" r="2" fill="#000088" />
                    <!-- Green Stripe -->
                    <path
                      d="M7 20 L25 20 L22 25 C19 28 13 28 10 25 Z"
                      fill="#138808"
                    />
                  </svg>
                  <span class="gi-tag-num">{prod.giRegNo}</span>
                </div>

                <!-- Quick Share Button -->
                <button
                  type="button"
                  class="gi-quick-share-btn"
                  onclick={() => copyProductShare(prod)}
                  title={t('giTagged.copyLinkTitle')}
                  aria-label={t('giTagged.shareAriaLabel')}
                >
                  <Icon name="share" size="0.85rem" />
                </button>
              </div>

              <!-- Body Info -->
              <div class="gi-card-body">
                <div class="gi-card-meta">
                  <span class="craft-chip">{t(prod.craftNameKey)}</span>
                  <span class="state-chip">{prod.state}</span>
                </div>

                <h3 class="gi-product-title">
                  <a href={`/listing/${prod.id}`}>{t(prod.titleKey)}</a>
                </h3>

                <p class="artisan-byline">
                  <span>{t('giTagged.byMasterArtisan')}</span> <strong>{t(prod.artisanNameKey)}</strong>
                </p>

                <!-- Pricing with strike-through MRP and discount (Image 3) -->
                <div class="gi-price-row">
                  <span class="gi-current-price">
                    <Money paise={prod.price} />
                  </span>
                  <span class="gi-mrp-price">
                    <Money paise={prod.mrp} />
                  </span>
                  <span class="gi-discount-tag">
                    {t('listingCard.discountPercent', { percent: String(prod.discountPct) })}
                  </span>
                </div>

                <div class="gi-card-actions">
                  <a href={`/listing/${prod.id}`} class="gi-acquire-btn">
                    <span>{t('giTagged.directOrder')}</span>
                    <Icon name="arrow-right" size="0.85rem" />
                  </a>
                  <Tooltip text={tooltip('tooltip.wishlist')}>
                  {#snippet trigger(tp)}
                    <button
                      type="button"
                      class="gi-wishlist-btn"
                      class:is-wishlisted={wishlist.has(prod.id)}
                      onclick={(e) => { e.preventDefault(); wishlist.toggle(prod.id, t(prod.titleKey)); }}
                      title={wishlist.has(prod.id) ? t('listingCard.removeFromWishlist') : t('listingCard.addToWishlist')}
                      aria-label={t('listingCard.wishlistAriaLabel')}
                      {...tp}
                    >
                      <svg viewBox="0 0 24 24" width="16" height="16" fill={wishlist.has(prod.id) ? '#e11d48' : 'none'} stroke={wishlist.has(prod.id) ? '#e11d48' : 'currentColor'} stroke-width="2">
                        <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
                      </svg>
                      <span>{wishlist.has(prod.id) ? t('listingCard.saved') : t('listingCard.wishlist')}</span>
                    </button>
                  {/snippet}
                </Tooltip>
                </div>
              </div>
            </article>
          {/each}
        </div>
      {/if}
    </main>
  </div>
</div>

<style>
  .gi-page-container {
    max-inline-size: 80rem;
    margin-inline: auto;
    padding: 1rem 1rem 3rem 1rem;
    font-family: inherit;
  }

  .breadcrumbs-row {
    margin-block-end: 1rem;
  }

  .gi-page-header {
    border-block-end: 1px solid var(--k-stone-300, var(--k-border-hairline));
    padding-block-end: 1rem;
    margin-block-end: 1.5rem;
  }

  .gi-page-title {
    font-family: var(--k-font-display, serif);
    font-size: 2rem;
    font-weight: 700;
    color: var(--k-accent-danger-strong); /* Official heritage burgundy */
    margin: 0 0 0.35rem 0;
  }

  .gi-page-subtitle {
    font-size: 0.85rem;
    color: var(--k-text-secondary, var(--k-text-tertiary));
    margin: 0;
  }

  /* Two Column Layout */
  .gi-layout {
    display: grid;
    grid-template-columns: 18rem 1fr;
    gap: 2rem;
    align-items: start;
  }

  @media (max-width: 64rem) {
    .gi-layout {
      grid-template-columns: 1fr;
    }
  }

  /* SIDEBAR: SHOPPING OPTIONS */
  .gi-sidebar {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: 4px;
    padding: 1.25rem;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-block-end: 1.5px solid var(--k-madder-800);
    padding-block-end: 0.75rem;
    margin-block-end: 1rem;
  }

  .sidebar-title {
    font-size: 0.95rem;
    font-weight: 700;
    color: var(--k-accent-danger-strong);
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .sidebar-clear-btn {
    background: none;
    border: none;
    color: var(--k-accent-primary-text);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    text-decoration: underline;
  }

  .filter-accordion {
    border-block-end: 1px solid var(--k-stone-200, var(--k-border-subtle));
    padding-block: 0.85rem;
  }

  .filter-accordion:last-child {
    border-block-end: none;
  }

  .filter-accordion-toggle {
    inline-size: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: none;
    border: none;
    padding: 0;
    font-size: 0.82rem;
    font-weight: 700;
    color: var(--k-text-primary);
    cursor: pointer;
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .filter-list {
    list-style: none;
    margin: 0.75rem 0 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .filter-option-btn {
    inline-size: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: none;
    border: none;
    padding: 0.25rem 0.35rem;
    font-size: 0.82rem;
    color: var(--k-text-secondary);
    cursor: pointer;
    text-align: start;
    border-radius: 3px;
    transition: all 0.15s ease;
  }

  .filter-option-btn:hover {
    background-color: var(--k-surface-base);
    color: var(--k-accent-danger-strong);
  }

  .filter-option-btn.active {
    background-color: var(--k-surface-raised);
    color: var(--k-accent-danger-strong);
    font-weight: 700;
  }

  .option-count {
    font-size: 0.75rem;
    color: var(--k-text-secondary, var(--k-text-tertiary));
  }

  /* 12 Color Swatches Grid (Image 4) */
  .swatches-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    margin-block-start: 0.85rem;
  }

  .swatch-circle {
    inline-size: 2.2rem;
    block-size: 2.2rem;
    border-radius: 50%;
    border: 1.5px solid var(--k-border-hairline);
    cursor: pointer;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }

  .swatch-circle:hover {
    transform: scale(1.1);
  }

  .swatch-circle.active {
    outline: 2px solid var(--k-madder-800);
    outline-offset: 2px;
  }

  .swatch-check {
    color: var(--k-text-on-accent);
    font-weight: 900;
    font-size: 0.85rem;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.6);
  }

  /* MAIN CONTENT: TOOLBAR */
  .gi-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: 4px;
    padding: 0.65rem 1rem;
    margin-block-end: 1.5rem;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .view-switchers {
    display: flex;
    border: 1px solid var(--k-border-hairline);
    border-radius: 3px;
    overflow: hidden;
  }

  .view-btn {
    background-color: var(--k-surface-base);
    border: none;
    padding: 0.4rem 0.5rem;
    color: var(--k-text-tertiary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .view-btn.active {
    background-color: var(--k-madder-800);
    color: var(--k-text-on-accent);
  }

  .items-counter {
    font-size: 0.82rem;
    color: var(--k-text-secondary);
    font-weight: 500;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .toolbar-select-wrap {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .toolbar-select-wrap select {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    border-radius: 3px;
    padding: 0.4rem 0.65rem;
    font-size: 0.82rem;
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .sort-label {
    font-size: 0.82rem;
    color: var(--k-text-tertiary);
  }

  .active-filter-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.35rem 0.75rem;
    border-radius: 4px;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-neem-600);
    color: var(--k-accent-success-muted);
    font-size: 0.75rem;
    font-weight: 600;
  }

  /* PRODUCTS GRID (3x3 matching Image 3) */
  .gi-products-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 54rem) {
    .gi-products-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 36rem) {
    .gi-products-grid {
      grid-template-columns: 1fr;
    }
  }

  /* LIST VIEW */
  .gi-products-list {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .gi-products-list .gi-product-card {
    display: grid;
    grid-template-columns: 14rem 1fr;
    align-items: stretch;
  }

  /* Stack the media frame above the copy below 720px so the info panel
     does not get squeezed into a sliver on a phone. */
  @media (max-width: 45rem) {
    .gi-products-list .gi-product-card {
      grid-template-columns: 1fr;
    }
  }

  /* CARD STYLING */
  .gi-product-card {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 6px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .gi-product-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    border-color: var(--k-border-accent);
  }

  .gi-media-frame {
    position: relative;
    aspect-ratio: 1;
    background-color: var(--k-surface-raised);
    overflow: hidden;
  }

  .gi-media-link {
    display: block;
    inline-size: 100%;
    block-size: 100%;
  }

  .gi-product-img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  /* Official GI Tricolor Pin Emblem */
  .gi-official-pin-emblem {
    position: absolute;
    inset-block-start: 0.65rem;
    inset-inline-start: 0.65rem;
    display: flex;
    align-items: center;
    gap: 0.35rem;
    background-color: rgba(255, 255, 255, 0.95);
    border-radius: 999px;
    padding: 0.2rem 0.5rem 0.2rem 0.25rem;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
    border: 1px solid var(--k-border-subtle);
  }

  .gi-tag-num {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--k-accent-danger-strong);
  }

  .gi-quick-share-btn {
    position: absolute;
    inset-block-start: 0.65rem;
    inset-inline-end: 0.65rem;
    inline-size: 2rem;
    block-size: 2rem;
    border-radius: 50%;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    color: var(--k-text-tertiary);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    transition: all 0.15s ease;
  }

  .gi-quick-share-btn:hover {
    color: var(--k-accent-danger-strong);
    transform: scale(1.1);
  }

  /* Card Body */
  .gi-card-body {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    flex: 1;
    justify-content: space-between;
  }

  .gi-card-meta {
    display: flex;
    gap: 0.35rem;
    margin-block-end: 0.4rem;
  }

  .craft-chip {
    font-size: 0.68rem;
    text-transform: uppercase;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--k-accent-danger-strong);
  }

  .state-chip {
    font-size: 0.68rem;
    color: var(--k-text-tertiary);
  }

  .gi-product-title {
    font-size: 0.92rem;
    font-weight: 600;
    margin: 0 0 0.35rem 0;
    line-height: 1.35;
  }

  .gi-product-title a {
    color: var(--k-text-primary);
    text-decoration: none;
  }

  .gi-product-title a:hover {
    color: var(--k-accent-danger-strong);
    text-decoration: underline;
  }

  .artisan-byline {
    font-size: 0.78rem;
    color: var(--k-text-tertiary);
    margin: 0 0 0.75rem 0;
  }

  /* Pricing with MRP Strikethrough (Image 3) */
  .gi-price-row {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    margin-block-end: 0.85rem;
  }

  .gi-current-price {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .gi-mrp-price {
    font-size: 0.85rem;
    color: var(--k-stone-400);
    text-decoration: line-through;
  }

  .gi-discount-tag {
    font-size: 0.78rem;
    font-weight: 700;
    color: var(--k-accent-success-muted); /* Fair trade saving green */
  }

  .gi-card-actions {
    margin-block-start: auto;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .gi-acquire-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.4rem;
    padding: 0.55rem 0.75rem;
    border-radius: 4px;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-border-hairline);
    color: var(--k-accent-danger-strong);
    text-decoration: none;
    font-size: 0.82rem;
    font-weight: 600;
    transition: all 0.15s ease;
  }

  .gi-acquire-btn:hover {
    background-color: var(--k-madder-800);
    color: var(--k-text-on-accent);
    border-color: var(--k-madder-800);
  }

  .gi-wishlist-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    padding: 0.55rem 0.75rem;
    border-radius: 4px;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    color: var(--k-stone-600);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
    white-space: nowrap;
  }

  .gi-wishlist-btn:hover {
    border-color: var(--k-border-danger);
    color: var(--k-accent-danger-muted);
    background-color: var(--k-surface-neutral);
  }

  .gi-wishlist-btn.is-wishlisted {
    border-color: var(--k-border-danger);
    color: var(--k-accent-danger-muted);
    background-color: var(--k-surface-neutral);
  }

  .no-results-panel {
    text-align: center;
    padding: 3rem 1.5rem;
    background-color: var(--k-surface-base);
    border: 1px dashed var(--k-border-hairline);
    border-radius: 6px;
    color: var(--k-text-tertiary);
  }

  .no-results-panel h3 {
    color: var(--k-text-primary);
    margin: 0.5rem 0;
  }

  .reset-btn {
    margin-block-start: 1rem;
    padding: 0.5rem 1.25rem;
    background-color: var(--k-madder-800);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 600;
  }
</style>
