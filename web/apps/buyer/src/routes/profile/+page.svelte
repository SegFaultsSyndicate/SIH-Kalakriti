<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { locale } from '@kalakriti/i18n';

  const t = $derived(locale.t);

  onMount(() => {
    void goto('/account', { replaceState: true });
  });
</script>

<div style="padding: 2rem; text-align: center;">
  <p>{t('profile.redirecting')}</p>
</div>
