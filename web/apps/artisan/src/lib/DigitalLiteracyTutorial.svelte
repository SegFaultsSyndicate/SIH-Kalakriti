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
    icon: IconName;
    badge: string;
    speakHindi: string;

  }

  const STEPS: TutorialStep[] = [
    {
      titleKey: 'literacy.tutorial.step1Title',
      descKey: 'literacy.tutorial.step1Desc',
      icon: 'microphone',
      badge: 'बोलकर सूची बनाएं (Voice-First)',
      speakHindi: 'कलाकृति में टाइप करने की कोई ज़रूरत नहीं है। बस अपनी मातृभाषा में अपने शिल्प के बारे में बोलें। हमारा एआई इसे अपने आप लिख लेगा।',
    },
    {
      titleKey: 'literacy.tutorial.step2Title',
      descKey: 'literacy.tutorial.step2Desc',
      icon: 'camera',
      badge: 'एआई फोटो स्टूडियो (AI Studio)',
      speakHindi: 'अपने करघे पर ही साधारण मोबाइल फोटो खींचें। हमारा एआई अपने आप पीछे की हलचल हटाकर साफ सफेद पृष्ठभूमि और सही रोशनी बना देगा।',
    },
    {
      titleKey: 'literacy.tutorial.step3Title',
      descKey: 'literacy.tutorial.step3Desc',
      icon: 'fair-price',
      badge: 'उचित मूल्य सलाहकार (Fair Pricing)',
      speakHindi: 'कभी घाटे में न बेचें। कलाकृति कच्चे माल और आपकी दैनिक मजदूरी जोड़कर सही सरकारी मूल्य सुझाती है।',
    },
    {
      titleKey: 'literacy.tutorial.step4Title',
      descKey: 'literacy.tutorial.step4Desc',
      icon: 'income-statement',
      badge: 'सीधे खाते में भुगतान (Direct DBT)',
      speakHindi: 'ग्राहक और सरकारी खरीद का पूरा पैसा बिना किसी दलाल के सीधे आपके बैंक खाते में पहुंचेगा। शून्य कमीशन।',
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
          <span>DIGITAL SAHAYAK • डिजिटल साक्षरता सहायता</span>
        </div>
        <button type="button" class="close-btn" onclick={onclose} aria-label="Close">
          <Icon name="close" />
        </button>
      </header>

      <!-- Step Card Hero -->
      <div class="step-card">
        <div class="step-icon-wrap">
          <Icon name={activeStep.icon} size="2.5rem" />
        </div>
        <div class="step-badge">{activeStep.badge}</div>
        <h2 class="step-title">{t(activeStep.titleKey)}</h2>
        <p class="step-desc">{t(activeStep.descKey)}</p>

        <!-- Large Voice Audio Button -->
        <div class="voice-row">
          <SpeakButton text={activeStep.speakHindi} label="हिन्दी में सुनें (Listen in Hindi)" />
          <span class="voice-hint">Tap to listen in spoken Hindi</span>
        </div>
      </div>

      <!-- Step Dots Indicator -->
      <div class="dots-row">
        {#each STEPS as _, i}
          <button
            type="button"
            class="dot-btn"
            class:dot-btn--active={i === currentStep}
            onclick={() => (currentStep = i)}
            aria-label={`Go to step ${i + 1}`}
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
    background: rgba(15, 23, 42, 0.75);
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
    border: 1px solid var(--k-haldi-700);
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

  /* Step Card */
  .step-card {
    border-radius: var(--k-radius-lg);
    background: var(--k-surface-inverse);
    color: var(--k-text-on-inverse);
    padding: var(--k-space-6);
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--k-space-3);
  }

  .step-icon-wrap {
    inline-size: 5rem;
    block-size: 5rem;
    border-radius: 50%;
    background: color-mix(in srgb, var(--k-text-on-inverse) 18%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-block-end: var(--k-space-1);
  }

  .step-badge {
    background: rgba(255, 255, 255, 0.25);
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
    color: var(--k-ink-950) !important;
    font-weight: 700;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
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
    background: var(--k-accent-primary-bg, var(--k-accent-primary-bg));
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
