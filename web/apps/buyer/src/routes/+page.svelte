<!--
  apps/buyer/src/routes/+page.svelte

  The editorial marketplace home. Structure follows the IndiaHandmade
  content model (hero, shop-by-craft, editorial rails, artisan of the
  month, trust strip, material browse) executed with this system's design
  law: hairline separation, Section's khadi/indigo rhythm, no card-grid
  sameness, no gradient hero.

  Every section here reads real data through GET /listings and GET /search
  (gi_only and listing_type both apply server-side, not by filtering a
  fetched page client-side -- see services/bff/internal/bff/handler/api.go's
  searchFilters). "Artisan of the month" and native-script craft names have
  no backend source (documented in ml_wiring.md): the artisan is a
  deterministic pick, not a fabricated ranking, and craft tiles show only
  the ontology's one display_name.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { SectionHeader, Skeleton } from '@kalakriti/ui';
  import { Section } from '@kalakriti/patterns';
  import { HeroBackdrop } from '@kalakriti/illustrations';
  import { Divider } from '@kalakriti/ornament';
  import { Icon } from '@kalakriti/icons';
  import { listListings, listCrafts, search, getListingSummary, getArtisanStorefront, type components } from '@kalakriti/api';
  import ListingCard from '$lib/ListingCard.svelte';
  import { craftIcon } from '$lib/craft-icon';

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
  let heroImage = $state<string | undefined>(undefined);
  let loading = $state(true);

  async function hydrate(ids: string[]): Promise<ListingSummary[]> {
    const settled = await Promise.allSettled(ids.map((id) => getListingSummary(id)));
    return settled.filter((r): r is PromiseFulfilledResult<ListingSummary> => r.status === 'fulfilled').map((r) => r.value);
  }

  async function load(): Promise<void> {
    loading = true;
    try {
      const [craftsRes, arrivalsRes, giRes, motoRes] = await Promise.all([
        listCrafts(),
        listListings({ state: 'PUBLISHED' }),
        search({ q: '', gi_tagged: 'true' }),
        search({ q: '', made_to_order: 'true' }),
      ]);

      crafts = craftsRes.crafts ?? [];

      const arrivalIds = (arrivalsRes.listings ?? []).slice(0, 6).map((l) => l.id!).filter(Boolean);
      newArrivals = await hydrate(arrivalIds);

      const giIds = (giRes.results ?? []).slice(0, 6).map((h) => h.listing_id!).filter(Boolean);
      giListings = await hydrate(giIds);

      const motoIds = (motoRes.results ?? []).slice(0, 6).map((h) => h.listing_id!).filter(Boolean);
      madeToOrder = await hydrate(motoIds);

      heroImage = newArrivals.find((l) => l.image_url)?.image_url ?? giListings.find((l) => l.image_url)?.image_url;

      // Artisan of the month: a deterministic pick over real data (the first
      // artisan with a published listing under the first craft that has
      // one) -- not a ranking the backend produces. See file header.
      for (const craft of crafts) {
        if (!craft.id) continue;
        const detail = await getCraftArtisans(craft.id);
        if (detail) break;
      }
    } finally {
      loading = false;
    }
  }

  async function getCraftArtisans(craftId: string): Promise<boolean> {
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
  }

  const materials = $derived.by(() => {
    const set = new Set<string>();
    for (const c of crafts) for (const m of c.materials ?? []) set.add(m);
    return Array.from(set).slice(0, 10);
  });

  $effect(() => {
    void load();
  });
</script>

<svelte:head>
  <title>{t('buyer.home.title')}</title>
</svelte:head>

<h1 class="visually-hidden">{t('buyer.home.title')}</h1>

