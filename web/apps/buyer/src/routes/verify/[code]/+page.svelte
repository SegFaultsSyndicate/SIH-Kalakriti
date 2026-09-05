<script lang="ts">
  import { page } from '$app/state';
  import { API_BASE } from '@kalakriti/api';
  import { locale } from '@kalakriti/i18n';
  import { Button } from '@kalakriti/ui';
  import Seal from '@kalakriti/identity/src/seal.svg';
  import Crest from '@kalakriti/identity/src/heritage-crest-coarse.svg';

  type VerificationRecord = {
    short_code: string;
    content_hash: string;
    signature: string;
    signature_algo: string;
    public_key_id: string;
    technique_matched: boolean;
    sealed_at: string;
    verify_url: string;
    media_hashes: string[];
    listing_title?: string;
    artisan_name?: string;
    artisan_district?: string;
    craft_name?: string;
    signature_valid?: boolean;
  };

  type Status = 'loading' | 'verified' | 'mismatch' | 'insufficient' | 'invalid' | 'error';
  const t = $derived(locale.t);
  const code = $derived(page.url.pathname.split('/').filter(Boolean).slice(-1)[0] ?? '');
  const publicBase = $derived(API_BASE.replace(/\/api\/v1\/?$/, ''));
  let status = $state<Status>('loading');
  let record = $state<VerificationRecord | undefined>();

  async function load(): Promise<void> {
    status = 'loading';
    record = undefined;
    if (!code) {
      status = 'invalid';
      return;
    }

    try {
      const response = await fetch(`${publicBase}/v/${encodeURIComponent(code)}/verify.json`, {
        headers: { Accept: 'application/json' },
      });
      if (response.status === 404) {
        status = 'invalid';
        return;
      }
      if (!response.ok) {
        status = 'error';
        return;
      }

      const data: unknown = await response.json();
      if (!isVerificationRecord(data)) {
        status = 'error';
        return;
      }
      record = data;
      status =
        data.signature_valid === false
          ? 'insufficient'
          : data.technique_matched
            ? 'verified'
            : 'mismatch';
    } catch {
      status = 'error';
    }
  }

  function isVerificationRecord(value: unknown): value is VerificationRecord {
    if (typeof value !== 'object' || value === null) return false;
    const item = value as Record<string, unknown>;
    return (
      typeof item.short_code === 'string' &&
      typeof item.content_hash === 'string' &&
      typeof item.technique_matched === 'boolean' &&
      typeof item.sealed_at === 'string' &&
      Array.isArray(item.media_hashes) &&
      item.media_hashes.every((hash): hash is string => typeof hash === 'string')
    );
  }

  $effect(() => {
    void load();
  });
</script>

<svelte:head>
  <title>{t('buyer.verify.title')} — {t('app.name')}</title>
</svelte:head>

