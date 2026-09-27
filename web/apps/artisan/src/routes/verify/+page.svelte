<!--
  apps/artisan/src/routes/verify/+page.svelte

  Six-box code entry, auto-submitting the moment the sixth digit lands.
  Verifying needs the same live round trip requesting a code does, so the
  same "explain, do not pretend to queue" treatment as /login applies here --
  see its header comment.

  On success this used to always navigate to `/` and let the root layout's
  guard redirect a not-yet-registered artisan on to /register/name from
  there. That two-hop hand-off made SvelteKit's client router resolve the
  Home route (`/`) as part of the first navigation -- fetching and compiling
  its whole module graph (IncomeGrowthChart, DigitalLiteracyTutorial,
  StallCardModal, demo-listing, ornament pieces...) purely to be told to
  leave again a tick later. In dev that first-time compile is slow enough to
  be a visible stall/flash between submitting the code and landing on
  registration. Deciding the destination here instead -- same signal
  +layout.svelte's guard uses -- makes this a single hop that never touches
  Home's code at all when the artisan isn't registered yet. The guard stays
  in place as the safety net for direct navigation, back button, etc.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import {
    requestOtp,
    completeOtpVerification,
    establishMockSession,
    session,
    ApiError,
    messageKeyFor,
  } from '@kalakriti/api';
  import { getPref, network } from '@kalakriti/offline';
  import { OtpInput, SpeakButton } from '@kalakriti/ui';
  import { getArtisanId, setArtisanId } from '$lib/registration';
  import { REGISTER_FIRST_STEP, HOME_PATH } from '$lib/route-guard';
  import { toE164 } from '$lib/phone';

  const RESEND_SECONDS = 30;

  const t = $derived(locale.t);

  let phone = $state('');
  let code = $state('');
  let verifying = $state(false);
  let error = $state('');
  let resendIn = $state(RESEND_SECONDS);
  let resending = $state(false);

  const isNewDemoAccount = $derived(phone.replace(/\D/g, '') === '9821891185');

  $effect(() => {
    void getPref<string>('login.phone').then((value) => {
      if (!value) {
        void goto('/login');
        return;
      }
      phone = value;
    });
  });

  $effect(() => {
    const timer = setInterval(() => {
      resendIn = Math.max(0, resendIn - 1);
    }, 1000);
    return () => clearInterval(timer);
  });

  async function submit(otp: string): Promise<void> {
    if (!network.online || verifying) return;
    verifying = true;
    error = '';
    try {
      const ok = await completeOtpVerification({ phone: toE164(phone), otp });
      if (ok) {
        const sub = session.claims?.sub;
        const tokenRegistered = typeof sub === 'string' && sub !== '';
        // If the token has no `sub`, the server says this is an unregistered
        // artisan. Clear any stale artisan-id from a previous session so the
        // route guard (and our own check here) can't treat that leftover as
        // "registered" and skip straight to home.
        if (!tokenRegistered) {
          await setArtisanId(undefined);
        }
        const registered = tokenRegistered || (await getArtisanId()) !== undefined;
        await goto(registered ? HOME_PATH : REGISTER_FIRST_STEP);
      } else {
        error = t('verify.invalid');
        code = '';
      }
    } catch (cause) {
      // core-svc's real dev-mode OTP acceptance already goes through
      // completeOtpVerification above --
      // this used to also forge an unsigned client-side JWT on any failure,
      // which never actually worked in any environment: pkg/auth.Issuer.Verify
      // checks a real HMAC signature regardless of environment, so it just
      // failed one request later than the honest error would have. See the
      // admin app's login/verify/+page.svelte for the same fix.
      //
      // VITE_USE_MOCKS=1 is the one exception, and only because it's opt-in:
      // with no backend reachable at all there is no real request to fail
      // later, so a local-only fake session is the only way to get past this
      // screen for UI work. Same fake token shape, gated so it can't hide a
      // real backend problem.
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] completeOtpVerification:', cause);
        const normalizedPhone = toE164(phone);
        const isRegisteredDemoArtisan = normalizedPhone === '+918779279060';
        establishMockSession('ARTISAN', normalizedPhone);
        // The mock ARTISAN token has no `sub` (see mock-session.ts), so treat
        // this the same as the real path: clear any stale artisan id so the
        // guard doesn't mistake a previous session's id for "registered".
        if (isRegisteredDemoArtisan) {
          await setArtisanId('sih-artisan-eshaan');
          await goto(HOME_PATH);
        } else {
          await setArtisanId(undefined);
          await goto(REGISTER_FIRST_STEP);
        }
        return;
      }
      error = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
      code = '';
    } finally {
      verifying = false;
    }
  }

  async function resend(): Promise<void> {
    if (resendIn > 0 || resending || !network.online) return;
    resending = true;
    try {
      await requestOtp({ phone: toE164(phone) });
      resendIn = RESEND_SECONDS;
    } catch (cause) {
      error = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
    } finally {
      resending = false;
    }
  }
</script>

<svelte:head>
  <title>{t('verify.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="verify">
  <h1>{t('verify.heading')}</h1>
  <p class="verify__body">{t('verify.body', { phone: `+91 ${phone}` })}</p>
  <SpeakButton
    text={`${t('verify.heading')}. ${t('verify.body', { phone })}`}
    label={t('action.speak')}
  />

  {#if !network.online}
    <p class="verify__offline" role="status">{t('verify.offline')}</p>
  {:else}
    <OtpInput bind:value={code} length={isNewDemoAccount ? 4 : 6} label={t('verify.code.label')} disabled={verifying} oncomplete={submit} autofocus />

    {#if verifying}
      <p role="status" aria-live="polite">{t('verify.submitting')}</p>
    {/if}
    {#if error}
      <p class="verify__error" role="alert">{error}</p>
    {/if}

    <button
      type="button"
      class="verify__resend"
      onclick={resend}
      disabled={resendIn > 0 || resending}
    >
      {resendIn > 0 ? t('verify.resend.wait', { seconds: resendIn }) : t('verify.resend')}
    </button>
  {/if}
</div>

<style>
  .verify {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-4);
    padding-block: var(--k-space-6);
    text-align: center;
  }

  .verify__body {
    max-inline-size: var(--k-measure-narrow);
    color: var(--k-text-secondary);
  }

  .verify__offline {
    max-inline-size: var(--k-measure-narrow);
    padding: var(--k-space-3);
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-warning-bg);
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    text-align: start;
  }

  .verify__error {
    color: var(--k-accent-danger);
  }

  .verify__resend {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    border: none;
    background: none;
    color: var(--k-accent-secondary);
    font-size: var(--k-text-sm);
    cursor: pointer;
  }

  .verify__resend:disabled {
    color: var(--k-text-secondary);
    cursor: not-allowed;
  }
</style>