<HeroBackdrop element="section" class="hero">
  {#if heroImage}
    <img class="hero__image" src={heroImage} alt="" fetchpriority="high" />
  {/if}
  <div class="hero__overlay">
    <p class="hero__tagline">{t('app.tagline')}</p>
    <a class="hero__cta" href="/search">{t('home.hero.cta')}</a>
  </div>
</HeroBackdrop>

<Section variant="khadi-plain">
  <SectionHeader kicker={t('home.shopByCraft.kicker')} heading={t('home.shopByCraft.heading')} />
  {#if loading && crafts.length === 0}
    <Skeleton shape="card" height="8rem" />
  {:else}
    <ul class="craft-tiles" role="list">
      {#each crafts.slice(0, 12) as craft (craft.id)}
        <li>
          <a class="craft-tile" href={`/craft/${craft.slug}`}>
            <Icon name={craftIcon(craft.slug ?? craft.display_name ?? '')} />
            <span>{craft.display_name}</span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</Section>

<Divider variant="blockprint-running" density="medium" />

{#if giListings.length > 0}
  <Section variant="khadi-weft">
    <SectionHeader kicker={t('home.giTagged.kicker')} heading={t('home.giTagged.heading')} href="/search?gi_tagged=true" />
    <div class="rail">
      {#each giListings as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

{#if newArrivals.length > 0}
  <Section variant="khadi-plain">
    <SectionHeader kicker={t('home.newArrivals.kicker')} heading={t('home.newArrivals.heading')} href="/search" />
    <div class="rail">
      {#each newArrivals as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

{#if madeToOrder.length > 0}
  <Section variant="khadi-weft">
    <SectionHeader kicker={t('home.madeToOrder.kicker')} heading={t('home.madeToOrder.heading')} href="/search?made_to_order=true" />
    <div class="rail">
      {#each madeToOrder as listing (listing.id)}
        <ListingCard {listing} href={`/listing/${listing.id}`} />
      {/each}
    </div>
  </Section>
{/if}

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

<Section variant="khadi-plain">
  <ul class="trust-strip" role="list">
    <li><Icon name="verified-artisan" />{t('home.trust.artisanVerified')}</li>
    <li><Icon name="fair-price" />{t('home.trust.directPayment')}</li>
    <li><Icon name="provenance" />{t('home.trust.provenanceVerified')}</li>
    <li><Icon name="lock" />{t('home.trust.securePayment')}</li>
  </ul>
</Section>

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

<style>
  .visually-hidden {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }

  :global(.hero) {
    position: relative;
    aspect-ratio: 16 / 9;
    min-block-size: 16rem;
    border-radius: var(--k-radius-lg);
    overflow: hidden;
    background-color: var(--k-surface-sunken);
  }

  .hero__image {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .hero__overlay {
    position: absolute;
    inset-block-end: 0;
    inset-inline-start: 0;
    padding: var(--k-space-5);
    color: white;
    background: linear-gradient(to top, rgb(0 0 0 / 0.55), transparent);
    inline-size: 100%;
    box-sizing: border-box;
  }

  .hero__tagline {
    font-family: var(--k-font-display);
    font-size: var(--k-text-xl);
    max-inline-size: 32ch;
    margin-block-end: var(--k-space-3);
  }

  .hero__cta {
    display: inline-block;
    padding: var(--k-space-2) var(--k-space-4);
    background-color: white;
    color: var(--k-text-primary);
    border-radius: var(--k-radius-sm);
    text-decoration: none;
    font-weight: var(--k-weight-semibold);
  }

  .craft-tiles {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(7rem, 1fr));
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-5);
  }

  .craft-tile {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-4) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    color: inherit;
    text-decoration: none;
    text-align: center;
    font-size: var(--k-text-sm);
  }

  .rail {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(11rem, 1fr);
    gap: var(--k-space-4);
    overflow-x: auto;
    margin-block-start: var(--k-space-5);
    padding-block-end: var(--k-space-2);
  }

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

  .trust-strip {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3) var(--k-space-6);
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

  .material-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-4);
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

  @media (min-width: 48rem) {
    .rail {
      grid-auto-columns: minmax(13rem, 1fr);
    }
  }
</style>
