<!--
  apps/artisan/src/routes/learn/[code]/+page.svelte

  One F15 lesson: three short cards, a practice task where the lesson has
  one (a real photo or voice upload the server checks, or a confirm tap),
  then a picture quiz. Progress is recorded per step, so leaving halfway
  keeps whatever was already done (the server only ever sets flags true).
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, SpeakButton, showToast } from '@kalakriti/ui';
  import { recordLessonProgress, ApiError, messageKeyFor } from '@kalakriti/api';
  import { lessonContent } from '$lib/lessons';
  import { uploadFile } from '$lib/upload';

  const t = $derived(locale.t);
  const lesson = $derived(lessonContent(page.params.code ?? ''));

  type Stage = 'cards' | 'practice' | 'quiz' | 'done';
  let stage = $state<Stage>('cards');
  let card = $state(0);
  let busy = $state(false);
  let wrong = $state(false);
  let file = $state<File | null>(null);

  function fail(cause: unknown): void {
    showToast({ message: t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown'), variant: 'error' });
  }

  function next(): void {
    if (!lesson) return;
    if (card < lesson.cards.length - 1) card += 1;
    else stage = lesson.practice ? 'practice' : 'quiz';
  }

  async function practise(): Promise<void> {
    if (!lesson?.practice) return;
    busy = true;
    try {
      const mediaId = lesson.practice.evidence === 'CONFIRM' ? undefined : file ? await uploadFile(file) : undefined;
      if (lesson.practice.evidence !== 'CONFIRM' && !mediaId) return;
      await recordLessonProgress(lesson.code, {
        practice_done: true,
        ...(mediaId ? { practice_media_id: mediaId } : {}),
      });
      stage = 'quiz';
    } catch (cause) {
      fail(cause);
    } finally {
      busy = false;
    }
  }

  async function answer(i: number): Promise<void> {
    if (!lesson) return;
    if (i !== lesson.quiz.correct) {
      wrong = true;
      return;
    }
    wrong = false;
    busy = true;
    try {
      await recordLessonProgress(lesson.code, { quiz_passed: true });
      stage = 'done';
    } catch (cause) {
      fail(cause);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head>
  <title>{lesson ? t(lesson.title) : t('learn.title')} — {t('app.name')}</title>
</svelte:head>

<div class="lesson">
  {#if !lesson}
    <h1>{t('learn.title')}</h1>
    <p>{t('api.error.not_found')}</p>
    <a href="/learn">{t('action.back')}</a>
  {:else}
    <h1><Icon name={lesson.icon} size="1.5rem" /> {t(lesson.title)}</h1>

    {#if stage === 'cards'}
      {@const text = t(lesson.cards[card])}
      <p class="lesson__step">{card + 1} / {lesson.cards.length}</p>
      <div class="lesson__card" aria-live="polite">
        <p>{text}</p>
        <SpeakButton {text} label={t('action.speak')} />
      </div>
      <Button size="xl" onclick={next}>{t('action.next')} →</Button>
    {:else if stage === 'practice' && lesson.practice}
      {@const text = t(lesson.practice.key)}
      <h2 class="lesson__h2">{t('learn.practice')}</h2>
      <div class="lesson__card">
        <p>{text}</p>
        <SpeakButton {text} label={t('action.speak')} />
      </div>
      {#if lesson.practice.evidence !== 'CONFIRM'}
        <input
          type="file"
          accept={lesson.practice.evidence === 'IMAGE' ? 'image/*' : 'audio/*'}
          capture={lesson.practice.evidence === 'IMAGE' ? 'environment' : 'user'}
          aria-label={text}
          onchange={(e) => (file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null)}
        />
      {/if}
      <Button
        size="xl"
        loading={busy}
        disabled={lesson.practice.evidence !== 'CONFIRM' && !file}
        onclick={practise}>{t('learn.practiceDone')}</Button
      >
    {:else if stage === 'quiz'}
      {@const question = t(lesson.quiz.question)}
      <h2 class="lesson__h2">{t('learn.quiz')}</h2>
      <div class="lesson__card">
        <p>{question}</p>
        <SpeakButton text={question} label={t('action.speak')} />
      </div>
      <div class="lesson__options">
        {#each lesson.quiz.options as opt, i (opt.key)}
          <button type="button" class="lesson__option" disabled={busy} onclick={() => answer(i)}>
            <Icon name={opt.icon} size="2rem" />
            <span>{t(opt.key)}</span>
          </button>
        {/each}
      </div>
      {#if wrong}
        <p class="lesson__wrong" role="alert">{t('learn.wrong')}</p>
      {/if}
    {:else}
      <p class="lesson__done" role="status">{t('learn.done')}</p>
      <Button size="xl" onclick={() => goto('/learn')}>{t('learn.title')}</Button>
    {/if}
  {/if}
</div>

<style>
  .lesson {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
  }

  .lesson h1 {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin: 0;
    font-size: var(--k-text-xl);
  }

  .lesson__h2 {
    margin: 0;
    font-size: var(--k-text-md);
  }

  .lesson__step {
    margin: 0;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .lesson__card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    font-size: var(--k-text-lg);
    line-height: 1.4;
  }

  .lesson__card p {
    margin: 0;
  }

  .lesson__options {
    display: grid;
    gap: var(--k-space-2);
  }

  .lesson__option {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    min-block-size: calc(var(--k-touch-min) * 1.5);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
    text-align: start;
    cursor: pointer;
  }

  .lesson__wrong {
    margin: 0;
    color: var(--k-accent-danger);
    font-weight: 600;
  }

  .lesson__done {
    margin: 0;
    font-size: var(--k-text-lg);
    font-weight: 700;
    color: var(--k-accent-success-muted);
  }
</style>
