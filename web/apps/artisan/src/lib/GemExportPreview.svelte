<!--
  apps/artisan/src/lib/GemExportPreview.svelte

  Formatted preview of a published listing in GeM (Government e-Marketplace)
  catalog format. This is a mock preview — no real GeM API is called. It
  renders a Dialog with a structured product card that the artisan can print
  or copy as plain text to paste into a real GeM registration later.

  The card mirrors the fields GeM requires for a seller catalog entry:
  product name, HSN code, unit price, seller details, and category.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, Dialog, showToast } from '@kalakriti/ui';
  import type { Listing } from '$lib/listings';
  import { titleFor } from '$lib/listings';

  interface Props {
    open: boolean;
    listing: Listing;
    /** Artisan name from registration, if known on this device. */
    sellerName?: string;
    /** District from registration, if known on this device. */
    sellerDistrict?: string;
    /** Pehchan / MSME ID from registration, if known on this device. */
    sellerId?: string;
  }

  let {
    open = $bindable(false),
    listing,
    sellerName = '',
    sellerDistrict = '',
    sellerId = '',
  }: Props = $props();

  const t = $derived(locale.t);

  const productName = $derived(titleFor(listing, locale.code) || t('listings.untitled'));

  const description = $derived(
    listing.translations?.find((tr) => tr.language === locale.code)?.description ??
    listing.translations?.[0]?.description ??
    ''
  );

  const priceRupees = $derived(
    listing.price?.amount_paise != null
      ? `₹${(listing.price.amount_paise / 100).toLocaleString('en-IN', { minimumFractionDigits: 2 })}`
      : '—'
  );

  const listingType = $derived(
    listing.type === 'MADE_TO_ORDER' ? t('gem.madeToOrder') : t('gem.readyStock')
  );

  const catalogId = $derived(
    t('gem.catalogIdPlaceholder', { id: (listing.id ?? '').slice(0, 8).toUpperCase() })
  );

  function buildPlainText(): string {
    const lines = [
      '══════════════════════════════════════════',
      '  GeM — Government e-Marketplace',
      '  Product Catalog Entry (Draft)',
      '══════════════════════════════════════════',
      '',
      `${t('gem.catalogId')}:    ${catalogId}`,
      `${t('gem.productName')}:  ${productName}`,
      `${t('gem.hsnCode')}:      ${t('gem.hsnPlaceholder')}`,
      `${t('gem.unitPrice')}:    ${priceRupees}`,
      `${t('gem.listingType')}:  ${listingType}`,
      `${t('gem.category')}:     ${t('gem.categoryPlaceholder')}`,
      '',
      `── ${t('gem.sellerDetails')} ──`,
      `${t('gem.sellerName')}:     ${sellerName || '—'}`,
      `${t('gem.sellerDistrict')}: ${sellerDistrict || '—'}`,
      `${t('gem.sellerId')}:       ${sellerId || t('gem.sellerIdPlaceholder')}`,
      '',
    ];
    if (description) {
      lines.push(`── ${t('gem.description')} ──`, description, '');
    }
    lines.push(
      '──────────────────────────────────────────',
      t('gem.note'),
      '──────────────────────────────────────────',
    );
    return lines.join('\n');
  }

  async function copyAsText(): Promise<void> {
    try {
      await navigator.clipboard.writeText(buildPlainText());
      showToast({ variant: 'success', message: t('gem.copied') });
    } catch {
      showToast({ variant: 'error', message: t('api.error.unknown') });
    }
  }

  function printPreview(): void {
    window.print();
  }
</script>

