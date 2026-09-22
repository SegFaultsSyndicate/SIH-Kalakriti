<!--
  apps/artisan/src/routes/trends/+page.svelte

  Dual-Curated Market Trends & Social Inspiration Board.
  Provides master artisans with actionable market intelligence curated by BOTH:
  1. Ministry & Cluster Development Officers (identifying export & seasonal demand)
  2. Fellow Artisan Contributors (sharing viral Instagram, Pinterest & boutique trends)

  Mobile-first (360px optimized), voice-first with screen readout for low literacy.
  Zero emoji, strict hairline separation, Svelte 5 runes.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import {
    Button,
    Input,
    FieldGroup,
    Select,
    Dialog,
    SpeakButton,
    Skeleton,
    showToast,
  } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    listTrendLinks,
    createTrendLink,
    pinTrendLink,
    type TrendLink,
    type CreateTrendLinkBody,
  } from '@kalakriti/api';

  const t = $derived(locale.t);
  const userRole = $derived(session.claims?.['role'] as string | undefined);
  const isOfficer = $derived(userRole === 'CLUSTER_OFFICER' || userRole === 'MINISTRY');

  type SourceFilter = 'ALL' | 'MINISTRY' | 'ARTISAN' | 'INSTAGRAM' | 'PINTEREST';
  let activeFilter = $state<SourceFilter>('ALL');

  let trends = $state<TrendLink[]>([]);
  let loading = $state(false);

  // Modal State for New Trend Contribution
  let submitModalOpen = $state(false);
  let submitting = $state(false);
  let newUrl = $state('');
  let newTitle = $state('');
  let newSourceType = $state<'INSTAGRAM' | 'PINTEREST' | 'YOUTUBE' | 'BLOG' | 'NEWS' | 'OTHER'>('INSTAGRAM');
  let newCraftId = $state('craft-bagru');

  const CRAFT_SELECT_OPTIONS = [
    { value: 'craft-bagru', label: 'Bagru Block Print (Rajasthan)' },
    { value: 'craft-sanganeri', label: 'Sanganeri Print (Rajasthan)' },
    { value: 'craft-chanderi', label: 'Chanderi Weaving (Madhya Pradesh)' },
    { value: 'craft-banarasi-brocade', label: 'Banarasi Brocade (Uttar Pradesh)' },
    { value: 'craft-paithani', label: 'Paithani Silk (Maharashtra)' },
    { value: 'craft-madhubani', label: 'Madhubani Painting (Bihar)' },
    { value: 'craft-kutch-embroidery', label: 'Kutch Embroidery (Gujarat)' },
  ];

  const SOURCE_TYPE_OPTIONS = [
    { value: 'INSTAGRAM', label: 'Instagram Reel / Post' },
    { value: 'PINTEREST', label: 'Pinterest Moodboard / Pin' },
    { value: 'YOUTUBE', label: 'YouTube Short / Video' },
    { value: 'OTHER', label: 'Boutique / Design Article' },
  ];

  const MOCK_TRENDS: TrendLink[] = [
    {
      id: 'trend-01',
      title: 'Earth Tones & Minimalist Ajrakh in European Summer Collections',
      url: 'https://instagram.com/p/C_heritage_weaves_2026',
      source_type: 'INSTAGRAM',
      thumbnail_url: 'https://cdn.kalakriti.org.in/trends/ajrakh_minimalist.webp',
      craft_id: 'craft-bagru',
      description: 'Natural dyes, earth-tones, and resort wear silhouettes in rising boutique demand.',
      created_by: 'MINISTRY_OFFICER_01',
      curator_name: 'Meera Rao (Ministry Sourcing Advisor)',
      curator_role: 'MINISTRY',
      pinned: true,
      created_at: '2026-09-10T08:00:00Z',
    },
    {
      id: 'trend-02',
      title: 'Contemporary Zardozi Trim on Khadi Festive Co-ord Sets',
      url: 'https://pinterest.com/pin/zardozi_festive_coords_2026',
      source_type: 'PINTEREST',
      thumbnail_url: 'https://cdn.kalakriti.org.in/trends/zardozi_coords.webp',
      craft_id: 'craft-banarasi-brocade',
      description: 'Zardozi borders on everyday khadi co-ord sets trending across domestic boutiques.',
      created_by: 'ARTISAN_108',
      curator_name: 'Rajesh Singhania (Master Weaver, Varanasi)',
      curator_role: 'ARTISAN',
      pinned: true,
      created_at: '2026-09-12T11:30:00Z',
    },
    {
      id: 'trend-03',
      title: 'Handcrafted Madhubani Table Linens & Ceramic Accents',
      url: 'https://instagram.com/p/D_mithila_home_interiors',
      source_type: 'INSTAGRAM',
      thumbnail_url: 'https://cdn.kalakriti.org.in/trends/madhubani_home.webp',
      craft_id: 'craft-madhubani',
      description: 'Dining linens and ceramic accents using traditional Mithila natural pigments.',
      created_by: 'ARTISAN_042',
      curator_name: 'Sita Devi (Folk Artist, Madhubani)',
      curator_role: 'ARTISAN',
      pinned: false,
      created_at: '2026-09-13T14:15:00Z',
    },
    {
      id: 'trend-04',
      title: 'Pastel Chanderi Saris for Corporate Handloom Fridays',
      url: 'https://youtube.com/shorts/chanderi_pastel_styling',
      source_type: 'YOUTUBE',
      thumbnail_url: 'https://cdn.kalakriti.org.in/trends/chanderi_pastels.webp',
      craft_id: 'craft-chanderi',
      description: 'Pastel palette Chanderi weaves gaining traction for urban corporate workwear.',
      created_by: 'MINISTRY_OFFICER_03',
      curator_name: 'Dr. Sunita Patel (Cluster Dev Officer, MP)',
      curator_role: 'MINISTRY',
      pinned: false,
      created_at: '2026-09-08T09:45:00Z',
    },
  ];

  async function loadTrends(): Promise<void> {
    loading = true;
    try {
      const res = await listTrendLinks({ exclude_expired: true });
      if (res.trend_links && res.trend_links.length > 0) {
        trends = res.trend_links;
      } else if (import.meta.env.VITE_USE_MOCKS === '1') {
        trends = MOCK_TRENDS;
      } else {
        trends = [];
      }
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] loadTrends:', cause);
        trends = MOCK_TRENDS;
      } else {
        trends = [];
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void loadTrends();
  });

  const filteredTrends = $derived.by(() => {
    if (activeFilter === 'ALL') return trends;
    if (activeFilter === 'MINISTRY') {
      return trends.filter((t) => t.curator_role === 'MINISTRY' || t.curator_role === 'CLUSTER_OFFICER');
    }
    if (activeFilter === 'ARTISAN') {
      return trends.filter((t) => t.curator_role === 'ARTISAN' || !t.curator_role);
    }
    if (activeFilter === 'INSTAGRAM') {
      return trends.filter((t) => t.source_type === 'INSTAGRAM');
    }
    if (activeFilter === 'PINTEREST') {
      return trends.filter((t) => t.source_type === 'PINTEREST');
    }
    return trends;
  });

  async function handleTogglePin(trend: TrendLink): Promise<void> {
    const next = !trend.pinned;
    try {
      await pinTrendLink(trend.id, next);
      trends = trends.map((item) => (item.id === trend.id ? { ...item, pinned: next } : item));
      showToast({
        message: next ? t('trends.toast.saved') : t('trends.toast.shared'),
        variant: 'info',
      });
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] pinTrendLink:', cause);
        trends = trends.map((item) => (item.id === trend.id ? { ...item, pinned: next } : item));
      } else {
        showToast({ message: t('api.error.unknown'), variant: 'error' });
      }
    }
  }

  async function handleCreateTrend(): Promise<void> {
    if (!newUrl.trim() || !newTitle.trim()) {
      showToast({ message: 'Please provide both URL and title', variant: 'error' });
      return;
    }

    submitting = true;
    try {
      const body: CreateTrendLinkBody = {
        title: newTitle.trim(),
        url: newUrl.trim(),
        source_type: newSourceType,
        craft_id: newCraftId,
        description: `Curated trend for ${newCraftId}`,
      };

      const created = await createTrendLink(body);
      trends = [created, ...trends];
      submitModalOpen = false;
      newUrl = '';
      newTitle = '';

      showToast({
        message: t('trends.toast.shared'),
        variant: 'success',
      });
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] createTrendLink:', cause);
        const mockCreated: TrendLink = {
          id: 'trend-' + Date.now().toString(36),
          title: newTitle.trim(),
          url: newUrl.trim(),
          source_type: newSourceType,
          craft_id: newCraftId,
          curator_name: 'You (Artisan Contributor)',
          curator_role: 'ARTISAN',
          pinned: false,
          created_at: new Date().toISOString(),
        };
        trends = [mockCreated, ...trends];
        submitModalOpen = false;
        showToast({
          message: t('trends.toast.saved'),
          variant: 'info',
        });
      } else {
        showToast({ message: t('api.error.unknown'), variant: 'error' });
      }
    } finally {
      submitting = false;
    }
  }

  function getDomain(url: string): string {
    try {
      return new URL(url).hostname.replace('www.', '');
    } catch {
      return url;
    }
  }

  function getLocalizedTitle(item: TrendLink): string {
    if (item.id === 'trend-01') return t('trends.item1.title');
    if (item.id === 'trend-02') return t('trends.item2.title');
    if (item.id === 'trend-03') return t('trends.item3.title');
    if (item.id === 'trend-04') return t('trends.item4.title');
    return item.title;
  }

  function getLocalizedDesc(item: TrendLink): string | undefined {
    if (item.id === 'trend-01') return t('trends.item1.desc');
    if (item.id === 'trend-02') return t('trends.item2.desc');
    if (item.id === 'trend-03') return t('trends.item3.desc');
    if (item.id === 'trend-04') return t('trends.item4.desc');
    return item.description;
  }

  function getLocalizedCurator(item: TrendLink): string {
    if (item.id === 'trend-01') return t('trends.item1.curator');
    if (item.id === 'trend-02') return t('trends.item2.curator');
    if (item.id === 'trend-03') return t('trends.item3.curator');
    if (item.id === 'trend-04') return t('trends.item4.curator');
    return item.curator_name || t('trends.role.communityMember');
  }

  function getLocalizedRole(role?: string): string {
    if (role === 'MINISTRY' || role === 'CLUSTER_OFFICER') {
      return t('trends.role.ministryAdvisor');
    }
    return t('trends.role.artisanContributor');
  }
