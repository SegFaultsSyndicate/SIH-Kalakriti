<!--
  packages/ui/src/Tabs.svelte

    <Tabs
      tabs={[{ id: 'photos', label: 'Photographs' }, { id: 'details', label: 'Details' }]}
      bind:selected
    >
      {#snippet children(activeId)}
        {#if activeId === 'photos'}...{/if}
      {/snippet}
    </Tabs>

  WAI-ARIA APG tabs pattern, automatic activation: arrow keys move focus AND
  select in one step (Home/End jump to the ends), which is the model most
  screen-reader users expect and the simpler of the two APG allows. Roving
  tabindex -- only the selected tab is in the Tab order -- so a keyboard user
  reaches the tablist once and moves within it with arrows, not Tab.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface TabDef {
    id: string;
    label: string;
  }

  interface Props {
    tabs: TabDef[];
    selected?: string;
    class?: string;
    children: Snippet<[string]>;
  }

  let { tabs, selected = $bindable(tabs[0]?.id ?? ''), class: className, children }: Props =
    $props();

  const uid = $props.id();
  const tabId = (id: string) => `${uid}-tab-${id}`;
  const panelId = (id: string) => `${uid}-panel-${id}`;

  let tabRefs: Record<string, HTMLButtonElement> = {};

  function select(id: string, focus = false): void {
    selected = id;
    if (focus) tabRefs[id]?.focus();
  }

  function onkeydown(event: KeyboardEvent): void {
    const index = tabs.findIndex((tab) => tab.id === selected);
    if (index === -1) return;

    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      event.preventDefault();
      select(tabs[(index + 1) % tabs.length].id, true);
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      event.preventDefault();
      select(tabs[(index - 1 + tabs.length) % tabs.length].id, true);
    } else if (event.key === 'Home') {
      event.preventDefault();
      select(tabs[0].id, true);
    } else if (event.key === 'End') {
      event.preventDefault();
      select(tabs[tabs.length - 1].id, true);
    }
  }
</script>

<div class="k-tabs {className || ''}">
  <div class="k-tabs__list" role="tablist" tabindex="-1" {onkeydown}>
    {#each tabs as tab (tab.id)}
      <button
        bind:this={tabRefs[tab.id]}
        type="button"
        role="tab"
        class="k-tabs__tab"
        id={tabId(tab.id)}
        aria-selected={tab.id === selected}
        aria-controls={panelId(tab.id)}
        tabindex={tab.id === selected ? 0 : -1}
        onclick={() => select(tab.id)}
      >
        {tab.label}
      </button>
    {/each}
  </div>
  {#each tabs as tab (tab.id)}
    <div
      class="k-tabs__panel"
      role="tabpanel"
      id={panelId(tab.id)}
      aria-labelledby={tabId(tab.id)}
      hidden={tab.id !== selected}
      tabindex="0"
    >
      {#if tab.id === selected}
        {@render children(tab.id)}
      {/if}
    </div>
  {/each}
</div>
