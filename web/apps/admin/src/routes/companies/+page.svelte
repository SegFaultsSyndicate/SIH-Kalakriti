<!--
  apps/admin/src/routes/companies/+page.svelte

  B2B Company & Boutique Verification Queue (5th Admin Tab).
  Enforces administrative legitimacy checks on companies registering through
  the buyer portal. Authenticates business legitimacy via GSTIN and income
  statement audit, and monitors platform commission ledger (0.5% for Retailer/
  Boutique/Institution, 1.0% for Exporter).

  Design Law compliant: hairline rules, no shadow-card floating boxes, Svelte 5 runes.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Button, Skeleton, Money, Dialog, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    listCompanies,
    verifyCompany,
    getCommissionStats,
    type Company,
    type CommissionStatsResponse,
  } from '@kalakriti/api';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'CLUSTER_OFFICER' || role === 'MINISTRY');

  type Tab = 'pending' | 'verified' | 'rejected' | 'all';
  let activeTab = $state<Tab>('pending');

  let companies = $state<Company[]>([]);
  let stats = $state<CommissionStatsResponse>({
    total_companies: 4,
    pending_verifications: 2,
    verified_companies: 2,
    total_sales_paise: 245000000,
    total_commission_paise: 1475000,
  });
  let loading = $state(false);

  let selectedDocUrl = $state<string | null>(null);
  let selectedCompanyName = $state<string>('');
  let rejectionDrafts = $state<Record<string, string>>({});
  let rejectingId = $state<string | null>(null);
  let acting = $state<Record<string, boolean>>({});

  const MOCK_COMPANIES: Company[] = [
    {
      id: 'comp-01',
      name: 'Anokhi Heritage Handlooms Pvt Ltd',
      type: 'RETAILER',
      gstin: '07AAAAA0000A1Z5',
      contact_name: 'Rajesh Singhania',
      contact_phone: '+919810012345',
      contact_email: 'procurement@anokhiheritage.in',
      website: 'https://anokhiheritage.in',
      verification_status: 'PENDING',
      verified: false,
      income_statement_url: 'https://cdn.kalakriti.org.in/statements/anokhi_fy25_audited.pdf',
      commission_rate_bps: 50,
      total_sales_paise: 0,
      commission_earned_paise: 0,
      region: { state_code: 'IN-RJ', district: 'Jaipur' },
      preferred_craft_ids: ['craft-bagru', 'craft-sanganeri'],
      accepts_consignment: true,
      min_order_value_paise: 5000000,
    },
    {
      id: 'comp-02',
      name: 'Kala Mandir Boutique Studio',
      type: 'BOUTIQUE',
      gstin: '27AABCK1234F1Z8',
      contact_name: 'Pooja Bhattacharya',
      contact_phone: '+919820054321',
      contact_email: 'curator@kalamandir.studio',
      website: 'https://kalamandir.studio',
      verification_status: 'PENDING',
      verified: false,
      income_statement_url: 'https://cdn.kalakriti.org.in/statements/kalamandir_gst_pnl_2025.pdf',
      commission_rate_bps: 50,
      total_sales_paise: 0,
      commission_earned_paise: 0,
      region: { state_code: 'IN-MH', district: 'Mumbai Suburban' },
      preferred_craft_ids: ['craft-paithani', 'craft-chanderi'],
      store_location: {
        latitude: 19.076,
        longitude: 72.8777,
        address: 'Boutique 4, Heritage Lane, Kala Ghoda',
        city: 'Mumbai',
        pincode: '400001',
      },
      accepts_consignment: true,
      min_order_value_paise: 2500000,
    },
    {
      id: 'comp-03',
      name: 'Indus Weaves Global Exports',
      type: 'EXPORTER',
      gstin: '09AAACI9876E1Z2',
      contact_name: 'Vikramaditya Oberoi',
      contact_phone: '+919711098765',
      contact_email: 'trade@indusweaves.com',
      website: 'https://indusweaves.com',
      verification_status: 'VERIFIED',
      verified: true,
      verified_by: 'MINISTRY_OFFICER_04',
      verified_at: '2026-08-12T10:30:00Z',
      income_statement_url: 'https://cdn.kalakriti.org.in/statements/indus_audit_2025.pdf',
      commission_rate_bps: 100,
      total_sales_paise: 185000000,
      commission_earned_paise: 1850000,
      region: { state_code: 'IN-UP', district: 'Varanasi' },
      preferred_craft_ids: ['craft-banarasi-brocade', 'craft-zardozi'],
      accepts_consignment: false,
      min_order_value_paise: 20000000,
    },
    {
      id: 'comp-04',
      name: 'National Handloom Development Society',
      type: 'INSTITUTION',
      gstin: '08AAATN1122D1Z9',
      contact_name: 'Dr. Alok Verma',
      contact_phone: '+919414033445',
      contact_email: 'director@nhds.gov.in',
      verification_status: 'VERIFIED',
      verified: true,
      verified_by: 'MINISTRY_OFFICER_01',
      verified_at: '2026-07-20T14:15:00Z',
      income_statement_url: 'https://cdn.kalakriti.org.in/statements/nhds_annual_report.pdf',
      commission_rate_bps: 50,
      total_sales_paise: 60000000,
      commission_earned_paise: 300000,
      region: { state_code: 'IN-DL', district: 'New Delhi' },
      preferred_craft_ids: ['craft-madhubani', 'craft-chanderi'],
      accepts_consignment: false,
      min_order_value_paise: 10000000,
    },
  ];

  async function loadData(): Promise<void> {
    loading = true;
    try {
      const [compRes, statsRes] = await Promise.allSettled([
        listCompanies(),
        getCommissionStats(),
      ]);

      if (compRes.status === 'fulfilled' && compRes.value?.companies && compRes.value.companies.length > 0) {
        companies = compRes.value.companies;
      } else if (import.meta.env.VITE_USE_MOCKS === '1') {
        if (compRes.status === 'rejected') console.warn('[mock fallback] listCompanies:', compRes.reason);
        companies = MOCK_COMPANIES;
      } else {
        companies = [];
      }

      if (statsRes.status === 'fulfilled' && statsRes.value) {
        stats = statsRes.value;
      }
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] loadData companies:', cause);
        companies = MOCK_COMPANIES;
      } else {
        companies = [];
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void loadData();
  });

  const filteredCompanies = $derived.by(() => {
    if (activeTab === 'all') return companies;
    if (activeTab === 'pending') {
      return companies.filter(
        (c) => c.verification_status === 'PENDING' || c.verification_status === 'VERIFICATION_STATUS_PENDING' || !c.verified,
      );
    }
    if (activeTab === 'verified') {
      return companies.filter(
        (c) => c.verification_status === 'VERIFIED' || c.verification_status === 'VERIFICATION_STATUS_VERIFIED' || c.verified,
      );
    }
    return companies.filter(
      (c) => c.verification_status === 'REJECTED' || c.verification_status === 'VERIFICATION_STATUS_REJECTED',
    );
  });

  const pendingCount = $derived(
    companies.filter(
      (c) => c.verification_status === 'PENDING' || c.verification_status === 'VERIFICATION_STATUS_PENDING' || !c.verified,
    ).length,
  );

  const verifiedCount = $derived(
    companies.filter(
      (c) => c.verification_status === 'VERIFIED' || c.verification_status === 'VERIFICATION_STATUS_VERIFIED' || c.verified,
    ).length,
  );

  async function handleApprove(c: Company): Promise<void> {
    if (acting[c.id]) return;
    acting = { ...acting, [c.id]: true };

    try {
      await verifyCompany(c.id, { decision: 'VERIFIED' });
      companies = companies.map((item) =>
        item.id === c.id
          ? { ...item, verification_status: 'VERIFIED', verified: true, verified_at: new Date().toISOString() }
          : item,
      );
      showToast({
        message: `${c.name} verified. Platform commission schedule activated.`,
        variant: 'success',
      });
    } catch (err) {
      showToast({
        message: err instanceof Error ? err.message : 'Verification failed',
        variant: 'error',
      });
    } finally {
      acting = { ...acting, [c.id]: false };
    }
  }

  async function handleConfirmReject(c: Company): Promise<void> {
    const reason = rejectionDrafts[c.id] || 'Inadequate income statement documentation';
    acting = { ...acting, [c.id]: true };

    try {
      await verifyCompany(c.id, { decision: 'REJECTED', rejection_reason: reason });
      companies = companies.map((item) =>
        item.id === c.id
          ? { ...item, verification_status: 'REJECTED', verified: false, rejection_reason: reason }
          : item,
      );
      rejectingId = null;
      showToast({
        message: `${c.name} rejected with logged reason.`,
        variant: 'info',
      });
    } catch (err) {
      showToast({
        message: err instanceof Error ? err.message : 'Rejection failed',
        variant: 'error',
      });
    } finally {
      acting = { ...acting, [c.id]: false };
    }
  }

  let docModalOpen = $state(false);

  function openDocument(c: Company): void {
    selectedDocUrl = c.income_statement_url || 'https://cdn.kalakriti.org.in/statements/sample_statement.pdf';
    selectedCompanyName = c.name;
    docModalOpen = true;
  }

  function formatType(type: string): string {
    const clean = type.replace('COMPANY_TYPE_', '');
    switch (clean) {
      case 'BOUTIQUE':
        return 'Boutique Studio';
      case 'EXPORTER':
        return 'Global Exporter';
      case 'INSTITUTION':
        return 'Public / State Institution';
      default:
        return 'Retailer / Brand';
    }
  }

  function commissionRateText(c: Company): string {
    const type = c.type.replace('COMPANY_TYPE_', '');
    if (type === 'EXPORTER') return '1.0% Platform Fee';
    return '0.5% Platform Fee';
  }
