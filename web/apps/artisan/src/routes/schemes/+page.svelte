<!-- apps/artisan/src/routes/schemes/+page.svelte -->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { SectionHeader, EmptyState, SpeakButton } from '@kalakriti/ui';
  import type { PageData } from './$types';

  interface Props {
    data: PageData;
  }
  let { data }: Props = $props();

  const t = $derived(locale.t);

  const mayQualify = $derived(
    data.matches.filter((m) => m.status === 'MATCH_STATUS_MAY_QUALIFY'),
  );
  const checkRequired = $derived(
    data.matches.filter((m) => m.status === 'MATCH_STATUS_CHECK_REQUIRED'),
  );
  const unlikely = $derived(
    data.matches.filter((m) => m.status === 'MATCH_STATUS_UNLIKELY'),
  );

  function schemeName(scheme: { name_i18n_key?: string; name_text?: string }): string {
    return scheme.name_i18n_key ? t(scheme.name_i18n_key as MessageKey) : (scheme.name_text ?? '');
  }

  function schemeSummary(scheme: { summary_i18n_key?: string; summary_text?: string }): string {
    return scheme.summary_i18n_key
      ? t(scheme.summary_i18n_key as MessageKey)
      : (scheme.summary_text ?? '');
  }

  function checkText(check: { i18n_key?: string; check_text?: string }): string {
    return check.i18n_key ? t(check.i18n_key as MessageKey) : (check.check_text ?? '');
  }

  function formatMatchedCriteria(labels: string[]): string {
    const translated = labels.map((l) => t(l as MessageKey)).join(', ');
    return t('schemes.matchedOn', { criteria: translated });
  }
</script>

<svelte:head>
  <title>{t('schemes.title')} — {t('app.name')}</title>
</svelte:head>

<main id="main-content" class="schemes-page">
  <SectionHeader heading={t('schemes.title')} kicker={t('schemes.subtitle')} />

  {#if mayQualify.length === 0 && checkRequired.length === 0}
    <EmptyState
      illustration="empty-no-search-results"
      heading={t('schemes.empty')}
      body={t('schemes.subtitle')}
    />
  {/if}

  {#if mayQualify.length > 0 || checkRequired.length > 0}
    <div class="schemes-list" role="feed" aria-label={t('schemes.title')}>
      {#each [...mayQualify, ...checkRequired] as match (match.scheme.id)}
        {@const isMayQualify = match.status === 'MATCH_STATUS_MAY_QUALIFY'}
        {@const title = schemeName(match.scheme)}
        {@const summary = schemeSummary(match.scheme)}
        <article class="scheme-card" class:scheme-card--may-qualify={isMayQualify}>
          <div class="scheme-card__header">
            <span
              class="scheme-card__status"
              class:scheme-card__status--may-qualify={isMayQualify}
              class:scheme-card__status--check-required={!isMayQualify}
            >
              {isMayQualify ? t('schemes.status.mayQualify') : t('schemes.status.checkRequired')}
            </span>

            <SpeakButton text={`${title}. ${summary}`} label={t('schemes.readAloud')} />
          </div>

          <h2 class="scheme-card__title">{title}</h2>
          <p class="scheme-card__summary">{summary}</p>

          {#if match.matched_criteria_labels && match.matched_criteria_labels.length > 0}
            <p class="scheme-card__matched">
              {formatMatchedCriteria(match.matched_criteria_labels)}
            </p>
          {/if}

          {#if match.manual_checks && match.manual_checks.length > 0}
            <div class="scheme-card__checklist">
              <h3 class="scheme-card__checklist-title">{t('schemes.manualChecklist')}</h3>
              <ul class="scheme-card__checklist-items">
                {#each match.manual_checks as check (check.id)}
                  <li class="scheme-card__check-item">
                    <label class="scheme-card__check-label">
                      <input type="checkbox" class="scheme-card__checkbox" />
                      <span>{checkText(check)}</span>
                    </label>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}

          <div class="scheme-card__footer">
            <p class="scheme-card__confirm">{t('schemes.confirmOnPortal')}</p>
            <a
              href={match.scheme.official_url}
              target="_blank"
              rel="noopener noreferrer"
              class="scheme-card__official-link"
            >
              {t('schemes.officialLink')} ↗
            </a>
          </div>
        </article>
      {/each}
    </div>
  {/if}

  {#if unlikely.length > 0}
    <details class="schemes-unlikely">
      <summary class="schemes-unlikely__summary">
        {t('schemes.status.unlikely')} ({unlikely.length})
      </summary>
      <div class="schemes-unlikely__content">
        {#each unlikely as match (match.scheme.id)}
          <div class="schemes-unlikely__row">
            <h4>{schemeName(match.scheme)}</h4>
            <p>{schemeSummary(match.scheme)}</p>
          </div>
        {/each}
      </div>
    </details>
  {/if}
</main>

<style>
  .schemes-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding-block: var(--k-space-4);
    max-inline-size: 48rem;
    margin-inline: auto;
  }

  .schemes-list {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .scheme-card {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    box-shadow: none;
  }

  .scheme-card--may-qualify {
    border-inline-start-width: var(--k-rule);
    border-inline-start-color: var(--k-accent-primary-bg);
  }

  .scheme-card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
  }

  .scheme-card__status {
    display: inline-block;
    padding-inline: var(--k-space-2);
    padding-block: var(--k-space-1);
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .scheme-card__status--may-qualify {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
    border: var(--k-hairline) solid var(--k-border-interactive);
  }

  .scheme-card__status--check-required {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    border: var(--k-hairline) solid var(--k-border-subtle);
  }

  .scheme-card__title {
    font-size: var(--k-text-lg);
    font-weight: 600;
    color: var(--k-text-primary);
    line-height: 1.3;
    margin: 0;
  }

  .scheme-card__summary {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    line-height: 1.4;
    margin: 0;
  }

  .scheme-card__matched {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    background-color: var(--k-surface-sunken);
    padding: var(--k-space-2);
    border-radius: var(--k-radius-sm);
    margin: 0;
  }

  .scheme-card__checklist {
    padding: var(--k-space-3);
    border: var(--k-hairline) dashed var(--k-border-subtle);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-base);
  }

  .scheme-card__checklist-title {
    font-size: var(--k-text-xs);
    font-weight: 600;
    color: var(--k-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0 0 var(--k-space-2) 0;
  }

  .scheme-card__checklist-items {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .scheme-card__check-label {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .scheme-card__checkbox {
    margin-block-start: 0.2rem;
    inline-size: 1rem;
    block-size: 1rem;
    cursor: pointer;
  }

  .scheme-card__footer {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-2);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-subtle);
  }

  .scheme-card__confirm {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    margin: 0;
  }

  .scheme-card__official-link {
    align-self: flex-start;
    font-size: var(--k-text-sm);
    font-weight: 600;
    color: var(--k-text-primary);
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  .scheme-card__official-link:hover {
    color: var(--k-accent-primary-bg);
  }

  .schemes-unlikely {
    border: var(--k-hairline) solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3);
    background-color: var(--k-surface-sunken);
  }

  .schemes-unlikely__summary {
    font-size: var(--k-text-sm);
    font-weight: 600;
    color: var(--k-text-secondary);
    cursor: pointer;
  }

  .schemes-unlikely__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-subtle);
  }

  .schemes-unlikely__row h4 {
    font-size: var(--k-text-sm);
    font-weight: 600;
    color: var(--k-text-secondary);
    margin: 0 0 var(--k-space-1) 0;
  }

  .schemes-unlikely__row p {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    margin: 0;
  }
</style>
