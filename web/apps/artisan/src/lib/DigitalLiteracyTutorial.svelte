<!--
  apps/artisan/src/lib/DigitalLiteracyTutorial.svelte

  Guided Onboarding & Digital Literacy Mode:
  Provides a warm, visual, Indic-voice-accompanied 4-step interactive
  tutorial designed specifically for artisans with low digital literacy.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Icon, type IconName } from '@kalakriti/icons';
  import { Button, SpeakButton } from '@kalakriti/ui';
  import { setPref } from '@kalakriti/offline';

  interface Props {
    open: boolean;
    onclose: () => void;
    onstartDemo?: () => void;
  }

  let { open, onclose, onstartDemo }: Props = $props();

  const t = $derived(locale.t);
  let currentStep = $state(0);

  interface TutorialStep {
    titleKey: MessageKey;
    descKey: MessageKey;
    badgeKey: MessageKey;
    icon: IconName;
    speakHindi: string;
    /* Each step carries one accent from the craft palette -- voice is clay,
       the photo studio is indigo, pricing is turmeric, payment is neem. Four
       flat fills, no gradient: the colour identifies the step, it is not a
       decoration laid over it. Both tokens are named so the pairing stays a
       measured one (see palette.css) rather than whatever reads nicely. */
    accent: string;
    onAccent: string;
  }

  const STEPS: TutorialStep[] = [
    {
      titleKey: 'literacy.tutorial.step1Title',
      descKey: 'literacy.tutorial.step1Desc',
      badgeKey: 'literacy.tutorial.step1Badge',
      icon: 'microphone',
      speakHindi: 'कलाकृति में टाइप करने की कोई ज़रूरत नहीं है। बस अपनी मातृभाषा में अपने शिल्प के बारे में बोलें। हमारा एआई इसे अपने आप लिख लेगा।',
      accent: '--k-accent-primary-bg',
      onAccent: '--k-text-on-accent',
    },
    {
      titleKey: 'literacy.tutorial.step2Title',
      descKey: 'literacy.tutorial.step2Desc',
      badgeKey: 'literacy.tutorial.step2Badge',
      icon: 'camera',
      speakHindi: 'अपने करघे पर ही साधारण मोबाइल फोटो खींचें। हमारा एआई अपने आप पीछे की हलचल हटाकर साफ सफेद पृष्ठभूमि और सही रोशनी बना देगा।',
      accent: '--k-accent-secondary',
      onAccent: '--k-text-on-accent',
    },
    {
      titleKey: 'literacy.tutorial.step3Title',
      descKey: 'literacy.tutorial.step3Desc',
      badgeKey: 'literacy.tutorial.step3Badge',
      icon: 'fair-price',
      speakHindi: 'कभी घाटे में न बेचें। कलाकृति कच्चे माल और आपकी दैनिक मजदूरी जोड़कर सही सरकारी मूल्य सुझाती है।',
      accent: '--k-accent-warning-bg',
      onAccent: '--k-accent-warning-text',
    },
    {
      titleKey: 'literacy.tutorial.step4Title',
      descKey: 'literacy.tutorial.step4Desc',
      badgeKey: 'literacy.tutorial.step4Badge',
      icon: 'income-statement',
      speakHindi: 'ग्राहक और सरकारी खरीद का पूरा पैसा बिना किसी दलाल के सीधे आपके बैंक खाते में पहुंचेगा। शून्य कमीशन।',
      accent: '--k-accent-success-bg',
      onAccent: '--k-text-on-accent',
    },
  ];

  const activeStep = $derived(STEPS[currentStep]);

  async function handleFinish(): Promise<void> {
    await setPref('literacy.tutorial_completed', true);
    onclose();
  }

  function handleNext(): void {
    if (currentStep < STEPS.length - 1) {
      currentStep += 1;
    } else {
      void handleFinish();
    }
  }

  function handlePrev(): void {
    if (currentStep > 0) {
      currentStep -= 1;
    }
  }
</script>

