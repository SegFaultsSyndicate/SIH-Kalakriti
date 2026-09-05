<!--
  packages/ui/src/Accordion.svelte

    <Accordion items={[{ id: 'materials', heading: 'Materials used' }, ...]}>
      {#snippet children(id)}
        {#if id === 'materials'}...{/if}
      {/snippet}
    </Accordion>

    <Accordion type="multiple" items={sections} bind:open>...</Accordion>

  `type="single"` (default): opening one closes any other, like a native
  <details> group. `type="multiple"`: independent. Arrow Up/Down move focus
  between headers (Home/End to the ends); Enter/Space toggle is native
  <button> behaviour, not reimplemented.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Icon } from '@kalakriti/icons';

  interface Item {
    id: string;
    heading: string;
  }

  interface Props {
    items: Item[];
    type?: 'single' | 'multiple';
    open?: string[];
    class?: string;
    children: Snippet<[string]>;
  }

  let { items, type = 'single', open = $bindable([]), class: className, children }: Props =
    $props();

  const uid = $props.id();
  const headingId = (id: string) => `${uid}-heading-${id}`;
  const panelId = (id: string) => `${uid}-panel-${id}`;

  let headerRefs: Record<string, HTMLButtonElement> = {};

  function isOpen(id: string): boolean {
    return open.includes(id);
  }

  function toggle(id: string): void {
    if (isOpen(id)) {
      open = open.filter((openId) => openId !== id);
    } else {
      open = type === 'single' ? [id] : [...open, id];
    }
  }

  function onkeydown(event: KeyboardEvent, index: number): void {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      headerRefs[items[(index + 1) % items.length].id]?.focus();
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      headerRefs[items[(index - 1 + items.length) % items.length].id]?.focus();
    } else if (event.key === 'Home') {
      event.preventDefault();
      headerRefs[items[0].id]?.focus();
    } else if (event.key === 'End') {
      event.preventDefault();
      headerRefs[items[items.length - 1].id]?.focus();
    }
  }
</script>

<div class="k-accordion {className || ''}">
  {#each items as item, index (item.id)}
    <div class="k-accordion__item">
      <h3 class="k-accordion__heading">
        <button
          bind:this={headerRefs[item.id]}
          type="button"
          class="k-accordion__trigger"
          id={headingId(item.id)}
          aria-expanded={isOpen(item.id)}
          aria-controls={panelId(item.id)}
          onclick={() => toggle(item.id)}
          onkeydown={(event) => onkeydown(event, index)}
        >
          <Icon name="chevron-right" class="k-accordion__chevron" />
          {item.heading}
        </button>
      </h3>
      <div
        class="k-accordion__panel"
        id={panelId(item.id)}
        role="region"
        aria-labelledby={headingId(item.id)}
        hidden={!isOpen(item.id)}
      >
        {#if isOpen(item.id)}
          {@render children(item.id)}
        {/if}
      </div>
    </div>
  {/each}
</div>
