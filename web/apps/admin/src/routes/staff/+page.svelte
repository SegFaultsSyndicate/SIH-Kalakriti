<!--
  apps/admin/src/routes/staff/+page.svelte

  F14 staff and assisted mode, for officers and the ministry:
    - field agent productivity (artisans onboarded, listings created);
    - voice-consent links awaiting review -- listen, then mark reviewed;
    - staff accounts (MINISTRY only): create, activate, deactivate. A phone
      on an active account logs in through ordinary OTP as that role.
  Cluster officers only ever see their own scope (clamped by core-svc).
  When VITE_USE_MOCKS=1 is set, $lib/stubs.ts appends sample rows to any
  table real rows don't already cover (deduped) so the page never looks
  empty; unreachable endpoints fall back to the stubs and create/activate
  actions are applied to the local list (a "[mock fallback]" console.warn is
  logged).
-->
<script lang="ts">
  import { locale, formatDate, formatNumber, type MessageKey } from '@kalakriti/i18n';
  import { Button, FieldGroup, Input, Select, showToast } from '@kalakriti/ui';
  import {
    session,
    listAgentProductivity,
    listLinksForReview,
    markLinkReviewed,
    getMediaUrl,
    listStaff,
    createStaff,
    setStaffActive,
    ApiError,
    messageKeyFor,
    type AgentProductivity,
    type LinkForReview,
    type StaffAccount,
    type CreateStaffBody,
  } from '@kalakriti/api';
  import { AGENT_PRODUCTIVITY, LINKS_FOR_REVIEW, STAFF_ACCOUNTS, mergeWithStubs } from '$lib/stubs';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const authorized = $derived(role === 'MINISTRY' || role === 'CLUSTER_OFFICER');
  const ministry = $derived(role === 'MINISTRY');

  const ROLES: Record<string, MessageKey> = {
    FIELD_AGENT: 'staff.role.agent',
    CLUSTER_OFFICER: 'staff.role.officer',
    MINISTRY: 'staff.role.ministry',
  };
  const roleLabel = (r?: string): string => (r && ROLES[r] ? t(ROLES[r]) : (r ?? ''));

  let agents = $state<AgentProductivity[]>([]);
  let reviews = $state<LinkForReview[]>([]);
  let staff = $state<StaffAccount[]>([]);
  let audio = $state<Record<string, string>>({});
  let error = $state('');

  let phone = $state('');
  let name = $state('');
  let newRole = $state('FIELD_AGENT');
  let stateCode = $state('');
  let district = $state('');
  let csc = $state('');
  let creating = $state(false);

  function fail(cause: unknown): string {
    return t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown');
  }

  async function load(): Promise<void> {
    error = '';
    try {
      const [a, r, s] = await Promise.all([
        listAgentProductivity(),
        listLinksForReview(),
        ministry ? listStaff() : Promise.resolve({ staff: [] as StaffAccount[] }),
      ]);
      agents = mergeWithStubs(a.agents ?? [], AGENT_PRODUCTIVITY, (x) => x.agent_id);
      reviews = mergeWithStubs(r.links ?? [], LINKS_FOR_REVIEW, (x) => x.link_id);
      staff = ministry ? mergeWithStubs(s.staff ?? [], STAFF_ACCOUNTS, (x) => x.id) : (s.staff ?? []);
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] staff load:', cause);
        agents = AGENT_PRODUCTIVITY;
        reviews = LINKS_FOR_REVIEW;
        staff = ministry ? STAFF_ACCOUNTS : [];
      } else {
        error = fail(cause);
      }
    }
  }

  $effect(() => {
    if (authorized) void load();
  });

  async function listen(link: LinkForReview): Promise<void> {
    if (!link.voice_media_id || !link.link_id) return;
    try {
      const r = await getMediaUrl(link.voice_media_id);
      if (r.url) audio = { ...audio, [link.link_id]: r.url };
    } catch (cause) {
      showToast({ message: fail(cause), variant: 'error' });
    }
  }

  async function reviewed(link: LinkForReview): Promise<void> {
    if (!link.link_id) return;
    try {
      await markLinkReviewed(link.link_id);
      reviews = reviews.filter((r) => r.link_id !== link.link_id);
    } catch (cause) {
      showToast({ message: fail(cause), variant: 'error' });
    }
  }

  async function create(): Promise<void> {
    creating = true;
    try {
      await createStaff({
        phone_e164: phone.trim(),
        display_name: name.trim(),
        role: newRole as CreateStaffBody['role'],
        ...(stateCode.trim() ? { state_code: stateCode.trim() } : {}),
        ...(district.trim() ? { district: district.trim() } : {}),
        ...(csc.trim() ? { csc_id: csc.trim() } : {}),
      });
      phone = name = stateCode = district = csc = '';
      showToast({ message: t('staff.created'), variant: 'success' });
      await load();
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] createStaff:', cause);
        const fresh: StaffAccount = {
          id: `staff-demo-${Date.now()}`,
          phone_e164: phone.trim(),
          display_name: name.trim(),
          role: newRole as StaffAccount['role'],
          state_code: stateCode.trim() || undefined,
          district: district.trim() || undefined,
          csc_id: csc.trim() || undefined,
          active: true,
          created_at: new Date().toISOString(),
        };
        staff = [fresh, ...staff];
        phone = name = stateCode = district = csc = '';
        newRole = 'FIELD_AGENT';
        showToast({ message: t('staff.created'), variant: 'success' });
      } else {
        showToast({ message: fail(cause), variant: 'error' });
      }
    } finally {
      creating = false;
    }
  }

  async function toggle(s: StaffAccount): Promise<void> {
    if (!s.id) return;
    try {
      await setStaffActive(s.id, !s.active);
      await load();
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1') {
        console.warn('[mock fallback] setStaffActive:', cause);
        staff = staff.map((x) => (x.id === s.id ? { ...x, active: !x.active } : x));
      } else {
        showToast({ message: fail(cause), variant: 'error' });
      }
    }
  }

  const num = (n?: number): string => formatNumber(n ?? 0, locale.code);
  const place = (x: { district?: string; state_code?: string }): string => [x.district, x.state_code].filter(Boolean).join(', ');
