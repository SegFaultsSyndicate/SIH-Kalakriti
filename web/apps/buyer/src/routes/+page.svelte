<!--
  apps/buyer/src/routes/+page.svelte

  The Editorial Marketplace Home for Kalakriti.
  Strictly adheres to the Kalakriti Design Law:
  - Separation by line and space, not shadow (hairline rules, generous whitespace).
  - Editorial section pattern from IndiaHandmade (lowercase kicker, large heading, 'View all →' link).
  - Product-forward: authentic craft photography is the hero.
  - Government-portal trust cues (GIGW 3.0): clear telemetry, plain language, visible accessibility.
  - Texture over gloss: khadi/paper surfaces, structural ornament, zero neon gradients or glassmorphism.
  - Asymmetry over uniform card grids.
  - Craft names, GI tags, and artisan status are first-class content.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { locale } from '@kalakriti/i18n';
  import { SectionHeader } from '@kalakriti/ui';
  import { Section } from '@kalakriti/patterns';
  import { Divider } from '@kalakriti/ornament';
  import { Icon } from '@kalakriti/icons';
  import {
    listListings,
    listCrafts,
    search,
    getListingSummary,
    getArtisanStorefront,
    type components,
  } from '@kalakriti/api';
  import ListingCard from '$lib/ListingCard.svelte';

  // Modular cultural marketplace components
  import VoicesReelCarousel from '$lib/VoicesReelCarousel.svelte';
  import RegionalBeltNavigator from '$lib/RegionalBeltNavigator.svelte';
  import InstitutionalProcurementBanner from '$lib/InstitutionalProcurementBanner.svelte';
  import ProvenanceTerminal from '$lib/ProvenanceTerminal.svelte';
  import AtelierCommissionCard from '$lib/AtelierCommissionCard.svelte';
  import ArtisanConciergeModal from '$lib/ArtisanConciergeModal.svelte';
  import FaqAccordion from '$lib/FaqAccordion.svelte';
  import SellerShowcaseBanner from '$lib/SellerShowcaseBanner.svelte';
  import ArtisanCraftGrid from '$lib/ArtisanCraftGrid.svelte';

  type ListingSummary = components['schemas']['ListingSummary'];
  type Craft = components['schemas']['Craft'];
  type ArtisanStorefront = components['schemas']['ArtisanStorefront'];

  const t = $derived(locale.t);

  let crafts = $state<Craft[]>([]);
  let giListings = $state<ListingSummary[]>([]);
  let newArrivals = $state<ListingSummary[]>([]);
  let madeToOrder = $state<ListingSummary[]>([]);
  let artisan = $state<ArtisanStorefront | undefined>(undefined);
  let artisanWork = $state<ListingSummary[]>([]);
  let loading = $state(true);

  // VIP Concierge modal
  let isConciergeOpen = $state(false);

  // Editorial Curated Themes
  let activeHeroIndex = $state(0);
  const HERO_SLIDES = [
    {
      titleKey: 'home.hero.slide1.title',
      subtitleKey: 'home.hero.slide1.subtitle',
      ctaKey: 'home.hero.slide1.cta',
      searchQuery: 'ajrakh',
      tag: 'Kutch & Dhamadka, Gujarat',
      theme: 'Botanical Indigo & Mud-Resist',
      image: 'https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=1600&q=80',
    },
    {
      titleKey: 'home.hero.slide2.title',
      subtitleKey: 'home.hero.slide2.subtitle',
      ctaKey: 'home.hero.slide2.cta',
      searchQuery: 'banarasi',
      tag: 'Varanasi Weavers Colony, UP',
      theme: 'Pure Zari & Kadwa Pit-Loom',
      image: 'https://images.unsplash.com/photo-1610030469983-98e550d6193c?auto=format&fit=crop&w=1600&q=80',
    },
    {
      titleKey: 'home.hero.slide3.title',
      subtitleKey: 'home.hero.slide3.subtitle',
      ctaKey: 'home.hero.slide3.cta',
      searchQuery: 'pashmina',
      tag: 'Old Srinagar Valley, J&K',
      theme: 'Microscopic Sozni Needlework',
      image: 'https://images.unsplash.com/photo-1590736969955-71cc94801759?auto=format&fit=crop&w=1600&q=80',
    },
  ];

  const currentSlide = $derived(HERO_SLIDES[activeHeroIndex]);

  // Curated Geographical Traditions
  const DISCOVERY_TAGS = [
    { name: 'Kutch Ajrakh', query: 'ajrakh' },
    { name: 'Banarasi Kadwa', query: 'banarasi' },
    { name: 'Kashmir Pashmina', query: 'pashmina' },
    { name: 'Pochampally Double-Ikat', query: 'pochampally' },
    { name: 'Bastar Lost-Wax Dhokra', query: 'dhokra' },
    { name: 'Kanchipuram Temple Silk', query: 'kanchipuram' },
    { name: 'Chanderi Gossamer', query: 'chanderi' },
    { name: 'Nizamabad Black Pottery', query: 'pottery' },
  ];

  onMount(() => {
    const timer = setInterval(() => {
      activeHeroIndex = (activeHeroIndex + 1) % HERO_SLIDES.length;
    }, 8000);
    return () => clearInterval(timer);
  });

  async function hydrate(ids: string[]): Promise<ListingSummary[]> {
    const settled = await Promise.allSettled(ids.map((id) => getListingSummary(id)));
    return settled
      .filter((r): r is PromiseFulfilledResult<ListingSummary> => r.status === 'fulfilled')
      .map((r) => r.value);
  }

  const FALLBACK_CRAFTS: Craft[] = [
    {
      id: 'craft-banarasi',
      slug: 'banarasi-brocade-weaving',
      display_name: 'Banarasi Brocade',
      gi_registration_no: 'GI-99',
      gi_certified: true,
      regions: ['Varanasi, Uttar Pradesh'],
      techniques: ['jacquard-weaving', 'zari-brocade'],
      materials: ['silk', 'zari'],
    },
    {
      id: 'craft-pashmina',
      slug: 'pashmina-weaving',
      display_name: 'Kashmir Pashmina',
      gi_registration_no: 'GI-46',
      gi_certified: true,
      regions: ['Srinagar, Jammu & Kashmir'],
      techniques: ['hand-spinning', 'twill-weaving'],
      materials: ['pashmina-wool'],
    },
    {
      id: 'craft-patola',
      slug: 'patan-patola',
      display_name: 'Patan Patola Ikat',
      gi_registration_no: 'GI-232',
      gi_certified: true,
      regions: ['Patan, Gujarat'],
      techniques: ['double-ikat-weaving'],
      materials: ['mulberry-silk', 'natural-dye'],
    },
    {
      id: 'craft-chanderi',
      slug: 'chanderi-weaving',
      display_name: 'Chanderi Gossamer',
      gi_registration_no: 'GI-14',
      gi_certified: true,
      regions: ['Chanderi, Madhya Pradesh'],
      techniques: ['pit-loom-weaving'],
      materials: ['silk-cotton', 'zari'],
    },
    {
      id: 'craft-dhokra',
      slug: 'dhokra-casting',
      display_name: 'Dhokra Bell Metal',
      gi_registration_no: 'GI-117',
      gi_certified: true,
      regions: ['Bastar, Chhattisgarh'],
      techniques: ['cire-perdue-lost-wax'],
      materials: ['brass', 'bronze', 'beeswax'],
    },
    {
      id: 'craft-bidriware',
      slug: 'bidriware',
      display_name: 'Bidriware Silver Inlay',
      gi_registration_no: 'GI-19',
      gi_certified: true,
      regions: ['Bidar, Karnataka'],
      techniques: ['zinc-copper-alloy-inlay'],
      materials: ['pure-silver', 'zinc-copper'],
    },
    {
      id: 'craft-madhubani',
      slug: 'madhubani-painting',
      display_name: 'Mithila / Madhubani',
      gi_registration_no: 'GI-75',
      gi_certified: true,
      regions: ['Madhubani, Bihar'],
      techniques: ['nib-and-twig-painting'],
      materials: ['handmade-paper', 'botanical-pigments'],
    },
    {
      id: 'craft-kutch-rogan',
      slug: 'kutch-rogan',
      display_name: 'Kutch Rogan Art',
      gi_registration_no: 'GI-312',
      gi_certified: true,
      regions: ['Nirona, Gujarat'],
      techniques: ['castor-oil-stylus-painting'],
      materials: ['castor-oil', 'natural-pigments'],
    },
  ];

  const FALLBACK_GI_LISTINGS: ListingSummary[] = [
    {
      id: 'listing-gi-1',
      product_id: 'prod-banarasi-kadwa',
      artisan_id: 'artisan-kabir',
      artisan_name: 'Mohammad Kabir Ansari',
      craft_name: 'Banarasi Brocade Weaving',
      craft_slug: 'banarasi-brocade-weaving',
      craft_gi_registration_no: 'GI-99',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Varanasi',
      artisan_state_code: 'UP',
      type: 'READY_STOCK',
      price: { amount_paise: 2450000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1610030469983-98e550d6193c?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Saree',
          description: 'Handwoven pure mulberry silk with electroplated real silver zari bootis.',
        },
      ],
    },
    {
      id: 'listing-gi-2',
      product_id: 'prod-pashmina-kani',
      artisan_id: 'artisan-mir',
      artisan_name: 'Ghulam Nabi Mir',
      craft_name: 'Kashmir Pashmina Weaving',
      craft_slug: 'pashmina-weaving',
      craft_gi_registration_no: 'GI-46',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Srinagar',
      artisan_state_code: 'JK',
      type: 'READY_STOCK',
      price: { amount_paise: 3800000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1590736969955-71cc94801759?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Changthangi Micro-Spun Needlework Pashmina Shawl',
          description: 'Ultra-fine 12-micron grade hand-spun cashmere with needle Sozni flora.',
        },
      ],
    },
    {
      id: 'listing-gi-3',
      product_id: 'prod-patola-shikargah',
      artisan_id: 'artisan-salvi',
      artisan_name: 'Dinesh Salvi Guild',
      craft_name: 'Patan Patola',
      craft_slug: 'patan-patola',
      craft_gi_registration_no: 'GI-232',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Patan',
      artisan_state_code: 'GJ',
      type: 'MADE_TO_ORDER',
      price: { amount_paise: 12000000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1617627143750-d86bc21e42bb?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Patan Double-Ikat Shikargah Royal Heritage Saree',
          description: 'Sacred mathematical warp and weft double-resist silk handloom.',
        },
      ],
    },
    {
      id: 'listing-gi-4',
      product_id: 'prod-dhokra-nandi',
      artisan_id: 'artisan-budheshwar',
      artisan_name: 'Budheshwar Ghadwa',
      craft_name: 'Dhokra Metal Casting',
      craft_slug: 'dhokra-casting',
      craft_gi_registration_no: 'GI-117',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Bastar',
      artisan_state_code: 'CT',
      type: 'READY_STOCK',
      price: { amount_paise: 850000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Bastar Cire-Perdue Lost-Wax Sacred Bull Figurine',
          description: 'Unreplicated single-cast bell metal sculpture with beeswax ribbing.',
        },
      ],
    },
  ];

  const FALLBACK_NEW_ARRIVALS: ListingSummary[] = [
    {
      id: 'listing-arr-1',
      product_id: 'prod-ajrakh-stole',
      artisan_id: 'artisan-khatri',
      artisan_name: 'Dr. Ismail Mohammed Khatri',
      craft_name: 'Ajrakh Block Printing',
      craft_slug: 'ajrakh-printing',
      craft_gi_registration_no: 'GI-312',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Kutch',
      artisan_state_code: 'GJ',
      type: 'READY_STOCK',
      price: { amount_paise: 420000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: '16-Stage Natural Indigo & Harde Riverbed Stole',
          description: 'Double-sided mud resist blocked on handspun indigenous Kala cotton.',
        },
      ],
    },
    {
      id: 'listing-arr-2',
      product_id: 'prod-black-pottery-handi',
      artisan_id: 'artisan-prajapati',
      artisan_name: 'Ram Prakash Prajapati',
      craft_name: 'Nizamabad Black Clay Pottery',
      craft_slug: 'nizamabad-black-pottery',
      craft_gi_registration_no: 'GI-264',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Azamgarh',
      artisan_state_code: 'UP',
      type: 'READY_STOCK',
      price: { amount_paise: 320000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Nizamabad Mirror-Burnished Black Clay Vedic Handi',
          description: 'Clay reduction smoked cookware infused with zinc-mercury silvery inlay.',
        },
      ],
    },
    {
      id: 'listing-arr-3',
      product_id: 'prod-bidri-vase',
      artisan_id: 'artisan-rashid',
      artisan_name: 'Abdul Rashid Guild',
      craft_name: 'Bidriware Inlay',
      craft_slug: 'bidriware',
      craft_gi_registration_no: 'GI-19',
      gi_certified: true,
      artisan_verified: true,
      artisan_district: 'Bidar',
      artisan_state_code: 'KA',
      type: 'READY_STOCK',
      price: { amount_paise: 650000, currency_code: 'INR' },
      image_url: 'https://images.unsplash.com/photo-1596178065887-1198b6148b2b?auto=format&fit=crop&w=600&q=80',
      translations: [
        {
          language: 'en',
          title: 'Pure Silver Wire Tarkashi Inlay Flower Vessel',
          description: 'Oxidized soil blackened alloy with inlaid pure silver geometric creepers.',
        },
      ],
    },
  ];

  const displayCrafts = $derived(crafts.length > 0 ? crafts : FALLBACK_CRAFTS);
  const displayGiListings = $derived(giListings.length > 0 ? giListings : FALLBACK_GI_LISTINGS);
  const displayNewArrivals = $derived(newArrivals.length > 0 ? newArrivals : FALLBACK_NEW_ARRIVALS);
  const displayMadeToOrder = $derived(
    madeToOrder.length > 0
      ? madeToOrder
      : FALLBACK_GI_LISTINGS.filter((l) => l.type === 'MADE_TO_ORDER')
  );

  async function load(): Promise<void> {
    loading = true;
    try {
      const [craftsRes, arrivalsRes, giRes, motoRes] = await Promise.all([
        listCrafts().catch(() => ({ crafts: [] })),
        listListings({ state: 'PUBLISHED' }).catch(() => ({ listings: [] })),
        search({ q: '', gi_tagged: 'true' }).catch(() => ({ results: [] })),
        search({ q: '', made_to_order: 'true' }).catch(() => ({ results: [] })),
      ]);

      crafts = craftsRes.crafts ?? [];

      const arrivalIds = (arrivalsRes.listings ?? []).slice(0, 6).map((l) => l.id!).filter(Boolean);
      newArrivals = await hydrate(arrivalIds);

      const giIds = (giRes.results ?? []).slice(0, 6).map((h) => h.listing_id!).filter(Boolean);
      giListings = await hydrate(giIds);

      const motoIds = (motoRes.results ?? []).slice(0, 6).map((h) => h.listing_id!).filter(Boolean);
      madeToOrder = await hydrate(motoIds);

      // Deterministic artisan pick
      for (const craft of displayCrafts) {
        if (!craft.id) continue;
        const detail = await getCraftArtisans(craft.id).catch(() => false);
        if (detail) break;
      }
    } catch {
      // Offline fallback
    } finally {
      loading = false;
    }
  }

  async function getCraftArtisans(craftId: string): Promise<boolean> {
    try {
      const listings = await listListings({ craft_id: craftId, state: 'PUBLISHED' });
      const first = listings.listings?.find((l) => l.artisan_id);
      if (!first?.artisan_id) return false;
      const [profile, work] = await Promise.all([
        getArtisanStorefront(first.artisan_id),
        listListings({ artisan_id: first.artisan_id, state: 'PUBLISHED' }),
      ]);
      artisan = profile;
      artisanWork = await hydrate((work.listings ?? []).slice(0, 3).map((l) => l.id!).filter(Boolean));
      return true;
    } catch {
      return false;
    }
  }

  const materials = $derived.by(() => {
    const set = new Set<string>();
    for (const c of displayCrafts) for (const m of c.materials ?? []) set.add(m);
    return Array.from(set).slice(0, 10);
  });

  $effect(() => {
    void load();
  });