</script>

<svelte:head>
  <title>{t('company.title')} - Kalakriti Admin</title>
</svelte:head>

{#if !authorized}
  <div class="restricted-box">
    <h1>{t('company.title')}</h1>
    <p role="alert">{t('insights.accessRestricted')}</p>
  </div>
{:else}
<section class="companies-view" aria-labelledby="heading">
  <header class="companies-header">
    <div class="kicker">ministry oversight &amp; enterprise ledger</div>
    <div class="heading-row">
      <div>
        <h1 id="heading" class="title">{t('company.title')}</h1>
        <p class="subtitle">{t('company.subtitle')}</p>
      </div>
      <Button variant="secondary" size="md" onclick={() => void loadData()}>
        <Icon name="refresh" />
        <span>{t('insights.refresh')}</span>
      </Button>
    </div>

    <!-- Platform Commission Schedule Disclosure Banner -->
    <div class="commission-banner" role="region" aria-label={t('company.field.commissionRate')}>
      <Icon name="info" />
      <div class="banner-text">
        <span class="banner-strong">{t('company.field.commissionRate')}:</span>
        <span>{t('company.rate.desc')}. Credited automatically to platform treasury on every buy order.</span>
      </div>
    </div>

    <!-- Metrics Summary Strip -->
    <div class="stats-strip" role="region" aria-label="Platform Financial Overview">
      <div class="stat-card">
        <span class="stat-label">{t('company.stats.total')}</span>
        <span class="stat-value">{stats.total_companies}</span>
      </div>
      <div class="stat-card stat-pending">
        <span class="stat-label">{t('company.stats.pending')}</span>
        <span class="stat-value">{pendingCount}</span>
      </div>
      <div class="stat-card stat-verified">
        <span class="stat-label">{t('company.stats.verified')}</span>
        <span class="stat-value">{verifiedCount}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{t('company.stats.sales')}</span>
        <span class="stat-value stat-money">
          <Money paise={stats.total_sales_paise} />
        </span>
      </div>
      <div class="stat-card stat-treasury">
        <span class="stat-label">{t('company.stats.commission')}</span>
        <span class="stat-value stat-money">
          <Money paise={stats.total_commission_paise} />
        </span>
      </div>
    </div>

    <!-- Filter Tabs -->
    <div class="tabs" role="tablist">
      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'pending'}
        class="tab-btn"
        class:active={activeTab === 'pending'}
        onclick={() => (activeTab = 'pending')}
      >
        <span>{t('company.tab.pending')}</span>
        {#if pendingCount > 0}
          <span class="badge badge-pending">{pendingCount}</span>
        {/if}
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'verified'}
        class="tab-btn"
        class:active={activeTab === 'verified'}
        onclick={() => (activeTab = 'verified')}
      >
        <span>{t('company.tab.verified')}</span>
        {#if verifiedCount > 0}
          <span class="badge badge-verified">{verifiedCount}</span>
        {/if}
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'rejected'}
        class="tab-btn"
        class:active={activeTab === 'rejected'}
        onclick={() => (activeTab = 'rejected')}
      >
        <span>{t('company.tab.rejected')}</span>
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'all'}
        class="tab-btn"
        class:active={activeTab === 'all'}
        onclick={() => (activeTab = 'all')}
      >
        <span>{t('company.tab.all')}</span>
      </button>
    </div>
  </header>

  {#if loading && companies.length === 0}
    <div class="company-list" aria-busy="true" aria-hidden="true">
      {#each Array(2) as _, i (i)}
        <article class="company-card">
          <div class="card-header">
            <div class="title-group">
              <Skeleton shape="text" width="12rem" height="1.25rem" />
              <div class="badge-row">
                <Skeleton width="4rem" height="1.25rem" radius="var(--k-radius-pill)" />
                <Skeleton width="5rem" height="1.25rem" radius="var(--k-radius-pill)" />
                <Skeleton width="4.5rem" height="1.25rem" radius="var(--k-radius-pill)" />
              </div>
            </div>
          </div>
          <div class="details-grid">
            {#each Array(4) as __, j (j)}
              <div class="detail-cell">
                <Skeleton shape="text" width="5rem" height="0.75rem" />
                <Skeleton shape="text" width="8rem" height="0.9rem" />
              </div>
            {/each}
          </div>
          <div class="audit-strip">
            <div class="audit-left">
              <Skeleton width="1.5rem" height="1.5rem" radius="var(--k-radius-sm)" />
              <div class="doc-meta">
                <Skeleton shape="text" width="8rem" height="0.85rem" />
                <Skeleton shape="text" width="12rem" height="0.75rem" />
              </div>
            </div>
            <Skeleton width="7rem" height="2rem" radius="var(--k-radius-md)" />
          </div>
        </article>
      {/each}
    </div>
  {:else if filteredCompanies.length === 0}
    <div class="empty-state">
      <Icon name="package" />
      <p>{t(activeTab === 'pending' ? 'company.empty.pending' : 'company.empty.verified')}</p>
    </div>
  {:else}
    <div class="company-list" role="feed" aria-label="Company verification feed">
      {#each filteredCompanies as comp (comp.id)}
        <article class="company-card" class:is-verified={comp.verified}>
          <!-- Row 1: Top Identity & Type -->
          <div class="card-header">
            <div class="title-group">
              <h2 class="company-name">{comp.name}</h2>
              <div class="badge-row">
                <span class="type-pill">{formatType(comp.type)}</span>
                <span class="rate-pill">{commissionRateText(comp)}</span>
                {#if comp.verified || comp.verification_status === 'VERIFIED'}
                  <span class="status-pill verified">
                    <Icon name="check" />
                    <span>{t('company.status.verified')}</span>
                  </span>
                {:else if comp.verification_status === 'REJECTED'}
                  <span class="status-pill rejected">
                    <Icon name="close" />
                    <span>{t('company.status.rejected')}</span>
                  </span>
                {:else}
                  <span class="status-pill pending">
                    <Icon name="clock" />
                    <span>{t('company.status.pending')}</span>
                  </span>
                {/if}
              </div>
            </div>

            <!-- Header Quick Financial Summary -->
            {#if (comp.total_sales_paise ?? 0) > 0}
              <div class="revenue-summary">
                <span class="revenue-label">{t('company.field.totalSales')}</span>
                <span class="revenue-val"><Money paise={comp.total_sales_paise ?? 0} /></span>
                <span class="fee-val">
                  {t('company.field.commissionEarned')}: <Money paise={comp.commission_earned_paise ?? 0} />
                </span>
              </div>
            {/if}
          </div>

          <!-- Row 2: Comprehensive Details Grid -->
          <div class="details-grid">
            <div class="detail-cell">
              <span class="cell-label">{t('company.field.gstin')}</span>
              <span class="cell-val gstin-val">{comp.gstin || 'Unspecified'}</span>
            </div>
            <div class="detail-cell">
              <span class="cell-label">{t('company.field.contact')}</span>
              <span class="cell-val">{comp.contact_name} ({comp.contact_phone})</span>
            </div>
            <div class="detail-cell">
              <span class="cell-label">{t('company.field.email')}</span>
              <span class="cell-val">{comp.contact_email || '—'}</span>
            </div>
            <div class="detail-cell">
              <span class="cell-label">{t('company.field.region')}</span>
              <span class="cell-val">{comp.region?.district || ''}, {comp.region?.state_code || ''}</span>
            </div>

            {#if comp.store_location}
              <div class="detail-cell full-width">
                <span class="cell-label">Physical Store Location (Boutique)</span>
                <span class="cell-val">
                  {comp.store_location.address}, {comp.store_location.city} ({comp.store_location.pincode})
                </span>
              </div>
            {/if}

            {#if comp.rejection_reason}
              <div class="detail-cell full-width rejection-callout">
                <span class="cell-label">{t('company.field.rejectionReason')}</span>
                <span class="cell-val danger-text">{comp.rejection_reason}</span>
              </div>
            {/if}
          </div>

          <!-- Row 3: Income Statement Audit Strip -->
          <div class="audit-strip">
            <div class="audit-left">
              <Icon name="income-statement" />
              <div class="doc-meta">
                <span class="doc-title">{t('company.field.incomeStatement')}</span>
                <span class="doc-url">{comp.income_statement_url || 'Financial statement attached'}</span>
              </div>
            </div>
            <Button variant="secondary" size="sm" onclick={() => openDocument(comp)}>
              <Icon name="eye" />
              <span>{t('company.action.viewDocument')}</span>
            </Button>
          </div>

          <!-- Row 4: Action Tray -->
          {#if !comp.verified && comp.verification_status !== 'REJECTED'}
            {#if rejectingId === comp.id}
              <div class="reject-box">
                <label for={`reject-reason-${comp.id}`} class="cell-label">
                  Specify audit rejection reason (communicated to applicant):
                </label>
                <textarea
                  id={`reject-reason-${comp.id}`}
                  class="rejection-textarea"
                  rows={2}
                  value={rejectionDrafts[comp.id] || ''}
                  oninput={(e) => {
                    rejectionDrafts = { ...rejectionDrafts, [comp.id]: (e.target as HTMLTextAreaElement).value };
                  }}
                  placeholder="e.g. Income statement does not match GSTIN turnover or entity name."
                ></textarea>
                <div class="reject-actions">
                  <Button variant="danger" size="sm" onclick={() => handleConfirmReject(comp)}>
                    <span>{t('company.action.confirmReject')}</span>
                  </Button>
                  <Button variant="ghost" size="sm" onclick={() => (rejectingId = null)}>
                    <span>{t('company.action.cancel')}</span>
                  </Button>
                </div>
              </div>
            {:else}
              <div class="card-actions">
                <Button
                  variant="primary"
                  size="md"
                  onclick={() => handleApprove(comp)}
                  disabled={acting[comp.id]}
                >
                  <Icon name="check" />
                  <span>{t('company.action.approve')}</span>
                </Button>
                <Button
                  variant="danger"
                  size="md"
                  onclick={() => (rejectingId = comp.id)}
                  disabled={acting[comp.id]}
                >
                  <Icon name="close" />
                  <span>{t('company.action.reject')}</span>
                </Button>
              </div>
            {/if}
          {:else if comp.verified}
            <div class="verified-footer">
              <Icon name="success" />
              <span>Verified by {comp.verified_by || 'Ministry Officer'} on {comp.verified_at ? comp.verified_at.slice(0, 10) : '2026-08-12'}. Commission schedule active.</span>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  {/if}

  <!-- Income Statement Document Inspector Dialog -->
  {#if docModalOpen}
    <Dialog bind:open={docModalOpen} title={`${selectedCompanyName} — Income Statement`}>
      <div class="doc-modal-content">
        <div class="doc-viewer-placeholder">
          <Icon name="income-statement" />
          <h3>Legitimacy &amp; Turnover Verification</h3>
          <p class="doc-url-line">Document URL: <code class="doc-url-code">{selectedDocUrl}</code></p>
          <div class="audit-checklist">
            <h4>Mandatory Audit Checklist:</h4>
            <ul>
              <li>GSTIN matches Ministry of Corporate Affairs / GST portal records</li>
              <li>Entity turnover exceeds ₹20 Lakhs per annum</li>
              <li>Authorized signatory matches contact person on profile</li>
            </ul>
          </div>
          <div class="doc-modal-actions">
            <a href={selectedDocUrl} target="_blank" rel="noopener noreferrer" class="doc-external-link">
              <Icon name="download" />
              <span>Download Full Statement PDF</span>
            </a>
            <Button variant="secondary" size="md" onclick={() => (docModalOpen = false)}>
              <span>Close Inspector</span>
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  {/if}
</section>
{/if}

<style>
  @layer components {
    .restricted-box {
      max-width: 600px;
      margin-inline: auto;
      margin-block: 4rem;
      padding: 2rem;
      border: 1px solid var(--k-border-hairline);
      text-align: center;
    }

    .companies-view {
      max-width: 1200px;
      margin-inline: auto;
      padding-inline: 1.5rem;
      padding-block: 1.5rem 4rem;
    }

    .kicker {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--k-terracotta);
      font-weight: 600;
      margin-block-end: 0.25rem;
    }

    .heading-row {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-block-end: 1rem;
      gap: 1rem;
      flex-wrap: wrap;
    }

    .title {
      font-family: var(--k-font-display);
      font-size: 1.75rem;
      color: var(--k-ink);
      margin: 0;
      line-height: 1.2;
    }

    .subtitle {
      font-size: 0.875rem;
      color: var(--k-text-muted);
      margin-block-start: 0.25rem;
      margin-block-end: 0;
    }

    /* Commission Schedule Banner */
    .commission-banner {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: color-mix(in srgb, var(--k-haldi) 12%, transparent);
      border: 1px solid color-mix(in srgb, var(--k-haldi) 30%, transparent);
      margin-block-end: 1.25rem;
      font-size: 0.8125rem;
      color: var(--k-ink);
    }

    .banner-strong {
      font-weight: 600;
      margin-inline-end: 0.25rem;
    }

    /* Metrics Strip */
    .stats-strip {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 0.75rem;
      margin-block-end: 1.5rem;
    }

    .stat-card {
      padding: 0.75rem 1rem;
      background: var(--k-surface-raised, #ffffff);
      border: var(--k-border-hairline);
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }

    .stat-card.stat-pending {
      border-inline-start: 3px solid var(--k-haldi);
    }

    .stat-card.stat-verified {
      border-inline-start: 3px solid var(--k-neem);
    }

    .stat-card.stat-treasury {
      border-inline-start: 3px solid var(--k-terracotta);
    }

    .stat-label {
      font-size: 0.6875rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--k-text-muted);
    }

    .stat-value {
      font-size: 1.25rem;
      font-weight: 700;
      color: var(--k-ink);
      font-variant-numeric: tabular-nums;
    }

    .stat-money {
      font-size: 1.125rem;
    }

    /* Tabs */
    .tabs {
      display: flex;
      gap: 0.5rem;
      border-block-end: var(--k-border-hairline);
      margin-block-end: 1.5rem;
      overflow-x: auto;
    }

    .tab-btn {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      background: none;
      border: none;
      border-block-end: 2px solid transparent;
      color: var(--k-text-muted);
      font-size: 0.875rem;
      cursor: pointer;
      font-weight: 500;
      transition: color var(--k-dur-fast) ease;
    }

    .tab-btn:hover {
      color: var(--k-ink);
    }

    .tab-btn.active {
      color: var(--k-ink);
      border-block-end-color: var(--k-terracotta);
      font-weight: 600;
    }

    .badge {
      font-size: 0.6875rem;
      padding: 0.125rem 0.375rem;
      border-radius: 2px;
    }

    .badge-pending {
      background: color-mix(in srgb, var(--k-haldi) 20%, transparent);
      color: var(--k-ink);
    }

    .badge-verified {
      background: color-mix(in srgb, var(--k-neem) 20%, transparent);
      color: var(--k-neem);
    }

    /* Company Card (Line & Space separation, NO floating shadows) */
    .company-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .company-card {
      background: var(--k-surface-raised, #ffffff);
      border: var(--k-border-hairline);
      padding: 1.25rem;
      display: flex;
      flex-direction: column;
      gap: 1rem;
      position: relative;
    }

    .company-card.is-verified {
      border-inline-start: 3px solid var(--k-neem);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      flex-wrap: wrap;
    }

    .title-group {
      display: flex;
      flex-direction: column;
      gap: 0.375rem;
    }

    .company-name {
      font-family: var(--k-font-display);
      font-size: 1.25rem;
      color: var(--k-ink);
      margin: 0;
      line-height: 1.25;
    }

    .badge-row {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex-wrap: wrap;
    }

    .type-pill {
      font-size: 0.6875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      padding: 0.2rem 0.5rem;
      background: color-mix(in srgb, var(--k-ink) 8%, transparent);
      color: var(--k-ink);
    }

    .rate-pill {
      font-size: 0.6875rem;
      font-weight: 600;
      padding: 0.2rem 0.5rem;
      background: color-mix(in srgb, var(--k-terracotta) 12%, transparent);
      color: var(--k-terracotta);
    }

    .status-pill {
      font-size: 0.6875rem;
      display: inline-flex;
      align-items: center;
      gap: 0.25rem;
      padding: 0.2rem 0.5rem;
    }

    .status-pill.pending {
      background: color-mix(in srgb, var(--k-haldi) 15%, transparent);
      color: var(--k-ink);
    }

    .status-pill.verified {
      background: color-mix(in srgb, var(--k-neem) 15%, transparent);
      color: var(--k-neem);
      font-weight: 600;
    }

    .status-pill.rejected {
      background: color-mix(in srgb, var(--k-madder) 15%, transparent);
      color: var(--k-madder);
    }

    .revenue-summary {
      text-align: right;
      display: flex;
      flex-direction: column;
      gap: 0.125rem;
    }

    .revenue-label {
      font-size: 0.6875rem;
      text-transform: uppercase;
      color: var(--k-text-muted);
    }

    .revenue-val {
      font-size: 1rem;
      font-weight: 700;
      color: var(--k-ink);
    }

    .fee-val {
      font-size: 0.75rem;
      color: var(--k-terracotta);
      font-weight: 500;
    }

    /* Details Grid */
    .details-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 0.75rem;
      background: color-mix(in srgb, var(--k-khadi) 40%, transparent);
      padding: 0.75rem;
      border: var(--k-border-hairline);
    }

    .detail-cell {
      display: flex;
      flex-direction: column;
      gap: 0.125rem;
    }

    .detail-cell.full-width {
      grid-column: 1 / -1;
    }

    .cell-label {
      font-size: 0.6875rem;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: var(--k-text-muted);
    }

    .cell-val {
      font-size: 0.8125rem;
      color: var(--k-ink);
    }

    .gstin-val {
      font-family: monospace;
      font-weight: 600;
    }

    .danger-text {
      color: var(--k-madder);
      font-weight: 500;
    }

    .rejection-callout {
      border-inline-start: 2px solid var(--k-madder);
      padding-inline-start: 0.5rem;
    }

    /* Audit Strip */
    .audit-strip {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 1rem;
      padding: 0.625rem 0.875rem;
      background: color-mix(in srgb, var(--k-indigo) 8%, transparent);
      border: 1px dashed color-mix(in srgb, var(--k-indigo) 25%, transparent);
    }

    .audit-left {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      min-width: 0;
    }

    .doc-meta {
      display: flex;
      flex-direction: column;
      min-width: 0;
    }

    .doc-title {
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--k-ink);
    }

    .doc-url {
      font-size: 0.6875rem;
      color: var(--k-text-muted);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    /* Action Tray */
    .card-actions {
      display: flex;
      gap: 0.75rem;
      align-items: center;
    }

    .reject-box {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      background: color-mix(in srgb, var(--k-madder) 6%, transparent);
      padding: 0.75rem;
      border: 1px solid color-mix(in srgb, var(--k-madder) 25%, transparent);
    }

    .rejection-textarea {
      width: 100%;
      box-sizing: border-box;
      padding: 0.5rem;
      font-size: 0.8125rem;
      border: var(--k-border-hairline);
      background: var(--k-surface-raised, #ffffff);
      color: var(--k-ink);
      font-family: inherit;
    }

    .reject-actions {
      display: flex;
      gap: 0.5rem;
    }

    .verified-footer {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.75rem;
      color: var(--k-neem);
      font-weight: 500;
    }

    /* Doc Modal */
    .doc-modal-content {
      padding: 1.5rem;
      max-width: 100%;
      overflow-x: hidden;
    }

    .doc-viewer-placeholder {
      display: flex;
      flex-direction: column;
      align-items: center;
      text-align: center;
      gap: 0.75rem;
      padding: 2rem;
      border: 1px dashed var(--k-border-hairline);
      background: var(--k-khadi);
      max-width: 100%;
      box-sizing: border-box;
      overflow-x: hidden;
    }

    .doc-url-line {
      max-width: 100%;
      overflow-wrap: anywhere;
    }

    .doc-url-code {
      word-break: break-all;
      overflow-wrap: anywhere;
    }

    .audit-checklist {
      text-align: left;
      width: 100%;
      max-width: 480px;
      margin-block: 1rem;
      font-size: 0.8125rem;
    }

    .audit-checklist h4 {
      margin-block-end: 0.5rem;
      color: var(--k-ink);
    }

    .audit-checklist ul {
      padding-inline-start: 1.25rem;
      margin: 0;
      color: var(--k-text-muted);
    }

    .doc-modal-actions {
      display: flex;
      gap: 1rem;
      align-items: center;
      margin-block-start: 1rem;
    }

    .doc-external-link {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      background: var(--k-terracotta);
      color: var(--k-surface, #ffffff);
      text-decoration: none;
      font-size: 0.875rem;
      font-weight: 500;
    }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 0.75rem;
      padding: 3rem 1.5rem;
      border: var(--k-border-hairline);
      color: var(--k-text-muted);
    }
  }
</style>
