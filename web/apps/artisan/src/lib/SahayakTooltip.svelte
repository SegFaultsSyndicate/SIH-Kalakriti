<!--
  apps/artisan/src/lib/SahayakTooltip.svelte

  Digital Literacy Assistant: Contextual Help Tooltip with spoken voice assistance.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { SpeakButton } from '@kalakriti/ui';

  interface Props {
    text: string;
    speakText?: string;
  }

  let { text, speakText }: Props = $props();
  const t = $derived(locale.t);
</script>

<aside class="sahayak-tooltip" role="note">
  <div class="sahayak-tooltip__icon">
    <Icon name="verified-artisan" size="0.95rem" />
  </div>
  <div class="sahayak-tooltip__content">
    <span class="sahayak-tooltip__kicker">SAHAYAK TIP / सहायता</span>
    <p class="sahayak-tooltip__text">{text}</p>
  </div>
  <div class="sahayak-tooltip__speak">
    <SpeakButton text={speakText || text} label={t('action.speak')} />
  </div>
</aside>

<style>
  .sahayak-tooltip {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-3);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: linear-gradient(135deg, rgba(245, 158, 11, 0.1), rgba(217, 119, 6, 0.04));
    border: 1px solid rgba(217, 119, 6, 0.35);
    margin-block: var(--k-space-1);
  }

  .sahayak-tooltip__icon {
    color: var(--k-haldi-700);
    margin-block-start: 2px;
    flex-shrink: 0;
  }

  .sahayak-tooltip__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .sahayak-tooltip__kicker {
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.06em;
    color: var(--k-accent-primary-text);
  }

  .sahayak-tooltip__text {
    font-size: var(--k-text-xs);
    color: var(--k-text-primary);
    line-height: var(--k-leading-normal);
    margin: 0;
  }

  .sahayak-tooltip__speak {
    flex-shrink: 0;
  }
</style>
