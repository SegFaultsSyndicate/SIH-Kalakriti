<!--
  apps/artisan/src/lib/BusinessCardModal.svelte

  Artisan Digital Business Card Modal:
  Provides master artisans with an interactive preview of their standalone
  visiting card, instant WhatsApp sharing, and printing options.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button, showToast } from '@kalakriti/ui';
  import { CardEdge, KolamCorner } from '@kalakriti/ornament';

  interface Props {
    open: boolean;
    artisanName: string;
    craftName: string;
    districtName: string;
    clusterName: string;
    pehchanId: string;
    avatarUrl?: string;
    onclose: () => void;
  }

  let {
    open,
    artisanName,
    craftName,
    districtName,
    clusterName,
    pehchanId,
    avatarUrl,
    onclose,
  }: Props = $props();

  const t = $derived(locale.t);
  let qrDataUrl = $state('');

  const slug = $derived(encodeURIComponent(artisanName.toLowerCase().replace(/\s+/g, '-')));
  const cardUrl = $derived(`http://localhost:5174/card/${slug}`);

  $effect(() => {
    if (!open) return;
    void (async () => {
      try {
        const QRCode = await import('qrcode');
        qrDataUrl = await QRCode.toDataURL(cardUrl, {
          margin: 1,
          width: 320,
          color: {
            dark: '#1e293b',
            light: '#ffffff',
          },
        });
      } catch {
        qrDataUrl = '';
      }
    })();
  });

  function copyLink(): void {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(cardUrl);
      showToast({ message: t('card.linkCopied'), variant: 'success' });
    }
  }

  function shareWhatsApp(): void {
    const text = t('card.shareMessage', { craft: craftName, name: artisanName, district: districtName, url: cardUrl });
    window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank');
  }

  function handlePrint(): void {
    window.print();
  }
</script>

