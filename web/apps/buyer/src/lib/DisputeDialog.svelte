<!--
  apps/buyer/src/lib/DisputeDialog.svelte

    <DisputeDialog bind:open={disputeOpen} {orderId} />

  The dispute "entry point" the brief asks for -- honestly bounded. There is
  no dispute RPC anywhere in fulfilment.proto or collab-svc, so this cannot
  submit anything to the backend without inventing an endpoint. What it
  offers instead is real: a mailto: link carrying the actual order id, which
  a person on the support side can act on. Documented as a gap in
  ml_wiring.md -- a real in-app dispute flow needs a backend RPC first.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Dialog, Button } from '@kalakriti/ui';

  interface Props {
    open: boolean;
    orderId: string;
  }

  let { open = $bindable(false), orderId }: Props = $props();

  const t = $derived(locale.t);
  const mailHref = $derived(
    `mailto:support@kalakriti.example?subject=${encodeURIComponent(t('orders.dispute.mailSubject', { orderId }))}`,
  );
</script>

<Dialog bind:open title={t('orders.dispute.heading')}>
  <p>{t('orders.dispute.body')}</p>
  <p class="dispute-dialog__id">{orderId}</p>
  <div class="dispute-dialog__actions">
    <a class="dispute-dialog__mail-link" href={mailHref}>{t('orders.dispute.mailLink')}</a>
    <Button variant="secondary" onclick={() => (open = false)}>{t('orders.dispute.close')}</Button>
  </div>
</Dialog>

<style>
  .dispute-dialog__id {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    background: var(--k-surface-sunken);
    padding: var(--k-space-2);
    border-radius: var(--k-radius-sm);
    word-break: break-all;
  }

  .dispute-dialog__actions {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-4);
  }

  .dispute-dialog__mail-link {
    display: inline-flex;
    align-items: center;
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-radius: var(--k-radius-md);
    text-decoration: none;
    font-weight: var(--k-weight-semibold);
  }
</style>
