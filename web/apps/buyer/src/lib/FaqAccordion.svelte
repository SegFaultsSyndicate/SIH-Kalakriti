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
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  interface FaqItem {
    id: string;
    question: string;
    answer: string;
    tag: string;
  }

  const FAQS: FaqItem[] = [
    {
      id: 'gi-verification',
      tag: 'Cryptographic Provenance',
      question: 'How is Geographical Indication (GI) authenticity verified on Kalakriti?',
      answer: 'Every GI-certified piece undergoes cluster verification where physical loom warp-weft weave structures and authentic raw materials are validated. Upon approval, the Ministry of Social Justice & Empowerment issues an Ed25519 cryptographic seal and SHA-256 hash printed on an tamper-proof QR label, allowing anyone to verify its provenance on-chain.',
    },
    {
      id: 'direct-payouts',
      tag: '0% Platform Fee',
      question: 'How does 100% direct-to-artisan payout work with zero middleman deductions?',
      answer: 'Kalakriti is built as National Public Digital Goods sponsored by the Government of India. Unlike private marketplaces that extract 25-40% broker commissions, 100% of your payment is disbursed directly into the verified DBT bank account of the artisan or Self-Help Group (SHG) upon dispatch.',
    },
    {
      id: 'bespoke-lead-times',
      tag: 'Loom Timelines',
      question: 'What is the lead time for Made-To-Order and bespoke heirloom commissions?',
      answer: 'Ready-stock craft lots dispatch within 48 to 72 hours. For Made-to-Order and custom bridal or architectural pieces, lead times typically range from 2 to 6 weeks depending on pit-loom complexity, natural vegetable dye fermentation, and warp tensioning. Buyers receive real-time loom video milestones as their piece is woven.',
    },
    {
      id: 'authenticity-guarantee',
      tag: 'Buyer Protection',
      question: 'What is the Kalakriti transit damage and authenticity guarantee?',
      answer: 'Every shipment is packed in climate-sealed khadi-lined archival packaging and covered under the Central Craft Transit Guarantee. In the rare event of transit damage or motif mismatch, we offer 100% free restoration at the artisan guild or an immediate full refund.',
    },
    {
      id: 'loom-video-tour',
      tag: 'VIP Concierge',
      question: 'Can buyers schedule a live video consultation with the artisan before acquiring high-value pieces?',
      answer: 'Yes. Through our VIP Concierge service, prospective collectors and institutional procurement officers can reserve a 10-minute live video meeting with the master weaver directly at their loom to inspect the drape, zari count, and natural indigo sheen before placing an order.',
    },
  ];

  let openId = $state<string | null>(FAQS[0].id);

  function toggle(id: string) {
    openId = openId === id ? null : id;
  }
</script>

<div class="faq-container">
  <div class="faq-header">
    <span class="faq-kicker">Transparent Guild Standards</span>
    <h2 class="faq-title">Frequently Asked Questions</h2>
    <p class="faq-subhead">Everything you need to know about certified provenance, direct payments, and bespoke heirloom fulfillment.</p>
  </div>

  <div class="faq-accordion" role="region" aria-label="Frequently Asked Questions">
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
            <span class="faq-tag">{item.tag}</span>
            <span class="faq-question">{item.question}</span>
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
            <p class="faq-text">{item.answer}</p>
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
    background-color: var(--k-surface-card, #ffffff);
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
    background-color: var(--k-surface-sunken, #fbf9f6);
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