</script>

<svelte:head>
  <title>{t('staff.title')} — {t('admin.home.title')}</title>
</svelte:head>

<div class="staff">
  <h1>{t('staff.title')}</h1>

  {#if !authorized}
    <p>{t('admin.home.noAccess')}</p>
  {:else}
    {#if error}
      <p class="staff__error" role="alert">{error}</p>
    {/if}

    <section class="staff__section">
      <h2>{t('staff.productivity')}</h2>
      <div class="staff__scroll">
        <table class="staff__table">
          <thead>
            <tr>
              <th scope="col">{t('staff.name')}</th>
              <th scope="col">{t('staff.role')}</th>
              <th scope="col">{t('insights.district')}</th>
              <th scope="col">{t('staff.onboarded')}</th>
              <th scope="col">{t('staff.listings')}</th>
              <th scope="col">{t('staff.lastActive')}</th>
            </tr>
          </thead>
          <tbody>
            {#each agents as a (a.agent_id)}
              <tr class:staff__off={!a.active}>
                <th scope="row">{a.display_name}</th>
                <td>{roleLabel(a.role)}</td>
                <td>{place(a)}</td>
                <td>{num(a.artisans_onboarded)}</td>
                <td>{num(a.listings_created)}</td>
                <td>{a.last_active_at ? formatDate(a.last_active_at, locale.code) : '—'}</td>
              </tr>
            {:else}
              <tr><td colspan="6">{t('insights.noData')}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>

    <section class="staff__section">
      <h2>{t('staff.review.title')}</h2>
      <p class="staff__muted">{t('staff.review.desc')}</p>
      {#if reviews.length === 0}
        <p class="staff__muted">{t('staff.review.empty')}</p>
      {:else}
        <ul class="staff__reviews" role="list">
          {#each reviews as r (r.link_id)}
            <li class="staff__review">
              <p><strong>{r.artisan_name}</strong> · {place(r)}</p>
              <p class="staff__muted">{r.agent_name}{r.created_at ? ` · ${formatDate(r.created_at, locale.code)}` : ''}</p>
              {#if r.link_id && audio[r.link_id]}
                <audio controls src={audio[r.link_id]}></audio>
              {:else if r.voice_media_id}
                <Button size="sm" variant="secondary" onclick={() => listen(r)}>{t('staff.review.listen')}</Button>
              {:else}
                <p class="staff__muted">{t('staff.review.noRecording')}</p>
              {/if}
              <Button size="sm" onclick={() => reviewed(r)}>{t('staff.review.done')}</Button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    {#if ministry}
      <section class="staff__section">
        <h2>{t('staff.accounts')}</h2>
        <form
          class="staff__form"
          onsubmit={(e) => {
            e.preventDefault();
            void create();
          }}
        >
          <FieldGroup label={t('staff.phone')}>
            {#snippet children({ id })}
              <Input {id} type="tel" bind:value={phone} placeholder="+919876543210" />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('staff.name')}>
            {#snippet children({ id })}
              <Input {id} bind:value={name} />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('staff.role')}>
            {#snippet children({ id })}
              <Select {id} bind:value={newRole} options={Object.keys(ROLES).map((r) => ({ value: r, label: roleLabel(r) }))} />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('impact.filter.state')} optional>
            {#snippet children({ id })}
              <Input {id} bind:value={stateCode} placeholder="IN-UP" />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('insights.district')} optional>
            {#snippet children({ id })}
              <Input {id} bind:value={district} />
            {/snippet}
          </FieldGroup>
          <FieldGroup label={t('staff.csc')} optional>
            {#snippet children({ id })}
              <Input {id} bind:value={csc} />
            {/snippet}
          </FieldGroup>
          <Button type="submit" loading={creating}>{t('staff.create')}</Button>
        </form>

        <div class="staff__scroll">
          <table class="staff__table">
            <thead>
              <tr>
                <th scope="col">{t('staff.name')}</th>
                <th scope="col">{t('staff.phone')}</th>
                <th scope="col">{t('staff.role')}</th>
                <th scope="col">{t('insights.district')}</th>
                <th scope="col"><span class="k-visually-hidden">{t('staff.status')}</span></th>
              </tr>
            </thead>
            <tbody>
              {#each staff as s (s.id)}
                <tr class:staff__off={!s.active}>
                  <th scope="row">{s.display_name}</th>
                  <td>{s.phone_e164}</td>
                  <td>{roleLabel(s.role)}</td>
                  <td>{place(s)}</td>
                  <td>
                    <Button size="sm" variant={s.active ? 'ghost' : 'secondary'} onclick={() => toggle(s)}>
                      {s.active ? t('staff.deactivate') : t('staff.activate')}
                    </Button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .staff {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-5, 1.5rem);
  }

  .staff h1 {
    margin: 0;
  }

  .staff h2 {
    margin: 0;
    font-size: var(--k-text-lg);
  }

  .staff__section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .staff__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .staff__error {
    margin: 0;
    color: var(--k-accent-danger);
  }

  .staff__scroll {
    overflow-x: auto;
  }

  .staff__table {
    inline-size: 100%;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .staff__table th,
  .staff__table td {
    padding: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    text-align: start;
  }

  .staff__off {
    opacity: 0.6;
  }

  .staff__reviews {
    display: grid;
    gap: var(--k-space-3);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .staff__review {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
  }

  .staff__review p {
    margin: 0;
  }

  .staff__form {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--k-space-3);
    align-items: end;
  }

  .staff__form :global(.k-field-group) {
    margin-block-end: 0;
  }
</style>
