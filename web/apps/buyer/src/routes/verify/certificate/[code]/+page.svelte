<!--
  apps/buyer/src/routes/verify/certificate/[code]/+page.svelte

  F15: the page a literacy certificate's QR code opens. Public, no sign-in
  -- the one new public route in tier 4. Shows only what the printed
  certificate already shows (name, craft, place, date); insight-svc checks
  the ed25519 signature before answering valid.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale, formatDate } from '@kalakriti/i18n';
  import { verifyLiteracyCertificate, ApiError } from '@kalakriti/api';

  type Result = Awaited<ReturnType<typeof verifyLiteracyCertificate>>;

  const t = $derived(locale.t);
  const code = $derived(page.params.code ?? '');

  let status = $state<'loading' | 'valid' | 'invalid' | 'error'>('loading');
  let result = $state<Result | null>(null);

  $effect(() => {
    const c = code;
    status = 'loading';
    verifyLiteracyCertificate(c)
      .then((r) => {
        result = r;
        status = r.valid ? 'valid' : 'invalid';
      })
      .catch((cause) => {
        status = cause instanceof ApiError && cause.status === 404 ? 'invalid' : 'error';
      });
  });
</script>

<svelte:head>
  <title>{t('certVerify.title')} — {t('app.name')}</title>
</svelte:head>

<div class="cert">
  <h1>{t('certVerify.title')}</h1>
  <p class="cert__code">{code}</p>

  {#if status === 'loading'}
    <p role="status">{t('state.loading')}</p>
  {:else if status === 'valid' && result}
    <p class="cert__ok" role="status">{t('certVerify.valid')}</p>
    <dl class="cert__facts">
      <dt>{t('certVerify.issuedTo')}</dt>
      <dd>{result.artisan_name}</dd>
      {#if result.craft_name}
        <dt>{t('register.craft.heading')}</dt>
        <dd>{result.craft_name}</dd>
      {/if}
      {#if result.district}
        <dt>{t('register.district.heading')}</dt>
        <dd>{result.district}</dd>
      {/if}
      {#if result.issued_at}
        <dt>{t('certVerify.issuedOn')}</dt>
        <dd>{formatDate(new Date(result.issued_at), locale.code)}</dd>
      {/if}
    </dl>
    <p class="cert__what">{t('certVerify.what')}</p>
  {:else if status === 'invalid'}
    <p class="cert__bad" role="alert">{t('certVerify.invalid')}</p>
  {:else}
    <p role="alert">{t('state.error.body')}</p>
  {/if}
</div>

<style>
  .cert {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    max-inline-size: 36rem;
    margin-inline: auto;
    padding: var(--k-space-6, 2rem) var(--k-space-4);
  }

  .cert h1 {
    margin: 0;
  }

  .cert__code {
    margin: 0;
    font-family: var(--k-font-mono, monospace);
    color: var(--k-text-secondary);
  }

  .cert__ok {
    margin: 0;
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: var(--k-accent-success-bg);
    color: var(--k-text-on-accent);
    font-weight: 700;
  }

  .cert__bad {
    margin: 0;
    color: var(--k-accent-danger);
    font-weight: 700;
  }

  .cert__facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--k-space-1) var(--k-space-3);
    margin: 0;
  }

  .cert__facts dt {
    color: var(--k-text-secondary);
  }

  .cert__facts dd {
    margin: 0;
    font-weight: 600;
  }

  .cert__what {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }
</style>