{#if open}
  <div class="tutorial-backdrop" onclick={onclose} role="presentation">
    <div
      class="tutorial-dialog"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-label={t('literacy.tutorial.title')}
      tabindex="-1"
    >
      <header class="tutorial-header">
        <div class="sahayak-badge">
          <span>{t('literacy.tutorial.sahayakBadge')}</span>
        </div>
        <button type="button" class="close-btn" onclick={onclose} aria-label={t('ui.dialog.close')}>
          <Icon name="close" />
        </button>
      </header>

      <!-- Step Card Hero. Keyed on the step so every part of it is torn down
           and rebuilt together -- and so the card reads as a new card, not as
           the old one with a swapped icon. -->
      {#key currentStep}
        <div
          class="step-card"
          style:--step-accent="var({activeStep.accent})"
          style:--step-on-accent="var({activeStep.onAccent})"
        >
          <div class="step-icon-wrap">
            <Icon name={activeStep.icon} size="2.5rem" />
          </div>
          <div class="step-badge">{t(activeStep.badgeKey)}</div>
          <h2 class="step-title">{t(activeStep.titleKey)}</h2>
          <p class="step-desc">{t(activeStep.descKey)}</p>

          <!-- Large Voice Audio Button -->
          <div class="voice-row">
            <SpeakButton text={activeStep.speakHindi} label={t('literacy.tutorial.listenHindi')} />
            <span class="voice-hint">{t('literacy.tutorial.voiceHint')}</span>
          </div>
        </div>
      {/key}

      <!-- Step Dots Indicator -->
      <div class="dots-row">
        {#each STEPS as _, i}
          <button
            type="button"
            class="dot-btn"
            class:dot-btn--active={i === currentStep}
            onclick={() => (currentStep = i)}
            aria-label={t('literacy.tutorial.dotAriaLabel', { step: String(i + 1) })}
          ></button>
        {/each}
      </div>

      <!-- Footer Navigation Actions -->
      <footer class="tutorial-footer">
        <div class="left-actions">
          {#if onstartDemo}
            <Button
              variant="secondary"
              size="sm"
              onclick={() => {
                onclose();
                onstartDemo();
              }}
            >
              <Icon name="play" size="0.85rem" />
              {t('literacy.demo.start')}
            </Button>
          {/if}
        </div>

        <div class="right-actions">
          {#if currentStep > 0}
            <Button variant="secondary" size="md" onclick={handlePrev}>
              {t('action.back')}
            </Button>
          {/if}

          <Button variant="primary" size="md" onclick={handleNext}>
            {currentStep === STEPS.length - 1 ? t('literacy.tutorial.close') : t('action.next')}
          </Button>
        </div>
      </footer>
    </div>
  </div>
{/if}

<style>
  .tutorial-backdrop {
    position: fixed;
    inset: 0;
    background: color-mix(in srgb, var(--k-overlay-scrim) 75%, transparent);
    backdrop-filter: blur(6px);
    z-index: 99999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--k-space-4);
  }

  .tutorial-dialog {
    background: var(--k-surface-base);
    border-radius: var(--k-radius-xl, 1.25rem);
    box-shadow: 0 20px 48px rgba(0, 0, 0, 0.35);
    max-inline-size: 36rem;
    inline-size: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-5);
    border: var(--k-hairline) solid var(--k-border-hairline);
    animation: zoomIn 0.2s ease-out;
  }

  @keyframes zoomIn {
    from {
      transform: scale(0.96);
      opacity: 0;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }

  .tutorial-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .sahayak-badge {
    display: inline-flex;
    align-items: center;
    background: var(--k-surface-pressed);
    border: 1px solid var(--k-border-warning);
    color: var(--k-accent-primary-text);
    font-size: 0.68rem;
    font-weight: 800;
    letter-spacing: 0.05em;
    padding: 3px 10px;
    border-radius: 999px;
  }

  .close-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--k-text-secondary);
    padding: var(--k-space-1);
    border-radius: var(--k-radius-pill);
  }

  /* Step Card. Flat inverse ground with the step's accent carried by a top
     rule, the icon disc and the badge -- separation by line and fill, never
     by gradient or shadow. */
  .step-card {
    border-radius: var(--k-radius-lg);
    /* A tint of the step's own accent falling into the inverse surface. Both
       stops are tokens, so the four cards differ by hue without anyone
       inventing a colour pair, and the dark and high-contrast themes still
       move underneath it. */
    background:
      radial-gradient(
        120% 90% at 50% 0%,
        color-mix(in srgb, var(--step-accent, var(--k-accent-primary-bg)) 30%, var(--k-surface-inverse)),
        var(--k-surface-inverse) 68%
      );
    color: var(--k-text-on-inverse);
    border-block-start: 4px solid var(--step-accent, var(--k-accent-primary-bg));
    animation: stepIn 0.22s ease-out;
    /* Inline padding trimmed from the block padding: on a narrow phone this is
       the difference between the Hindi listen button fitting on one line and
       wrapping mid-word (measured: 229px available vs 281px needed at the
       old --k-space-6 inline padding). */
    padding-block: var(--k-space-6);
    padding-inline: var(--k-space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--k-space-3);
  }

  @keyframes stepIn {
    from {
      opacity: 0;
      transform: translateY(6px);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }

  .step-icon-wrap {
    inline-size: 5rem;
    block-size: 5rem;
    border-radius: 50%;
    background: var(--step-accent, var(--k-accent-primary-bg));
    color: var(--step-on-accent, var(--k-text-on-accent));
    display: flex;
    align-items: center;
    justify-content: center;
    margin-block-end: var(--k-space-1);
  }

  .step-badge {
    background: var(--step-accent, var(--k-accent-primary-bg));
    color: var(--step-on-accent, var(--k-text-on-accent));
    font-size: 0.75rem;
    font-weight: 700;
    padding: 2px 12px;
    border-radius: 999px;
    letter-spacing: 0.04em;
  }

  .step-title {
    font-size: 1.35rem;
    font-weight: 800;
    margin: 0;
    line-height: 1.25;
  }

  .step-desc {
    font-size: var(--k-text-sm);
    line-height: 1.5;
    opacity: 0.95;
    margin: 0;
    max-inline-size: 28ch;
  }

  .voice-row {
    margin-block-start: var(--k-space-2);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .voice-row :global(button) {
    background: var(--k-surface-base) !important;
    color: var(--k-text-primary) !important;
    font-weight: 700;
    font-size: var(--k-text-base);
    border: none;
  }

  .voice-hint {
    font-size: 0.68rem;
    opacity: 0.85;
  }

  /* Dots */
  .dots-row {
    display: flex;
    justify-content: center;
    gap: var(--k-space-2);
  }

  .dot-btn {
    inline-size: 0.65rem;
    block-size: 0.65rem;
    border-radius: 50%;
    border: none;
    background: var(--k-border-interactive);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .dot-btn--active {
    inline-size: 2rem;
    border-radius: 999px;
    background: var(--k-accent-primary-bg);
  }

  /* One step's colour is not worth a vestibular migraine. */
  @media (prefers-reduced-motion: reduce) {
    .step-card,
    .tutorial-dialog {
      animation: none;
    }
  }

  /* Footer */
  .tutorial-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-block-start: 1px solid var(--k-border-hairline);
    padding-block-start: var(--k-space-3);
    margin-block-start: var(--k-space-1);
  }

  .right-actions {
    display: flex;
    gap: var(--k-space-2);
  }
</style>
