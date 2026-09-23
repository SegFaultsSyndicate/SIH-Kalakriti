<!--
  apps/artisan/src/routes/learn/+page.svelte

  F15: the 8-lesson digital literacy track, progress from the server, and
  the signed certificate once every lesson is complete. The certificate's
  QR opens the public /verify/certificate/{code} page (buyer app).
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, showToast } from '@kalakriti/ui';
  import {
    listLessons,
    getLiteracyCertificate,
    issueLiteracyCertificate,
    type Lesson,
    type LiteracyCertificate,
  } from '@kalakriti/api';
  import { LESSONS } from '$lib/lessons';

  const t = $derived(locale.t);

  let progress = $state<Lesson[] | null>(null);
  let completed = $state(0);
  let cert = $state<LiteracyCertificate | null>(null);
  let failed = $state(false);
  let issuing = $state(false);

  async function load(): Promise<void> {
    failed = false;
    try {
      const r = await listLessons();
      progress = r.lessons ?? [];
      completed = r.completed_count ?? 0;
      if (r.certificate_issued) cert = (await getLiteracyCertificate()).certificate ?? null;
    } catch {
      failed = true;
    }
  }

  $effect(() => {
    void load();
  });

  const done = (code: string): boolean => progress?.find((p) => p.code === code)?.completed ?? false;

  async function issue(): Promise<void> {
    issuing = true;
    try {
      cert = (await issueLiteracyCertificate()).certificate ?? null;
    } catch {
      showToast({ message: t('api.error.unknown'), variant: 'error' });
    } finally {
      issuing = false;
    }
  }
</script>

<svelte:head>
  <title>{t('learn.title')} — {t('app.name')}</title>
</svelte:head>

<div class="learn">
  <h1>{t('learn.title')}</h1>
  <p class="learn__muted">{t('learn.intro')}</p>

  {#if failed}
    <p class="learn__muted">{t('state.error.body')}</p>
    <Button variant="ghost" onclick={load}>{t('state.error.retry')}</Button>
  {:else if progress}
    <p class="learn__progress">{t('learn.progress', { done: completed, total: LESSONS.length })}</p>
    <progress max={LESSONS.length} value={completed} aria-hidden="true"></progress>

    <ol class="learn__list">
      {#each LESSONS as lesson (lesson.code)}
        <li>
          <a class="learn__item" class:learn__item--done={done(lesson.code)} href="/learn/{lesson.code}">
            <Icon name={done(lesson.code) ? 'success' : lesson.icon} size="1.5rem" />
            <span>{t(lesson.title)}</span>
          </a>
        </li>
      {/each}
    </ol>

    <section class="learn__cert" aria-labelledby="cert-title">
      <h2 id="cert-title">{t('learn.cert.title')}</h2>
      {#if cert}
        <p>{t('learn.cert.code', { code: cert.short_code ?? '' })}</p>
        {#if cert.download_url}
          <a class="learn__download" href={cert.download_url} target="_blank" rel="noopener">{t('learn.cert.download')}</a>
        {/if}
      {:else if completed >= LESSONS.length}
        <Button size="xl" loading={issuing} onclick={issue}>{t('learn.cert.get')}</Button>
      {:else}
        <p class="learn__muted">{t('learn.cert.locked')}</p>
      {/if}
    </section>
  {/if}
</div>

<style>
  .learn {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
  }

  .learn h1 {
    margin: 0;
    font-size: var(--k-text-xl);
  }

  .learn__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .learn__progress {
    margin: 0;
    font-weight: 700;
  }

  progress {
    inline-size: 100%;
    accent-color: var(--k-accent-success-bg);
  }

  .learn__list {
    display: grid;
    gap: var(--k-space-2);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .learn__item {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    min-block-size: calc(var(--k-touch-min) * 1.25);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-weight: 600;
    text-decoration: none;
  }

  .learn__item--done {
    border-color: var(--k-accent-success-bg);
    color: var(--k-accent-success-muted);
  }

  .learn__cert {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-4);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-sunken);
  }

  .learn__cert h2 {
    margin: 0;
    font-size: var(--k-text-md);
  }

  .learn__cert p {
    margin: 0;
  }

  .learn__download {
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }
</style>
