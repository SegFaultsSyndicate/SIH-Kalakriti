<!--
  apps/admin/src/routes/schemes/+page.svelte

  Government Scheme Catalog & Eligibility Criteria Administration.
  Allows MINISTRY officers to browse active welfare schemes, register new
  schemes with criteria and manual checklists, edit existing definitions,
  and remove obsolete programs.

  Design Law compliant: hairline rules, no shadow-card floating boxes, Svelte 5 runes.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Button, Input, FieldGroup, Dialog, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    listSchemes,
    upsertScheme,
    deleteScheme,
    type GovernmentScheme,
    type UpsertSchemeBody,
  } from '@kalakriti/api';
  import type { PageData } from './$types';

  interface Props {
    data: PageData;
  }
  let { data }: Props = $props();

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY');

  let schemes = $state<GovernmentScheme[]>(data.schemes ?? []);
  let loading = $state(false);

  // Modal / Form state
  let formOpen = $state(false);
  let editingId = $state<string | null>(null);
  let saving = $state(false);

  // Form fields
  let code = $state('');
  let authority = $state<'CENTRAL' | 'STATE'>('CENTRAL');
  let ministry = $state('');
  let officialUrl = $state('');
  let stateCode = $state('');
  let nameText = $state('');
  let summaryText = $state('');
  let sortOrder = $state(10);

  type CriterionType =
    | 'SOCIAL_CATEGORY'
    | 'STATE_CODE'
    | 'CRAFT_ID'
    | 'MIN_YEARS_EXPERIENCE'
    | 'HAS_PEHCHAN_ID'
    | 'HAS_PM_VISHWAKARMA_ID'
    | 'CLUSTER_MEMBER'
    | 'SHG_MEMBER';

  interface FormCriterion {
    type: CriterionType;
    stringValues: string;
    intValue?: number;
    negate: boolean;
  }

  let criteria = $state<FormCriterion[]>([]);
  let manualChecks = $state<string[]>([]);

  // Delete confirm state
  let deleteConfirmId = $state<string | null>(null);
  let deleting = $state(false);

  const CRITERION_TYPES: readonly CriterionType[] = [
    'SOCIAL_CATEGORY',
    'STATE_CODE',
    'CRAFT_ID',
    'MIN_YEARS_EXPERIENCE',
    'HAS_PEHCHAN_ID',
    'HAS_PM_VISHWAKARMA_ID',
    'CLUSTER_MEMBER',
    'SHG_MEMBER',
  ];

  async function refresh(): Promise<void> {
    loading = true;
    try {
      const res = await listSchemes();
      schemes = res.schemes ?? [];
    } catch (err) {
      showToast({
        message: err instanceof Error ? err.message : 'Failed to refresh schemes',
        variant: 'error',
      });
    } finally {
      loading = false;
    }
  }

  function openCreate(): void {
    editingId = null;
    code = '';
    authority = 'CENTRAL';
    ministry = '';
    officialUrl = 'https://';
    stateCode = '';
    nameText = '';
    summaryText = '';
    sortOrder = (schemes.length + 1) * 10;
    criteria = [];
    manualChecks = [];
    formOpen = true;
  }

  function openEdit(s: GovernmentScheme): void {
    editingId = s.id;
    code = s.code;
    authority = (s.authority as 'CENTRAL' | 'STATE') || 'CENTRAL';
    ministry = s.ministry;
    officialUrl = s.official_url;
    stateCode = s.state_code ?? '';
    nameText = s.name_text ?? (s.name_i18n_key ? t(s.name_i18n_key as MessageKey) : '');
    summaryText = s.summary_text ?? (s.summary_i18n_key ? t(s.summary_i18n_key as MessageKey) : '');
    sortOrder = s.sort_order;
    criteria = [];
    manualChecks = [];
    formOpen = true;
  }

  function addCriterion(): void {
    criteria = [...criteria, { type: 'SOCIAL_CATEGORY', stringValues: '', negate: false }];
  }

  function removeCriterion(idx: number): void {
    criteria = criteria.filter((_, i) => i !== idx);
  }

  function addManualCheck(): void {
    manualChecks = [...manualChecks, ''];
  }

  function removeManualCheck(idx: number): void {
    manualChecks = manualChecks.filter((_, i) => i !== idx);
  }

  async function handleSave(): Promise<void> {
    if (!code.trim() || !ministry.trim() || !officialUrl.trim() || !nameText.trim()) {
      showToast({
        message: 'Please fill in code, ministry, official URL, and scheme name.',
        variant: 'error',
      });
      return;
    }

    saving = true;
    try {
      const body: UpsertSchemeBody = {
        id: editingId ?? undefined,
        code: code.trim(),
        authority,
        ministry: ministry.trim(),
        official_url: officialUrl.trim(),
        state_code: authority === 'STATE' && stateCode.trim() ? stateCode.trim().toUpperCase() : undefined,
        name_text: nameText.trim(),
        summary_text: summaryText.trim() || undefined,
        sort_order: Number(sortOrder) || 10,
        criteria: criteria.map((c) => ({
          type: c.type,
          string_values: c.stringValues
            ? c.stringValues
                .split(',')
                .map((v) => v.trim())
                .filter(Boolean)
            : [],
          int_value: c.intValue !== undefined ? Number(c.intValue) : undefined,
          negate: c.negate,
        })),
        manual_checks: manualChecks
          .map((text) => ({ check_text: text.trim() }))
          .filter((c) => c.check_text !== ''),
      };

      await upsertScheme(body, editingId ?? undefined);
      showToast({
        message: editingId ? 'Scheme updated successfully.' : 'Scheme created successfully.',
        variant: 'success',
      });
      formOpen = false;
      await refresh();
    } catch (err) {
      showToast({
        message: err instanceof Error ? err.message : 'Failed to save scheme',
        variant: 'error',
      });
    } finally {
      saving = false;
    }
  }

  async function handleDelete(id: string): Promise<void> {
    deleting = true;
    try {
      await deleteScheme(id);
      showToast({
        message: 'Scheme removed from catalog.',
        variant: 'info',
      });
      deleteConfirmId = null;
      await refresh();
    } catch (err) {
      showToast({
        message: err instanceof Error ? err.message : 'Failed to delete scheme',
        variant: 'error',
      });
    } finally {
      deleting = false;
    }
  }

  function displayName(s: GovernmentScheme): string {
    return s.name_i18n_key ? t(s.name_i18n_key as MessageKey) : (s.name_text ?? s.code);
  }

  function displaySummary(s: GovernmentScheme): string {
    return s.summary_i18n_key ? t(s.summary_i18n_key as MessageKey) : (s.summary_text ?? '');
  }
