<!--
  apps/artisan/src/lib/LogSaleSheet.svelte

  F13 "Log a sale": a sale made outside Kalakriti (a fair, the local
  market, a neighbour). Queued through the outbox like every other write,
  so it works with no signal; the entry carries its own id, which the
  server uses as the row id, so a replay after a flaky connection cannot
  count the same sale twice.
-->
<script lang="ts">
  import { locale, type MessageKey } from '@kalakriti/i18n';
  import { Sheet, Button, FieldGroup, Input, showToast } from '@kalakriti/ui';
  import { enqueue } from '@kalakriti/offline';
  import { getActingFor } from '@kalakriti/api';

  interface Props {
    open?: boolean;
  }

  let { open = $bindable(false) }: Props = $props();

  const t = $derived(locale.t);

  const CHANNELS: readonly { code: string; key: MessageKey }[] = [
    { code: 'FAIR', key: 'income.sale.fair' },
    { code: 'LOCAL_MARKET', key: 'income.sale.market' },
    { code: 'DIRECT', key: 'income.sale.direct' },
    { code: 'OTHER', key: 'income.sale.other' },
  ];

  const today = (): string => new Date().toLocaleDateString('en-CA'); // YYYY-MM-DD, local day

  let channel = $state('FAIR');
  let rupees = $state('');
  let soldOn = $state(today());
  let eventName = $state('');
  let saving = $state(false);

  const amountPaise = $derived(Math.round(Number(rupees.replace(/[^\d.]/g, '')) * 100));
  const valid = $derived(Number.isFinite(amountPaise) && amountPaise > 0 && soldOn !== '' && soldOn <= today());

  async function save(): Promise<void> {
    if (!valid) return;
    saving = true;
    try {
      await enqueue({
        kind: 'income.sale',
        onBehalfOf: getActingFor(),
        payload: {
          id: crypto.randomUUID(),
          channel,
          amount_paise: amountPaise,
          sold_on: soldOn,
          ...(channel === 'FAIR' && eventName.trim() ? { event_name: eventName.trim() } : {}),
        },
      });
      showToast({ message: t('income.sale.saved'), variant: 'success' });
      rupees = '';
      eventName = '';
      soldOn = today();
      open = false;
    } finally {
      saving = false;
    }
  }
</script>

<Sheet bind:open title={t('income.sale.log')}>
  <div class="sale">
    <div class="sale__channels" role="radiogroup" aria-label={t('income.sale.where')}>
      {#each CHANNELS as c (c.code)}
        <button
          type="button"
          role="radio"
          aria-checked={channel === c.code}
          class="sale__channel"
          class:sale__channel--on={channel === c.code}
          onclick={() => (channel = c.code)}
        >
          {t(c.key)}
        </button>
      {/each}
    </div>

    <FieldGroup label={t('income.sale.amount')}>
      {#snippet children({ id })}
        <Input {id} bind:value={rupees} inputmode="decimal" autocomplete="off" />
      {/snippet}
    </FieldGroup>

    <FieldGroup label={t('income.sale.date')}>
      {#snippet children({ id })}
        <Input {id} type="date" bind:value={soldOn} max={today()} />
      {/snippet}
    </FieldGroup>

    {#if channel === 'FAIR'}
      <FieldGroup label={t('income.sale.event')} optional>
        {#snippet children({ id })}
          <Input {id} bind:value={eventName} maxlength={120} />
        {/snippet}
      </FieldGroup>
    {/if}

    <Button size="xl" disabled={!valid} loading={saving} onclick={save}>{t('action.save')}</Button>
  </div>
</Sheet>

<style>
  .sale {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .sale__channels {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--k-space-2);
  }

  .sale__channel {
    min-block-size: var(--k-touch-min);
    padding: var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
    cursor: pointer;
  }

  .sale__channel--on {
    border-color: var(--k-accent-primary-bg);
    border-width: var(--k-rule);
    font-weight: 700;
  }
</style>
