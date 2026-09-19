<!--
  apps/admin/src/routes/login/verify/+page.svelte

  Second step of the same real OTP flow as /login: POST /auth/otp/verify via
  completeOtpVerification (packages/api/src/auth-flow.ts), which stores the
  token and establishes the session in one call.

  dev_role: 'MINISTRY' requests a real MINISTRY-role token instead of the
  default ARTISAN one -- core-svc only honors it with dev OTP enabled
  (AUTH_DEV_OTP_ENABLED) and *rejects the whole login* with a 400 otherwise
  (VerifyOtp treats a dev_role it can't honor as invalid input, not a no-op --
  see services/core-svc/internal/core/service/auth.go). So it's only sent in
  a dev build (import.meta.env.DEV): a production build omits it and falls
  through to the ordinary OTP path, which still logs the caller in, just as
  ARTISAN -- the same "no real MINISTRY issuance path exists yet" gap
  documented in CLAUDE.md, not a broken login. There is currently no other
  login path anywhere in the product that can mint a MINISTRY token at all.
  This used to fall back, on any verify failure, to forging an unsigned JWT
  client-side and calling session.establish on it directly -- which never
  actually worked: pkg/auth.Issuer.Verify checks a real HMAC signature
  regardless of environment, so the very first real API call the admin
  console made with that forged token 401'd. It just failed one request
  later than the honest error would have, in every environment, not just
  production.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { locale } from '@kalakriti/i18n';
  import { completeOtpVerification, ApiError, messageKeyFor } from '@kalakriti/api';
  import { Button, Input, FieldGroup } from '@kalakriti/ui';

  const t = $derived(locale.t);
  const phone = $derived($page.url.searchParams.get('phone') ?? '');
  const redirect = $derived($page.url.searchParams.get('redirect') ?? '/');

  let otp = $state('');
  let verifying = $state(false);
  let error = $state('');

  async function submit(): Promise<void> {
    verifying = true;
    error = '';
    try {
      const ok = await completeOtpVerification({
        phone,
        otp,
        ...(import.meta.env.DEV ? { dev_role: 'MINISTRY' } : {}),
      });
      if (ok) {
        await goto(redirect);
      } else {
        error = t('verify.invalid');
      }
    } catch (cause) {
      error = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
    } finally {
      verifying = false;
    }
  }
</script>

<svelte:head>
  <title>{t('verify.heading')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="verify">
  <h1>{t('verify.heading')}</h1>
  <p class="verify__body">{t('verify.body', { phone })}</p>

  <form onsubmit={(e) => { e.preventDefault(); void submit(); }} class="verify__form">
    <FieldGroup label={t('verify.code.label')}>
      {#snippet children({ id })}
        <Input {id} type="text" bind:value={otp} placeholder="123456" autocomplete="one-time-code" />
      {/snippet}
    </FieldGroup>

    {#if error}
      <p class="verify__error" role="alert">{error}</p>
    {/if}

    <Button type="submit" loading={verifying} disabled={otp.trim().length === 0}>
      {t('verify.submit')}
    </Button>
  </form>
</div>

<style>
  .verify {
    max-inline-size: 24rem;
    margin-inline: auto;
    padding-block: var(--k-space-8);
  }

  .verify__body {
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-5);
  }

  .verify__form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .verify__error {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }
</style>
