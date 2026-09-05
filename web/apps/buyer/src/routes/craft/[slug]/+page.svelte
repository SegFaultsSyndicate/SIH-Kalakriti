<!--
  apps/buyer/src/routes/craft/[slug]/+page.svelte

  Craft landing page: what the craft is, its region, its GI status, and the
  artisans who practise it -- the education layer that justifies the price
  premium the brief asks for. "The process" beyond techniques/materials has
  no backend field (ontology.proto's Craft carries no free-text process
  description) -- documented in ml_wiring.md rather than invented here.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { EmptyState, Skeleton } from '@kalakriti/ui';
  import { ProcessSequence } from '@kalakriti/illustrations';
  import { Icon } from '@kalakriti/icons';
  import { getCraft, type components } from '@kalakriti/api';
  import { craftIcon } from '$lib/craft-icon';

  type CraftDetail = components['schemas']['CraftDetail'];

  const t = $derived(locale.t);
  const slug = $derived(page.params.slug ?? '');

  let loading = $state(true);
  let craft = $state<CraftDetail | undefined>(undefined);

  $effect(() => {
    void (async () => {
      loading = true;
      try {
        craft = await getCraft(slug);
      } catch {
        craft = undefined;
      } finally {
        loading = false;
      }
    })();
  });
</script>

<svelte:head>
  <title>{craft?.display_name ?? t('search.heading')} — {t('app.name')}</title>
</svelte:head>

{#if loading}
  <Skeleton shape="card" height="20rem" />
{:else if !craft}
  <EmptyState illustration="empty-error" heading={t('craft.notFound')} />
{:else}
  <header class="craft-header">
    <span class="craft-header__icon"><Icon name={craftIcon(craft.slug ?? craft.display_name ?? '')} /></span>
    <div>
      <h1>{craft.display_name}</h1>
      {#if craft.gi_certified}
        <p class="craft-header__gi">
          <Icon name="gi-tagged" />
          {t('craft.giCertified')}
          {#if craft.gi_registration_no}<span>· {t('craft.giRegistrationNo', { number: craft.gi_registration_no })}</span>{/if}
        </p>
      {/if}
    </div>
  </header>

  <ProcessSequence
    craft={craft.slug === 'pottery'
      ? 'pottery'
      : craft.slug === 'block-printing'
        ? 'blockprint'
        : 'weaving'}
  />

  <div class="craft-facts">
    {#if craft.regions && craft.regions.length > 0}
      <section>
        <h2>{t('craft.regions.heading')}</h2>
        <p>{craft.regions.join(', ')}</p>
      </section>
    {/if}
    {#if craft.techniques && craft.techniques.length > 0}
      <section>
        <h2>{t('craft.techniques.heading')}</h2>
        <ul role="list">{#each craft.techniques as technique (technique)}<li>{technique}</li>{/each}</ul>
      </section>
    {/if}
    {#if craft.materials && craft.materials.length > 0}
      <section>
        <h2>{t('craft.materials.heading')}</h2>
        <ul role="list">{#each craft.materials as material (material)}<li>{material}</li>{/each}</ul>
      </section>
    {/if}
  </div>

  <section class="craft-artisans">
    <h2>{t('craft.artisans.heading')}</h2>
    {#if !craft.artisans || craft.artisans.length === 0}
      <p>{t('craft.artisans.empty')}</p>
    {:else}
      <ul class="craft-artisans__list" role="list">
        {#each craft.artisans as artisan (artisan.id)}
          <li>
            <a href={`/artisan/${artisan.id}`}>
              <Icon name="user" />
              {artisan.display_name}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}

<style>
  .craft-header {
    display: flex;
    align-items: center;
    gap: var(--k-space-4);
    margin-block-end: var(--k-space-6);
  }

  .craft-header__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 4rem;
    block-size: 4rem;
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-sunken);
    flex: none;
  }

  .craft-header__icon :global(svg) {
    inline-size: 2rem;
    block-size: 2rem;
  }

  .craft-header__gi {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-block-start: var(--k-space-1);
  }

  .craft-facts {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--k-space-5);
    padding-block: var(--k-space-5);
    border-block: var(--k-hairline) solid var(--k-border-hairline);
  }

  .craft-facts h2 {
    font-size: var(--k-text-sm);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .craft-artisans {
    margin-block-start: var(--k-space-6);
  }

  .craft-artisans__list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
  }

  .craft-artisans__list a {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    color: inherit;
    text-decoration: none;
  }
</style>
