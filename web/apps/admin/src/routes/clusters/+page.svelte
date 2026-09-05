<!--
  apps/admin/src/routes/clusters/+page.svelte

  Cluster development officer tools. CLUSTER_OFFICER or MINISTRY only.

  There is no ListClusters/ListSelfHelpGroups RPC anywhere in identity.proto
  -- only fetch-by-id (GetCluster/GetSelfHelpGroup) and create
  (CreateCluster/CreateSelfHelpGroup). A cluster or SHG this page creates is
  shown immediately with its id; loading an existing one means typing its id
  in (or following a link that already carries it). That is a real backend
  gap, not a UI shortcut -- see ml_wiring.md #10.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Button, Input, FieldGroup, Select, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    createCluster,
    getCluster,
    listClusterMembers,
    addClusterMember,
    removeClusterMember,
    onboardClusterArtisan,
    createSelfHelpGroup,
    getSelfHelpGroup,
    setSelfHelpGroupMembers,
    ApiError,
    messageKeyFor,
  } from '@kalakriti/api';
  import { parseOnboardSheet, type OnboardRow } from '$lib/bulk-onboard';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'CLUSTER_OFFICER' || role === 'MINISTRY');

  // --- Cluster load/create -------------------------------------------------
  type ClusterRecord = Awaited<ReturnType<typeof getCluster>>;
  type MemberRecord = NonNullable<Awaited<ReturnType<typeof listClusterMembers>>['members']>[number];

  let clusterIdInput = $state('');
  let cluster = $state<ClusterRecord | undefined>();
  let members = $state<MemberRecord[]>([]);
  let clusterError = $state('');
  let loadingCluster = $state(false);

  let newClusterName = $state('');
  let newClusterState = $state('');
  let newClusterDistrict = $state('');
  let creatingCluster = $state(false);

  async function loadCluster(id: string): Promise<void> {
    loadingCluster = true;
    clusterError = '';
    try {
      const [c, m] = await Promise.all([getCluster(id), listClusterMembers(id)]);
      cluster = c;
      members = m.members ?? [];
      clusterIdInput = id;
    } catch (cause) {
      clusterError = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      cluster = undefined;
      members = [];
    } finally {
      loadingCluster = false;
    }
  }

  async function createNewCluster(): Promise<void> {
    creatingCluster = true;
    try {
      const created = await createCluster({
        name: newClusterName,
        state_code: newClusterState,
        district: newClusterDistrict || undefined,
      });
      showToast({ variant: 'success', message: t('clusters.created') });
      if (created.id) await loadCluster(created.id);
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    } finally {
      creatingCluster = false;
    }
  }

  // --- Member management -----------------------------------------------
  let newMemberArtisanId = $state('');
  let newMemberRole = $state('MEMBER');

  async function addMember(): Promise<void> {
    if (!cluster?.id) return;
    try {
      await addClusterMember(cluster.id, { artisan_id: newMemberArtisanId, role: newMemberRole as 'MEMBER' | 'COORDINATOR' | 'MASTER' });
      newMemberArtisanId = '';
      await loadCluster(cluster.id);
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    }
  }

  async function removeMember(artisanId: string): Promise<void> {
    if (!cluster?.id) return;
    try {
      await removeClusterMember(cluster.id, artisanId);
      await loadCluster(cluster.id);
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    }
  }

  // --- Bulk onboarding ----------------------------------------------------
  let sheetRows = $state<OnboardRow[]>([]);
  let sheetHeaderError = $state('');
  let committing = $state(false);
  let commitResults = $state<{ rowNumber: number; ok: boolean; message: string }[]>([]);

  const validRows = $derived(sheetRows.filter((r) => r.errors.length === 0));
  const invalidRows = $derived(sheetRows.filter((r) => r.errors.length > 0));

  async function onFileSelected(e: Event): Promise<void> {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const text = await file.text();
    const parsed = parseOnboardSheet(text);
    sheetRows = parsed.rows;
    sheetHeaderError = parsed.headerError ?? '';
    commitResults = [];
  }

  async function commitSheet(): Promise<void> {
    if (!cluster?.id || validRows.length === 0) return;
    committing = true;
    commitResults = [];
    for (const row of validRows) {
      try {
        await onboardClusterArtisan(cluster.id, {
          display_name: row.display_name,
          phone_e164: row.phone_e164,
          craft_ids: row.craft_ids,
          languages: row.languages,
          region: { state_code: row.state_code, district: row.district },
        });
        commitResults = [...commitResults, { rowNumber: row.rowNumber, ok: true, message: t('clusters.onboarded') }];
      } catch (cause) {
        const message = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
        commitResults = [...commitResults, { rowNumber: row.rowNumber, ok: false, message }];
      }
    }
    committing = false;
    if (cluster?.id) await loadCluster(cluster.id);
  }

  // --- SHG management -------------------------------------------------
  type SHGRecord = Awaited<ReturnType<typeof getSelfHelpGroup>>;
  let shgIdInput = $state('');
  let shg = $state<SHGRecord | undefined>();
  let shgError = $state('');

  let newShgName = $state('');
  let newShgRegNo = $state('');

  interface ShareRow {
    artisan_id: string;
    share_pct: number;
  }
  let shareRows = $state<ShareRow[]>([{ artisan_id: '', share_pct: 0 }]);
  const shareSum = $derived(shareRows.reduce((sum, r) => sum + (r.share_pct || 0), 0));
  const shareValid = $derived(shareSum === 100 && shareRows.every((r) => r.artisan_id.trim() !== ''));

  async function loadShg(id: string): Promise<void> {
    shgError = '';
    try {
      shg = await getSelfHelpGroup(id);
      shgIdInput = id;
      const existing = (shg as { members?: ShareRow[] }).members ?? [];
      shareRows = existing.length > 0 ? existing.map((m) => ({ artisan_id: m.artisan_id, share_pct: m.share_pct })) : [{ artisan_id: '', share_pct: 0 }];
    } catch (cause) {
      shgError = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      shg = undefined;
    }
  }

  async function createNewShg(): Promise<void> {
    if (!shareValid) return;
    try {
      const created = await createSelfHelpGroup({
        name: newShgName,
        registration_no: newShgRegNo,
        cluster_id: cluster?.id,
        members: shareRows,
      });
      showToast({ variant: 'success', message: t('clusters.shgCreated') });
      if (created.id) await loadShg(created.id);
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    }
  }

  async function saveShgMembers(): Promise<void> {
    if (!shg?.id || !shareValid) return;
    try {
      await setSelfHelpGroupMembers(shg.id, { members: shareRows });
      showToast({ variant: 'success', message: t('clusters.shgSaved') });
      await loadShg(shg.id);
    } catch (cause) {
      showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
    }
  }

  function addShareRow(): void {
    shareRows = [...shareRows, { artisan_id: '', share_pct: 0 }];
  }
  function removeShareRow(i: number): void {
    shareRows = shareRows.filter((_, idx) => idx !== i);
  }
