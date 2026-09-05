<!-- apps/artisan/src/routes/register/craft/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Input, Button } from '@kalakriti/ui';
  import { listen, listenSupported } from '@kalakriti/voice';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft } from '$lib/registration';
  import { CRAFTS, matchesQuery } from '$lib/ontology';

  const t = $derived(locale.t);

  let selected = $state('');
  let query = $state('');
  let listening = $state(false);

  $effect(() => {
    void getDraft().then((draft) => {
      selected = draft.craftId ?? '';
    });
  });

  const filtered = $derived(CRAFTS.filter((craft) => matchesQuery(t(craft.nameKey), query)));

  function choose(id: string): void {
    selected = id;
    void patchDraft({ craftId: id });
  }

  async function useVoice(): Promise<void> {
    listening = true;
    try {
      const { result } = listen({ tag: locale.meta.tag });
      query = (await result).trim();
    } catch {
      // No mic permission, or recognition failed -- typed search still works.
    } finally {
      listening = false;
    }
  }

  async function next(): Promise<void> {
    if (selected === '') return;
    await goto('/register/district');
  }
</script>

<svelte:head>
  <title>{t('register.craft.heading')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep index={1} heading={t('register.craft.heading')} backHref="/register/name">
  {#snippet children()}
    <div class="craft-search">
      <Input
        bind:value={query}
        type="search"
        placeholder={t('register.craft.search')}
        aria-label={t('register.craft.search')}
      />
      {#if listenSupported()}
        <button type="button" class="voice-alt" onclick={useVoice} disabled={listening}>
          {listening ? t('ui.voice.recording') : t('register.craft.voice')}
        </button>
      {/if}
    </div>

    {#if filtered.length === 0}
      <p class="craft-empty">{t('register.craft.empty')}</p>
    {:else}
      <div class="craft-grid" role="radiogroup" aria-label={t('register.craft.heading')}>
        {#each filtered as craft (craft.id)}
          <button
            type="button"
            class="craft-tile"
            class:craft-tile--selected={selected === craft.id}
            role="radio"
            aria-checked={selected === craft.id}
            onclick={() => choose(craft.id)}
          >
            <Icon name={craft.icon} class="craft-tile__icon" />
            <span>{t(craft.nameKey)}</span>
          </button>
        {/each}
      </div>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={selected === ''} onclick={next}>{t('action.next')}</Button>
  {/snippet}
</RegisterStep>

<style>
  .craft-search {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .voice-alt {
    align-self: start;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: none;
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .craft-empty {
    color: var(--k-text-secondary);
  }

  .craft-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-3);
  }

  .craft-tile {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    min-block-size: calc(var(--k-touch-min) * 1.6);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    text-align: center;
    cursor: pointer;
  }

  .craft-tile--selected {
    border-color: var(--k-accent-primary-bg);
    border-width: var(--k-rule);
    background-color: var(--k-surface-sunken);
  }

  .craft-tile :global(.craft-tile__icon) {
    inline-size: 1.75rem;
    block-size: 1.75rem;
  }
</style>