{#if open}
  <div class="modal-backdrop" onclick={onclose} role="presentation">
    <div
      class="modal-dialog"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-label={t('card.title')}
      tabindex="-1"
    >
      <header class="modal-header">
        <div>
          <h2>{t('card.title')}</h2>
          <p class="modal-subhead">{t('card.subtitle')}</p>
        </div>
        <button type="button" class="close-btn" onclick={onclose} aria-label={t('ui.dialog.close')}>
          <Icon name="close" />
        </button>
      </header>

      <!-- Live Card Preview -->
      <div class="card-preview-stage" id="artisan-printable-card">
        <CardEdge class="preview-card">
          <KolamCorner corner="tl" />
          <KolamCorner corner="tr" />
          <KolamCorner corner="bl" />
          <KolamCorner corner="br" />

          <div class="preview-content">
            <div class="preview-header">
              <span class="preview-gov">{t('card.ministryLine')}</span>
              <span class="preview-badge">{t('card.pmVishwakarmaVerified')}</span>
            </div>

            <div class="preview-main">
              <div class="preview-profile">
                {#if avatarUrl}
                  <img src={avatarUrl} alt={artisanName} class="preview-avatar" />
                {:else}
                  <div class="preview-avatar-fallback">
                    {artisanName.charAt(0).toUpperCase()}
                  </div>
                {/if}
                <div>
                  <h3 class="preview-name">{artisanName}</h3>
                  <p class="preview-craft">{craftName}</p>
                  <p class="preview-loc">{districtName} • {clusterName}</p>
                  <span class="preview-id">{t('card.idLabel', { id: pehchanId })}</span>
                </div>
              </div>

              <div class="preview-qr-box">
                {#if qrDataUrl}
                  <img src={qrDataUrl} alt={t('card.qrAlt')} class="preview-qr" />
                {:else}
                  <div class="preview-qr-ph">QR</div>
                {/if}
                <span class="preview-scan-text">{t('card.scanToBuy')}</span>
              </div>
            </div>

            <footer class="preview-footer">
              <span>{t('card.dbtProvenance')}</span>
              <span class="preview-url">{cardUrl}</span>
            </footer>
          </div>
        </CardEdge>
      </div>

      <!-- Action Buttons Strip -->
      <footer class="modal-actions">
        <Button variant="secondary" size="sm" onclick={copyLink}>
          <Icon name="link" size="0.85rem" />
          <span>{t('card.copyLink')}</span>
        </Button>
        <Button variant="secondary" size="sm" onclick={shareWhatsApp}>
          <Icon name="whatsapp" size="0.85rem" />
          <span>{t('card.whatsapp')}</span>
        </Button>
        <Button variant="primary" size="sm" onclick={handlePrint}>
          <Icon name="print" size="0.85rem" />
          <span>{t('card.printCard')}</span>
        </Button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(4px);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--k-space-4);
  }

  .modal-dialog {
    background: var(--k-surface-base);
    border-radius: var(--k-radius-lg);
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.28);
    max-inline-size: 38rem;
    inline-size: 100%;
    max-block-size: 90vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-5);
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    border-block-end: 1px solid var(--k-border-hairline);
    padding-block-end: var(--k-space-3);
  }

  .modal-header h2 {
    font-size: var(--k-text-lg);
    font-weight: 800;
    margin: 0;
  }

  .modal-subhead {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    margin: 2px 0 0;
  }

  .close-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--k-text-secondary);
  }

  /* Card Preview */
  .card-preview-stage {
    background: var(--k-surface-base);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3);
    border: 1px solid var(--k-border-on-inverse);
  }

  :global(.preview-card) {
    position: relative;
    background: var(--k-surface-base);
    padding: 2rem 1.75rem;
    color: var(--k-text-primary);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08);
  }

  :global(.preview-card .k-kolam-corner) {
    color: var(--k-accent-primary-text);
    inline-size: 2.2rem;
    block-size: 2.2rem;
    opacity: 0.75;
  }

  .preview-content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .preview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-block-end: 2px solid var(--k-border-accent);
    padding-block-end: var(--k-space-2);
  }

  .preview-gov {
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--k-accent-primary-text);
  }

  .preview-badge {
    font-size: 0.62rem;
    font-weight: 700;
    background: var(--k-surface-pressed);
    color: var(--k-accent-primary-text);
    padding: 2px 6px;
    border-radius: 4px;
    border: 1px solid var(--k-border-warning);
  }

  .preview-main {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--k-space-3);
  }

  .preview-profile {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
  }

  .preview-avatar {
    inline-size: 3.8rem;
    block-size: 3.8rem;
    border-radius: 50%;
    object-fit: cover;
    border: 2px solid var(--k-border-warning);
  }

  .preview-avatar-fallback {
    inline-size: 3.8rem;
    block-size: 3.8rem;
    border-radius: 50%;
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: 1.5rem;
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .preview-name {
    font-size: 1.15rem;
    font-weight: 800;
    color: var(--k-text-primary);
    margin: 0;
  }

  .preview-craft {
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--k-accent-primary-text);
    margin: 0;
  }

  .preview-loc {
    font-size: 0.68rem;
    color: var(--k-stone-600);
    margin: 2px 0 0;
  }

  .preview-id {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--k-text-tertiary);
  }

  .preview-qr-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .preview-qr {
    inline-size: 5rem;
    block-size: 5rem;
    border: 1px solid var(--k-border-on-inverse);
    border-radius: 4px;
    padding: 2px;
  }

  .preview-qr-ph {
    inline-size: 5rem;
    block-size: 5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    color: var(--k-indigo-400);
  }

  .preview-scan-text {
    font-size: 0.6rem;
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  .preview-footer {
    border-block-start: 1px solid var(--k-border-on-inverse);
    padding-block-start: var(--k-space-2);
    display: flex;
    justify-content: space-between;
    font-size: 0.62rem;
    color: var(--k-text-tertiary);
  }

  .preview-url {
    font-family: monospace;
    color: var(--k-stone-300);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
    border-block-start: 1px solid var(--k-border-hairline);
    padding-block-start: var(--k-space-3);
  }

  :global(.modal-actions .k-button) {
    min-block-size: 2.2rem !important;
    padding: 0.35rem 0.85rem !important;
    font-size: 0.78rem !important;
    font-weight: 600 !important;
    border-radius: var(--k-radius-sm, 6px);
  }

  :global(.modal-actions .k-button .k-button__content) {
    gap: 0.35rem;
    white-space: nowrap;
  }

  @media (max-width: 560px) {
    .modal-actions {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: var(--k-space-1.5);
    }
    :global(.modal-actions .k-button) {
      min-block-size: 2.1rem !important;
      padding: 0.3rem 0.4rem !important;
      font-size: 0.72rem !important;
    }
  }

  @media print {
    :global(body *) {
      visibility: hidden;
    }
    .modal-backdrop {
      position: absolute;
      inset: 0;
      background: none;
      padding: 0;
    }
    .modal-dialog {
      box-shadow: none;
      border: none;
      max-inline-size: 100%;
      padding: 0;
    }
    .modal-header,
    .modal-actions {
      display: none !important;
    }
    #artisan-printable-card,
    #artisan-printable-card * {
      visibility: visible;
    }
    #artisan-printable-card {
      position: fixed;
      inset: 0;
      margin: auto;
      inline-size: 3.5in;
      block-size: 2in;
    }
    :global(.preview-card) {
      border: 2px solid var(--k-border-accent) !important;
      box-shadow: none;
    }
  }
</style>