</script>

<svelte:head>
  <title>{t('buyer.home.title')} — Kalakriti</title>
</svelte:head>

<h1 class="visually-hidden">{t('buyer.home.title')}</h1>

<!-- 1. AUTHORITATIVE GOVERNMENT TELEMETRY STRIP -->
<div class="k-telemetry-bar" role="region" aria-label={t('home.pulse.liveLabel')}>
  <div class="telemetry-inner">
    <div class="telemetry-authority">
      <Icon name="cluster" size="1rem" />
      <span class="authority-text">{t('home.pulse.liveLabel')}</span>
    </div>

    <ul class="telemetry-stats" role="list">
      <li class="stat-node">
        <Icon name="verified-artisan" size="0.9rem" />
        <span>{t('home.pulse.activeClusters')}</span>
      </li>
      <li class="stat-node">
        <Icon name="weaving" size="0.9rem" />
        <span>{t('home.pulse.loomsOperating')}</span>
      </li>
      <li class="stat-node">
        <Icon name="fair-price" size="0.9rem" />
        <span>{t('home.pulse.directPayouts')}</span>
      </li>
      <li class="stat-node">
        <Icon name="provenance" size="0.9rem" />
        <span>{t('home.pulse.cryptographicSeals')}</span>
      </li>
    </ul>

    <button
      type="button"
      class="k-concierge-pill"
      onclick={() => (isConciergeOpen = true)}
    >
      <Icon name="video" size="0.9rem" />
      <span>{t('home.concierge.bookButton')}</span>
    </button>
  </div>
