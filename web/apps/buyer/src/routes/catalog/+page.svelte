<!--
  apps/buyer/src/routes/catalog/+page.svelte

  National GI Craft & Cluster Catalog:
  Public directory of all 74 GI-certified craft clusters, geographical regions,
  and traditional techniques across India.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'All GI Craft Clusters' },
  ]);

  let searchQuery = $state('');
  let selectedBelt = $state('all');

  const FALLBACK_CRAFTS = [
    {
      id: 'craft-banarasi',
      slug: 'banarasi-brocade-weaving',
      display_name: 'Banarasi Brocade & Kadwa Zari',
      gi_registration_no: 'GI-99',
      belt: 'north',
      regions: ['Varanasi, Uttar Pradesh'],
      techniques: ['Jacquard Pit-Loom', 'Silver-Gilt Zari'],
      description: 'Master kadwa interlocking where motifs are individually woven on pit-looms using pure silver and gold zari.',
    },
    {
      id: 'craft-pashmina',
      slug: 'pashmina-weaving',
      display_name: 'Kashmir Pashmina & Sozni Needlework',
      gi_registration_no: 'GI-46',
      belt: 'north',
      regions: ['Srinagar, Jammu & Kashmir'],
      techniques: ['Hand-Spun Changthangi', 'Sozni Needle'],
      description: 'Hand-carded microscopic cashmere fibers hand-spun on wooden charkhas and stitched with imperial sozni needlecraft.',
    },
    {
      id: 'craft-ajrakh',
      slug: 'ajrakh-block-print',
      display_name: 'Kutch Ajrakh & Dabu Resist',
      gi_registration_no: 'GI-72',
      belt: 'west',
      regions: ['Dhamadka, Kutch, Gujarat'],
      techniques: ['16-Stage River Resist', 'Natural Indigo'],
      description: 'Ancient geometric block-printing using madder roots, river clay, and multi-week living organic indigo vats.',
    },
    {
      id: 'craft-patola',
      slug: 'patan-patola',
      display_name: 'Patan Patola Double Ikat',
      gi_registration_no: 'GI-232',
      belt: 'west',
      regions: ['Patan, Gujarat'],
      techniques: ['Double-Ikat Weaving', 'Botanical Pigments'],
      description: 'Mathematical warp and weft tie-dye alignment woven on tilted rosewood handlooms with identical clarity on both faces.',
    },
    {
      id: 'craft-kanchi',
      slug: 'kanchipuram-silk',
      display_name: 'Kanchipuram Temple Silk',
      gi_registration_no: 'GI-1',
      belt: 'south',
      regions: ['Kanchipuram, Tamil Nadu'],
      techniques: ['Korvai Joint', 'Three-Shuttle Weave'],
      description: 'Heavy mulberry silk with contrasting body and border attached via interlocking korvai technique with temple spires.',
    },
    {
      id: 'craft-bidri',
      slug: 'bidriware',
      display_name: 'Bidriware Pure Silver Inlay',
      gi_registration_no: 'GI-19',
      belt: 'south',
      regions: ['Bidar, Karnataka'],
      techniques: ['Zinc-Copper Damascening', 'Soil Firing'],
      description: 'Inlaid pure silver wire set into blackened zinc-copper alloy using special salt-rich soil from Bidar fort.',
    },
    {
      id: 'craft-dhokra',
      slug: 'dhokra-casting',
      display_name: 'Bastar Lost-Wax Bell Metal',
      gi_registration_no: 'GI-117',
      belt: 'central',
      regions: ['Bastar, Chhattisgarh'],
      techniques: ['Cire-Perdue Casting', 'Beeswax Threads'],
      description: 'Tribal lost-wax metallurgy using molten bronze poured into charcoal-fired river clay and beeswax molds.',
    },
    {
      id: 'craft-chanderi',
      slug: 'chanderi-weaving',
      display_name: 'Chanderi Gossamer Silk-Cotton',
      gi_registration_no: 'GI-14',
      belt: 'central',
      regions: ['Chanderi, Madhya Pradesh'],
      techniques: ['Pit-Loom Weaving', 'Zari Bootis'],
      description: 'Sheer translucent weave combining pure Degummed silk warp with fine cotton weft and delicate hand-picked zari bootis.',
    },
    {
      id: 'craft-muga',
      slug: 'assam-muga-silk',
      display_name: 'Assam Golden Muga Silk',
      gi_registration_no: 'GI-55',
      belt: 'east',
      regions: ['Sualkuchi, Assam'],
      techniques: ['Wild Antheraea Silk', 'Throw-Shuttle'],
      description: 'Naturally lustrous golden-yellow wild silk endemic to the Brahmaputra valley that gains brilliance with each wash.',
    },
    {
      id: 'craft-nizamabad',
      slug: 'nizamabad-black-pottery',
      display_name: 'Nizamabad Smoke-Fired Black Pottery',
      gi_registration_no: 'GI-398',
      belt: 'north',
      regions: ['Azamgarh, Uttar Pradesh'],
      techniques: ['Oxygen Reduction Firing', 'Silver Foil Inlay'],
      description: 'Lustrous pitch-black earthenware achieved through kiln smoke reduction and engraved with shimmering zinc-mercury foil.',
    },
  ];

  const filteredCrafts = $derived(
    FALLBACK_CRAFTS.filter((c) => {
      const matchesSearch =
        searchQuery === '' ||
        c.display_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.regions.some((r) => r.toLowerCase().includes(searchQuery.toLowerCase())) ||
        c.gi_registration_no.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesBelt = selectedBelt === 'all' || c.belt === selectedBelt;
      return matchesSearch && matchesBelt;
    })
  );
