<!--
  apps/artisan/src/lib/PriceAdvisory.svelte

    <PriceAdvisory {advice} />

  Renders POST /pricing/advise's response (real, wired since batch 8 --
  see @kalakriti/api's advisePricing). This component only displays; it
  never sets a price. The artisan's own price field lives on the caller's
  page and stays visually primary -- this sits below it as reference, one of
  several inputs to a decision that is always theirs.

  cost_floor gets its own locked, bordered row: BuildAdvisory
  (core-svc/internal/core/domain/pricing.go) clamps recommended_min up to it
  server-side, so the range above can never actually fall below it -- this
  is what makes that guarantee legible rather than just true. anomaly, when
  chosen_price was supplied, is server-computed against that same floor
  (CheckAnomaly) -- this component doesn't re-derive it.

  Every driver row explains itself via its own explanation_key rather than a
  hardcoded string, and carries a SpeakButton, per the batch 9 brief: an
  artisan who cannot read the pricing breakdown should still be able to hear
  it, row by row, in their own language.
-->
<script lang="ts">
  import { locale, formatMoneyRange, type MessageKey } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Money, SpeakButton } from '@kalakriti/ui';
  import { Card } from '@kalakriti/patterns';
  import type { components } from '@kalakriti/api';

  type Advice = components['schemas']['PriceAdvisory'];
  type Driver = NonNullable<Advice['drivers']>[number];

  interface Props {
    advice: Advice;
  }

  let { advice }: Props = $props();

  const t = $derived(locale.t);

  const floor = $derived(advice.drivers?.find((d) => d.name === 'cost_floor'));
  // recommended_min/max repeat the headline range as driver rows; cost_floor
  // gets its own distinct treatment above. Showing all three again here would
  // read as the same number three times, not as more information.
  const otherDrivers = $derived(
    (advice.drivers ?? []).filter((d) => !['cost_floor', 'recommended_min', 'recommended_max'].includes(d.name ?? '')),
  );

  const MONEY_DRIVERS = new Set(['wage_rate', 'market_p25', 'market_p50', 'market_p75']);

  function driverLabel(driver: Driver): MessageKey {
    return `pricing.driver.${driver.name}` as MessageKey;
  }

  function speakTextFor(driver: Driver): string {
    return `${t(driverLabel(driver))}`;
  }
</script>

<div class="k-price-advisory">
  {#if advice.recommended_min?.amount_paise != null && advice.recommended_max?.amount_paise != null}
    <div class="k-price-advisory__range">
      <p class="k-price-advisory__range-label">{t('pricing.advisory.rangeLabel')}</p>
      <p class="k-price-advisory__range-value">
        {formatMoneyRange(advice.recommended_min.amount_paise, advice.recommended_max.amount_paise, locale.code)}
      </p>
      <p class="k-price-advisory__caption">{t('pricing.advisory.caption')}</p>
    </div>
  {/if}

  {#if floor?.value}
    <Card variant="hairline" element="div" class="k-price-advisory__floor">
      <Icon name="lock" class="k-price-advisory__floor-icon" />
      <div>
        <p class="k-price-advisory__floor-label">{t('pricing.advisory.floorLabel')}</p>
        <p class="k-price-advisory__floor-value"><Money paise={Number(floor.value)} /></p>
        <p class="k-price-advisory__floor-explain">{t('pricing.driver.cost_floor')}</p>
      </div>
      <SpeakButton text={t('pricing.driver.cost_floor')} label={t('pricing.advisory.listen')} iconOnly />
    </Card>
  {/if}

  {#if advice.anomaly?.level === 'UNDERPRICED'}
    <div class="k-price-advisory__warning" role="alert">
      <Icon name="warning" />
      <div>
        <p>{t('pricing.anomaly.underpriced')}</p>
        {#if advice.anomaly.shortfall_paise != null}
          <p class="k-price-advisory__shortfall">
            {t('pricing.advisory.shortfall')} <Money paise={advice.anomaly.shortfall_paise} />
          </p>
        {/if}
      </div>
    </div>
  {:else if advice.anomaly?.level === 'OVERPRICED'}
    <div class="k-price-advisory__note" role="status">
      <Icon name="info" />
      <p>{t('pricing.anomaly.overpriced')}</p>
    </div>
  {/if}

  {#if otherDrivers.length > 0}
    <ul class="k-price-advisory__drivers" role="list">
      {#each otherDrivers as driver (driver.name)}
        <li class="k-price-advisory__driver">
          <div class="k-price-advisory__driver-text">
            <p class="k-price-advisory__driver-name">{t(driverLabel(driver))}</p>
            <p class="k-price-advisory__driver-value">
              {#if MONEY_DRIVERS.has(driver.name ?? '') && driver.value}
                <Money paise={Number(driver.value)} />
              {:else}
                {driver.value}
              {/if}
            </p>
          </div>
          <SpeakButton text={speakTextFor(driver)} label={t('pricing.advisory.listen')} iconOnly />
        </li>
      {/each}
    </ul>
  {/if}

  <p class="k-price-advisory__disclaimer">{t('pricing.advisory.disclaimer')}</p>
</div>

<style>
  .k-price-advisory {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .k-price-advisory__range {
    text-align: center;
  }

  .k-price-advisory__range-label {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-price-advisory__range-value {
    font-size: var(--k-text-xl);
    font-weight: 700;
  }

  .k-price-advisory__caption {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  :global(.k-price-advisory__floor) {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border-color: var(--k-text-primary) !important;
  }

  :global(.k-price-advisory__floor-icon) {
    flex-shrink: 0;
    inline-size: 1.5rem;
    block-size: 1.5rem;
  }

  .k-price-advisory__floor-label {
    font-size: var(--k-text-sm);
    font-weight: 600;
  }

  .k-price-advisory__floor-value {
    font-size: var(--k-text-lg);
    font-weight: 700;
  }

  .k-price-advisory__floor-explain {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-price-advisory__warning,
  .k-price-advisory__note {
    display: flex;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
  }

  .k-price-advisory__warning {
    border: var(--k-hairline) solid var(--k-accent-danger);
    color: var(--k-accent-danger);
  }

  .k-price-advisory__note {
    border: var(--k-hairline) solid var(--k-border-hairline);
    color: var(--k-text-secondary);
  }

  .k-price-advisory__shortfall {
    font-weight: 600;
  }

  .k-price-advisory__drivers {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .k-price-advisory__driver {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
    padding-block: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .k-price-advisory__driver-name {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .k-price-advisory__driver-value {
    font-weight: 600;
  }

  .k-price-advisory__disclaimer {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    text-align: center;
  }
</style>
