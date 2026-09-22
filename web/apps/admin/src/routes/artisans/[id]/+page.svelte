<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Button, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { session, grantBadge, revokeBadge, type ArtisanBadge } from '@kalakriti/api';
  import type { PageData } from './$types';

  interface Props { data: PageData }
  let { data }: Props = $props();

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY');

  let artisanBadges = $state<ArtisanBadge[]>(data.artisanBadges);
  const conferredCatalog = $derived(data.badgeCatalog.filter((b) => b.kind === 'CONFERRED'));
  const grantedIds = $derived(new Set(artisanBadges.map((g) => g.badge.id)));
  const availableBadges = $derived(conferredCatalog.filter((b) => !grantedIds.has(b.id)));

  let selectedBadgeId = $state('');
  let revokeReason = $state('');
  let activeRevokeCode = $state('');
  let submitting = $state(false);

  async function handleGrant() {
    if (!selectedBadgeId) return;
    submitting = true;
    try {
      const res = await grantBadge(data.artisanId, { badge_id: selectedBadgeId });
      artisanBadges = [...artisanBadges, res];
      selectedBadgeId = '';
      showToast({ variant: 'success', message: 'Badge granted successfully' });
    } catch (err) {
      showToast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to grant badge' });
    } finally {
      submitting = false;
    }
  }

  async function handleRevoke(code: string) {
    if (!revokeReason.trim()) {
      showToast({ variant: 'error', message: 'Revocation reason is required' });
      return;
    }
    submitting = true;
    try {
      await revokeBadge(data.artisanId, code, { reason: revokeReason });
      artisanBadges = artisanBadges.filter((g) => g.badge.code !== code);
      revokeReason = '';
      activeRevokeCode = '';
      showToast({ variant: 'success', message: 'Badge revoked' });
    } catch (err) {
      showToast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to revoke badge' });
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>{t('badges.title')} — Admin</title>
</svelte:head>

<div class="admin-badges-layout">
  <header class="admin-badges-header">
    <a href="/artisans" class="admin-back-link">
      <Icon name="arrow-left" size="1rem" />
      <span>Back to Artisans</span>
    </a>
    <h1>Artisan Badges: {data.artisanId}</h1>
    <p class="admin-badges-subtitle">Manage conferred recognition badges for this artisan.</p>
  </header>

  {#if !authorized}
    <div class="admin-unauthorized-banner">
      <Icon name="warning" size="1.25rem" />
      <p>Only MINISTRY officers have permission to grant or revoke conferred badges.</p>
    </div>
  {:else}
    <section class="admin-badges-panel">
      <h2>Active Badges ({artisanBadges.length})</h2>
      {#if artisanBadges.length === 0}
        <p class="admin-empty-text">No badges currently granted to this artisan.</p>
      {:else}
        <ul class="admin-badge-list">
          {#each artisanBadges as grant (grant.badge.id)}
            <li class="admin-badge-item">
              <div class="admin-badge-item__info">
                <span class="admin-badge-item__code">{grant.badge.code}</span>
                <span class="admin-badge-item__name">{t(`badge.${grant.badge.code}.name` as MessageKey)}</span>
                <span class="admin-badge-item__meta">Granted on {grant.granted_at} by {grant.granted_by}</span>
              </div>
              {#if grant.badge.kind === 'CONFERRED'}
                {#if activeRevokeCode === grant.badge.code}
                  <div class="admin-revoke-form">
                    <input
                      type="text"
                      placeholder="Reason for revocation..."
                      bind:value={revokeReason}
                      class="admin-revoke-input"
                    />
                    <Button
                      variant="danger"
                      size="sm"
                      disabled={submitting || !revokeReason.trim()}
                      onclick={() => handleRevoke(grant.badge.code)}
                    >
                      Confirm Revoke
                    </Button>
                    <Button variant="ghost" size="sm" onclick={() => { activeRevokeCode = ''; revokeReason = ''; }}>
                      Cancel
                    </Button>
                  </div>
                {:else}
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={submitting}
                    onclick={() => { activeRevokeCode = grant.badge.code; revokeReason = ''; }}
                  >
                    {t('common.revoke')}
                  </Button>
                {/if}
              {:else}
                <span class="admin-badge-item__earned-tag">Earned via Activity</span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="admin-badges-panel">
      <h2>Grant Conferred Badge</h2>
      {#if availableBadges.length === 0}
        <p class="admin-empty-text">All available conferred badges have already been granted.</p>
      {:else}
        <form class="admin-grant-form" onsubmit={(e) => { e.preventDefault(); handleGrant(); }}>
          <select bind:value={selectedBadgeId} class="admin-select">
            <option value="">-- {t('common.selectBadge')} --</option>
            {#each availableBadges as b (b.id)}
              <option value={b.id}>{b.code} ({t(`badge.${b.code}.name` as MessageKey)})</option>
            {/each}
          </select>
          <Button type="submit" variant="primary" disabled={submitting || !selectedBadgeId}>
            {t('common.grant')}
          </Button>
        </form>
      {/if}
    </section>
  {/if}
</div>

<style>
  .admin-badges-layout {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    max-width: 900px;
    margin: 0 auto;
    padding: var(--k-space-4);
  }

  .admin-badges-header {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .admin-back-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-muted);
    font-size: var(--k-text-sm);
    text-decoration: none;
  }

  .admin-badges-subtitle {
    color: var(--k-text-muted);
    font-size: var(--k-text-sm);
  }

  .admin-unauthorized-banner {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface-muted, #fdf6e2);
    border-radius: var(--k-radius-sm, 4px);
  }

  .admin-badges-panel {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    background-color: var(--k-surface);
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .admin-badges-panel h2 {
    font-size: var(--k-text-base);
    font-weight: 600;
    margin: 0;
  }

  .admin-empty-text {
    color: var(--k-text-muted);
    font-size: var(--k-text-sm);
    margin: 0;
  }

  .admin-badge-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .admin-badge-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
  }

  .admin-badge-item__info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .admin-badge-item__code {
    font-weight: 600;
    font-size: var(--k-text-sm);
  }

  .admin-badge-item__name {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
  }

  .admin-badge-item__meta {
    font-size: 11px;
    color: var(--k-text-muted);
  }

  .admin-badge-item__earned-tag {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    font-style: italic;
  }

  .admin-revoke-form {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .admin-revoke-input {
    padding: var(--k-space-1) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    font-size: var(--k-text-sm);
  }

  .admin-grant-form {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
  }

  .admin-select {
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    background-color: var(--k-surface);
    flex: 1;
    max-width: 400px;
    font-size: var(--k-text-sm);
  }
</style>
