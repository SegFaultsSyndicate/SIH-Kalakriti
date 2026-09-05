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
</script>

<svelte:head>
  <title>{t('login.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="login">
  <h1>{t('login.heading')}</h1>
  <p class="login__body">{t('login.body')}</p>
  <SpeakButton text={`${t('login.heading')}. ${t('login.body')}`} label={t('action.speak')} />

  <output class="login__readout" aria-label={t('login.phone.label')}>
    <span class="login__code">{t('login.phone.countryCode')}</span>
    <span class="login__digits">{digits || '—'}</span>
  </output>

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
    gap: var(--k-space-3);
    align-items: baseline;
    margin-block-start: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-5);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    font-size: var(--k-text-2xl);
    font-variant-numeric: var(--k-numeric-tabular);
  }

  .login__code {
    color: var(--k-text-secondary);
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
