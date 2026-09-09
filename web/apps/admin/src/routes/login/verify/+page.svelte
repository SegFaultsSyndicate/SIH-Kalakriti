<!--
  apps/admin/src/routes/login/verify/+page.svelte

  Second step of the same real OTP flow as /login: POST /auth/otp/verify via
  completeOtpVerification (packages/api/src/auth-flow.ts), which stores the
  token and establishes the session in one call.
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
      const ok = await completeOtpVerification({ phone, otp });
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