<article class="verify" aria-labelledby="verify-heading">
  <div class="verify__identity" aria-hidden="true">
    <img src={Crest} alt="" />
    <img src={Seal} alt="" />
  </div>
  <p class="verify__eyebrow">{t('buyer.verify.eyebrow')}</p>
  <h1 id="verify-heading">{t('buyer.verify.title')}</h1>

  {#if status === 'loading'}
    <p role="status" aria-live="polite">{t('a11y.loading')}</p>
  {:else if status === 'invalid'}
    <section class="verify__status verify__status--invalid" role="alert">
      <h2>{t('buyer.verify.invalid.title')}</h2>
      <p>{t('buyer.verify.invalid.body', { code })}</p>
    </section>
  {:else if status === 'error'}
    <section class="verify__status verify__status--invalid" role="alert">
      <h2>{t('buyer.verify.error.title')}</h2>
      <p>{t('buyer.verify.error.body')}</p>
      <Button variant="secondary" onclick={() => load()}>{t('state.error.retry')}</Button>
    </section>
  {:else if record}
    <section
      class:verify__status--valid={status === 'verified'}
      class:verify__status--warning={status === 'mismatch' || status === 'insufficient'}
      class="verify__status"
      role="status"
    >
      <h2>
        {status === 'verified'
          ? t('buyer.verify.verified.title')
          : status === 'mismatch'
            ? t('buyer.verify.mismatch.title')
            : t('buyer.verify.insufficient.title')}
      </h2>
      <p>
        {status === 'verified'
          ? t('buyer.verify.verified.body')
          : status === 'mismatch'
            ? t('buyer.verify.mismatch.body')
            : t('buyer.verify.insufficient.body')}
      </p>
    </section>

    <dl class="verify__fields">
      <div>
        <dt>{t('buyer.verify.code')}</dt>
        <dd><code>{record.short_code}</code></dd>
      </div>
      <div>
        <dt>{t('buyer.verify.technique')}</dt>
        <dd>
          {record.technique_matched ? t('buyer.verify.match') : t('buyer.verify.noMatch')}
        </dd>
      </div>
      <div>
        <dt>{t('buyer.verify.sealed')}</dt>
        <dd>{new Date(record.sealed_at).toLocaleDateString(locale.code)}</dd>
      </div>
      <div>
        <dt>{t('buyer.verify.hash')}</dt>
        <dd><code>{record.content_hash}</code></dd>
      </div>
      {#if record.media_hashes.length > 0}
        <div>
          <dt>{t('buyer.verify.mediaHashes')}</dt>
          <dd>
            <ul>
              {#each record.media_hashes as hash}
                <li><code>{hash}</code></li>
              {/each}
            </ul>
          </dd>
        </div>
      {/if}
      {#if record.listing_title}
        <div>
          <dt>{t('buyer.verify.listing')}</dt>
          <dd>{record.listing_title}</dd>
        </div>
      {/if}
      {#if record.artisan_name}
        <div>
          <dt>{t('buyer.verify.artisan')}</dt>
          <dd>{record.artisan_name}</dd>
        </div>
      {/if}
      {#if record.artisan_district}
        <div>
          <dt>{t('buyer.verify.district')}</dt>
          <dd>{record.artisan_district}</dd>
        </div>
      {/if}
      {#if record.craft_name}
        <div>
          <dt>{t('buyer.verify.craft')}</dt>
          <dd>{record.craft_name}</dd>
        </div>
      {/if}
      <div>
        <dt>{t('buyer.verify.handloom')}</dt>
        <dd>{t('buyer.verify.handloomUnavailable')}</dd>
      </div>
    </dl>
  {/if}

  <p class="verify__note">{t('buyer.verify.note')}</p>
  <a class="verify__back" href="/">{t('nav.home')}</a>
</article>

<style>
  .verify {
    position: relative;
    max-inline-size: 42rem;
    margin-inline: auto;
    display: grid;
    gap: var(--k-space-4);
  }
  .verify__identity {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    color: var(--k-text-secondary);
    opacity: 0.72;
  }
  .verify__identity img:first-child {
    inline-size: 4rem;
    block-size: 4rem;
    opacity: 0.2;
  }
  .verify__identity img:last-child {
    inline-size: 3rem;
    block-size: 3rem;
  }
  .verify__eyebrow {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    text-transform: uppercase;
    letter-spacing: var(--k-tracking-wide);
  }
  .verify__status {
    padding: var(--k-space-4);
    border-block: var(--k-rule) solid var(--k-border-interactive);
  }
  .verify__status--valid {
    border-color: var(--k-accent-success);
  }
  .verify__status--warning {
    border-color: var(--k-accent-warning);
  }
  .verify__status--invalid {
    border-color: var(--k-accent-danger);
  }
  .verify__status h2 {
    margin-block: 0 var(--k-space-2);
  }
  .verify__status p {
    margin: 0;
  }
  .verify__fields {
    display: grid;
    gap: var(--k-space-2);
  }
  .verify__fields div {
    display: flex;
    justify-content: space-between;
    gap: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    padding-block: var(--k-space-2);
  }
  .verify__fields dt {
    color: var(--k-text-secondary);
  }
  .verify__fields dd {
    margin: 0;
    text-align: end;
  }
  .verify__fields ul {
    display: grid;
    gap: var(--k-space-1);
    margin: 0;
    padding-inline-start: var(--k-space-4);
    text-align: start;
  }
  code {
    overflow-wrap: anywhere;
    font-family: var(--k-font-mono, monospace);
  }
  .verify__note {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
