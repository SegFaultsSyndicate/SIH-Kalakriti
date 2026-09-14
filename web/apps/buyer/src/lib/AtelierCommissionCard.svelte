<!--
  apps/buyer/src/lib/AtelierCommissionCard.svelte

  Kalakriti Atelier & Bespoke Commissions.
  Allows buyers to commission one-of-a-kind heirlooms directly with master weavers.
  Built to the Kalakriti design law: hairline separation, generous whitespace,
  editorial rhythm, and wage-floor transparency.
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { SectionHeader, Button } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  let { onOpenConcierge = () => {} }: { onOpenConcierge?: () => void } = $props();

  const t = $derived(locale.t);

  const STEPS = [
    {
      num: '01',
      titleKey: 'home.atelier.step1.title',
      descKey: 'home.atelier.step1.desc',
      icon: 'edit' as const,
    },
    {
      num: '02',
      titleKey: 'home.atelier.step2.title',
      descKey: 'home.atelier.step2.desc',
      icon: 'verified-artisan' as const,
    },
    {
      num: '03',
      titleKey: 'home.atelier.step3.title',
      descKey: 'home.atelier.step3.desc',
      icon: 'process-video' as const,
    },
    {
      num: '04',
      titleKey: 'home.atelier.step4.title',
      descKey: 'home.atelier.step4.desc',
      icon: 'provenance' as const,
    },
  ];
</script>

<div class="atelier-wrapper">
  <SectionHeader
    kicker={t('home.atelier.kicker')}
    heading={t('home.atelier.heading')}
  />

  <p class="atelier-intro">{t('home.atelier.subheading')}</p>

  <div class="milestones-row">
    {#each STEPS as step}
      <div class="milestone-box">
        <div class="milestone-header">
          <Icon name={step.icon} size="1.2rem" />
          <span class="milestone-step-num">{step.num}</span>
        </div>
        <h3 class="milestone-title">
          {step.num === '01'
            ? t('home.atelier.step1.title')
            : step.num === '02'
              ? t('home.atelier.step2.title')
              : step.num === '03'
                ? t('home.atelier.step3.title')
                : t('home.atelier.step4.title')}
        </h3>
        <p class="milestone-desc">
          {step.num === '01'
            ? t('home.atelier.step1.desc')
            : step.num === '02'
              ? t('home.atelier.step2.desc')
              : step.num === '03'
                ? t('home.atelier.step3.desc')
                : t('home.atelier.step4.desc')}
        </p>
      </div>
    {/each}
  </div>

  <div class="atelier-guarantee-bar">
    <div class="guarantee-message">
      <Icon name="fair-price" size="1.1rem" />
      <span>100% Direct Payout to Master Weaver • Zero Platform Fee Below ₹1.5L Floor</span>
    </div>
    <Button variant="primary" onclick={onOpenConcierge} tooltip={tooltip('tooltip.openConcierge')}>
      <span>{t('home.atelier.cta')}</span>
      <Icon name="arrow-right" />
    </Button>
  </div>
</div>

<style>
  .atelier-wrapper {
    margin-block: var(--k-space-5);
  }

  .atelier-intro {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0 0 var(--k-space-5) 0;
    max-inline-size: 70ch;
  }

  .milestones-row {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-4);
    margin-block-end: var(--k-space-5);
  }

  @media (min-width: 44rem) {
    .milestones-row {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 60rem) {
    .milestones-row {
      grid-template-columns: repeat(4, 1fr);
    }
  }

  .milestone-box {
    padding: var(--k-space-4);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .milestone-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    color: var(--k-accent-primary-text);
  }

  .milestone-step-num {
    font-family: var(--k-font-display);
    font-size: var(--k-text-sm);
    font-weight: bold;
    color: var(--k-text-secondary);
  }

  .milestone-title {
    font-family: var(--k-font-display);
    font-size: var(--k-text-base);
    color: var(--k-text-primary);
    margin: 0;
  }

  .milestone-desc {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .atelier-guarantee-bar {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    background-color: var(--k-surface-sunken);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    align-items: flex-start;
  }

  @media (min-width: 48rem) {
    .atelier-guarantee-bar {
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
    }
  }

  .guarantee-message {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
  }
</style>
