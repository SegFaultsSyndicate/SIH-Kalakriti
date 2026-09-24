<!--
  apps/artisan/src/routes/finance/+page.svelte

  F12: link a MoSJE finance-corporation loan to the shop and see whether
  this month's sales cover the EMI. The full loan number never leaves this
  form except to the server, which hashes it and keeps the last 4
  characters only. Linking needs explicit DPDP consent; removing a link is
  consent withdrawal and a hard delete, and only the artisan themself can
  do it (hidden while an agent is helping).
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Button, Checkbox, FieldGroup, Input, Money, Select, Dialog, showToast } from '@kalakriti/ui';
  import {
    listFinanceLinks,
    linkFinance,
    deleteFinanceLink,
    getRepaymentCoverage,
    ApiError,
    messageKeyFor,
    type FinanceLink,
    type RepaymentCoverage,
    type LinkFinanceBody,
  } from '@kalakriti/api';
  import { acting } from '$lib/acting.svelte';
  import { MOCK_FINANCE_LINKS, MOCK_REPAYMENT_COVERAGE } from '$lib/mock-data';

  const t = $derived(locale.t);

  // Must match core-svc's domain.FinanceConsentVersion: the server records
  // which consent text the artisan actually saw.
  const CONSENT_VERSION = 'finance-consent-2026-09';

  const CORPORATIONS: { value: string; label: string }[] = [
    { value: 'NSFDC', label: 'NSFDC' },
    { value: 'NBCFDC', label: 'NBCFDC' },
    { value: 'NSKFDC', label: 'NSKFDC' },
    { value: 'NDFDC', label: 'NDFDC' },
    { value: 'PM_DAKSH', label: 'PM-DAKSH' },
    { value: 'PM_AJAY', label: 'PM-AJAY' },
  ];
  const corpLabel = (v?: string): string => CORPORATIONS.find((c) => c.value === v)?.label ?? t('income.sale.other');

  const STATUS: Record<string, MessageKey> = {
    SELF_REPORTED: 'finance.status.self',
    VERIFIED: 'finance.status.verified',
    REJECTED: 'finance.status.rejected',
  };
  const COVERAGE: Record<string, MessageKey> = {
    COVERED: 'finance.coverage.covered',
    ALMOST: 'finance.coverage.almost',
    NOT_YET: 'finance.coverage.notYet',
  };

  let links = $state<FinanceLink[] | null>(null);
  let coverage = $state<RepaymentCoverage | null>(null);
  let failed = $state(false);

  let adding = $state(false);
  let corporation = $state('NSFDC');
  let reference = $state('');
  let emiRupees = $state('');
  let emiDay = $state('');
  let consent = $state(false);
  let saving = $state(false);
  let formError = $state('');

  let removing = $state<FinanceLink | null>(null);
  let removeOpen = $state(false);

  async function load(): Promise<void> {
    failed = false;
    try {
      const [l, c] = await Promise.all([listFinanceLinks(), getRepaymentCoverage()]);
      links = l.links ?? [];
      coverage = c.coverage ?? null;
      if (links.length === 0 && (import.meta.env.VITE_USE_MOCKS === '1' || import.meta.env.DEV)) {
        links = MOCK_FINANCE_LINKS;
        coverage = MOCK_REPAYMENT_COVERAGE;
      }
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1' || import.meta.env.DEV) {
        console.warn('[mock fallback] finance load:', cause);
        links = MOCK_FINANCE_LINKS;
        coverage = MOCK_REPAYMENT_COVERAGE;
        failed = false;
      } else {
        failed = true;
      }
    }
  }

  $effect(() => {
    void load();
  });

  const emiPaise = $derived(emiRupees.trim() ? Math.round(Number(emiRupees.replace(/[^\d.]/g, '')) * 100) : undefined);
  const dayNum = $derived(emiDay.trim() ? Number(emiDay) : undefined);
  const valid = $derived(
    consent &&
      reference.trim().length >= 4 &&
      (emiPaise === undefined || emiPaise > 0) &&
      (dayNum === undefined || (Number.isInteger(dayNum) && dayNum >= 1 && dayNum <= 28)),
  );

  async function save(): Promise<void> {
    if (!valid) return;
    saving = true;
    formError = '';
    try {
      await linkFinance({
        corporation: corporation as LinkFinanceBody['corporation'],
        reference: reference.trim(),
        consent_given: true,
        consent_version: CONSENT_VERSION,
        ...(emiPaise !== undefined ? { emi_paise: emiPaise } : {}),
        ...(dayNum !== undefined ? { emi_day_of_month: dayNum } : {}),
      });
      reference = '';
      emiRupees = '';
      emiDay = '';
      consent = false;
      adding = false;
      showToast({ message: t('finance.linked'), variant: 'success' });
      await load();
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1' || import.meta.env.DEV) {
        console.warn('[mock fallback] linkFinance:', cause);
        const newLink: FinanceLink = {
          id: `link-mock-${Date.now()}`,
          corporation: corporation as any,
          reference_last4: reference.trim().slice(-4),
          emi_paise: emiPaise ?? 250000,
          emi_day_of_month: dayNum ?? 10,
          status: 'SELF_REPORTED',
          created_at: new Date().toISOString(),
        };
        links = [newLink, ...(links ?? [])];
        reference = '';
        emiRupees = '';
        emiDay = '';
        consent = false;
        adding = false;
        showToast({ message: t('finance.linked'), variant: 'success' });
      } else {
        formError = t(cause instanceof ApiError ? messageKeyFor(cause) : 'api.error.unknown');
      }
    } finally {
      saving = false;
    }
  }

  async function remove(): Promise<void> {
    if (!removing?.id) return;
    try {
      await deleteFinanceLink(removing.id);
      removeOpen = false;
      await load();
    } catch (cause) {
      if (import.meta.env.VITE_USE_MOCKS === '1' || import.meta.env.DEV) {
        console.warn('[mock fallback] deleteFinanceLink:', cause);
        links = (links ?? []).filter((l) => l.id !== removing?.id);
        removeOpen = false;
        removing = null;
        showToast({ message: t('finance.link.deleted'), variant: 'success' });
      } else {
        showToast({ message: t('api.error.unknown'), variant: 'error' });
      }
    }
  }