</div>

<!-- 2. EDITORIAL ASYMMETRIC HERITAGE HERO -->
<section class="k-editorial-hero" aria-label="Curated Craft Edition">
  <div class="hero-asymmetric-grid">
    <!-- Image showcase: Product-Forward -->
    <div class="hero-photo-frame">
      <img
        class="hero-image"
        src={currentSlide.image}
        alt={currentSlide.theme}
        fetchpriority="high"
      />
      <div class="hero-image-badge">
        <span class="badge-dot"></span>
        <span>{currentSlide.theme}</span>
      </div>
    </div>

    <!-- Editorial context block -->
    <div class="hero-editorial-copy">
      <div class="hero-kicker-row">
        <p class="hero-kicker">{t('home.hero.thematicBadge')}</p>
        <span class="hero-slide-counter">0{activeHeroIndex + 1} / 0{HERO_SLIDES.length}</span>
      </div>

      <h2 class="hero-display-title">
        {activeHeroIndex === 0
          ? t('home.hero.slide1.title')
          : activeHeroIndex === 1
            ? t('home.hero.slide2.title')
            : t('home.hero.slide3.title')}
      </h2>

      <p class="hero-narrative">
        {activeHeroIndex === 0
          ? t('home.hero.slide1.subtitle')
          : activeHeroIndex === 1
            ? t('home.hero.slide2.subtitle')
            : t('home.hero.slide3.subtitle')}
      </p>

      <div class="hero-registry-meta">
        <span class="meta-item"><Icon name="location" size="1rem" /> {currentSlide.tag}</span>
        <span class="meta-item"><Icon name="verified-artisan" size="1rem" /> {t('home.hero.artisanLoomTag')}</span>
      </div>

      <div class="hero-cta-group">
        <a
          class="hero-primary-link"
          href={`/search?q=${encodeURIComponent(currentSlide.searchQuery)}`}
        >
          <span>
            {activeHeroIndex === 0
              ? t('home.hero.slide1.cta')
              : activeHeroIndex === 1
                ? t('home.hero.slide2.cta')
                : t('home.hero.slide3.cta')}
          </span>
          <Icon name="arrow-right" size="1rem" />
        </a>

        <button
          type="button"
          class="hero-secondary-btn"
          onclick={() => (isConciergeOpen = true)}
        >
          <Icon name="video" size="1rem" />
          <span>{t('home.concierge.title')}</span>
        </button>
      </div>

      <!-- Slide tabs -->
      <div class="hero-tab-selectors" role="tablist">
        {#each HERO_SLIDES as slide, idx}
          <button
            type="button"
            class="slide-tab-btn"
            class:active={activeHeroIndex === idx}
            onclick={() => (activeHeroIndex = idx)}
            aria-label={`Switch to slide ${idx + 1}: ${slide.theme}`}
          >
            <span class="tab-indicator"></span>
            <span class="tab-text">{slide.theme.split(' ')[0]}</span>
          </button>
        {/each}
      </div>
    </div>
  </div>
</section>

<!-- Discovery Traditions Strip -->
<div class="discovery-strip" role="navigation" aria-label={t('home.hero.quickTags')}>
  <span class="discovery-label">{t('home.hero.quickTags')}:</span>
  <ul class="discovery-list" role="list">
    {#each DISCOVERY_TAGS as tag}
      <li>
        <a href={`/search?q=${encodeURIComponent(tag.query)}`} class="discovery-chip">
          {tag.name}
        </a>
      </li>
    {/each}
  </ul>
</div>

<!-- 3. VOICES FROM THE LOOMS (REELS) -->
<Section variant="khadi-plain">
  <SectionHeader
    kicker={t('home.voices.kicker')}
    heading={t('home.voices.heading')}
    href="/feed"
  />
  <VoicesReelCarousel />
</Section>

<!-- 4. 12 INDIGENOUS ARTISAN CRAFT DISCIPLINES -->
<Section variant="khadi-plain">
  {#if loading}
    <span class="sr-only">Loading verified artisan catalog...</span>
  {/if}
  <ArtisanCraftGrid />
</Section>

<Divider variant="blockprint-running" density="medium" />

<!-- 5. REGIONAL HERITAGE BELT NAVIGATOR -->
<Section variant="khadi-weft">
  <RegionalBeltNavigator />
</Section>

<!-- 6. CURATED LIFESTYLE & OCCASION EDITS (ASYMMETRIC EDITORIAL GRID) -->
<Section variant="khadi-plain">
  <SectionHeader kicker={t('home.edits.kicker')} heading={t('home.edits.heading')} />
  <p class="section-subhead">{t('home.edits.subheading')}</p>

  <div class="editorial-lifestyle-grid">
    <!-- Featured Large Card -->
    <a href="/search?q=wedding" class="lifestyle-card feature-tile">
      <div class="tile-photo-wrap">
        <img
          src="https://images.unsplash.com/photo-1617627143750-d86bc21e42bb?auto=format&fit=crop&w=1200&q=80"
          alt={t('home.edits.edit1.title')}
          loading="lazy"
        />
        <span class="tile-tag">Heirloom Wardrobe</span>
      </div>
      <div class="tile-details">
        <h3 class="tile-title">{t('home.edits.edit1.title')}</h3>
        <p class="tile-desc">{t('home.edits.edit1.desc')}</p>
      </div>
    </a>

    <!-- Subordinate Card 1 -->
    <a href="/search?q=terracotta" class="lifestyle-card">
      <div class="tile-photo-wrap">
        <img
          src="https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=800&q=80"
          alt={t('home.edits.edit2.title')}
          loading="lazy"
        />
        <span class="tile-tag">Vedic Culinary</span>
      </div>
      <div class="tile-details">
        <h3 class="tile-title">{t('home.edits.edit2.title')}</h3>
        <p class="tile-desc">{t('home.edits.edit2.desc')}</p>
      </div>
    </a>

    <!-- Subordinate Card 2 -->
    <a href="/search?q=brass" class="lifestyle-card">
      <div class="tile-photo-wrap">
        <img
          src="https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=800&q=80"
          alt={t('home.edits.edit3.title')}
          loading="lazy"
        />
        <span class="tile-tag">Architectural Metal</span>
      </div>
      <div class="tile-details">
        <h3 class="tile-title">{t('home.edits.edit3.title')}</h3>
        <p class="tile-desc">{t('home.edits.edit3.desc')}</p>
      </div>
    </a>

    <!-- Subordinate Card 3 -->
    <a href="/search?q=jute" class="lifestyle-card">
      <div class="tile-photo-wrap">
        <img
          src="https://images.unsplash.com/photo-1596178065887-1198b6148b2b?auto=format&fit=crop&w=800&q=80"
          alt={t('home.edits.edit4.title')}
          loading="lazy"
        />
        <span class="tile-tag">Natural Fibers</span>
      </div>
      <div class="tile-details">
        <h3 class="tile-title">{t('home.edits.edit4.title')}</h3>
        <p class="tile-desc">{t('home.edits.edit4.desc')}</p>
      </div>
    </a>
  </div>
</Section>

<!-- 7. GI-TAGGED SPECIFIC CATALOGUE RAIL -->
{#if displayGiListings.length > 0}
  <Section variant="khadi-weft">
    <SectionHeader kicker={t('home.giTagged.kicker')} heading={t('home.giTagged.heading')} href="/search?gi_tagged=true" />
    <div class="rail">
      {#each displayGiListings as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

<!-- 8. NEW ARRIVALS RAIL -->
{#if displayNewArrivals.length > 0}
  <Section variant="khadi-plain">
    <SectionHeader kicker={t('home.newArrivals.kicker')} heading={t('home.newArrivals.heading')} href="/search" />
    <div class="rail">
      {#each displayNewArrivals as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

<!-- 9. MADE TO ORDER RAIL -->
{#if displayMadeToOrder.length > 0}
  <Section variant="khadi-weft">
    <SectionHeader kicker={t('home.madeToOrder.kicker')} heading={t('home.madeToOrder.heading')} href="/search?made_to_order=true" />
    <div class="rail">
      {#each displayMadeToOrder as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

<!-- 10. VIRASAT CULTURAL JOURNAL -->
<Section variant="khadi-plain">
  <SectionHeader kicker={t('home.journal.kicker')} heading={t('home.journal.heading')} />
  <p class="section-subhead">{t('home.journal.subheading')}</p>

  <div class="journal-editorial-grid">
    <article class="journal-entry">
      <div class="journal-metadata">
        <span class="journal-cluster-tag">Dhamadka Cluster</span>
        <span class="journal-time">{t('home.journal.article1.readTime')}</span>
      </div>
      <h3 class="journal-headline">{t('home.journal.article1.title')}</h3>
      <p class="journal-byline">{t('home.journal.article1.author')}</p>
      <p class="journal-lead">
        How the Khatri master dyers of Kutch sustain 16 chemical-free natural resist phases along the seasonal riverbanks of Dhamadka.
      </p>
      <a href="/search?q=ajrakh" class="journal-read-link">
        <span>{t('home.journal.readArticle')}</span>
        <Icon name="arrow-right" size="0.9rem" />
      </a>
    </article>

    <article class="journal-entry">
      <div class="journal-metadata">
        <span class="journal-cluster-tag">Patan Guild</span>
        <span class="journal-time">{t('home.journal.article2.readTime')}</span>
      </div>
      <h3 class="journal-headline">{t('home.journal.article2.title')}</h3>
      <p class="journal-byline">{t('home.journal.article2.author')}</p>
      <p class="journal-lead">
        Decoding the sacred geometric algorithms and double-resist warp alignments of Gujarat’s legendary 800-year Patan guild.
      </p>
      <a href="/search?q=patola" class="journal-read-link">
        <span>{t('home.journal.readArticle')}</span>
        <Icon name="arrow-right" size="0.9rem" />
      </a>
    </article>

    <article class="journal-entry">
      <div class="journal-metadata">
        <span class="journal-cluster-tag">Bastar Ghadwa</span>
        <span class="journal-time">{t('home.journal.article3.readTime')}</span>
      </div>
      <h3 class="journal-headline">{t('home.journal.article3.title')}</h3>
      <p class="journal-byline">{t('home.journal.article3.author')}</p>
      <p class="journal-lead">
        Inside the forest furnaces of Bastar where Ghadwa metalsmiths transform wild honey wax, red clay, and scrap bronze into animist deities.
      </p>
      <a href="/search?q=dhokra" class="journal-read-link">
        <span>{t('home.journal.readArticle')}</span>
        <Icon name="arrow-right" size="0.9rem" />
      </a>
    </article>
  </div>
</Section>

<!-- 11. ENTERPRISE & INSTITUTIONAL PROCUREMENT HUB -->
<Section variant="khadi-weft">
  <InstitutionalProcurementBanner />
</Section>

<!-- 12. CRYPTOGRAPHIC PROVENANCE TERMINAL -->
<Section variant="khadi-plain">
  <ProvenanceTerminal />
</Section>

<!-- 13. KALAKRITI ATELIER BESPOKE COMMISSIONS -->
<Section variant="khadi-weft">
  <AtelierCommissionCard onOpenConcierge={() => (isConciergeOpen = true)} />
</Section>

<!-- 14. DETERMINISTIC ARTISAN OF THE MONTH -->
{#if artisan}
  <Section variant="indigo-panel">
    <SectionHeader kicker={t('home.artisanOfMonth.kicker')} heading={artisan.display_name ?? ''} />
    <div class="artisan-of-month">
      {#if artisan.image_url}
        <img class="artisan-of-month__portrait" src={artisan.image_url} alt="" />
      {/if}
      <div class="artisan-of-month__body">
        {#if artisan.district || artisan.state_code}
          <p class="artisan-of-month__location">
            <Icon name="location" />
            {[artisan.district, artisan.state_code].filter(Boolean).join(', ')}
          </p>
        {/if}
        {#if artisan.verified}
          <p class="artisan-of-month__verified"><Icon name="verified-artisan" />{t('artisan.verified')}</p>
        {/if}
        {#if artisan.bio}<p class="artisan-of-month__bio">{artisan.bio}</p>{/if}
        <div class="artisan-of-month__work">
          {#each artisanWork as w (w.id)}
            {#if w.image_url}<img src={w.image_url} alt="" />{/if}
          {/each}
        </div>
        <a class="artisan-of-month__link" href={`/artisan/${artisan.slug}`}>{t('home.artisanOfMonth.storefrontLink')}</a>
      </div>
    </div>
  </Section>
{/if}

<!-- 15. TRUST STRIP -->
<Section variant="khadi-plain">
  <ul class="trust-strip" role="list">
    <li><Icon name="verified-artisan" />{t('home.trust.artisanVerified')}</li>
    <li><Icon name="fair-price" />{t('home.trust.directPayment')}</li>
    <li><Icon name="provenance" />{t('home.trust.provenanceVerified')}</li>
    <li><Icon name="lock" />{t('home.trust.securePayment')}</li>
  </ul>
</Section>

<!-- 16. MATERIALS DIRECTORY -->
{#if materials.length > 0}
  <Section variant="khadi-plain">
    <SectionHeader kicker={t('home.browseByMaterial.kicker')} heading={t('home.browseByMaterial.heading')} />
    <ul class="material-row" role="list">
      {#each materials as material (material)}
        <li><a href={`/search?material=${encodeURIComponent(material)}`}>{material}</a></li>
      {/each}
    </ul>
  </Section>
{/if}

<!-- 17. PUBLIC CRAFT DIRECTORY & IMPACT CASE STUDIES -->
<Section variant="khadi-weft">
  <div class="directory-case-studies-banner">
    <div class="banner-col">
      <p class="banner-kicker">Digital Master Register</p>
      <h3 class="banner-title">74 National Geographic Indication Clusters</h3>
      <p class="banner-desc">Explore our comprehensive public directory of registered craft corridors, artisan cooperatives, and verified master craftspersons across all 28 states and 8 union territories.</p>
      <a href="/catalog" class="banner-action-link">
        <span>Browse Complete Craft Catalog</span>
        <Icon name="arrow-right" size="0.9rem" />
      </a>
    </div>
    <div class="banner-divider" aria-hidden="true"></div>
    <div class="banner-col">
      <p class="banner-kicker">Guild Provenance & Economics</p>
      <h3 class="banner-title">Empirical Field Case Studies</h3>
      <p class="banner-desc">Review in-depth field research on how cryptographic provenance seals, zero-middlemen direct payouts, and collective B2B export lots transform rural artisan earnings.</p>
      <a href="/case-studies" class="banner-action-link">
        <span>Read Guild Case Studies</span>
        <Icon name="arrow-right" size="0.9rem" />
      </a>
    </div>
  </div>
</Section>

<!-- 18. SELLER EMPOWERMENT & ARTISAN TESTIMONIALS -->
<Section variant="khadi-plain">
  <SellerShowcaseBanner />
</Section>

<!-- 19. FREQUENTLY ASKED QUESTIONS -->
<Section variant="khadi-plain">
  <SectionHeader kicker="Authenticity & Fair Trade" heading="Frequently Answered Inquiries" />
  <FaqAccordion />
</Section>

<!-- 20. VIP CONCIERGE VIDEO MODAL -->
<ArtisanConciergeModal bind:isOpen={isConciergeOpen} />

<style>
  .visually-hidden {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }

  .section-subhead {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0 0 var(--k-space-5) 0;
    max-inline-size: 70ch;
  }

  /* 1. Government Telemetry Bar */
  .k-telemetry-bar {
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    padding: var(--k-space-2) var(--k-space-4);
    margin-block-end: var(--k-space-5);
  }

  .telemetry-inner {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .telemetry-authority {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-primary-text);
  }

  .telemetry-stats {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-4);
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .stat-node {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-text-primary);
  }

  .k-concierge-pill {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    background: none;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    padding: var(--k-space-1) var(--k-space-3);
    font-family: inherit;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    cursor: pointer;
    transition: background-color 0.15s ease;
  }

  .k-concierge-pill:hover {
    background-color: var(--k-surface-sunken);
  }

  /* 2. Asymmetric Editorial Hero */
  .k-editorial-hero {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    overflow: hidden;
    margin-block-end: var(--k-space-4);
  }

  .hero-asymmetric-grid {
    display: grid;
    grid-template-columns: 1fr;
  }

  @media (min-width: 52rem) {
    .hero-asymmetric-grid {
      grid-template-columns: 1.25fr 1fr;
    }
  }

  .hero-photo-frame {
    position: relative;
    aspect-ratio: 16 / 10;
    background-color: var(--k-surface-sunken);
    overflow: hidden;
  }

  .hero-image {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .hero-image-badge {
    position: absolute;
    inset-block-start: var(--k-space-3);
    inset-inline-start: var(--k-space-3);
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: 3px 10px;
    background-color: var(--k-surface-inverse);
    color: var(--k-text-on-inverse);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    border-radius: var(--k-radius-sm);
  }

  .badge-dot {
    inline-size: 6px;
    block-size: 6px;
    border-radius: 50%;
    background-color: var(--k-accent-primary-text);
  }

  .hero-editorial-copy {
    padding: var(--k-space-5);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  @media (min-width: 52rem) {
    .hero-editorial-copy {
      padding: var(--k-space-6);
      border-inline-start: var(--k-hairline) solid var(--k-border-hairline);
    }
  }

  .hero-kicker-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .hero-kicker {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--k-accent-primary-text);
    margin: 0;
  }

  .hero-slide-counter {
    font-family: monospace;
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .hero-display-title {
    font-family: var(--k-font-display);
    font-size: clamp(1.5rem, 3vw, 2.25rem);
    line-height: 1.2;
    color: var(--k-text-primary);
    margin: 0;
  }

  .hero-narrative {
    font-size: var(--k-text-sm);
    line-height: 1.6;
    color: var(--k-text-secondary);
    margin: 0;
  }

  .hero-registry-meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3);
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .meta-item {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
  }

  .hero-cta-group {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3);
    align-items: center;
    margin-block-start: var(--k-space-2);
  }

  .hero-primary-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-4);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    text-decoration: none;
    font-weight: var(--k-weight-semibold);
    font-size: var(--k-text-sm);
    border-radius: var(--k-radius-sm);
  }

  .hero-secondary-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background: none;
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-primary);
    font-family: inherit;
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
  }

  .hero-tab-selectors {
    display: flex;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .slide-tab-btn {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: inherit;
    text-align: left;
    flex: 1;
  }

  .tab-indicator {
    display: block;
    inline-size: 100%;
    block-size: 2px;
    background-color: var(--k-border-hairline);
    transition: background-color 0.2s ease;
  }

  .slide-tab-btn.active .tab-indicator {
    background-color: var(--k-accent-primary-bg);
  }

  .tab-text {
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  .slide-tab-btn.active .tab-text {
    color: var(--k-text-primary);
    font-weight: var(--k-weight-semibold);
  }

  /* Discovery Chips Strip */
  .discovery-strip {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding-block: var(--k-space-2);
    margin-block-end: var(--k-space-4);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .discovery-label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-text-secondary);
    white-space: nowrap;
  }

  .discovery-list {
    display: flex;
    gap: var(--k-space-2);
    overflow-x: auto;
    list-style: none;
    padding: 0;
    margin: 0;
    scrollbar-width: none;
  }

  .discovery-chip {
    display: inline-block;
    padding: var(--k-space-1) var(--k-space-3);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-full, 999px);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
    text-decoration: none;
    white-space: nowrap;
    transition: border-color 0.15s ease;
  }

  .discovery-chip:hover {
    border-color: var(--k-border-interactive);
  }

  /* Asymmetric Lifestyle Grid */
  .editorial-lifestyle-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-4);
  }

  @media (min-width: 44rem) {
    .editorial-lifestyle-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 60rem) {
    .editorial-lifestyle-grid {
      grid-template-columns: 1.5fr 1fr 1fr;
    }

    .feature-tile {
      grid-row: span 2;
    }
  }

  .lifestyle-card {
    display: flex;
    flex-direction: column;
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    overflow: hidden;
    text-decoration: none;
    color: inherit;
    transition: border-color 0.15s ease;
  }

  .lifestyle-card:hover {
    border-color: var(--k-border-interactive);
  }

  .tile-photo-wrap {
    position: relative;
    aspect-ratio: 16 / 10;
    overflow: hidden;
    background-color: var(--k-surface-sunken);
  }

  .tile-photo-wrap img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .tile-tag {
    position: absolute;
    inset-block-start: var(--k-space-2);
    inset-inline-start: var(--k-space-2);
    padding: 2px 8px;
    background-color: var(--k-surface-inverse);
    color: var(--k-text-on-inverse);
    font-size: 0.65rem;
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-radius: var(--k-radius-sm);
  }

  .tile-details {
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .tile-title {
    font-family: var(--k-font-display);
    font-size: var(--k-text-base);
    color: var(--k-text-primary);
    margin: 0;
  }

  .tile-desc {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: 1.4;
    margin: 0;
  }

  /* Virasat Journal Editorial Layout */
  .journal-editorial-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-4);
  }

  @media (min-width: 50rem) {
    .journal-editorial-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  .journal-entry {
    padding: var(--k-space-5);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .journal-metadata {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .journal-cluster-tag {
    font-size: 0.7rem;
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    color: var(--k-accent-primary-text);
    letter-spacing: 0.05em;
  }

  .journal-time {
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  .journal-headline {
    font-family: var(--k-font-display);
    font-size: var(--k-text-lg);
    color: var(--k-text-primary);
    line-height: 1.3;
    margin: 0;
  }

  .journal-byline {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-secondary);
    margin: 0;
  }

  .journal-lead {
    font-size: var(--k-text-xs);
    line-height: 1.6;
    color: var(--k-text-secondary);
    margin: 0;
  }

  .journal-read-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-accent-secondary);
    text-decoration: none;
  }

  /* Product Rails */
  .rail {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(11rem, 1fr);
    gap: var(--k-space-4);
    overflow-x: auto;
    margin-block-start: var(--k-space-4);
    padding-block-end: var(--k-space-2);
  }

  @media (min-width: 48rem) {
    .rail {
      grid-auto-columns: minmax(13rem, 1fr);
    }
  }

  /* Artisan of the Month */
  .artisan-of-month {
    display: grid;
    grid-template-columns: 10rem 1fr;
    gap: var(--k-space-5);
    margin-block-start: var(--k-space-5);
  }

  .artisan-of-month__portrait {
    inline-size: 100%;
    aspect-ratio: 1;
    object-fit: cover;
    border-radius: var(--k-radius-md);
  }

  .artisan-of-month__location,
  .artisan-of-month__verified {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-sm);
  }

  .artisan-of-month__work {
    display: flex;
    gap: var(--k-space-2);
    margin-block: var(--k-space-3);
  }

  .artisan-of-month__work img {
    inline-size: 5rem;
    block-size: 5rem;
    object-fit: cover;
    border-radius: var(--k-radius-sm);
  }

  .artisan-of-month__link {
    color: inherit;
    font-weight: var(--k-weight-semibold);
  }

  /* Trust Strip */
  .trust-strip {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3) var(--k-space-6);
    list-style: none;
    padding: 0;
  }

  .trust-strip li {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding-inline-end: var(--k-space-6);
    border-inline-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .trust-strip li:last-child {
    border-inline-end: none;
  }

  /* Materials */
  .material-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-4);
    list-style: none;
    padding: 0;
  }

  .material-row a {
    display: inline-block;
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-full, 999px);
    color: inherit;
    text-decoration: none;
    text-transform: capitalize;
  }

  /* 17. Directory & Case Studies Banner */
  .directory-case-studies-banner {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-6);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-lg);
    padding: var(--k-space-6);
  }

  @media (min-width: 48rem) {
    .directory-case-studies-banner {
      grid-template-columns: 1fr 1px 1fr;
      align-items: stretch;
      padding: var(--k-space-8);
    }
  }

  .banner-col {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }

  .banner-divider {
    background-color: var(--k-border-hairline);
    inline-size: 100%;
    block-size: 1px;
  }

  @media (min-width: 48rem) {
    .banner-divider {
      inline-size: 1px;
      block-size: 100%;
    }
  }

  .banner-kicker {
    font-size: var(--k-text-xs);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
    color: var(--k-terracotta-700, #96381e);
    font-weight: var(--k-weight-semibold);
    margin: 0 0 var(--k-space-2) 0;
  }

  .banner-title {
    font-family: var(--k-font-display, serif);
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-3) 0;
    line-height: 1.25;
  }

  .banner-desc {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    line-height: 1.6;
    margin: 0 0 var(--k-space-5) 0;
    flex-grow: 1;
  }

  .banner-action-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    color: var(--k-terracotta-700, #96381e);
    text-decoration: none;
    border-bottom: 1.5px solid var(--k-terracotta-600, #b24526);
    padding-bottom: 2px;
    align-self: flex-start;
    transition: color 0.15s ease, border-color 0.15s ease;
  }

  .banner-action-link:hover {
    color: var(--k-terracotta-800, #7a2010);
    border-color: var(--k-terracotta-800, #7a2010);
  }
</style>