</script>

<svelte:head>
  <title>{t('admin.schemes.title')} - Kalakriti Admin</title>
</svelte:head>

{#if !authorized}
  <div class="restricted-box">
    <h1>{t('admin.schemes.title')}</h1>
    <p role="alert">{t('insights.accessRestricted')}</p>
  </div>
{:else}
  <section class="schemes-view" aria-labelledby="heading">
    <header class="schemes-header">
      <div class="kicker">ministry welfare administration</div>
      <div class="heading-row">
        <div>
          <h1 id="heading" class="title">{t('admin.schemes.title')}</h1>
          <p class="subtitle">{t('admin.schemes.subtitle')}</p>
        </div>
        <div class="header-actions">
          <Button variant="secondary" size="md" onclick={refresh} disabled={loading}>
            <Icon name="refresh" />
            <span>{t('insights.refresh')}</span>
          </Button>
          <Button variant="primary" size="md" onclick={openCreate}>
            <Icon name="plus" />
            <span>{t('admin.schemes.add')}</span>
          </Button>
        </div>
      </div>
    </header>

    <!-- Schemes Table / List -->
    <div class="table-card">
      <table class="schemes-table">
        <thead>
          <tr>
            <th scope="col">Code</th>
            <th scope="col">Scheme Name</th>
            <th scope="col">Authority</th>
            <th scope="col">Ministry</th>
            <th scope="col">Official URL</th>
            <th scope="col" class="th-center">Order</th>
            <th scope="col" class="th-actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#if schemes.length === 0}
            <tr>
              <td colspan="7" class="empty-cell">
                {t('schemes.empty')}
              </td>
            </tr>
          {/if}
          {#each schemes as scheme (scheme.id)}
            <tr>
              <td class="td-code"><code>{scheme.code}</code></td>
              <td>
                <div class="scheme-name-cell">
                  <strong>{displayName(scheme)}</strong>
                  {#if displaySummary(scheme)}
                    <span class="scheme-summary-cell">{displaySummary(scheme)}</span>
                  {/if}
                </div>
              </td>
              <td>
                <span class="badge badge--{scheme.authority?.toLowerCase()}">
                  {scheme.authority}
                  {#if scheme.state_code}
                    ({scheme.state_code})
                  {/if}
                </span>
              </td>
              <td class="td-ministry">{scheme.ministry}</td>
              <td>
                <a
                  href={scheme.official_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="portal-link"
                >
                  Visit Portal ↗
                </a>
              </td>
              <td class="td-center">{scheme.sort_order}</td>
              <td class="td-actions">
                <button
                  type="button"
                  class="action-btn"
                  onclick={() => openEdit(scheme)}
                  title="Edit scheme"
                >
                  <Icon name="edit" />
                </button>
                <button
                  type="button"
                  class="action-btn action-btn--danger"
                  onclick={() => (deleteConfirmId = scheme.id)}
                  title="Delete scheme"
                >
                  <Icon name="trash" />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  <!-- Create / Edit Scheme Dialog -->
  <Dialog
    open={formOpen}
    title={editingId ? t('admin.schemes.edit') : t('admin.schemes.add')}
  >
    <div class="scheme-form">
      <div class="form-row">
        <FieldGroup label="Scheme Code">
          {#snippet children({ id })}
            <Input {id} bind:value={code} placeholder="e.g. pm_vishwakarma" disabled={editingId != null} />
          {/snippet}
        </FieldGroup>

        <FieldGroup label="Authority">
          {#snippet children({ id })}
            <select {id} bind:value={authority} class="form-select">
              <option value="CENTRAL">CENTRAL</option>
              <option value="STATE">STATE</option>
            </select>
          {/snippet}
        </FieldGroup>
      </div>

      <div class="form-row">
        <FieldGroup label="Ministry / Department">
          {#snippet children({ id })}
            <Input {id} bind:value={ministry} placeholder="e.g. Ministry of MSME" />
          {/snippet}
        </FieldGroup>

        {#if authority === 'STATE'}
          <FieldGroup label="State Code (2-letter)">
            {#snippet children({ id })}
              <Input {id} bind:value={stateCode} placeholder="e.g. RJ" maxlength={2} />
            {/snippet}
          </FieldGroup>
        {/if}
      </div>

      <FieldGroup label="Official Government Portal URL">
        {#snippet children({ id })}
          <Input {id} type="url" bind:value={officialUrl} placeholder="https://pmvishwakarma.gov.in" />
        {/snippet}
      </FieldGroup>

      <FieldGroup label="Scheme Name (Display Text)">
        {#snippet children({ id })}
          <Input {id} bind:value={nameText} placeholder="e.g. PM Vishwakarma Scheme" />
        {/snippet}
      </FieldGroup>

      <FieldGroup label="Summary Description">
        {#snippet children({ id })}
          <Input {id} bind:value={summaryText} placeholder="Short overview of benefits and eligibility" />
        {/snippet}
      </FieldGroup>

      <FieldGroup label="Sort Order">
        {#snippet children({ id })}
          <Input {id} type="number" bind:value={sortOrder} />
        {/snippet}
      </FieldGroup>

      <!-- Criteria Builder -->
      <section class="form-section">
        <div class="form-section__header">
          <h3>Eligibility Criteria</h3>
          <Button variant="secondary" size="sm" onclick={addCriterion}>
            + Add Criterion
          </Button>
        </div>

        {#if criteria.length === 0}
          <p class="section-empty">No criteria defined (scheme matches all artisans).</p>
        {/if}

        {#each criteria as crit, idx}
          <div class="criterion-row">
            <select bind:value={crit.type} class="form-select">
              {#each CRITERION_TYPES as ct}
                <option value={ct}>{ct}</option>
              {/each}
            </select>

            {#if crit.type === 'MIN_YEARS_EXPERIENCE'}
              <Input
                type="number"
                bind:value={crit.intValue}
                placeholder="Min years"
              />
            {:else if crit.type === 'SOCIAL_CATEGORY' || crit.type === 'STATE_CODE' || crit.type === 'CRAFT_ID'}
              <Input
                bind:value={crit.stringValues}
                placeholder="Comma-separated values (e.g. SC,ST or RJ)"
              />
            {/if}

            <label class="checkbox-label">
              <input type="checkbox" bind:checked={crit.negate} />
              <span>Negate</span>
            </label>

            <button
              type="button"
              class="icon-remove-btn"
              onclick={() => removeCriterion(idx)}
              title="Remove criterion"
            >
              <Icon name="trash" />
            </button>
          </div>
        {/each}
      </section>

      <!-- Manual Checklist Builder -->
      <section class="form-section">
        <div class="form-section__header">
          <h3>Manual Checklist Items</h3>
          <Button variant="secondary" size="sm" onclick={addManualCheck}>
            + Add Check Item
          </Button>
        </div>

        {#if manualChecks.length === 0}
          <p class="section-empty">No manual checks defined.</p>
        {/if}

        {#each manualChecks as check, idx}
          <div class="check-row">
            <Input
              bind:value={manualChecks[idx]}
              placeholder="e.g. Must have an active bank account"
            />
            <button
              type="button"
              class="icon-remove-btn"
              onclick={() => removeManualCheck(idx)}
              title="Remove check"
            >
              <Icon name="trash" />
            </button>
          </div>
        {/each}
      </section>

      <div class="form-actions">
        <Button variant="secondary" size="md" onclick={() => (formOpen = false)}>
          Cancel
        </Button>
        <Button variant="primary" size="md" onclick={handleSave} loading={saving}>
          Save Scheme
        </Button>
      </div>
    </div>
  </Dialog>

  <!-- Delete Confirm Dialog -->
  <Dialog
    open={deleteConfirmId != null}
    title={t('admin.schemes.deleteConfirm')}
  >
    <p>{t('admin.schemes.deleteConfirm')}</p>
    <div class="form-actions">
      <Button variant="secondary" size="md" onclick={() => (deleteConfirmId = null)}>
        Cancel
      </Button>
      <Button
        variant="primary"
        size="md"
        onclick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
        loading={deleting}
      >
        Delete
      </Button>
    </div>
  </Dialog>
{/if}

<style>
  .schemes-view {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-5);
    padding: var(--k-space-4);
  }

  .schemes-header {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .kicker {
    font-size: var(--k-text-xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--k-text-secondary);
  }

  .heading-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--k-space-4);
  }

  .title {
    font-size: var(--k-text-2xl);
    font-weight: 700;
    margin: 0;
    color: var(--k-text-primary);
  }

  .subtitle {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: var(--k-space-1) 0 0 0;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .table-card {
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    overflow-x: auto;
  }

  .schemes-table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
    text-align: start;
  }

  .schemes-table th,
  .schemes-table td {
    padding: var(--k-space-3) var(--k-space-4);
    border-block-end: var(--k-hairline) solid var(--k-border-subtle);
  }

  .schemes-table th {
    background-color: var(--k-surface-sunken);
    font-weight: 600;
    color: var(--k-text-secondary);
    text-transform: uppercase;
    font-size: var(--k-text-xs);
    letter-spacing: 0.05em;
  }

  .th-center,
  .td-center {
    text-align: center;
  }

  .th-actions,
  .td-actions {
    text-align: end;
  }

  .td-code code {
    font-size: var(--k-text-xs);
    background-color: var(--k-surface-sunken);
    padding: var(--k-space-1) var(--k-space-2);
    border-radius: var(--k-radius-sm);
  }

  .scheme-name-cell {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .scheme-summary-cell {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
  }

  .badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-xs);
    font-weight: 600;
  }

  .badge--central {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
    border: var(--k-hairline) solid var(--k-border-interactive);
  }

  .badge--state {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    border: var(--k-hairline) solid var(--k-border-subtle);
  }

  .td-ministry {
    color: var(--k-text-secondary);
  }

  .portal-link {
    color: var(--k-text-primary);
    text-decoration: underline;
    text-underline-offset: 2px;
    font-weight: 500;
  }

  .portal-link:hover {
    color: var(--k-accent-primary-bg);
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 2rem;
    block-size: 2rem;
    border: var(--k-hairline) solid var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-base);
    color: var(--k-text-secondary);
    cursor: pointer;
    margin-inline-start: var(--k-space-1);
  }

  .action-btn:hover {
    color: var(--k-text-primary);
    background-color: var(--k-surface-sunken);
  }

  .action-btn--danger:hover {
    color: var(--k-error);
    border-color: var(--k-error);
  }

  .empty-cell {
    text-align: center;
    padding: var(--k-space-6);
    color: var(--k-text-secondary);
  }

  /* Form modal styles */
  .scheme-form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    max-inline-size: 36rem;
    padding-block-start: var(--k-space-2);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--k-space-3);
  }

  .form-select {
    inline-size: 100%;
    min-block-size: var(--k-touch-min);
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-sunken);
  }

  .form-section__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .form-section__header h3 {
    font-size: var(--k-text-sm);
    font-weight: 600;
    margin: 0;
  }

  .section-empty {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    margin: 0;
  }

  .criterion-row,
  .check-row {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    cursor: pointer;
  }

  .icon-remove-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 2rem;
    block-size: 2rem;
    border: none;
    background: none;
    color: var(--k-text-muted);
    cursor: pointer;
    flex-shrink: 0;
  }

  .icon-remove-btn:hover {
    color: var(--k-error);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
  }

  .restricted-box {
    padding: var(--k-space-6);
    text-align: center;
  }
</style>
