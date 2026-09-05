<!--
  apps/artisan/src/routes/+error.svelte

  SvelteKit's routing-error page: a bad URL, a load() throw, or a 5xx from
  the BFF. Distinct from ErrorBoundary (packages/observability) -- that one
  catches a render/effect crash inside an already-matched route; this one
  catches everything the router itself failed to resolve.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { EmptyState, Button } from '@kalakriti/ui';

  const t = $derived(locale.t);
  const isNotFound = $derived(page.status === 404);
</script>

<svelte:head>
  <title>{isNotFound ? t('error.404.heading') : t('error.500.heading')} — {t('app.name')}</title>
</svelte:head>

<EmptyState
  illustration="empty-error"
  heading={isNotFound ? t('error.404.heading') : t('error.500.heading')}
  body={isNotFound ? t('error.404.body') : t('error.500.body')}
>
  {#snippet action()}
    <Button onclick={() => goto('/')}>
      {isNotFound ? t('error.404.action') : t('error.500.action')}
    </Button>
  {/snippet}
</EmptyState>
