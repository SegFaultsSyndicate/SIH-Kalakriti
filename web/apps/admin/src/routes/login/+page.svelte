<!--
  apps/admin/src/routes/login/+page.svelte

  Desktop phone-entry login, reusing the same POST /auth/otp/request the
  artisan and buyer apps use -- there is no separate admin auth backend, a
  phone number logs in as whatever role its account carries in the token.
  A plain text input here, not the artisan app's Keypad: this is a desktop
  tool for field staff, not a phone-first onboarding flow.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { locale } from '@kalakriti/i18n';
  import { requestOtp, ApiError, messageKeyFor } from '@kalakriti/api';
  import { Button, Input, FieldGroup } from '@kalakriti/ui';

  const t = $derived(locale.t);

  let phone = $state('');
  let sending = $state(false);
  let error = $state('');

  async function submit(): Promise<void> {
    sending = true;
    error = '';
    try {
      await requestOtp({ phone });
      const redirect = $page.url.searchParams.get('redirect') ?? '/';
      await goto(`/login/verify?phone=${encodeURIComponent(phone)}&redirect=${encodeURIComponent(redirect)}`);
    } catch (cause) {
      error = cause instanceof ApiError ? t(messageKeyFor(cause)) : t('api.error.unknown');
    } finally {
      sending = false;
    }
  }
</script>

<svelte:head>
  <title>{t('login.heading')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="login">
  <h1>{t('login.heading')}</h1>
  <p class="login__body">{t('login.body')}</p>

  <form onsubmit={(e) => { e.preventDefault(); void submit(); }} class="login__form">
    <FieldGroup label={t('login.phone.label')} description={t('admin.login.phoneHint')}>
      {#snippet children({ id })}
        <Input {id} type="tel" bind:value={phone} placeholder="+91XXXXXXXXXX" autocomplete="tel" />
      {/snippet}
    </FieldGroup>

    {#if error}
      <p class="login__error" role="alert">{error}</p>
    {/if}

    <Button type="submit" loading={sending} disabled={phone.trim().length === 0}>
      {t('login.submit')}
    </Button>
  </form>
</div>

<style>
  .login {
    max-inline-size: 24rem;
    margin-inline: auto;
    padding-block: var(--k-space-8);
  }

  .login__body {
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-5);
  }

  .login__form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .login__error {
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }
</style>
