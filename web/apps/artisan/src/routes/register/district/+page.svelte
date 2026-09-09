<!-- apps/artisan/src/routes/register/district/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Input, Button } from '@kalakriti/ui';
  import { listen, listenSupported } from '@kalakriti/voice';
  import RegisterStep from '$lib/RegisterStep.svelte';
  import { getDraft, patchDraft } from '$lib/registration';
  import { DISTRICTS, matchesQuery } from '$lib/ontology';

  const t = $derived(locale.t);

  let selected = $state('');
  let freeText = $state('');
  let notListed = $state(false);
  let query = $state('');
  let listening = $state(false);

  $effect(() => {
    void getDraft().then((draft) => {
      selected = draft.districtId ?? '';
      freeText = draft.districtFreeText ?? '';
      notListed = freeText !== '';

      // If a district was previously selected, show it in the search box
      if (selected && !notListed) {
        const selectedDistrict = DISTRICTS.find((d) => d.id === selected);
        if (selectedDistrict) {
          query = `${selectedDistrict.name}, ${selectedDistrict.state}`;
        }
      }
    });
  });

  const filtered = $derived(
    DISTRICTS.filter((d) => matchesQuery(`${d.name} ${d.state}`, query)),
  );

  async function choose(id: string): Promise<void> {
    selected = id;
    notListed = false;
    freeText = '';
    await patchDraft({ districtId: id, districtFreeText: undefined });
    // Automatically proceed to next page after selection
    await goto('/register/pehchan');
  }

  function toggleNotListed(): void {
    notListed = true;
    selected = '';
    void patchDraft({ districtId: undefined });
  }

  function onFreeTextChange(): void {
    void patchDraft({ districtFreeText: freeText });
  }

  async function useVoice(): Promise<void> {
    listening = true;
    try {
      const { result } = listen({ tag: locale.meta.tag });
      const transcript = (await result).trim();
      if (notListed) {
        freeText = transcript;
        onFreeTextChange();
      } else {
        query = transcript;
      }
    } catch {
      // No mic permission, or recognition failed -- typed search still works.
    } finally {
      listening = false;
    }
  }

  const canProceed = $derived(selected !== '' || freeText.trim() !== '');

  async function next(): Promise<void> {
    if (!canProceed) return;
    await goto('/register/pehchan');
  }
</script>

<svelte:head>
  <title>{t('register.district.heading')} — {t('app.name')}</title>
</svelte:head>

<RegisterStep index={2} heading={t('register.district.heading')} backHref="/register/craft">
  {#snippet children()}
    {#if notListed}
      <Input
        bind:value={freeText}
        onchange={onFreeTextChange}
        placeholder={t('register.district.notListed.label')}
        aria-label={t('register.district.notListed.label')}
      />
      {#if listenSupported()}
        <button type="button" class="voice-alt" onclick={useVoice} disabled={listening}>
          {listening ? t('ui.voice.recording') : t('register.district.voice')}
        </button>
      {/if}
      <button type="button" class="text-link" onclick={() => (notListed = false)}>
        {t('action.back')}
      </button>
    {:else}
      <div class="district-search">
        <Input
          bind:value={query}
          type="search"
          placeholder={t('register.district.search')}
          aria-label={t('register.district.search')}
        />
        {#if listenSupported()}
          <button type="button" class="voice-alt" onclick={useVoice} disabled={listening}>
            {listening ? t('ui.voice.recording') : t('register.district.voice')}
          </button>
        {/if}
      </div>

      {#if filtered.length === 0}
        <p class="district-empty">{t('register.district.empty')}</p>
      {:else}
        <ul role="radiogroup" aria-label={t('register.district.heading')} class="district-list">
          {#each filtered as district (district.id)}
            <li>
              <button
                type="button"
                class="district-row"
                class:district-row--selected={selected === district.id}
                role="radio"
                aria-checked={selected === district.id}
                onclick={() => choose(district.id)}
              >
                <span>{district.name}</span>
                <span class="district-row__state">{district.state}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}

      <button type="button" class="text-link" onclick={toggleNotListed}>
        {t('register.district.notListed')}
      </button>
    {/if}
  {/snippet}
  {#snippet actions()}
    <Button size="xl" disabled={!canProceed} onclick={next}>{t('action.next')}</Button>
  {/snippet}
</RegisterStep>

<style>
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

  .district-search {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .district-empty {
    color: var(--k-text-secondary);
  }

  .district-list {
    display: flex;
    flex-direction: column;
    max-block-size: 18rem;
    overflow-y: auto;
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .district-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    inline-size: 100%;
    min-block-size: var(--k-touch-min);
    padding-block: var(--k-space-3);
    border: none;
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    background: none;
    color: var(--k-text-primary);
    text-align: start;
    cursor: pointer;
  }

  .district-row--selected {
    font-weight: var(--k-weight-semibold);
    background-color: var(--k-surface-sunken);
  }

  .district-row__state {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .text-link {
    align-self: start;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    padding-block: var(--k-space-2);
    border: none;
    border-radius: var(--k-radius-md);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    cursor: pointer;
    font-weight: 600;
  }
</style>
