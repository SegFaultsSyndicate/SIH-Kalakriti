<!--
  apps/artisan/src/routes/agent/add/+page.svelte

  F14: a field agent links an artisan, with the artisan's consent:
    - ARTISAN_OTP: an OTP goes to the artisan's own phone and they read it
      out -- proof the phone's owner agreed, same as their own login.
    - VOICE_RECORDING: no signal on the artisan's phone. The link is made
      flagged for officer review, then the artisan's spoken consent is
      recorded and attached (it can only be uploaded once the link exists,
      because media belongs to an artisan).
  A phone with no Kalakriti profile is registered in the same call, from
  the same fields the artisan's own /register flow collects.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Button, FieldGroup, Input, OtpInput, Select } from '@kalakriti/ui';
  import {
    startArtisanConsent,
    linkArtisan,
    attachVoiceConsent,
    ApiError,
    messageKeyFor,
    type LinkArtisanBody,
  } from '@kalakriti/api';
  import { isValidIndianMobile, toE164 } from '$lib/phone';
  import { buildRegisterBody } from '$lib/registration';
  import { loadCrafts, STATES, type Craft } from '$lib/ontology';
  import { acting } from '$lib/acting.svelte';
  import { uploadFile } from '$lib/upload';

  const t = $derived(locale.t);

  type Step = 'phone' | 'otp' | 'voice' | 'done';
  let step = $state<Step>('phone');
  let busy = $state(false);
  let error = $state('');

  let digits = $state('');
  let isNew = $state(false);
  let name = $state('');
  let craftId = $state('');
  let stateName = $state('');
  let district = $state('');
  let crafts = $state<Craft[]>([]);

  let challengeId = $state('');
  let otp = $state('');
  let linked = $state<{ id: string; name: string } | null>(null);
  let voiceFile = $state<File | null>(null);

  $effect(() => {
    void loadCrafts().then((c) => (crafts = c));
  });

  const phoneOk = $derived(isValidIndianMobile(digits));
  const regOk = $derived(!isNew || (name.trim() !== '' && craftId !== '' && stateName !== ''));

  function fail(cause: unknown): void {
    error = t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown');
  }

  function baseBody(method: 'ARTISAN_OTP' | 'VOICE_RECORDING'): LinkArtisanBody {
    const body: LinkArtisanBody = { phone_e164: toE164(digits), consent_method: method };
    if (isNew) {
      body.registration = buildRegisterBody(
        { name, craftId, stateFreeText: stateName, districtFreeText: district.trim() || undefined },
        locale.code,
      ) as unknown as Record<string, unknown>;
    }
    return body;
  }

  async function sendOtp(): Promise<void> {
    busy = true;
    error = '';
    try {
      const r = await startArtisanConsent({ phone_e164: toE164(digits) });
      challengeId = r.challenge_id ?? '';
      step = 'otp';
    } catch (cause) {
      fail(cause);
    } finally {
      busy = false;
    }
  }

  async function confirmOtp(code: string): Promise<void> {
    busy = true;
    error = '';
    try {
      const r = await linkArtisan({ ...baseBody('ARTISAN_OTP'), challenge_id: challengeId, otp: code });
      linked = { id: r.artisan_id ?? '', name: name.trim() || toE164(digits) };
      step = 'done';
    } catch (cause) {
      otp = '';
      fail(cause);
    } finally {
      busy = false;
    }
  }

  async function startVoice(): Promise<void> {
    busy = true;
    error = '';
    try {
      const r = await linkArtisan(baseBody('VOICE_RECORDING'));
      linked = { id: r.artisan_id ?? '', name: name.trim() || toE164(digits) };
      step = 'voice';
    } catch (cause) {
      fail(cause);
    } finally {
      busy = false;
    }
  }

  async function uploadVoice(): Promise<void> {
    if (!voiceFile || !linked) return;
    busy = true;
    error = '';
    try {
      // The recording is the artisan's media, so it is uploaded on their
      // behalf; attaching it to the link is the agent's own call.
      const media_id = await uploadFile(voiceFile, { onBehalfOf: linked.id });
      await attachVoiceConsent({ artisan_id: linked.id, media_id });
      step = 'done';
    } catch (cause) {
      fail(cause);
    } finally {
      busy = false;
    }
  }

  function startHelping(): void {
    if (!linked) return;
    acting.start(linked);
    void goto('/');
  }
