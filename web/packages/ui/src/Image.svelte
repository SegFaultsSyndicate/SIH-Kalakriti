<!--
  packages/ui/src/Image.svelte

    <Image src={photo.url} srcset={photo.srcset} alt="Handwoven Kanchipuram silk sari" ratio="4/3" />
    <Image src={photo.url} alt="" ratio="1/1" placeholder={photo.blurDataUrl} />
    <Image src={hero.url} srcset={hero.srcset} alt="..." ratio="4/3" eager />

  `ratio` reserves the box before the image decodes, so nothing shifts
  around it (CLS). `placeholder` is a tiny low-res image (already resized
  server-side, not generated here) shown until the real one has loaded,
  then swapped by a plain opacity crossfade. A failed load shows a
  purpose-built panel with a word, never the browser's broken-image icon --
  that icon means nothing to an artisan and looks like the product is
  broken rather than one photo.

  `eager` is for the one LCP image on a screen (a listing's hero photo) --
  every other <Image> stays loading="lazy" by default, which is what "lazy
  below the fold" means in practice with no viewport math required. On
  Save-Data or a 2G effectiveType, `lowSrc` (if given) is served instead of
  `src`/`srcset` -- a genuinely smaller asset, not the same image relabelled.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { shouldConserveData } from '@kalakriti/offline';

  interface Props {
    src: string;
    /** Required. Pass "" only for a genuinely decorative image. */
    alt: string;
    srcset?: string;
    sizes?: string;
    /** e.g. "4/3", "1/1", "16/9". Omit only when width/height attrs are supplied instead. */
    ratio?: string;
    /** A tiny low-res stand-in shown until `src` finishes loading. */
    placeholder?: string;
    /** A smaller asset to serve under Save-Data / a 2G effectiveType, instead of `src`. */
    lowSrc?: string;
    /** Mark the LCP image on the screen: loading="eager" + fetchpriority="high" instead of lazy. */
    eager?: boolean;
    class?: string;
    [key: string]: unknown;
  }

  let {
    src,
    alt,
    srcset,
    sizes,
    ratio,
    placeholder,
    lowSrc,
    eager = false,
    class: className,
    ...rest
  }: Props = $props();

  const t = $derived(locale.t);
  let loaded = $state(false);
  let failed = $state(false);

  const conserveData = shouldConserveData();
  const resolvedSrc = $derived(conserveData && lowSrc ? lowSrc : src);
  const resolvedSrcset = $derived(conserveData && lowSrc ? undefined : srcset);
</script>

<div
  class="k-image {className || ''}"
  class:k-image--failed={failed}
  style={ratio ? `aspect-ratio:${ratio}` : undefined}
>
  {#if !failed}
    {#if placeholder && !loaded}
      <img class="k-image__placeholder" src={placeholder} alt="" aria-hidden="true" />
    {/if}
    <img
      class="k-image__real"
      class:k-image__real--loaded={loaded}
      src={resolvedSrc}
      srcset={resolvedSrcset}
      {sizes}
      {alt}
      loading={eager ? 'eager' : 'lazy'}
      fetchpriority={eager ? 'high' : undefined}
      decoding="async"
      onload={() => (loaded = true)}
      onerror={() => (failed = true)}
      {...rest}
    />
  {:else}
    <div class="k-image__fallback" role="img" aria-label={alt || t('ui.image.failed')}>
      <Icon name="image" />
      <span>{t('ui.image.failed')}</span>
    </div>
  {/if}
</div>
