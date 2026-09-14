<!--
  apps/buyer/src/lib/FaqAccordion.svelte

  5 Core Heritage & Procurement FAQs:
  - GI Cryptographic Verification
  - 100% Direct Payouts
  - Made-To-Order & Lead Times
  - Transit Guarantee & Authenticity
  - Live Loom Video Consultations
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  interface FaqItem {
    id: string;
    questionKey: MessageKey;
    answerKey: MessageKey;
    tagKey: MessageKey;
  }

  const FAQS: FaqItem[] = [
    {
      id: 'gi-verification',
      tagKey: 'faq.giVerification.tag',
      questionKey: 'faq.giVerification.question',
      answerKey: 'faq.giVerification.answer',
    },
    {
      id: 'direct-payouts',
      tagKey: 'faq.directPayouts.tag',
      questionKey: 'faq.directPayouts.question',
      answerKey: 'faq.directPayouts.answer',
    },
    {
      id: 'bespoke-lead-times',
      tagKey: 'faq.bespokeLeadTimes.tag',
      questionKey: 'faq.bespokeLeadTimes.question',
      answerKey: 'faq.bespokeLeadTimes.answer',
    },
    {
      id: 'authenticity-guarantee',
      tagKey: 'faq.authenticityGuarantee.tag',
      questionKey: 'faq.authenticityGuarantee.question',
      answerKey: 'faq.authenticityGuarantee.answer',
    },
    {
      id: 'loom-video-tour',
      tagKey: 'faq.loomVideoTour.tag',
      questionKey: 'faq.loomVideoTour.question',
      answerKey: 'faq.loomVideoTour.answer',
    },
  ];

  let openId = $state<string | null>(FAQS[0].id);

  function toggle(id: string) {
    openId = openId === id ? null : id;
  }
</script>

<div class="faq-container">
  <div class="faq-header">
    <span class="faq-kicker">{t('faq.kicker')}</span>
    <h2 class="faq-title">{t('faq.title')}</h2>
    <p class="faq-subhead">{t('faq.subhead')}</p>
  </div>

  <div class="faq-accordion" role="region" aria-label={t('faq.ariaLabel')}>
    {#each FAQS as item (item.id)}
      {@const isOpen = openId === item.id}
      <div class="faq-item" class:open={isOpen}>
        <button
          type="button"
          class="faq-trigger"
          aria-expanded={isOpen}
          aria-controls={`faq-answer-${item.id}`}
          onclick={() => toggle(item.id)}
        >
          <div class="faq-trigger__content">
            <span class="faq-tag">{t(item.tagKey)}</span>
            <span class="faq-question">{t(item.questionKey)}</span>
          </div>
          <span class="faq-icon" class:rotate={isOpen} aria-hidden="true">
            <Icon name="chevron-down" size="1.1rem" />
          </span>
        </button>

        {#if isOpen}
          <div
            id={`faq-answer-${item.id}`}
            class="faq-answer"
            role="region"
            aria-labelledby={`faq-trigger-${item.id}`}
          >
            <p class="faq-text">{t(item.answerKey)}</p>
          </div>
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .faq-container {
    max-inline-size: 54rem;
    margin-inline: auto;
    padding-block: var(--k-space-6);
  }

  .faq-header {
    text-align: center;
    margin-block-end: var(--k-space-6);
  }

  .faq-kicker {
    display: inline-block;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-secondary);
    margin-block-end: var(--k-space-1);
  }

  .faq-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-2xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .faq-subhead {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    max-inline-size: 38rem;
    margin-inline: auto;
    line-height: 1.6;
  }

  .faq-accordion {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-card, var(--k-surface-base));
    overflow: hidden;
  }

  .faq-item {
    border-block-end: 1px solid var(--k-border-subtle);
    transition: background-color var(--k-duration-fast) ease;
  }

  .faq-item:last-child {
    border-block-end: none;
  }

  .faq-item.open {
    background-color: var(--k-surface-sunken, var(--k-surface-base));
  }

  .faq-trigger {
    display: flex;
    justify-content: space-between;
    align-items: center;
    inline-size: 100%;
    padding: var(--k-space-4) var(--k-space-5);
    background: none;
    border: none;
    cursor: pointer;
    text-align: start;
    font: inherit;
    gap: var(--k-space-3);
  }

  .faq-trigger:hover {
    background-color: rgba(198, 93, 59, 0.03);
  }

  .faq-trigger:focus-visible {
    outline: 2px solid var(--k-accent-secondary);
    outline-offset: -2px;
  }

  .faq-trigger__content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .faq-tag {
    font-size: 0.7rem;
    font-weight: var(--k-weight-semibold);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-accent-secondary);
  }

  .faq-question {
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
    line-height: 1.4;
  }

  .faq-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: var(--k-text-secondary);
    transition: transform var(--k-duration-fast) ease;
  }

  .faq-icon.rotate {
    transform: rotate(180deg);
    color: var(--k-accent-secondary);
  }

  .faq-answer {
    padding: 0 var(--k-space-5) var(--k-space-4) var(--k-space-5);
    border-block-start: 1px dashed var(--k-border-subtle);
  }

  .faq-text {
    font-size: var(--k-text-sm);
    line-height: 1.7;
    color: var(--k-text-secondary);
    margin: var(--k-space-3) 0 0 0;
  }
</style>
