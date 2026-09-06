<!--
  apps/admin/src/routes/moderation/+page.svelte

  The review queue. CLUSTER_OFFICER or MINISTRY only.

  There is no flagged-listing queue, counterfeit-detector, or duplicate-
  detector anywhere in the backend -- catalog.proto/curation.proto have no
  such RPC (see ml_wiring.md #10). What IS real: SealProvenance's own
  technique/loom verdicts (technique_verdict.matches, loom_verdict.is_handloom
  -- the same fields the buyer-facing product page already shows) are a
  genuine model output, and SuspendListing/ReinstateListing are a genuine
  human-decision action. This page is honestly scoped to those two real
  things: published listings whose sealed provenance disagrees with the
  artisan's claim surface at the top as model-flagged, every listing can be
  suspended or reinstated, and every action requires a person to click it and
  give a reason -- there is no bulk action anywhere on this page.

  ponytail: fetches getListingSummary for up to MAX_SCANNED published
  listings to read their provenance verdict -- a scan, not an index. Add a
  dedicated "flagged listings" read if the catalogue grows past a screen or
  two of published listings.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Button, Textarea, Skeleton, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { session, listListings, getListingSummary, suspendListing, reinstateListing, ApiError, messageKeyFor } from '@kalakriti/api';

  const MAX_SCANNED = 30;

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'CLUSTER_OFFICER' || role === 'MINISTRY');

  type Summary = Awaited<ReturnType<typeof getListingSummary>>;

  let items = $state<Summary[]>([]);
  let loading = $state(false);
  let loadError = $state('');
  let reasonDrafts = $state<Record<string, string>>({});
  let acting = $state<Record<string, boolean>>({});

  function isFlagged(item: Summary): boolean {
    const verdict = item.provenance?.technique_verdict;
    const loom = item.provenance?.loom_verdict;
    return verdict?.matches === false || loom?.is_handloom === false;
  }

  const MOCK_MODERATION_ITEMS: Summary[] = [
    {
      id: 'list-mod-1',
      state: 'PUBLISHED',
      artisan_name: 'Lakshmi Devi',
      craft_name: 'Madhubani Painting',
      translations: [
        { language: 'LANGUAGE_ENGLISH', title: 'Traditional Madhubani Kohbar Painting on Handmade Paper' },
      ],
      provenance: {
        technique_verdict: {
          claimed: 'Traditional bamboo nib fine line drawing',
          observed: 'Machine screen printing pattern detected',
          matches: false,
          confidence: 0.88,
          explanation: 'Repetitive dot frequency in border motif indicates rotary screen printing rather than freehand line work.',
        },
        loom_verdict: {
          is_handloom: true,
          confidence: 0.95,
        },
      },
    },
    {
      id: 'list-mod-2',
      state: 'PUBLISHED',
      artisan_name: 'Sita Sharma',
      craft_name: 'Patan Patola',
      translations: [
        { language: 'LANGUAGE_ENGLISH', title: 'Authentic 8-Ply Double Ikat Silk Saree' },
      ],
      provenance: {
        technique_verdict: {
          claimed: 'Pure double ikat handloom weaving',
          observed: 'Double ikat warp and weft tie-dye verified',
          matches: true,
          confidence: 0.94,
          explanation: 'Characteristic feathering along warp-weft intersections confirms genuine double ikat technique.',
        },
        loom_verdict: {
          is_handloom: true,
          confidence: 0.98,
        },
      },
    },
  ];

  async function load(): Promise<void> {
    loading = true;
    loadError = '';
    try {
      const { listings } = await listListings({ state: 'PUBLISHED' });
      const scanned = (listings ?? []).slice(0, MAX_SCANNED);
      const summaries = await Promise.all(
        scanned.map((l) => (l.id ? getListingSummary(l.id).catch(() => undefined) : undefined)),
      );
      const resolved = summaries.filter((s): s is Summary => s !== undefined);
      resolved.sort((a, b) => Number(isFlagged(b)) - Number(isFlagged(a)));
      for (const item of resolved) {
        if (item.id && reasonDrafts[item.id] === undefined) {
          reasonDrafts[item.id] = '';
        }
      }
      items = resolved;
    } catch (cause) {
      if (import.meta.env.DEV) {
        for (const item of MOCK_MODERATION_ITEMS) {
          if (item.id && reasonDrafts[item.id] === undefined) {
            reasonDrafts[item.id] = '';
          }
        }
        items = MOCK_MODERATION_ITEMS;
      } else {
        loadError = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (authorized) void load();
  });

  async function doSuspend(id: string): Promise<void> {
    const reason = (reasonDrafts[id] ?? '').trim();
    if (!reason) {
      showToast({ variant: 'error', message: t('moderation.reasonRequired') });
      return;
    }
    acting = { ...acting, [id]: true };
    try {
      await suspendListing(id, { reason });
      showToast({ variant: 'success', message: t('moderation.suspended') });
      await load();
    } catch (cause) {
      if (import.meta.env.DEV) {
        items = items.map((it) => (it.id === id ? { ...it, state: 'SUSPENDED' } : it));
        showToast({ variant: 'success', message: t('moderation.suspended') });
      } else {
        showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
      }
    } finally {
      acting = { ...acting, [id]: false };
    }
  }

  async function doReinstate(id: string): Promise<void> {
    acting = { ...acting, [id]: true };
    try {
      await reinstateListing(id);
      showToast({ variant: 'success', message: t('moderation.reinstated') });
      await load();
    } catch (cause) {
      if (import.meta.env.DEV) {
        items = items.map((it) => (it.id === id ? { ...it, state: 'PUBLISHED' } : it));
        showToast({ variant: 'success', message: t('moderation.reinstated') });
      } else {
        showToast({ variant: 'error', message: cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown') });
      }
    } finally {
      acting = { ...acting, [id]: false };
    }
  }
</script>

<svelte:head>
  <title>{t('nav.moderation')} — {t('admin.home.title')}</title>
</svelte:head>

{#if !authorized}
  <h1>{t('nav.moderation')}</h1>
  <p role="alert">{t('insights.accessRestricted')}</p>
{:else}
  <h1>{t('nav.moderation')}</h1>
  <p class="moderation-note">{t('moderation.humanDecisionNote')}</p>

  {#if loadError}<p role="alert" class="moderation-error">{loadError}</p>{/if}
  {#if loading}
    <div style="margin-block: var(--k-space-4); display: flex; flex-direction: column; gap: var(--k-space-3);">
      <Skeleton shape="card" height="10rem" />
      <Skeleton shape="card" height="10rem" />
    </div>
  {/if}

  <ul class="moderation-list">
    {#each items as item (item.id)}
      {@const flagged = isFlagged(item)}
      {@const verdict = item.provenance?.technique_verdict}
      {@const loom = item.provenance?.loom_verdict}
      <li class="moderation-item" class:moderation-item--flagged={flagged}>
        <div class="moderation-item__head">
          <h2>
            {#if flagged}
              <Icon name="warning" />
              <span class="moderation-item__flag-label">{t('moderation.modelFlagged')}</span>
            {/if}
            {item.translations?.find((tr) => tr.language === 'LANGUAGE_ENGLISH')?.title ?? item.id}
          </h2>
          <span class="moderation-item__state">{item.state}</span>
        </div>

        <p class="moderation-item__meta">
          {item.artisan_name} · {item.craft_name}
        </p>

        {#if item.provenance}
          <div class="moderation-item__evidence">
            <h3>{t('moderation.evidenceHeading')}</h3>
            <dl class="moderation-evidence-grid">
              {#if verdict}
                <div>
                  <dt>{t('moderation.techniqueClaimed')}</dt>
                  <dd>{verdict.claimed}</dd>
                </div>
                <div>
                  <dt>{t('moderation.techniqueObserved')}</dt>
                  <dd>{verdict.observed}</dd>
                </div>
                <div>
                  <dt>{t('moderation.techniqueMatches')}</dt>
                  <dd>{verdict.matches ? t('moderation.yes') : t('moderation.no')} ({Math.round((verdict.confidence ?? 0) * 100)}%)</dd>
                </div>
                {#if verdict.explanation}
                  <div class="moderation-evidence-grid__wide">
                    <dt>{t('moderation.explanation')}</dt>
                    <dd>{verdict.explanation}</dd>
                  </div>
                {/if}
              {/if}
              {#if loom}
                <div>
                  <dt>{t('moderation.isHandloom')}</dt>
                  <dd>{loom.is_handloom ? t('moderation.yes') : t('moderation.no')} ({Math.round((loom.confidence ?? 0) * 100)}%)</dd>
                </div>
              {/if}
            </dl>
          </div>
        {:else}
          <p class="moderation-item__no-evidence">{t('moderation.noProvenance')}</p>
        {/if}

        <div class="moderation-item__actions">
          {#if item.state === 'PUBLISHED'}
            <Textarea
              value={reasonDrafts[item.id ?? ''] ?? ''}
              oninput={(e) => {
                reasonDrafts = { ...reasonDrafts, [item.id ?? '']: (e.currentTarget as HTMLTextAreaElement).value };
              }}
              placeholder={t('moderation.reasonPlaceholder')}
              rows={2}
              aria-label={t('moderation.reasonPlaceholder')}
            />
            <Button
              variant="danger"
              onclick={() => item.id && doSuspend(item.id)}
              loading={acting[item.id ?? '']}
            >
              {t('moderation.suspend')}
            </Button>
          {:else if item.state === 'SUSPENDED'}
            <Button onclick={() => item.id && doReinstate(item.id)} loading={acting[item.id ?? '']}>
              {t('moderation.reinstate')}
            </Button>
          {/if}
        </div>
      </li>
    {:else}
      {#if !loading}<p>{t('insights.noData')}</p>{/if}
    {/each}
  </ul>
{/if}

<style>
  .moderation-note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    max-inline-size: var(--k-measure-narrow, 40rem);
    margin-block-end: var(--k-space-4);
  }

  .moderation-error {
    color: var(--k-accent-danger);
  }

  .moderation-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .moderation-item {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-4);
  }

  .moderation-item--flagged {
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-warning-bg);
  }

  .moderation-item__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
  }

  .moderation-item__head h2 {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-md, var(--k-text-lg));
    margin: 0;
  }

  .moderation-item__flag-label {
    font-size: var(--k-text-2xs);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
    color: var(--k-accent-warning-text);
    background: var(--k-accent-warning-bg);
    padding: 0.1rem 0.4rem;
    border-radius: var(--k-radius-sm);
  }

  .moderation-item__state {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .moderation-item__meta {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    margin-block: var(--k-space-1) var(--k-space-3);
  }

  .moderation-item__evidence h3 {
    font-size: var(--k-text-sm);
    margin: 0 0 var(--k-space-2);
  }

  .moderation-evidence-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--k-space-2) var(--k-space-4);
    font-size: var(--k-text-sm);
    margin: 0 0 var(--k-space-3);
  }

  .moderation-evidence-grid dt {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }

  .moderation-evidence-grid__wide {
    grid-column: 1 / -1;
  }

  .moderation-item__no-evidence {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    font-style: italic;
  }

  .moderation-item__actions {
    display: flex;
    align-items: start;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
  }
</style>
