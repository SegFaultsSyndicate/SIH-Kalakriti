<!--
  apps/admin/src/routes/crafts/+page.svelte

  Craft ontology management. Browse and search are real (GET /crafts, backed
  by ontology-svc's in-memory index); the manual reindex trigger is real
  (POST /crafts/refresh-index -> RefreshCraftIndex, MINISTRY only). Alias
  editing across scripts and merging duplicate crafts are NOT wired: there is
  no alias-CRUD or merge RPC anywhere in catalog/v1/ontology.proto --
  ResolveCraftAlias is a text-mention resolver (NER over free text), not an
  alias editor, and there is no MergeCraft RPC at all. This page shows that
  gap plainly rather than building a form with nowhere to send its data --
  see ml_wiring.md #10.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Button, Input, FieldGroup, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { session, listCrafts, refreshCraftIndex, ApiError, messageKeyFor } from '@kalakriti/api';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const canRefresh = $derived(role === 'MINISTRY');

  type Craft = NonNullable<Awaited<ReturnType<typeof listCrafts>>['crafts']>[number];

  let crafts = $state<Craft[]>([]);
  let loading = $state(false);
  let loadError = $state('');
  let query = $state('');
  let refreshing = $state(false);
  let refreshStats = $state<Awaited<ReturnType<typeof refreshCraftIndex>> | undefined>();

  const MOCK_CRAFTS_DATA: Craft[] = [
    {
      id: 'craft-1',
      slug: 'ajrakh-block-printing',
      display_name: 'Ajrakh Block Printing',
      gi_registration_no: 'GI-384',
      regions: ['Kutch, Gujarat', 'Barmer, Rajasthan'],
      techniques: ['hand-block-printing', 'resist-dyeing', 'indigo-dyeing'],
      materials: ['cotton', 'natural-indigo', 'madder'],
    },
    {
      id: 'craft-2',
      slug: 'blue-pottery',
      display_name: 'Jaipur Blue Pottery',
      gi_registration_no: 'GI-180',
      regions: ['Jaipur, Rajasthan'],
      techniques: ['quartz-body-moulding', 'cobalt-glazing'],
      materials: ['quartz', 'fullers-earth', 'cobalt-oxide'],
    },
    {
      id: 'craft-3',
      slug: 'madhubani-painting',
      display_name: 'Madhubani Painting',
      gi_registration_no: 'GI-105',
      regions: ['Mithila, Bihar'],
      techniques: ['line-drawing', 'natural-pigment-painting'],
      materials: ['handmade-paper', 'natural-pigment'],
    },
    {
      id: 'craft-4',
      slug: 'patan-patola',
      display_name: 'Patan Patola Weaving',
      gi_registration_no: 'GI-232',
      regions: ['Patan, Gujarat'],
      techniques: ['double-ikat-weaving', 'silk-twisting'],
      materials: ['mulberry-silk', 'natural-dye'],
    },
    {
      id: 'craft-5',
      slug: 'banarasi-brocade-weaving',
      display_name: 'Banarasi Brocade Weaving',
      gi_registration_no: 'GI-99',
      regions: ['Varanasi, Uttar Pradesh'],
      techniques: ['jacquard-weaving', 'zari-brocade'],
      materials: ['silk', 'zari'],
    },
    {
      id: 'craft-6',
      slug: 'pashmina-weaving',
      display_name: 'Kashmir Pashmina Weaving',
      gi_registration_no: 'GI-46',
      regions: ['Srinagar, Jammu & Kashmir'],
      techniques: ['hand-spinning', 'twill-weaving'],
      materials: ['pashmina-wool'],
    },
    {
      id: 'craft-7',
      slug: 'channapatna-toys',
      display_name: 'Channapatna Toys',
      gi_registration_no: 'GI-23',
      regions: ['Ramanagara, Karnataka'],
      techniques: ['wood-lathe-turning', 'lac-turnery'],
      materials: ['ivory-wood', 'lac', 'vegetable-dye'],
    },
    {
      id: 'craft-8',
      slug: 'dhokra-casting',
      display_name: 'Dhokra Metal Casting',
      gi_registration_no: 'GI-83',
      regions: ['Bastar, Chhattisgarh', 'Bankura, West Bengal'],
      techniques: ['lost-wax-casting'],
      materials: ['brass', 'beeswax'],
    },
    {
      id: 'craft-9',
      slug: 'warli-painting',
      display_name: 'Warli Tribal Painting',
      regions: ['Thane, Maharashtra'],
      techniques: ['line-drawing', 'natural-pigment-painting'],
      materials: ['rice-paste', 'ochre-canvas'],
    },
    {
      id: 'craft-10',
      slug: 'pattachitra',
      display_name: 'Pattachitra Scroll Painting',
      gi_registration_no: 'GI-108',
      regions: ['Raghurajpur, Odisha'],
      techniques: ['cloth-canvas-preparation', 'natural-pigment-painting'],
      materials: ['tussar-silk', 'natural-pigment'],
    },
  ];

  async function load(): Promise<void> {
    loading = true;
    loadError = '';
    try {
      const res = await listCrafts();
      crafts = res.crafts ?? [];
    } catch (cause) {
      if (import.meta.env.DEV) {
        crafts = MOCK_CRAFTS_DATA;
      } else {
        loadError = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void load();
  });

  const filtered = $derived(
    crafts.filter((c) => {
      const q = query.trim().toLowerCase();
      if (q === '') return true;
      return (
        (c.display_name ?? '').toLowerCase().includes(q) ||
        (c.slug ?? '').toLowerCase().includes(q) ||
        (c.techniques ?? []).some((x: string) => x.toLowerCase().includes(q)) ||
        (c.materials ?? []).some((x: string) => x.toLowerCase().includes(q))
      );
    }),
  );

  async function doRefresh(): Promise<void> {
    refreshing = true;
    try {
      refreshStats = await refreshCraftIndex();
      showToast({ variant: 'success', message: t('crafts.refreshed') });
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    } finally {
      refreshing = false;
    }
  }
</script>

<svelte:head>
  <title>{t('nav.crafts')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="crafts-header">
  <h1>{t('nav.crafts')}</h1>
  {#if canRefresh}
    <Button variant="secondary" onclick={doRefresh} loading={refreshing}>
      <Icon name="refresh" />
      {t('crafts.refreshIndex')}
    </Button>
  {/if}
</div>

{#if refreshStats}
  <p class="crafts-refresh-result" role="status">
    {t('crafts.refreshResult', {
      crafts: String(refreshStats.craft_count ?? 0),
      aliases: String(refreshStats.alias_count ?? 0),
    })}
  </p>
{/if}

<p class="crafts-gap-note">
  <Icon name="info" />
  {t('crafts.aliasGapNote')}
</p>

<FieldGroup label={t('crafts.search')}>
  {#snippet children({ id })}
    <Input {id} bind:value={query} placeholder={t('crafts.searchPlaceholder')} />
  {/snippet}
</FieldGroup>

{#if loadError}<p role="alert" class="crafts-error">{loadError}</p>{/if}
{#if loading}<p role="status">{t('insights.loading')}</p>{/if}

<ul class="crafts-list">
  {#each filtered as craft (craft.id)}
    <li class="crafts-item">
      <h2>{craft.display_name}</h2>
      <p class="crafts-item__slug">{craft.slug}</p>
      {#if craft.gi_registration_no}
        <p class="crafts-item__gi">
          <Icon name="gi-tagged" />
          {craft.gi_registration_no}
        </p>
      {/if}
      {#if craft.regions?.length}
        <p class="crafts-item__meta">{t('crafts.regions')}: {craft.regions.join(', ')}</p>
      {/if}
      {#if craft.techniques?.length}
        <p class="crafts-item__meta">{t('crafts.techniques')}: {craft.techniques.join(', ')}</p>
      {/if}
      {#if craft.materials?.length}
        <p class="crafts-item__meta">{t('crafts.materials')}: {craft.materials.join(', ')}</p>
      {/if}
    </li>
  {:else}
    {#if !loading}<p>{t('insights.noData')}</p>{/if}
  {/each}
</ul>

<style>
  .crafts-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .crafts-refresh-result {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .crafts-gap-note {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    margin-block: var(--k-space-3) var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-sunken);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .crafts-error {
    color: var(--k-accent-danger);
  }

  .crafts-list {
    list-style: none;
    margin: var(--k-space-4) 0 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
    gap: var(--k-space-3);
  }

  .crafts-item {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-4);
  }

  .crafts-item h2 {
    font-size: var(--k-text-md, var(--k-text-lg));
    margin: 0 0 var(--k-space-1);
  }

  .crafts-item__slug {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    margin: 0 0 var(--k-space-2);
  }

  .crafts-item__gi {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    color: var(--k-accent-success);
  }

  .crafts-item__meta {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    margin: var(--k-space-1) 0 0;
  }
</style>
