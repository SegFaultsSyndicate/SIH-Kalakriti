<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  onMount(() => {
    void goto('/account', { replaceState: true });
  });
</script>

<div style="padding: 2rem; text-align: center;">
  <p>Redirecting to your account dashboard...</p>
</div>