</script>

<svelte:head>
  <title>{t('agent.add')} — {t('app.name')}</title>
</svelte:head>

<div class="add">
  <h1>{t('agent.add')}</h1>

  {#if step === 'phone'}
    <FieldGroup label={t('agent.add.phone')}>
      {#snippet children({ id })}
        <Input {id} type="tel" bind:value={digits} inputmode="numeric" maxlength={10} autocomplete="off" />
      {/snippet}
    </FieldGroup>

    <label class="add__check">
      <input type="checkbox" bind:checked={isNew} />
      <span>{t('agent.add.isNew')}</span>
    </label>

    {#if isNew}
      <FieldGroup label={t('register.name.heading')}>
        {#snippet children({ id })}
          <Input {id} bind:value={name} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('register.craft.heading')}>
        {#snippet children({ id })}
          <Select {id} bind:value={craftId} options={[{ value: '', label: '—' }, ...crafts.map((c) => ({ value: c.id, label: c.displayName }))]} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('agent.add.state')}>
        {#snippet children({ id })}
          <Select {id} bind:value={stateName} options={[{ value: '', label: '—' }, ...STATES.map((s) => ({ value: s, label: s }))]} />
        {/snippet}
      </FieldGroup>
      <FieldGroup label={t('register.district.heading')} optional>
        {#snippet children({ id })}
          <Input {id} bind:value={district} />
        {/snippet}
      </FieldGroup>
    {/if}

    <p class="add__muted">{t('agent.add.consentNote')}</p>
    <Button size="xl" disabled={!phoneOk || !regOk} loading={busy} onclick={sendOtp}>{t('agent.add.sendOtp')}</Button>
    <Button variant="secondary" disabled={!phoneOk || !regOk || busy} onclick={startVoice}>{t('agent.add.noSignal')}</Button>
  {:else if step === 'otp'}
    <p class="add__muted">{t('agent.add.readOtp')}</p>
    <OtpInput bind:value={otp} label={t('verify.code.label')} disabled={busy} oncomplete={confirmOtp} autofocus />
  {:else if step === 'voice'}
    <p class="add__muted">{t('agent.voice.intro')}</p>
    <blockquote class="add__script">{t('agent.voice.script', { name: linked?.name ?? '' })}</blockquote>
    <FieldGroup label={t('agent.voice.record')}>
      {#snippet children({ id })}
        <input
          {id}
          type="file"
          accept="audio/*"
          capture="user"
          onchange={(e) => (voiceFile = (e.currentTarget as HTMLInputElement).files?.[0] ?? null)}
        />
      {/snippet}
    </FieldGroup>
    <Button size="xl" disabled={!voiceFile} loading={busy} onclick={uploadVoice}>{t('action.save')}</Button>
  {:else}
    <p class="add__done">{t('agent.add.done', { name: linked?.name ?? '' })}</p>
    <Button size="xl" onclick={startHelping}>{t('agent.help')}</Button>
    <Button variant="ghost" onclick={() => goto('/agent')}>{t('action.back')}</Button>
  {/if}

  {#if error}
    <p class="add__error" role="alert">{error}</p>
  {/if}
</div>

<style>
  .add {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
  }

  .add h1 {
    margin: 0;
    font-size: var(--k-text-xl);
  }

  .add__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .add__check {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    min-block-size: var(--k-touch-min);
  }

  .add__script {
    margin: 0;
    padding: var(--k-space-3);
    border-inline-start: 4px solid var(--k-accent-primary-bg);
    background: var(--k-surface-sunken);
    font-size: var(--k-text-md);
  }

  .add__done {
    margin: 0;
    font-weight: 700;
  }

  .add__error {
    margin: 0;
    color: var(--k-accent-danger);
  }
</style>
