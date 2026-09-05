<!--
  packages/ui/src/SkipLink.svelte

  The first focusable element on every page in all three apps. Required by
  GIGW 3.0 and by the keyboard contract; shared rather than reimplemented per
  app so no layout can accidentally ship without one.

  Its appearance lives in @kalakriti/tokens/base (.k-skip-link) on purpose:
  the rule that it must become visible on focus is not a component's to
  override.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';

  interface Props {
    /** id of the <main> it jumps to. Must exist, or the link is a lie. */
    target?: string;
  }

  let { target = 'main-content' }: Props = $props();

  const t = $derived(locale.t);
</script>

<a class="k-skip-link" href="#{target}">{t('a11y.skipToContent')}</a>
