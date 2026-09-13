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
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs, type BreadcrumbItem, Money, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { wishlist } from '$lib/wishlist.svelte';

  const t = $derived(locale.t);

  const breadcrumbs = $derived<BreadcrumbItem[]>([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'GI Tagged' },
  ]);

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

  interface GIProduct {
    id: string;
    title: string;
    craftName: string;
    giRegNo: string;
    state: string;
    category: string;
    price: number;
    mrp: number;
    discountPct: number;
    image: string;
    colorHex: string;
    artisanName: string;
    weavingStyle?: string;
    pattern?: string;
    fabric?: string;
  }

  const GI_PRODUCTS: GIProduct[] = [
    {
      id: 'gi-1',
      title: 'Aipanart Decoration Sri Goljyu Mahraz Painting 1212',
      craftName: 'Uttarakhand Aipan Art',
      giRegNo: 'GI-696',
      state: 'Uttarakhand',
      category: 'Paintings',
      price: 350000,
      mrp: 450000,
      discountPct: 22,
      image: '/craft-images/paintings/category_cover.jpg',
      colorHex: '#8c2323',
      artisanName: 'Bhawana Bhatt',
    },
    {
      id: 'gi-2',
      title: 'Aipanart Decorations Sri Krishna Balgopal 1212',
      craftName: 'Uttarakhand Aipan Art',
      giRegNo: 'GI-696',
      state: 'Uttarakhand',
      category: 'Paintings',
      price: 250000,
      mrp: 350000,
      discountPct: 29,
      image: '/craft-images/paintings/category_cover.jpg',
      colorHex: '#8c2323',
      artisanName: 'Kamla Devi',
    },
    {
      id: 'gi-3',
      title: 'Handcrafted Aipan Art Laxmi Aipan Gmr2424',
      craftName: 'Uttarakhand Aipan Art',
      giRegNo: 'GI-696',
      state: 'Uttarakhand',
      category: 'Paintings',
      price: 600000,
      mrp: 700000,
      discountPct: 14,
      image: '/craft-images/paintings/category_cover.jpg',
      colorHex: '#c68237',
      artisanName: 'Deepa Joshi',
    },
    {
      id: 'gi-4',
      title: 'Handcrafted Aipan Art Ganesh Aipan Gmr2424',
      craftName: 'Uttarakhand Aipan Art',
      giRegNo: 'GI-696',
      state: 'Uttarakhand',
      category: 'Paintings',
      price: 600000,
      mrp: 700000,
      discountPct: 14,
      image: '/craft-images/paintings/category_cover.jpg',
      colorHex: '#c68237',
      artisanName: 'Deepa Joshi',
    },
    {
      id: 'gi-5',
      title: 'Kutch Botanical Indigo Ajrakh Bedcover Set',
      craftName: 'Ajrakh Block Print',
      giRegNo: 'GI-384',
      state: 'Gujarat',
      category: 'Home and living',
      price: 420000,
      mrp: 550000,
      discountPct: 24,
      image: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
      colorHex: '#0033cc',
      artisanName: 'Ismail Khatri',
    },
    {
      id: 'gi-6',
      title: 'Varanasi Pure Kadwa Zari Brocade Silk Saree',
      craftName: 'Banarasi Brocade',
      giRegNo: 'GI-99',
      state: 'Uttar Pradesh',
      category: 'Women',
      price: 1850000,
      mrp: 2400000,
      discountPct: 23,
      image: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      colorHex: '#e60000',
      artisanName: 'Sita Sharma',
    },
    {
      id: 'gi-7',
      title: 'Kashmir Hand-Spun Sozni Pashmina Shawl',
      craftName: 'Kashmir Pashmina',
      giRegNo: 'GI-46',
      state: 'Jammu & Kashmir',
      category: 'Women',
      price: 1450000,
      mrp: 1800000,
      discountPct: 19,
      image: '/craft-images/embroidery/kashmir_pashmina_sozni_01.jpeg',
      colorHex: '#fdfcf0',
      artisanName: 'Ghulam Hassan',
    },
    {
      id: 'gi-8',
      title: 'Bastar Bell Metal Traditional Bull & Deity',
      craftName: 'Bastar Dhokra',
      giRegNo: 'GI-83',
      state: 'Chhattisgarh',
      category: 'Home and living',
      price: 480000,
      mrp: 600000,
      discountPct: 20,
      image: '/craft-images/metalwork/dhokra-casting.jpg',
      colorHex: '#f4c430',
      artisanName: 'Manglu Ghadwa',
    },
    {
      id: 'gi-9',
      title: 'Jaipur Hand-Turned Quartz Blue Pottery Vase',
      craftName: 'Blue Pottery',
      giRegNo: 'GI-180',
      state: 'Rajasthan',
      category: 'Home and living',
      price: 180000,
      mrp: 240000,
      discountPct: 25,
      image: '/craft-images/pottery/nizamabad-black-pottery.jpg',
      colorHex: '#007a78',
      artisanName: 'Kripal Kumbhar',
    },
    {
      id: 'gi-10',
      title: 'Madhubani Kohbar Mithila Fine Line Painting',
      craftName: 'Madhubani Painting',
      giRegNo: 'GI-105',
      state: 'Bihar',
      category: 'Paintings',
      price: 320000,
      mrp: 400000,
      discountPct: 20,
      image: '/craft-images/paintings/madhubani-painting.jpg',
      colorHex: '#ffd200',
      artisanName: 'Lakshmi Devi',
    },
    {
      id: 'gi-12',
      title: 'Patan Double-Ikat Silk Heritage Dupatta',
      craftName: 'Patan Patola',
      giRegNo: 'GI-232',
      state: 'Gujarat',
      category: 'Women',
      price: 2600000,
      mrp: 3200000,
      discountPct: 19,
      image: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      colorHex: '#730000',
      artisanName: 'Rohit Salvi',
    },
  ];

  // Colors from Image 4
  const COLOR_SWATCHES = [
    { name: 'Royal Blue', hex: '#0033cc' },
    { name: 'Raw Ivory', hex: '#fdfcf0' },
    { name: 'Kutch Ochre', hex: '#c68237' },
    { name: 'Madder Maroon', hex: '#8c2323' },
    { name: 'Pale Haldi', hex: '#fff9d2' },
    { name: 'Bright Haldi', hex: '#ffd200' },
    { name: 'Deep Crimson', hex: '#730000' },
    { name: 'Brass Gold', hex: '#f4c430' },
    { name: 'Marigold Orange', hex: '#ff9900' },
    { name: 'Gulabi Pink', hex: '#f8b9cb' },
    { name: 'Sindoor Red', hex: '#e60000' },
    { name: 'Forest Teal', hex: '#007a78' },
  ];

  // Price Tiers from Image 4
  const PRICE_TIERS = [
    { label: '₹0 - ₹999', min: 0, max: 99999, count: 20 },
    { label: '₹1,000 - ₹1,999', min: 100000, max: 199999, count: 9 },
    { label: '₹2,000 - ₹2,999', min: 200000, max: 299999, count: 9 },
    { label: '₹3,000 - ₹3,999', min: 300000, max: 399999, count: 7 },
    { label: '₹4,000 - ₹4,999', min: 400000, max: 499999, count: 10 },
    { label: '₹5,000 - ₹5,999', min: 500000, max: 599999, count: 13 },
    { label: '₹6,000 - ₹6,999', min: 600000, max: 699999, count: 5 },
    { label: '₹7,000 - ₹7,999', min: 700000, max: 799999, count: 4 },
    { label: '₹9,000 - ₹9,999', min: 900000, max: 999999, count: 2 },
    { label: '₹10,000 and above', min: 1000000, max: 999999999, count: 27 },
  ];

  const CATEGORY_FILTERS = [
    { label: 'Weaving', count: 38 },
    { label: 'Block printing', count: 14 },
    { label: 'Pottery', count: 11 },
    { label: 'Metalwork', count: 19 },
    { label: 'Woodwork', count: 16 },
    { label: 'Embroidery', count: 21 },
    { label: 'Painting', count: 61 },
    { label: 'Basketry', count: 8 },
    { label: 'Jewellery', count: 12 },
    { label: 'Leatherwork', count: 7 },
    { label: 'Stone carving', count: 9 },
    { label: 'Bamboo craft', count: 15 },
    { label: 'Home and living', count: 22 },
    { label: 'Furniture', count: 6 },
    { label: 'Discounted Products', count: 5 },
  ];

  // Weaving Style Filters (from IndiaHandmade reference)
  const WEAVING_STYLES = [
    { label: 'Handloom with extra-weft thread/zari work', count: 6 },
    { label: 'Kanjeevaram / Kanchipuram', count: 1 },
  ];

  // Print or Pattern Type (from IndiaHandmade reference)
  const PATTERN_TYPES = [
    { label: 'Checkered', count: 1 },
    { label: 'Ethnic Motif', count: 1 },
    { label: 'Others', count: 5 },
  ];

  // Fabric Filters (from IndiaHandmade reference)
  const FABRICS = [
    { label: 'Cotton', count: 4 },
    { label: 'Silk', count: 7 },
    { label: 'Wool', count: 5 },
    { label: 'Khadi', count: 11 },
    { label: 'Other', count: 1 },
  ];

  // Discount Tiers (from IndiaHandmade reference)
  const DISCOUNT_TIERS = [
    { label: '10% and above', minPct: 10, count: 95 },
    { label: '20% and above', minPct: 20, count: 51 },
    { label: '30% and above', minPct: 30, count: 11 },
    { label: '40% and above', minPct: 40, count: 5 },
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
      return a.title.localeCompare(b.title);
    }),
  );

  function copyProductShare(prod: GIProduct): void {
    const shareUrl = `${window.location.origin}/listing/${prod.id}`;
    if (navigator?.clipboard) {
      navigator.clipboard.writeText(shareUrl).then(() => {
        showToast({
          message: `Direct link for "${prod.title}" copied!`,
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
  <title>GI Tagged Product Directory — Ministry of Social Justice & Empowerment</title>
  <meta
    name="description"
    content="Official National Geographical Indications catalog for authentic certified Indian crafts directly from master artisan looms."
  />
</svelte:head>

<div class="gi-page-container">
  <!-- Top Breadcrumbs -->
  <div class="breadcrumbs-row">
    <Breadcrumbs items={breadcrumbs} />
  </div>

  <!-- Page Header -->
  <header class="gi-page-header">
    <h1 class="gi-page-title">GI Tagged Product</h1>
    <p class="gi-page-subtitle">
      Government Registered Geographical Indications • Direct Artisan Cooperative Procurement
    </p>
  </header>

  <!-- Layout: Sidebar + Catalog Grid -->
  <div class="gi-layout">
    <!-- LEFT SIDEBAR: SHOPPING OPTIONS -->
    <aside class="gi-sidebar" aria-label="Shopping Options">
      <div class="sidebar-header">
        <h2 class="sidebar-title">Shopping Options</h2>
        {#if selectedCategory !== 'all' || selectedPriceRange || selectedColor || selectedState !== 'all'}
          <button type="button" class="sidebar-clear-btn" onclick={resetAllFilters}>
            Clear All
          </button>
        {/if}
      </div>

      <!-- 1. CATEGORY ACCORDION -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isCategoryOpen = !isCategoryOpen)}
          aria-expanded={isCategoryOpen}
        >
          <span>CATEGORY</span>
          <Icon name={isCategoryOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isCategoryOpen}
          <ul class="filter-list" role="list">
            {#each CATEGORY_FILTERS as cat}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedCategory === cat.label}
                  onclick={() => (selectedCategory = selectedCategory === cat.label ? 'all' : cat.label)}
                >
                  <span class="option-name">{cat.label}</span>
                  <span class="option-count">({cat.count})</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 2. PRICE ACCORDION (IMAGE 4) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isPriceOpen = !isPriceOpen)}
          aria-expanded={isPriceOpen}
        >
          <span>PRICE</span>
          <Icon name={isPriceOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isPriceOpen}
          <ul class="filter-list" role="list">
            {#each PRICE_TIERS as tier}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedPriceRange === tier.label}
                  onclick={() => (selectedPriceRange = selectedPriceRange === tier.label ? null : tier.label)}
                >
                  <span class="option-name">{tier.label}</span>
                  <span class="option-count">({tier.count})</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 3. COLOR PALETTE SWATCHES (IMAGE 4) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isColorOpen = !isColorOpen)}
          aria-expanded={isColorOpen}
        >
          <span>COLOR</span>
          <Icon name={isColorOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isColorOpen}
          <div class="swatches-grid">
            {#each COLOR_SWATCHES as swatch}
              <button
                type="button"
                class="swatch-circle"
                class:active={selectedColor === swatch.hex}
                style="background-color: {swatch.hex};"
                title={swatch.name}
                aria-label={`Filter by ${swatch.name}`}
                onclick={() => (selectedColor = selectedColor === swatch.hex ? null : swatch.hex)}
              >
                {#if selectedColor === swatch.hex}
                  <span class="swatch-check">✓</span>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- 4. WEAVING STYLE ACCORDION (IMAGE 1) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isWeavingOpen = !isWeavingOpen)}
          aria-expanded={isWeavingOpen}
        >
          <span>WEAVING STYLE</span>
          <Icon name={isWeavingOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isWeavingOpen}
          <ul class="filter-list" role="list">
            {#each WEAVING_STYLES as style}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedWeavingStyle === style.label}
                  onclick={() => (selectedWeavingStyle = selectedWeavingStyle === style.label ? null : style.label)}
                >
                  <span class="option-name">{style.label}</span>
                  <span class="option-count">({style.count})</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 5. PRINT OR PATTERN TYPE ACCORDION (IMAGE 1) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isPatternOpen = !isPatternOpen)}
          aria-expanded={isPatternOpen}
        >
          <span>PRINT OR PATTERN TYPE</span>
          <Icon name={isPatternOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isPatternOpen}
          <ul class="filter-list" role="list">
            {#each PATTERN_TYPES as pat}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedPattern === pat.label}
                  onclick={() => (selectedPattern = selectedPattern === pat.label ? null : pat.label)}
                >
                  <span class="option-name">{pat.label}</span>
                  <span class="option-count">({pat.count})</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 6. FABRIC ACCORDION (IMAGE 2) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isFabricOpen = !isFabricOpen)}
          aria-expanded={isFabricOpen}
        >
          <span>FABRIC</span>
          <Icon name={isFabricOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isFabricOpen}
          <ul class="filter-list" role="list">
            {#each FABRICS as fab}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedFabric === fab.label}
                  onclick={() => (selectedFabric = selectedFabric === fab.label ? null : fab.label)}
                >
                  <span class="option-name">{fab.label}</span>
                  <span class="option-count">({fab.count})</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- 7. DISCOUNT ACCORDION (IMAGE 2) -->
      <div class="filter-accordion">
        <button
          type="button"
          class="filter-accordion-toggle"
          onclick={() => (isDiscountOpen = !isDiscountOpen)}
          aria-expanded={isDiscountOpen}
        >
          <span>DISCOUNT</span>
          <Icon name={isDiscountOpen ? 'chevron-up' : 'chevron-down'} size="0.85rem" />
        </button>

        {#if isDiscountOpen}
          <ul class="filter-list" role="list">
            {#each DISCOUNT_TIERS as disc}
              <li>
                <button
                  type="button"
                  class="filter-option-btn"
                  class:active={selectedDiscount === disc.minPct}
                  onclick={() => (selectedDiscount = selectedDiscount === disc.minPct ? null : disc.minPct)}
                >
                  <span class="option-name">{disc.label}</span>
                  <span class="option-count">({disc.count})</span>
                </button>
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
          <div class="view-switchers" role="group" aria-label="View format">
            <button
              type="button"
              class="view-btn"
              class:active={viewMode === 'grid'}
              onclick={() => (viewMode = 'grid')}
              aria-label="Grid view"
            >
              <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor">
                <rect x="1" y="1" width="6" height="6" rx="1" />
                <rect x="9" y="1" width="6" height="6" rx="1" />
                <rect x="1" y="9" width="6" height="6" rx="1" />
                <rect x="9" y="9" width="6" height="6" rx="1" />
              </svg>
            </button>
            <button
              type="button"
              class="view-btn"
              class:active={viewMode === 'list'}
              onclick={() => (viewMode = 'list')}
              aria-label="List view"
            >
              <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor">
                <rect x="1" y="2" width="14" height="2" rx="0.5" />
                <rect x="1" y="7" width="14" height="2" rx="0.5" />
                <rect x="1" y="12" width="14" height="2" rx="0.5" />
              </svg>
            </button>
          </div>

          <span class="items-counter">
            Items 1-{filteredProducts.length} of {GI_PRODUCTS.length}
          </span>
        </div>

        <div class="toolbar-right">
          <!-- State Filter Dropdown -->
          <div class="toolbar-select-wrap">
            <select bind:value={selectedState} aria-label="Select State">
              <option value="all">Select State</option>
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
            <span>GI Craft</span>
            <Icon name="check" size="0.75rem" />
          </span>

          <!-- Sort By Dropdown -->
          <div class="toolbar-select-wrap">
            <label for="sort-select" class="sort-label">Sort By</label>
            <select id="sort-select" bind:value={sortBy} aria-label="Sort products">
              <option value="name_asc">Product Name</option>
              <option value="price_asc">Price: Low to High</option>
              <option value="price_desc">Price: High to Low</option>
              <option value="discount">Highest Discount %</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Products Grid / List -->
      {#if filteredProducts.length === 0}
        <div class="no-results-panel">
          <Icon name="info" size="2rem" />
          <h3>No GI Products Match Selected Filters</h3>
          <p>Try clearing your price, color, or state filter to view more registered crafts.</p>
          <button type="button" class="reset-btn" onclick={resetAllFilters}>Reset Filters</button>
        </div>
      {:else}
        <div class={viewMode === 'grid' ? 'gi-products-grid' : 'gi-products-list'}>
          {#each filteredProducts as prod (prod.id)}
            <article class="gi-product-card">
              <div class="gi-media-frame">
                <a href={`/listing/${prod.id}`} class="gi-media-link">
                  <img
                    src={prod.image}
                    alt={`${prod.title} - ${prod.craftName}`}
                    loading="lazy"
                    class="gi-product-img"
                  />
                </a>

                <!-- Official India GI Tricolor Map Pin Emblem (Image 3) -->
                <div class="gi-official-pin-emblem" title={`Registered ${prod.giRegNo}`}>
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
                  title="Copy artisan piece link"
                  aria-label="Share listing"
                >
                  <Icon name="share" size="0.85rem" />
                </button>
              </div>

              <!-- Body Info -->
              <div class="gi-card-body">
                <div class="gi-card-meta">
                  <span class="craft-chip">{prod.craftName}</span>
                  <span class="state-chip">{prod.state}</span>
                </div>

                <h3 class="gi-product-title">
                  <a href={`/listing/${prod.id}`}>{prod.title}</a>
                </h3>

                <p class="artisan-byline">
                  <span>By master artisan</span> <strong>{prod.artisanName}</strong>
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
                    ({prod.discountPct}% OFF)
                  </span>
                </div>

                <div class="gi-card-actions">
                  <a href={`/listing/${prod.id}`} class="gi-acquire-btn">
                    <span>Direct Order</span>
                    <Icon name="arrow-right" size="0.85rem" />
                  </a>
                  <button
                    type="button"
                    class="gi-wishlist-btn"
                    class:is-wishlisted={wishlist.has(prod.id)}
                    onclick={(e) => { e.preventDefault(); wishlist.toggle(prod.id, prod.title); }}
                    title={wishlist.has(prod.id) ? 'Remove from Wishlist' : 'Add to Wishlist'}
                    aria-label="Wishlist"
                  >
                    <svg viewBox="0 0 24 24" width="16" height="16" fill={wishlist.has(prod.id) ? '#e11d48' : 'none'} stroke={wishlist.has(prod.id) ? '#e11d48' : 'currentColor'} stroke-width="2">
                      <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
                    </svg>
                    <span>{wishlist.has(prod.id) ? 'Saved' : 'Wishlist'}</span>
                  </button>
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
    color: var(--k-madder-800); /* Official heritage burgundy */
    margin: 0 0 0.35rem 0;
  }

  .gi-page-subtitle {
    font-size: 0.85rem;
    color: var(--k-text-secondary, var(--k-border-interactive));
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
    color: var(--k-madder-800);
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
    border-block-end: 1px solid var(--k-stone-200, var(--k-surface-sunken));
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
    color: var(--k-madder-800);
  }

  .filter-option-btn.active {
    background-color: var(--k-surface-raised);
    color: var(--k-madder-800);
    font-weight: 700;
  }

  .option-count {
    font-size: 0.75rem;
    color: var(--k-text-secondary, var(--k-border-interactive));
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
    color: var(--k-border-interactive);
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
    color: var(--k-border-interactive);
  }

  .active-filter-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.35rem 0.75rem;
    border-radius: 4px;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-neem-600);
    color: var(--k-neem-600);
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
    border: 1px solid var(--k-surface-sunken);
    border-radius: 6px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .gi-product-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    border-color: var(--k-accent-primary-bg);
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
    border: 1px solid var(--k-surface-sunken);
  }

  .gi-tag-num {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--k-madder-800);
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
    color: var(--k-border-interactive);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    transition: all 0.15s ease;
  }

  .gi-quick-share-btn:hover {
    color: var(--k-madder-800);
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
    color: var(--k-madder-800);
  }

  .state-chip {
    font-size: 0.68rem;
    color: var(--k-border-interactive);
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
    color: var(--k-madder-800);
    text-decoration: underline;
  }

  .artisan-byline {
    font-size: 0.78rem;
    color: var(--k-border-interactive);
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
    color: var(--k-neem-600); /* Fair trade saving green */
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
    color: var(--k-madder-800);
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
    border-color: var(--k-madder-600);
    color: var(--k-madder-600);
    background-color: var(--k-stone-100);
  }

  .gi-wishlist-btn.is-wishlisted {
    border-color: var(--k-madder-600);
    color: var(--k-madder-600);
    background-color: var(--k-stone-100);
  }

  .no-results-panel {
    text-align: center;
    padding: 3rem 1.5rem;
    background-color: var(--k-surface-base);
    border: 1px dashed var(--k-border-hairline);
    border-radius: 6px;
    color: var(--k-border-interactive);
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
