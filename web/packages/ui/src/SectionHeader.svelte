<!--
  packages/ui/src/SectionHeader.svelte

    <SectionHeader kicker={t('buyer.home.dupattaKicker')} heading={t('buyer.home.dupattaHeading')} href="/c/dupattas" />

  The editorial rhythm this design law requires everywhere a buyer page
  lists more than it shows: a small lowercase kicker, a large heading, and
  "View all" aligned to the far end -- never a centered block. `href`
  is optional; a section with nowhere further to go simply omits it and
  the heading runs full width.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    kicker?: string;
    heading: string;
    href?: string;
    /** Heading level, matched to where this sits in the page outline. */
    level?: 2 | 3 | 4;
    class?: string;
  }

  let { kicker, heading, href, level = 2, class: className }: Props = $props();

  const t = $derived(locale.t);
</script>

<div class="k-section-header {className || ''}">
  <div class="k-section-header__text">
    {#if kicker}
      <p class="k-section-header__kicker">{kicker}</p>
    {/if}
    <svelte:element this={`h${level}`} class="k-section-header__heading">{heading}</svelte:element>
  </div>
  {#if href}
    <a class="k-section-header__view-all" {href}>
      {t('ui.sectionHeader.viewAll')}
      <Icon name="arrow-right" />
    </a>
  {/if}
</div>
