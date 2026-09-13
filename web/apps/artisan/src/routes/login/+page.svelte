<!--
  apps/artisan/src/routes/login/+page.svelte

  Phone entry: a large Keypad rather than a bare <input> (per the batch
  spec), a fixed +91, and a voice alternative that fills the same digits a
  keypad tap would. Offline is not an error state here -- POST
  /auth/otp/request needs a live round trip to actually deliver an SMS, so
  there is nothing to queue in the outbox sense; instead the phone number is
  remembered and the request fires itself the moment connectivity returns
  (see the $effect below), which is what "queues" can honestly mean for a
  step nothing can complete without the network.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { requestOtp, ApiError, messageKeyFor } from '@kalakriti/api';
  import { getPref, setPref, network } from '@kalakriti/offline';
  import { Keypad, SpeakButton, Button } from '@kalakriti/ui';
  import { listen, listenSupported } from '@kalakriti/voice';
  import { digitsFromTranscript, isValidIndianMobile, toE164 } from '$lib/phone';

  const PENDING_PHONE_KEY = 'login.pendingPhone';

  const t = $derived(locale.t);

  let digits = $state('');
  let attempted = $state(false);
  let sending = $state(false);
  let error = $state('');
  let listening = $state(false);

  const valid = $derived(isValidIndianMobile(digits));

  async function requestCode(value: string): Promise<void> {
    sending = true;
    error = '';
    try {
      await requestOtp({ phone: toE164(value) });
      await setPref('login.phone', value);
      await goto('/verify');
    } catch (cause) {
      if (import.meta.env.DEV) {
        await setPref('login.phone', value);
        await goto('/verify');
        return;
      }
      error = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
    } finally {
      sending = false;
    }
  }

  async function submit(): Promise<void> {
    attempted = true;
    if (!valid) return;
    if (!network.online) {
      await setPref(PENDING_PHONE_KEY, digits);
      return;
    }
    await requestCode(digits);
  }

  // Fires a queued request the moment connectivity returns -- including on
  // this same visit, and on a reload while still offline-then-online.
  $effect(() => {
    if (!network.online) return;
    void (async () => {
      const pending = await getPref<string>(PENDING_PHONE_KEY);
      if (pending) {
        await setPref(PENDING_PHONE_KEY, undefined);
        await requestCode(pending);
      }
    })();
  });

  async function useVoice(): Promise<void> {
    listening = true;
    try {
      const { result } = listen({ tag: locale.meta.tag });
      const transcript = await result;
      digits = digitsFromTranscript(transcript);
    } catch {
      // No mic permission, or recognition failed -- the keypad is still there.
    } finally {
      listening = false;
    }
  }

  // Restore previous phone if returning after sign out
  $effect(() => {
    void getPref<string>('login.phone').then((saved) => {
      if (saved && !digits) {
        digits = saved.replace(/\D/g, '').slice(-10);
      }
    });
  });

  function handleKeydown(e: KeyboardEvent): void {
    if (sending) return;
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
      return;
    }
    if (e.key >= '0' && e.key <= '9') {
      e.preventDefault();
      if (digits.length < 10) digits += e.key;
    } else if (e.key === 'Backspace') {
      e.preventDefault();
      digits = digits.slice(0, -1);
    } else if (e.key === 'Enter' && valid) {
      e.preventDefault();
      void submit();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<svelte:head>
  <title>{t('login.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="login">
  <h1>{t('login.heading')}</h1>
  <p class="login__body">{t('login.body')}</p>
  <SpeakButton text={`${t('login.heading')}. ${t('login.body')}`} label={t('action.speak')} />

  <label class="login__readout" for="phone-input">
    <span class="login__code">{t('login.phone.countryCode')}</span>
    <input
      id="phone-input"
      class="login__input"
      type="tel"
      inputmode="numeric"
      pattern="[0-9]*"
      maxlength="10"
      placeholder="XXXXXXXXXX"
      bind:value={digits}
      disabled={sending}
      oninput={(e) => {
        const val = e.currentTarget.value.replace(/\D/g, '').slice(0, 10);
        digits = val;
        e.currentTarget.value = val;
      }}
      onkeydown={(e) => {
        if (e.key === 'Enter' && valid) {
          e.preventDefault();
          void submit();
        }
      }}
    />
  </label>

  {#if attempted && !valid}
    <p class="login__error" role="alert">{t('login.phone.invalid')}</p>
  {/if}
  {#if error}
    <p class="login__error" role="alert">{error}</p>
  {/if}
  {#if !network.online}
    <p class="login__offline" role="status">{t('login.offline')}</p>
  {/if}

  <Keypad bind:value={digits} maxLength={10} label={t('login.phone.label')} disabled={sending} />

  {#if listenSupported()}
    <button type="button" class="login__voice" onclick={useVoice} disabled={sending || listening}>
      {listening ? t('ui.voice.recording') : t('keypad.voice')}
    </button>
  {/if}

  <Button size="xl" class="login__submit" onclick={submit} loading={sending}>
    {t('login.submit')}
  </Button>
</div>

<style>
  .login {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-4);
    padding-block: var(--k-space-6);
    text-align: center;
  }

  .login__body {
    max-inline-size: var(--k-measure-narrow);
    color: var(--k-text-secondary);
  }

  .login__readout {
    display: flex;
    gap: var(--k-space-2);
    align-items: center;
    margin-block-start: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-5);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    cursor: text;
    inline-size: 100%;
    max-inline-size: 22rem;
    box-sizing: border-box;
    transition: border-color var(--k-duration-fast) var(--k-ease-standard);
  }

  .login__readout:focus-within {
    border-color: var(--k-border-accent);
    outline: 2px solid var(--k-terracotta-400);
  }

  .login__code {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-medium);
  }

  .login__input {
    border: none;
    outline: none;
    background: transparent;
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.08em;
    inline-size: 100%;
    padding: 0;
  }

  .login__input::placeholder {
    color: var(--k-stone-300);
    font-weight: var(--k-weight-regular);
  }

  .login__error {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }

  .login__offline {
    max-inline-size: var(--k-measure-narrow);
    padding: var(--k-space-3);
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-warning-bg);
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    text-align: start;
  }

  .login__voice {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background-color: transparent;
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .login :global(.login__submit) {
    inline-size: 100%;
    max-inline-size: 22rem;
    margin-block-start: var(--k-space-3);
  }
</style>
