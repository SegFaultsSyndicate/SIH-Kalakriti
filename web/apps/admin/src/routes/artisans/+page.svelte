<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Button } from '@kalakriti/ui';

  const t = $derived(locale.t);
  let artisanId = $state('');

  function handleSubmit() {
    if (artisanId.trim()) {
      goto(`/artisans/${encodeURIComponent(artisanId.trim())}`);
    }
  }
</script>

<svelte:head>
  <title>Artisans — Admin</title>
</svelte:head>

<div class="admin-artisans-page">
  <header>
    <h1>{t('nav.badges')} — Artisan Recognition</h1>
    <p class="admin-desc">Manage conferred recognition badges and inspect artisan achievements.</p>
  </header>

  <form class="admin-search-box" onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <label for="artisan-id-input">Enter Artisan ID:</label>
    <div class="admin-input-row">
      <input
        id="artisan-id-input"
        type="text"
        placeholder="e.g. 018d9f10-..."
        bind:value={artisanId}
        class="admin-text-input"
      />
      <Button type="submit" variant="primary" disabled={!artisanId.trim()}>
        Manage Badges
      </Button>
    </div>
  </form>
</div>

<style>
  .admin-artisans-page {
    padding: var(--k-space-4);
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .admin-desc {
    color: var(--k-text-muted);
    font-size: var(--k-text-sm);
  }

  .admin-search-box {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    background-color: var(--k-surface);
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .admin-input-row {
    display: flex;
    gap: var(--k-space-2);
  }

  .admin-text-input {
    flex: 1;
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    font-size: var(--k-text-sm);
  }
</style>