</script>

<svelte:head>
  <title>{t('finance.title')} — {t('app.name')}</title>
</svelte:head>

<div class="finance">
  <h1>{t('finance.title')}</h1>
  <p class="finance__intro">{t('finance.intro')}</p>

  {#if failed}
    <p class="finance__muted">{t('state.error.body')}</p>
    <Button variant="ghost" onclick={load}>{t('state.error.retry')}</Button>
  {:else if links !== null}
    {#if coverage && coverage.status && coverage.status !== 'NO_EMI'}
      <section class="cover cover--{coverage.status.toLowerCase()}" aria-labelledby="cover-title">
        <h2 id="cover-title" class="cover__title">{t(COVERAGE[coverage.status] ?? 'finance.coverage.notYet')}</h2>
        <dl class="cover__grid">
          <div>
            <dt>{t('finance.coverage.earned')}</dt>
            <dd><Money paise={(coverage.earned_platform_paise ?? 0) + (coverage.earned_offline_paise ?? 0)} /></dd>
          </div>
          <div>
            <dt>{t('finance.coverage.emi')}</dt>
            <dd><Money paise={coverage.emi_paise ?? 0} /></dd>
          </div>
        </dl>
        {#if (coverage.days_to_emi ?? -1) >= 0}
          <p class="finance__muted">{t('finance.coverage.days', { days: coverage.days_to_emi ?? 0 })}</p>
        {/if}
        {#if (coverage.pending_paise ?? 0) > 0}
          <p class="finance__muted">{t('income.card.pending')}: <Money paise={coverage.pending_paise ?? 0} /></p>
        {/if}
      </section>
    {/if}

    {#if links.length > 0}
      <ul class="links" role="list">
        {#each links as link (link.id)}
          <li class="link">
            <div>
              <p class="link__corp">{corpLabel(link.corporation)} <span class="link__ref">•••• {link.reference_last4}</span></p>
              <p class="finance__muted">
                {t(STATUS[link.status ?? ''] ?? 'finance.status.self')}
                {#if link.emi_paise}· {t('finance.coverage.emi')}: <Money paise={link.emi_paise} />{/if}
              </p>
              {#if link.status === 'REJECTED' && link.reject_reason}
                <p class="finance__muted">{link.reject_reason}</p>
              {/if}
            </div>
            {#if !acting.current}
              <Button
                size="md"
                variant="ghost"
                onclick={() => {
                  removing = link;
                  removeOpen = true;
                }}>{t('action.remove')}</Button
              >
            {/if}
          </li>
        {/each}
      </ul>
    {/if}

    {#if adding}
      <section class="form" aria-labelledby="form-title">
        <h2 id="form-title" class="cover__title">{t('finance.add')}</h2>
        <FieldGroup label={t('finance.corporation')}>
          {#snippet children({ id })}
            <Select {id} bind:value={corporation} options={CORPORATIONS} />
          {/snippet}
        </FieldGroup>
        <FieldGroup label={t('finance.reference')} description={t('finance.referenceHint')}>
          {#snippet children({ id, describedBy })}
            <Input {id} bind:value={reference} autocomplete="off" aria-describedby={describedBy} maxlength={64} />
          {/snippet}
        </FieldGroup>
        <FieldGroup label={t('finance.emi')} optional>
          {#snippet children({ id })}
            <Input {id} bind:value={emiRupees} inputmode="decimal" />
          {/snippet}
        </FieldGroup>
        <FieldGroup label={t('finance.emiDay')} optional>
          {#snippet children({ id })}
            <Input {id} bind:value={emiDay} inputmode="numeric" maxlength={2} />
          {/snippet}
        </FieldGroup>
        <Checkbox bind:checked={consent}>{t('finance.consent')}</Checkbox>
        {#if formError}
          <p class="finance__error" role="alert">{formError}</p>
        {/if}
        <div class="form__actions">
          <Button variant="ghost" onclick={() => (adding = false)}>{t('action.cancel')}</Button>
          <Button disabled={!valid} loading={saving} onclick={save}>{t('action.save')}</Button>
        </div>
      </section>
    {:else}
      <Button size="xl" onclick={() => (adding = true)}>{t('finance.add')}</Button>
    {/if}
  {/if}
</div>

<Dialog bind:open={removeOpen} title={t('action.remove')}>
  <p>{t('finance.removeConfirm')}</p>
  <div class="form__actions">
    <Button variant="ghost" onclick={() => (removeOpen = false)}>{t('action.cancel')}</Button>
    <Button variant="danger" onclick={remove}>{t('action.remove')}</Button>
  </div>
</Dialog>

<style>
  .finance {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
  }

  .finance h1 {
    margin: 0;
    font-size: var(--k-text-xl);
  }

  .finance__intro,
  .finance__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .finance__error {
    margin: 0;
    color: var(--k-accent-danger);
    font-size: var(--k-text-sm);
  }

  .cover,
  .form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
  }

  .cover--covered {
    border-color: var(--k-accent-success-bg);
    border-width: var(--k-rule);
  }

  .cover__title {
    margin: 0;
    font-size: var(--k-text-md);
  }

  .cover__grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--k-space-2);
    margin: 0;
  }

  .cover__grid dt {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .cover__grid dd {
    margin: 0;
    font-weight: 700;
  }

  .links {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .link {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding-block: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .link__corp {
    margin: 0;
    font-weight: 700;
  }

  .link__ref {
    font-weight: 400;
    font-variant-numeric: tabular-nums;
    color: var(--k-text-secondary);
  }

  .form__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
  }
</style>
