<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Icon, type IconName } from '@kalakriti/icons';
  import SpeakButton from './SpeakButton.svelte';

  interface BadgeCatalogEntry {
    code: string;
    kind: 'EARNED' | 'CONFERRED';
    tier?: 'BRONZE' | 'SILVER' | 'GOLD';
    icon_name: string;
    threshold?: number;
  }

  interface Props {
    badge: BadgeCatalogEntry;
    granted: boolean;
    grantedAt?: string;
    progress?: { current: number; total: number };
    class?: string;
  }

  let { badge, granted, grantedAt, progress, class: className }: Props = $props();

  const t = $derived(locale.t);
  const iconName = $derived((granted ? badge.icon_name : 'badge-locked') as IconName);
  const name = $derived(t(`badge.${badge.code}.name` as MessageKey));
  const desc = $derived(t(`badge.${badge.code}.desc` as MessageKey));
  const tierLabel = $derived(
    badge.tier === 'BRONZE' ? t('badges.tier.bronze') :
    badge.tier === 'SILVER' ? t('badges.tier.silver') :
    badge.tier === 'GOLD' ? t('badges.tier.gold') : null,
  );
  const speakText = $derived(
    tierLabel ? `${name}. ${tierLabel}. ${desc}` : `${name}. ${desc}`,
  );
</script>

<div class="k-badge-chip {className || ''}" class:k-badge-chip--granted={granted} class:k-badge-chip--locked={!granted}>
  <Icon name={iconName} size="2rem" />
  <div class="k-badge-chip__body">
    <span class="k-badge-chip__name">
      {name}
      {#if tierLabel}<span class="k-badge-chip__tier">— {tierLabel}</span>{/if}
    </span>
    <span class="k-badge-chip__desc">{desc}</span>
    {#if granted && grantedAt}
      <span class="k-badge-chip__meta">{t('badges.grantedOn', { date: grantedAt })}</span>
    {:else if progress}
      <div class="k-badge-chip__progress" role="progressbar" aria-valuenow={progress.current} aria-valuemin={0} aria-valuemax={progress.total} aria-label={t('badges.progressLabel', { current: String(progress.current), total: String(progress.total) })}>
        <div class="k-badge-chip__progress-fill" style="width: {Math.min(100, (progress.current / progress.total) * 100)}%"></div>
      </div>
      <span class="k-badge-chip__meta">{t('badges.progressLabel', { current: String(progress.current), total: String(progress.total) })}</span>
    {:else}
      <span class="k-badge-chip__meta">{t(`badge.${badge.code}.criteria` as MessageKey, { threshold: String(badge.threshold ?? '') })}</span>
    {/if}
  </div>
  <SpeakButton text={speakText} label={t('badges.readAloud')} />
</div>

<style>
  .k-badge-chip {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    background-color: var(--k-surface);
  }

  .k-badge-chip--locked {
    color: var(--k-text-muted);
    border-style: dashed;
  }

  .k-badge-chip__body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    flex: 1;
    min-width: 0;
  }

  .k-badge-chip__name {
    font-weight: 600;
  }

  .k-badge-chip__tier {
    font-weight: 400;
    color: var(--k-text-muted);
  }

  .k-badge-chip__desc,
  .k-badge-chip__meta {
    font-size: var(--k-text-sm);
    color: var(--k-text-muted);
  }

  .k-badge-chip__progress {
    height: 6px;
    border-radius: var(--k-radius-full, 999px);
    background-color: var(--k-border-hairline);
    overflow: hidden;
  }

  .k-badge-chip__progress-fill {
    height: 100%;
    background-color: var(--k-accent);
  }
</style>
