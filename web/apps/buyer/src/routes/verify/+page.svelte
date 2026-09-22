<!--
  apps/buyer/src/routes/verify/+page.svelte

  Bare /verify (footer, account menu, account order rail) had no page --
  only /verify/[code] existed. Reuses the home page's lookup terminal, which
  already links on to /verify/[code] for a found record.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import ProvenanceTerminal from '$lib/ProvenanceTerminal.svelte';

  const t = $derived(locale.t);
</script>

<svelte:head>
  <title>{t('footer.col.corridors.provenance')} — {t('app.name')}</title>
</svelte:head>

<ProvenanceTerminal />