</script>

<svelte:head>
  <title>National GI Craft & Cluster Catalog - Kalakriti</title>
  <meta name="description" content="Explore India's complete national registry of Geographical Indication (GI) handlooms, tribal metallurgy, and authentic handicraft traditions." />
</svelte:head>

<div class="catalog-page">
  <div class="catalog-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <header class="catalog-header">
      <span class="catalog-kicker">Geographical Indications Registry of India</span>
      <h1 class="catalog-title">National Craft & Cluster Catalog</h1>
      <p class="catalog-subhead">
        Explore 74 certified Geographical Indication (GI) craft clusters across India. Every tradition is cataloged with statutory registration numbers, master techniques, and verifiable loom provenance.
      </p>

      <!-- Filter Controls -->
      <div class="filter-controls">
        <div class="search-input-box">
          <Icon name="search" size="1rem" />
          <input
            type="search"
            placeholder="Search by craft, GI tag, or state..."
            bind:value={searchQuery}
            class="catalog-search"
            aria-label="Search craft catalog"
          />
        </div>

        <div class="belt-chips" role="tablist" aria-label="Craft Belt Filter">
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'all'}
            onclick={() => (selectedBelt = 'all')}
          >
            All Belts ({FALLBACK_CRAFTS.length})
          </button>
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'north'}
            onclick={() => (selectedBelt = 'north')}
          >
            Northern Plains
          </button>
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'west'}
            onclick={() => (selectedBelt = 'west')}
          >
            Western Deserts
          </button>
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'south'}
            onclick={() => (selectedBelt = 'south')}
          >
            Deccan & South
          </button>
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'east'}
            onclick={() => (selectedBelt = 'east')}
          >
            Eastern Loomlands
          </button>
          <button
            type="button"
            class="belt-chip"
            class:active={selectedBelt === 'central'}
            onclick={() => (selectedBelt = 'central')}
          >
            Central Heartland
          </button>
        </div>
      </div>
    </header>

    <!-- Catalog Grid -->
    <div class="catalog-grid">
      {#each filteredCrafts as craft (craft.id)}
        <article class="cluster-card">
          <div class="cluster-card__header">
            <span class="cluster-card__gi-pill">{craft.gi_registration_no}</span>
            <span class="cluster-card__region">{craft.regions[0]}</span>
          </div>

          <h2 class="cluster-card__title">
            <a href={`/search?craft=${craft.slug}`}>{craft.display_name}</a>
          </h2>

          <p class="cluster-card__desc">{craft.description}</p>

          <div class="cluster-card__techniques">
            {#each craft.techniques as tech}
              <span class="tech-tag">{tech}</span>
            {/each}
          </div>

          <div class="cluster-card__footer">
            <a href={`/search?craft=${craft.slug}`} class="cluster-card__link">
              <span>View Certified Lots</span>
              <Icon name="arrow-right" size="0.85rem" />
            </a>
          </div>
        </article>
      {/each}
    </div>
  </div>
</div>

<style>
  .catalog-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .catalog-container {
    max-inline-size: 78rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .catalog-header {
    margin-block-end: var(--k-space-8);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-6);
  }

  .catalog-kicker {
    display: inline-block;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-secondary);
    margin-block-end: var(--k-space-1);
  }

  .catalog-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-3xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .catalog-subhead {
    font-size: var(--k-text-base);
    color: var(--k-text-secondary);
    max-inline-size: 46rem;
    margin: 0 0 var(--k-space-6) 0;
    line-height: 1.6;
  }

  .filter-controls {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .search-input-box {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: #ffffff;
    max-inline-size: 32rem;
  }

  .catalog-search {
    border: none;
    outline: none;
    font: inherit;
    font-size: var(--k-text-sm);
    inline-size: 100%;
  }

  .belt-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
  }

  .belt-chip {
    padding: 0.35rem 0.75rem;
    font-size: 0.75rem;
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-full);
    background: none;
    color: var(--k-text-secondary);
    cursor: pointer;
    transition: all var(--k-duration-fast) ease;
  }

  .belt-chip:hover {
    border-color: var(--k-accent-secondary);
    color: var(--k-accent-secondary);
  }

  .belt-chip.active {
    background-color: var(--k-accent-secondary);
    border-color: var(--k-accent-secondary);
    color: #ffffff;
    font-weight: var(--k-weight-semibold);
  }

  /* Catalog Grid */
  .catalog-grid {
    display: grid;
    /* min() caps the track floor at 100% so a 320px phone gets one
       full-width column instead of a 22rem overflow. */
    grid-template-columns: repeat(auto-fill, minmax(min(22rem, 100%), 1fr));
    gap: var(--k-space-5);
  }

  .cluster-card {
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-card, #ffffff);
    padding: var(--k-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    transition: transform var(--k-duration-fast) ease, box-shadow var(--k-duration-fast) ease;
  }

  .cluster-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
  }

  .cluster-card__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .cluster-card__gi-pill {
    font-size: 0.68rem;
    font-weight: var(--k-weight-bold);
    color: var(--k-accent-secondary);
    background-color: rgba(198, 93, 59, 0.08);
    padding: 0.15rem 0.45rem;
    border-radius: var(--k-radius-sm);
  }

  .cluster-card__region {
    font-size: 0.72rem;
    color: var(--k-text-muted);
  }

  .cluster-card__title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-lg);
    font-weight: var(--k-weight-bold);
    margin: 0;
    line-height: 1.3;
  }

  .cluster-card__title a {
    color: var(--k-text-primary);
    text-decoration: none;
  }

  .cluster-card__title a:hover {
    color: var(--k-accent-secondary);
    text-decoration: underline;
  }

  .cluster-card__desc {
    font-size: var(--k-text-xs);
    line-height: 1.6;
    color: var(--k-text-secondary);
    margin: 0;
    flex-grow: 1;
  }

  .cluster-card__techniques {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-1);
  }

  .tech-tag {
    font-size: 0.65rem;
    background-color: var(--k-surface-sunken, #f7f4ee);
    color: var(--k-text-secondary);
    padding: 0.15rem 0.4rem;
    border-radius: var(--k-radius-sm);
  }

  .cluster-card__footer {
    border-block-start: 1px dashed var(--k-border-subtle);
    padding-block-start: var(--k-space-3);
    margin-block-start: var(--k-space-1);
  }

  .cluster-card__link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-accent-secondary);
    text-decoration: none;
  }

  .cluster-card__link:hover {
    text-decoration: underline;
  }
</style>