</script>

<svelte:head>
  <title>{t('nav.clusters')} — {t('admin.home.title')}</title>
</svelte:head>

{#if !authorized}
  <h1>{t('nav.clusters')}</h1>
  <p role="alert">{t('insights.accessRestricted')}</p>
{:else}
  <h1>{t('nav.clusters')}</h1>

  <section class="clusters-section" aria-labelledby="cluster-load-heading">
    <h2 id="cluster-load-heading">{t('clusters.loadHeading')}</h2>
    <div class="clusters-row">
      <FieldGroup label={t('clusters.clusterId')}>
        {#snippet children({ id })}
          <Input {id} bind:value={clusterIdInput} placeholder="uuid" />
        {/snippet}
      </FieldGroup>
      <Button onclick={() => loadCluster(clusterIdInput)} loading={loadingCluster}>{t('clusters.load')}</Button>
    </div>
    {#if clusterError}<p role="alert" class="clusters-error">{clusterError}</p>{/if}

    <details class="clusters-create">
      <summary>{t('clusters.createNew')}</summary>
      <div class="clusters-row">
        <FieldGroup label={t('clusters.name')}>
          {#snippet children({ id })}<Input {id} bind:value={newClusterName} />{/snippet}
        </FieldGroup>
        <FieldGroup label={t('insights.stateCode')}>
          {#snippet children({ id })}<Input {id} bind:value={newClusterState} placeholder="IN-GJ" />{/snippet}
        </FieldGroup>
        <FieldGroup label={t('insights.district')}>
          {#snippet children({ id })}<Input {id} bind:value={newClusterDistrict} />{/snippet}
        </FieldGroup>
        <Button onclick={createNewCluster} loading={creatingCluster} disabled={!newClusterName || !newClusterState}>
          {t('clusters.create')}
        </Button>
      </div>
    </details>
  </section>

  {#if cluster}
    <section class="clusters-section" aria-labelledby="cluster-roster-heading">
      <h2 id="cluster-roster-heading">
        {cluster.name} <span class="clusters-meta">({cluster.district ?? ''} {cluster.state_code}) · {t('clusters.capacity', { count: String(cluster.artisan_count ?? 0) })}</span>
      </h2>

      <div class="clusters-table-scroll">
        <table class="clusters-table">
          <caption class="sr-only">{t('clusters.rosterCaption')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('clusters.member')}</th>
              <th scope="col">{t('clusters.role')}</th>
              <th scope="col">{t('clusters.joined')}</th>
              <th scope="col"><span class="sr-only">{t('action.remove')}</span></th>
            </tr>
          </thead>
          <tbody>
            {#each members as member (member.artisan_id)}
              <tr>
                <td>{member.display_name}</td>
                <td>{member.role}</td>
                <td>{member.joined_at ? new Date(member.joined_at).toLocaleDateString() : ''}</td>
                <td><Button variant="ghost" onclick={() => removeMember(member.artisan_id ?? '')}>{t('action.remove')}</Button></td>
              </tr>
            {:else}
              <tr><td colspan="4">{t('insights.noData')}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="clusters-row">
        <FieldGroup label={t('clusters.artisanId')}>
          {#snippet children({ id })}<Input {id} bind:value={newMemberArtisanId} placeholder="uuid" />{/snippet}
        </FieldGroup>
        <FieldGroup label={t('clusters.role')}>
          {#snippet children({ id })}
            <Select {id} bind:value={newMemberRole} options={[
              { value: 'MEMBER', label: t('clusters.roleMember') },
              { value: 'COORDINATOR', label: t('clusters.roleCoordinator') },
              { value: 'MASTER', label: t('clusters.roleMaster') },
            ]} />
          {/snippet}
        </FieldGroup>
        <Button onclick={addMember} disabled={!newMemberArtisanId}>{t('clusters.addMember')}</Button>
      </div>
    </section>

    <section class="clusters-section" aria-labelledby="bulk-heading">
      <h2 id="bulk-heading">{t('clusters.bulkOnboarding')}</h2>
      <p class="clusters-hint">{t('clusters.bulkHint')}</p>
      <input type="file" accept=".csv,text/csv" onchange={onFileSelected} />

      {#if sheetHeaderError}
        <p role="alert" class="clusters-error">{sheetHeaderError}</p>
      {:else if sheetRows.length > 0}
        <p class="clusters-summary">
          {t('clusters.sheetSummary', { valid: String(validRows.length), invalid: String(invalidRows.length) })}
        </p>

        {#if invalidRows.length > 0}
          <div class="clusters-table-scroll">
            <table class="clusters-table clusters-table--errors">
              <caption class="sr-only">{t('clusters.errorRowsCaption')}</caption>
              <thead>
                <tr><th scope="col">{t('clusters.row')}</th><th scope="col">{t('clusters.name')}</th><th scope="col">{t('clusters.errors')}</th></tr>
              </thead>
              <tbody>
                {#each invalidRows as row (row.rowNumber)}
                  <tr>
                    <td>{row.rowNumber}</td>
                    <td>{row.display_name || '—'}</td>
                    <td>
                      <ul class="clusters-error-list">
                        {#each row.errors as err (err)}<li>{err}</li>{/each}
                      </ul>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}

        <div class="clusters-table-scroll">
          <table class="clusters-table">
            <caption class="sr-only">{t('clusters.previewCaption')}</caption>
            <thead>
              <tr><th scope="col">{t('clusters.row')}</th><th scope="col">{t('clusters.name')}</th><th scope="col">{t('clusters.phone')}</th><th scope="col">{t('insights.district')}</th></tr>
            </thead>
            <tbody>
              {#each validRows as row (row.rowNumber)}
                <tr>
                  <td>{row.rowNumber}</td>
                  <td>{row.display_name}</td>
                  <td>{row.phone_e164}</td>
                  <td>{row.district ?? row.state_code}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <Button onclick={commitSheet} loading={committing} disabled={validRows.length === 0}>
          {t('clusters.commitRows', { count: String(validRows.length) })}
        </Button>

        {#if commitResults.length > 0}
          <ul class="clusters-commit-results" aria-live="polite">
            {#each commitResults as result (result.rowNumber)}
              <li class={result.ok ? 'clusters-commit-ok' : 'clusters-commit-fail'}>
                {t('clusters.row')} {result.rowNumber}: {result.message}
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </section>

    <section class="clusters-section" aria-labelledby="shg-heading">
      <h2 id="shg-heading">
        <Icon name="shg" />
        {t('clusters.shgHeading')}
      </h2>
      <div class="clusters-row">
        <FieldGroup label={t('clusters.shgId')}>
          {#snippet children({ id })}<Input {id} bind:value={shgIdInput} placeholder="uuid" />{/snippet}
        </FieldGroup>
        <Button onclick={() => loadShg(shgIdInput)}>{t('clusters.load')}</Button>
      </div>
      {#if shgError}<p role="alert" class="clusters-error">{shgError}</p>{/if}

      {#if !shg}
        <details class="clusters-create">
          <summary>{t('clusters.shgCreateNew')}</summary>
          <div class="clusters-row">
            <FieldGroup label={t('clusters.name')}>
              {#snippet children({ id })}<Input {id} bind:value={newShgName} />{/snippet}
            </FieldGroup>
            <FieldGroup label={t('clusters.regNo')}>
              {#snippet children({ id })}<Input {id} bind:value={newShgRegNo} />{/snippet}
            </FieldGroup>
          </div>
        </details>
      {:else}
        <p>{shg.name} · {shg.registration_no}</p>
      {/if}

      <h3>{t('clusters.shareSplit')}</h3>
      <p class="clusters-hint">{t('clusters.shareHint')}</p>
      <div class="clusters-share-rows">
        {#each shareRows as row, i (i)}
          <div class="clusters-share-row">
            <Input bind:value={row.artisan_id} placeholder={t('clusters.artisanId')} aria-label={t('clusters.artisanId')} />
            <input
              type="number"
              min="0"
              max="100"
              bind:value={row.share_pct}
              aria-label={t('clusters.sharePct')}
              class="clusters-share-input"
            />
            <span>%</span>
            <Button variant="ghost" onclick={() => removeShareRow(i)}>{t('action.remove')}</Button>
          </div>
        {/each}
      </div>
      <Button variant="secondary" onclick={addShareRow}>{t('clusters.addShareRow')}</Button>

      <p class="clusters-share-total" class:clusters-share-total--bad={shareSum !== 100} role="status">
        {t('clusters.shareTotal', { total: String(shareSum) })}
        {#if shareSum !== 100}
          — {t('clusters.shareMustSum100')}
        {/if}
      </p>

      {#if shg}
        <Button onclick={saveShgMembers} disabled={!shareValid}>{t('clusters.saveShgMembers')}</Button>
      {:else}
        <Button onclick={createNewShg} disabled={!shareValid || !newShgName || !newShgRegNo}>{t('clusters.create')}</Button>
      {/if}
    </section>
  {/if}
{/if}

<style>
  .clusters-section {
    margin-block-end: var(--k-space-7);
    padding-block-end: var(--k-space-6);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .clusters-row {
    display: flex;
    align-items: end;
    gap: var(--k-space-3);
    flex-wrap: wrap;
    margin-block-end: var(--k-space-3);
  }

  .clusters-error {
    color: var(--k-accent-danger);
  }

  .clusters-meta {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-regular);
    color: var(--k-text-secondary);
  }

  .clusters-table-scroll {
    overflow-x: auto;
    margin-block-end: var(--k-space-3);
  }

  .clusters-table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .clusters-table th,
  .clusters-table td {
    text-align: start;
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .clusters-table--errors {
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-danger);
  }

  .clusters-error-list {
    margin: 0;
    padding-inline-start: var(--k-space-4);
    color: var(--k-accent-danger);
  }

  .clusters-hint,
  .clusters-summary {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block: var(--k-space-2);
  }

  .clusters-commit-results {
    margin-block-start: var(--k-space-3);
    padding: 0;
    list-style: none;
    font-size: var(--k-text-sm);
  }

  .clusters-commit-ok::before {
    content: '✓ ';
  }
  .clusters-commit-fail {
    color: var(--k-accent-danger);
  }
  .clusters-commit-fail::before {
    content: '✕ ';
  }

  .clusters-create {
    margin-block-start: var(--k-space-2);
  }

  .clusters-share-rows {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-2);
  }

  .clusters-share-row {
    display: grid;
    grid-template-columns: 1fr 6rem auto auto;
    align-items: center;
    gap: var(--k-space-2);
  }

  .clusters-share-input {
    font: inherit;
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
  }

  .clusters-share-total {
    margin-block: var(--k-space-3);
    font-weight: var(--k-weight-semibold);
  }

  .clusters-share-total--bad {
    color: var(--k-accent-danger);
  }

  .sr-only {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
</style>