<Dialog bind:open title={t('gem.dialogTitle')} class="gem-dialog">
  <div class="gem-card">
    <div class="gem-card__header">
      <div class="gem-card__header-emblem">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true">
          <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="1.5" />
          <text x="12" y="16" text-anchor="middle" font-size="11" font-weight="700">G</text>
        </svg>
      </div>
      <div class="gem-card__header-text">
        <p class="gem-card__header-title">Government e-Marketplace</p>
        <p class="gem-card__header-sub">gem.gov.in — Product Catalog Entry</p>
      </div>
    </div>

    <div class="gem-card__catalog-id">
      <span class="gem-card__field-label">{t('gem.catalogId')}</span>
      <span class="gem-card__field-value gem-card__field-value--mono">{catalogId}</span>
    </div>

    <table class="gem-card__table">
      <tbody>
        <tr>
          <th scope="row">{t('gem.productName')}</th>
          <td class="gem-card__product-name">{productName}</td>
        </tr>
        <tr>
          <th scope="row">{t('gem.hsnCode')}</th>
          <td><code>{t('gem.hsnPlaceholder')}</code></td>
        </tr>
        <tr>
          <th scope="row">{t('gem.unitPrice')}</th>
          <td class="gem-card__price">{priceRupees}</td>
        </tr>
        <tr>
          <th scope="row">{t('gem.listingType')}</th>
          <td>{listingType}</td>
        </tr>
        <tr>
          <th scope="row">{t('gem.category')}</th>
          <td>{t('gem.categoryPlaceholder')}</td>
        </tr>
      </tbody>
    </table>

    {#if description}
      <div class="gem-card__section">
        <h3>{t('gem.description')}</h3>
        <p class="gem-card__description">{description}</p>
      </div>
    {/if}

    <div class="gem-card__section">
      <h3>{t('gem.sellerDetails')}</h3>
      <table class="gem-card__table">
        <tbody>
          <tr>
            <th scope="row">{t('gem.sellerName')}</th>
            <td>{sellerName || '—'}</td>
          </tr>
          <tr>
            <th scope="row">{t('gem.sellerDistrict')}</th>
            <td>{sellerDistrict || '—'}</td>
          </tr>
          <tr>
            <th scope="row">{t('gem.sellerId')}</th>
            <td class="gem-card__field-value--placeholder">
              {sellerId || t('gem.sellerIdPlaceholder')}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p class="gem-card__note">
      <Icon name="info" />
      {t('gem.note')}
    </p>
  </div>

  <div class="gem-actions">
    <Button size="sm" variant="secondary" onclick={printPreview}>
      <Icon name="print" />
      {t('gem.printButton')}
    </Button>
    <Button size="sm" variant="secondary" onclick={copyAsText}>
      <Icon name="share" />
      {t('gem.copyButton')}
    </Button>
  </div>
</Dialog>

<style>
  :global(.gem-dialog) {
    max-inline-size: min(36rem, 95vw);
  }

  .gem-card {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    font-size: var(--k-text-sm);
  }

  .gem-card__header {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border-radius: var(--k-radius-md);
    background: linear-gradient(135deg, #1a237e 0%, #283593 100%);
    color: #fff;
  }

  .gem-card__header-emblem {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.5rem;
    block-size: 2.5rem;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
  }

  .gem-card__header-title {
    font-weight: 700;
    font-size: var(--k-text-md);
    margin: 0;
  }

  .gem-card__header-sub {
    font-size: var(--k-text-xs, 0.75rem);
    opacity: 0.8;
    margin: 0;
  }

  .gem-card__catalog-id {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-sm, 0.25rem);
    background: var(--k-surface-sunken);
    font-size: var(--k-text-xs, 0.75rem);
  }

  .gem-card__field-label {
    font-weight: 600;
    color: var(--k-text-secondary);
  }

  .gem-card__field-value--mono {
    font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', monospace;
    letter-spacing: 0.04em;
  }

  .gem-card__table {
    inline-size: 100%;
    border-collapse: collapse;
  }

  .gem-card__table th,
  .gem-card__table td {
    text-align: start;
    padding: var(--k-space-2) var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    vertical-align: top;
  }

  .gem-card__table th {
    inline-size: 40%;
    font-weight: 600;
    color: var(--k-text-secondary);
    white-space: nowrap;
  }

  .gem-card__product-name {
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .gem-card__price {
    font-weight: 700;
    font-variant-numeric: var(--k-numeric-tabular);
    color: #1b5e20;
  }

  .gem-card__table code {
    font-size: var(--k-text-xs, 0.75rem);
    padding: 2px var(--k-space-1);
    border-radius: var(--k-radius-xs, 0.2rem);
    background: var(--k-surface-sunken);
  }

  .gem-card__section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .gem-card__section h3 {
    font-size: var(--k-text-xs, 0.75rem);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #1a237e;
    margin: 0;
    padding-block-start: var(--k-space-2);
    border-block-start: 2px solid #1a237e;
  }

  .gem-card__description {
    color: var(--k-text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .gem-card__field-value--placeholder {
    color: var(--k-text-secondary);
    font-style: italic;
  }

  .gem-card__note {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-sm, 0.25rem);
    background: #fff3e0;
    color: #e65100;
    font-size: var(--k-text-xs, 0.75rem);
    line-height: 1.4;
    margin: 0;
  }

  .gem-actions {
    display: flex;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
    flex-wrap: wrap;
  }

  @media print {
    .gem-actions {
      display: none;
    }
    .gem-card__note {
      border: 1px solid #e65100;
    }
  }
</style>
