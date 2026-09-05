<!-- apps/admin/src/routes/accessibility/+page.svelte -->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { AccessibilityStatement } from '@kalakriti/ui';

  const t = $derived(locale.t);
</script>

<svelte:head>
  <title>{t('a11y.statement.title')} — {t('admin.home.title')}</title>
</svelte:head>

<AccessibilityStatement />