</script>

<svelte:head>
  <title>Market Trends &amp; Boutique Inspiration — Kalakriti</title>
</svelte:head>

<section class="trends-view" aria-labelledby="trends-heading">
  <!-- Header with Editorial Kicker & Voice Readout -->
  <header class="trends-header">
    <div class="header-top">
      <span class="kicker">{t('trends.kicker')}</span>
      <SpeakButton class="intro-speak" label={t('trends.intro.speakLabel')} text={t('trends.intro.speakText')} />
    </div>

    <div class="heading-row">
      <div>
        <h1 id="trends-heading" class="title">{t('trends.title')}</h1>
        <p class="subtitle">{t('trends.subtitle')}</p>
      </div>

      <Button variant="primary" size="md" onclick={() => (submitModalOpen = true)}>
        <Icon name="plus" />
        <span>{t('trends.submit')}</span>
      </Button>
    </div>

    <!-- Dual Curation Governance Callout -->
    <div class="curation-banner" role="region" aria-label={t('trends.curation.policyLabel')}>
      <Icon name="info" />
      <div class="banner-text">
        <strong>{t('trends.curation.policyLabel')}</strong>
        <span>{t('trends.curation.policyDesc')}</span>
      </div>
    </div>

    <!-- Filter Pills Navigation -->
    <div class="filter-strip" role="toolbar" aria-label={t('trends.title')}>
      <button
        type="button"
        class="filter-pill"
        class:is-active={activeFilter === 'ALL'}
        onclick={() => (activeFilter = 'ALL')}
      >
        {t('trends.filter.all', { count: String(trends.length) })}
      </button>
      <button
        type="button"
        class="filter-pill"
        class:is-active={activeFilter === 'MINISTRY'}
        onclick={() => (activeFilter = 'MINISTRY')}
      >
        {t('trends.filter.ministry')}
      </button>
      <button
        type="button"
        class="filter-pill"
        class:is-active={activeFilter === 'ARTISAN'}
        onclick={() => (activeFilter = 'ARTISAN')}
      >
        {t('trends.filter.artisan')}
      </button>
      <button
        type="button"
        class="filter-pill"
        class:is-active={activeFilter === 'INSTAGRAM'}
        onclick={() => (activeFilter = 'INSTAGRAM')}
      >
        {t('trends.filter.instagram')}
      </button>
      <button
        type="button"
        class="filter-pill"
        class:is-active={activeFilter === 'PINTEREST'}
        onclick={() => (activeFilter = 'PINTEREST')}
      >
        {t('trends.filter.pinterest')}
      </button>
    </div>
  </header>

  <!-- Trends Cards Grid -->
  {#if loading}
    <div class="loading-grid">
      <Skeleton shape="card" />
      <Skeleton shape="card" />
    </div>
  {:else if filteredTrends.length === 0}
    <div class="empty-box">
      <Icon name="link" />
      <p>{t('trends.filter.empty')}</p>
      <Button variant="secondary" size="md" onclick={() => (activeFilter = 'ALL')}>
        <span>{t('trends.filter.viewAll')}</span>
      </Button>
    </div>
  {:else}
    <div class="trends-grid" role="feed" aria-label="Market trends feed">
      {#each filteredTrends as item (item.id)}
        <article class="trend-card" class:is-pinned={item.pinned}>
          <!-- Top Row: Source Platform & Pinned Badge -->
          <div class="card-meta-top">
            <span class="source-tag {item.source_type.toLowerCase()}">
              {item.source_type}
            </span>

            <div class="meta-actions">
              {#if item.pinned}
                <span class="pinned-tag">
                  <Icon name="check" />
                  <span>{t('trends.clusterPriority')}</span>
                </span>
              {/if}

              {#if isOfficer}
                <button
                  type="button"
                  class="pin-toggle-btn"
                  onclick={() => void handleTogglePin(item)}
                  title={item.pinned ? 'Unpin from board' : 'Pin to top'}
                >
                  <Icon name="edit" />
                </button>
              {/if}
            </div>
          </div>

          <!-- Trend Title & Link -->
          <h2 class="trend-title">
            <a href={item.url} target="_blank" rel="noopener noreferrer" class="title-link">
              <span>{getLocalizedTitle(item)}</span>
              <Icon name="external-link" />
            </a>
          </h2>

          <!-- Voice Spoken Explanation for Low-Literacy Artisans -->
          <div class="voice-row">
            <SpeakButton
              label={t('trends.readAloud')}
              text={`${getLocalizedTitle(item)}. ${getLocalizedCurator(item)}. ${t('trends.sourceLabel', { domain: getDomain(item.url) })}`}
            />
            <span class="domain-text">{t('trends.sourceLabel', { domain: getDomain(item.url) })}</span>
          </div>

          <!-- Curator Badge (Dual Curation Visibility) -->
          <div class="curator-strip">
            <div class="curator-avatar" class:is-officer={item.curator_role === 'MINISTRY'}>
              <Icon name={item.curator_role === 'MINISTRY' ? 'cluster' : 'user'} />
            </div>
            <div class="curator-info">
              <span class="curator-role-label">
                {getLocalizedRole(item.curator_role)}
              </span>
              <span class="curator-name">{getLocalizedCurator(item)}</span>
            </div>
          </div>

          <!-- Description Strip -->
          {#if getLocalizedDesc(item)}
            <p class="trend-description">{getLocalizedDesc(item)}</p>
          {/if}

          <!-- Direct Production CTA -->
          <div class="card-actions">
            <a href="/listing/new/capture" class="k-btn k-btn--primary">
              <Icon name="camera" />
              <span>{t('trends.craftSimilar')}</span>
            </a>
            <a href={item.url} target="_blank" rel="noopener noreferrer" class="k-btn k-btn--secondary">
              <Icon name="link" />
              <span>{t('trends.viewSocialPost')}</span>
            </a>
          </div>
        </article>
      {/each}
    </div>
  {/if}

  <!-- Share Trend Link Dialog -->
  {#if submitModalOpen}
    <Dialog bind:open={submitModalOpen} title={t('trends.modal.heading')}>
      <div class="modal-body">
        <p class="modal-intro">
          {t('trends.modal.intro')}
        </p>

        <div class="modal-field-stack">
          <FieldGroup label={t('trends.modal.urlLabel')} description={t('trends.modal.urlDesc')}>
            {#snippet children({ id, describedBy })}
              <Input
                {id}
                type="url"
                aria-describedby={describedBy}
                placeholder="https://instagram.com/p/..."
                bind:value={newUrl}
              />
            {/snippet}
          </FieldGroup>

          <FieldGroup label={t('trends.modal.headlineLabel')} description={t('trends.modal.headlineDesc')}>
            {#snippet children({ id, describedBy })}
              <Input
                {id}
                aria-describedby={describedBy}
                placeholder={t('trends.modal.headlinePlaceholder')}
                bind:value={newTitle}
              />
            {/snippet}
          </FieldGroup>

          <div class="modal-two-col">
            <FieldGroup label={t('trends.modal.platformLabel')}>
              {#snippet children({ id, describedBy })}
                <Select
                  {id}
                  aria-describedby={describedBy}
                  options={SOURCE_TYPE_OPTIONS}
                  bind:value={newSourceType}
                />
              {/snippet}
            </FieldGroup>

            <FieldGroup label={t('trends.modal.craftLabel')}>
              {#snippet children({ id, describedBy })}
                <Select
                  {id}
                  aria-describedby={describedBy}
                  options={CRAFT_SELECT_OPTIONS}
                  bind:value={newCraftId}
                />
              {/snippet}
            </FieldGroup>
          </div>
        </div>

        <div class="modal-actions">
          <Button variant="secondary" size="md" onclick={() => (submitModalOpen = false)}>
            <span>{t('company.action.cancel')}</span>
          </Button>
          <Button
            variant="primary"
            size="md"
            disabled={submitting || !newUrl.trim() || !newTitle.trim()}
            onclick={handleCreateTrend}
          >
            {#if submitting}
              <span>{t('trends.modal.sharing')}</span>
            {:else}
              <span>{t('trends.modal.publish')}</span>
            {/if}
          </Button>
        </div>
      </div>
    </Dialog>
  {/if}
</section>

<style>
  /* Unlayered on purpose: SpeakButton's own styles are unlayered and would beat a rule in @layer. */
  /* One line, never a two-line pill squeezed beside the source label. */
  .voice-row :global(.k-speak) {
    white-space: nowrap;
    font-size: var(--k-text-sm);
  }

  .header-top :global(.intro-speak) {
    font-size: var(--k-text-sm);
  }

  @layer components {
    .trends-view {
      max-width: 900px;
      margin-inline: auto;
      padding-inline: 1rem;
      padding-block: 1.5rem 5rem;
      max-inline-size: 100%;
      box-sizing: border-box;
      overflow-x: hidden;
    }

    .trends-header {
      margin-block-end: 2rem;
      border-block-end: 1px solid var(--k-border-hairline);
      padding-block-end: 1.5rem;
      max-inline-size: 100%;
    }

    .header-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-block-end: 0.25rem;
      max-inline-size: 100%;
    }

    .kicker {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--k-terracotta);
      font-weight: 600;
    }

    .heading-row {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      margin-block-end: 1.25rem;
      flex-wrap: wrap;
      max-inline-size: 100%;
    }

    @media (max-width: 600px) {
      .heading-row {
        flex-direction: column;
      }
    }

    .title {
      font-size: clamp(1.4rem, 4vw, 1.9rem);
      font-weight: 600;
      color: var(--k-ink);
      line-height: 1.25;
      margin-block-end: 0.35rem;
      overflow-wrap: break-word;
    }

    .subtitle {
      font-size: 0.9375rem;
      color: var(--k-text-muted);
      line-height: 1.5;
      max-width: 650px;
      overflow-wrap: break-word;
    }

    .curation-banner {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.875rem 1rem;
      background-color: var(--k-surface-raised);
      border: 1px solid var(--k-border-hairline);
      font-size: 0.875rem;
      margin-block-end: 1.25rem;
      max-inline-size: 100%;
      box-sizing: border-box;
    }

    .banner-text strong {
      color: var(--k-ink);
      margin-inline-end: 0.35rem;
    }

    .banner-text span {
      color: var(--k-text-muted);
    }

    /* Filter Toolbar */
    .filter-strip {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
      max-inline-size: 100%;
    }

    .filter-pill {
      border: 1px solid var(--k-border-hairline);
      background-color: var(--k-surface);
      color: var(--k-text-muted);
      font-size: 0.8125rem;
      font-weight: 500;
      padding: 0.4rem 0.85rem;
      cursor: pointer;
      min-block-size: 38px;
      transition: all var(--k-dur-fast) var(--k-ease);
      max-inline-size: 100%;
    }

    .filter-pill:hover {
      border-color: var(--k-ink);
      color: var(--k-ink);
    }

    .filter-pill.is-active {
      background-color: var(--k-ink);
      border-color: var(--k-ink);
      color: var(--k-khadi);
      font-weight: 600;
    }

    /* Trends Grid */
    .trends-grid {
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
      max-inline-size: 100%;
    }

    .trend-card {
      border: 1px solid var(--k-border-hairline);
      background-color: var(--k-surface);
      padding: 1.5rem;
      display: flex;
      flex-direction: column;
      gap: 1rem;
      max-inline-size: 100%;
      box-sizing: border-box;
      overflow-x: hidden;
    }

    .trend-card.is-pinned {
      border-color: var(--k-terracotta);
      border-inline-start-width: 4px;
    }

    .card-meta-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .source-tag {
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      padding: 0.2rem 0.5rem;
      border: 1px solid var(--k-border-hairline);
    }

    .source-tag.instagram {
      color: var(--k-madder);
      border-color: var(--k-madder);
    }

    .source-tag.pinterest {
      color: var(--k-terracotta);
      border-color: var(--k-terracotta);
    }

    .source-tag.youtube {
      color: var(--k-danger);
      border-color: var(--k-danger);
    }

    .meta-actions {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .pinned-tag {
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      font-size: 0.6875rem;
      font-weight: 700;
      letter-spacing: 0.06em;
      color: var(--k-terracotta);
    }

    .pin-toggle-btn {
      border: none;
      background: transparent;
      cursor: pointer;
      color: var(--k-text-muted);
      padding: 0.25rem;
    }

    .trend-title {
      font-size: 1.15rem;
      font-weight: 600;
      line-height: 1.35;
      margin: 0;
    }

    .title-link {
      color: var(--k-ink);
      text-decoration: none;
      display: inline-flex;
      align-items: baseline;
      gap: 0.5rem;
    }

    .title-link:hover {
      color: var(--k-terracotta);
      text-decoration: underline;
    }

    .voice-row {
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 0.5rem 0.75rem;
    }

    .domain-text {
      font-size: 0.8125rem;
      color: var(--k-text-muted);
    }

    /* Curator Strip */
    .curator-strip {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.65rem 0.85rem;
      border: 1px solid var(--k-border-hairline);
      background-color: var(--k-surface-raised);
    }

    .curator-avatar {
      inline-size: 2rem;
      block-size: 2rem;
      border-radius: 50%;
      border: 1px solid var(--k-border-hairline);
      display: flex;
      align-items: center;
      justify-content: center;
      color: var(--k-terracotta);
    }

    .curator-avatar.is-officer {
      color: var(--k-indigo);
      border-color: var(--k-indigo);
    }

    .curator-info {
      display: flex;
      flex-direction: column;
      line-height: 1.25;
    }

    .curator-role-label {
      font-size: 0.6875rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--k-text-muted);
      font-weight: 600;
    }

    .curator-name {
      font-size: 0.875rem;
      font-weight: 600;
      color: var(--k-ink);
    }

    /* Tags Row */
    .tags-row {
      display: flex;
      flex-wrap: wrap;
      gap: 0.4rem;
    }

    .tag-chip {
      font-size: 0.75rem;
      color: var(--k-text-muted);
      padding: 0.2rem 0.5rem;
      border: 1px solid var(--k-border-hairline);
    }

    /* Card Actions */
    .card-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 0.75rem;
      margin-block-start: 0.5rem;
      border-block-start: 1px solid var(--k-border-hairline);
      padding-block-start: 1rem;
      max-inline-size: 100%;
    }

    .k-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.45rem;
      padding: 0.6rem 1rem;
      font-size: 0.875rem;
      font-weight: 600;
      text-decoration: none;
      cursor: pointer;
      min-block-size: 44px;
      max-inline-size: 100%;
      word-break: break-word;
    }

    .k-btn--primary {
      background-color: var(--k-terracotta);
      color: var(--k-khadi);
      border: 1px solid transparent;
    }

    .k-btn--secondary {
      background-color: var(--k-surface);
      color: var(--k-ink);
      border: 1px solid var(--k-border-hairline);
    }

    /* Modal Layout */
    .modal-body {
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
    }

    .modal-intro {
      font-size: 0.875rem;
      color: var(--k-text-muted);
      line-height: 1.5;
      margin: 0;
    }

    .modal-field-stack {
      display: flex;
      flex-direction: column;
      gap: 1.25rem;
    }

    .modal-two-col {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
    }

    @media (max-width: 500px) {
      .modal-two-col {
        grid-template-columns: 1fr;
      }
    }

    .modal-actions {
      display: flex;
      justify-content: flex-end;
      gap: 0.75rem;
      border-block-start: 1px solid var(--k-border-hairline);
      padding-block-start: 1rem;
    }

    .empty-box {
      border: 1px solid var(--k-border-hairline);
      background-color: var(--k-surface);
      padding: 3rem 1.5rem;
      text-align: center;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 1rem;
      color: var(--k-text-muted);
    }
  }
</style>
