<!--
  apps/artisan/src/lib/StallCardModal.svelte

  Exhibition-to-Digital Bridge: Official Exhibition QR Stall Placard
  Enables artisans at physical fairs (Surajkund, Shilp Samagam, Dilli Haat)
  to print and display an authentic framed QR standee so buyers can scan and
  reorder directly from their digital storefront year-round.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
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

  const FAIRS = [
    { value: 'surajkund-2026', label: 'Surajkund International Crafts Mela (Faridabad, Haryana)' },
    { value: 'shilp-samagam-2026', label: 'Shilp Samagam (Major Dhyan Chand Stadium, New Delhi)' },
    { value: 'dilli-haat-ina', label: 'Dilli Haat Master Crafts Fortnight (INA, New Delhi)' },
    { value: 'saras-mela-2026', label: 'Saras Mela National Exhibition (Patna, Bihar)' },
    { value: 'hunar-haat-mumbai', label: 'Hunar Haat Craft Pavilion (BKC, Mumbai)' },
  ];

  let selectedFair = $state('surajkund-2026');
  let stallNumber = $state('B-42');
  let qrDataUrl = $state('');

  const currentFairLabel = $derived(
    FAIRS.find((f) => f.value === selectedFair)?.label ?? 'National Craft Exhibition',
  );

  const storefrontSlug = $derived(encodeURIComponent(artisanName.toLowerCase().replace(/\s+/g, '-')));
  const targetUrl = $derived(
    `http://localhost:5174/artisan/${storefrontSlug}?fair=${selectedFair}&stall=${encodeURIComponent(stallNumber)}`,
  );

  $effect(() => {
    if (!open) return;
    void (async () => {
      try {
        const QRCode = await import('qrcode');
        qrDataUrl = await QRCode.toDataURL(targetUrl, {
          margin: 1,
          width: 420,
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

  function handlePrint(): void {
    window.print();
  }

  function handleShare(): void {
    const text = `Namaste! Visit ${artisanName}'s stall (${stallNumber}) at ${currentFairLabel} or scan to order authentic handmade ${craftName} anytime: ${targetUrl}`;
    window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank');
  }
</script>

{#if open}
  <div class="modal-backdrop" onclick={onclose} role="presentation">
    <div
      class="modal-dialog"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-label={t('exhibition.stallCard.title')}
      tabindex="-1"
    >
      <header class="modal-header">
        <div>
          <h2>{t('exhibition.stallCard.title')}</h2>
          <p class="modal-subhead">{t('exhibition.stallCard.subtitle')}</p>
        </div>
        <button type="button" class="close-btn" onclick={onclose} aria-label="Close">
          <Icon name="close" />
        </button>
      </header>

      <!-- Exhibition Controls -->
      <div class="controls-grid">
        <div class="field-item">
          <label for="fair-select" class="field-label">{t('exhibition.stallCard.selectFair')}</label>
          <select id="fair-select" bind:value={selectedFair} class="field-select">
            {#each FAIRS as fair}
              <option value={fair.value}>{fair.label}</option>
            {/each}
          </select>
        </div>

        <div class="field-item">
          <label for="stall-num" class="field-label">{t('exhibition.stallCard.stallNumber')}</label>
          <input id="stall-num" bind:value={stallNumber} placeholder="e.g. B-42" class="field-input" />
        </div>
      </div>

      <!-- Printable Placard View -->
      <div class="placard-container" id="printable-stall-placard">
        <CardEdge class="placard-card">
          <!-- Kolam Ornaments in 4 Corners -->
          <KolamCorner corner="tl" />
          <KolamCorner corner="tr" />
          <KolamCorner corner="bl" />
          <KolamCorner corner="br" />

          <div class="placard-content">
            <!-- Government Emblem & Header -->
            <div class="placard-gov-header">
              <div class="emblem-strip">
                <span class="emblem-text">GOVERNMENT OF INDIA • MINISTRY OF TEXTILES</span>
                <span class="emblem-sub">DEVELOPMENT COMMISSIONER (HANDICRAFTS & HANDLOOMS)</span>
              </div>
              <div class="fair-banner">
                <span class="fair-badge">OFFICIAL EXHIBITION STALL CARD</span>
                <h3 class="fair-name">{currentFairLabel}</h3>
                <div class="stall-pill">STALL NUMBER: {stallNumber}</div>
              </div>
            </div>

            <div class="placard-body">
              <!-- Artisan Details Left -->
              <div class="placard-artisan-info">
                {#if avatarUrl}
                  <img src={avatarUrl} alt={artisanName} class="placard-avatar" />
                {:else}
                  <div class="placard-avatar-fallback">
                    {artisanName.charAt(0).toUpperCase()}
                  </div>
                {/if}

                <h4 class="artisan-name">{artisanName}</h4>
                <p class="craft-title">{craftName}</p>

                <div class="artisan-tags">
                  <span class="placard-tag">🏛️ {clusterName || 'Varanasi Weavers Cluster'}</span>
                  <span class="placard-tag">📍 {districtName || 'Uttar Pradesh'}</span>
                  <span class="placard-tag placard-tag--gold">🎖️ PM Vishwakarma ID: {pehchanId || 'UP-VNS-2024-0982'}</span>
                  <span class="placard-tag placard-tag--gi">🇮🇳 Certified GI Handicraft</span>
                </div>
              </div>

              <!-- High-Res QR Code Right -->
              <div class="placard-qr-section">
                <div class="qr-box">
                  {#if qrDataUrl}
                    <img src={qrDataUrl} alt="Scan QR Code to order" class="qr-image" />
                  {:else}
                    <div class="qr-placeholder">Generating QR...</div>
                  {/if}
                </div>
                <p class="qr-prompt">
                  <strong>SCAN TO REORDER ANYTIME</strong>
                  <span>Direct Artisan Delivery • Zero Middlemen • Authentic GI</span>
                </p>
              </div>
            </div>

            <!-- Footer Seal Strip -->
            <footer class="placard-footer">
              <div class="footer-seal">
                <Icon name="verified-artisan" />
                <span>Verified Public Digital Infrastructure — Powered by Kalakriti</span>
              </div>
              <span class="footer-url">{targetUrl}</span>
            </footer>
          </div>
        </CardEdge>
      </div>

      <!-- Action Buttons -->
      <footer class="modal-actions">
        <Button variant="secondary" onclick={handleShare}>
          <Icon name="share" />
          {t('exhibition.stallCard.shareWhatsapp')}
        </Button>
        <Button variant="primary" onclick={handlePrint}>
          <Icon name="print" />
          {t('exhibition.stallCard.print')}
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
    max-inline-size: 44rem;
    inline-size: 100%;
    max-block-size: 92vh;
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
    font-weight: var(--k-weight-bold);
    margin: 0;
  }

  .modal-subhead {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    margin: var(--k-space-1) 0 0;
  }

  .close-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--k-text-secondary);
    padding: var(--k-space-1);
    border-radius: var(--k-radius-pill);
  }

  .controls-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: var(--k-space-3);
  }

  .field-item {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .field-label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .field-select,
  .field-input {
    inline-size: 100%;
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  /* Placard Card */
  .placard-container {
    background: #fdfbf7;
    border-radius: var(--k-radius-md);
    padding: var(--k-space-4);
    border: 1px solid #e7e5e4;
  }

  :global(.placard-card) {
    position: relative;
    background: #ffffff;
    color: #1c1917;
    border-radius: 8px;
    padding: var(--k-space-6);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
  }

  :global(.placard-card .k-kolam-corner) {
    color: #b45309;
    inline-size: 2.5rem;
    block-size: 2.5rem;
  }

  .placard-content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    text-align: center;
  }

  .placard-gov-header {
    border-block-end: 2px solid #78350f;
    padding-block-end: var(--k-space-3);
  }

  .emblem-strip {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-block-end: var(--k-space-2);
  }

  .emblem-text {
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    color: #78350f;
  }

  .emblem-sub {
    font-size: 0.65rem;
    font-weight: 600;
    color: #57534e;
    letter-spacing: 0.05em;
  }

  .fair-badge {
    display: inline-block;
    background: #78350f;
    color: #fff;
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    padding: 2px 8px;
    border-radius: 3px;
    margin-block-end: 4px;
  }

  .fair-name {
    font-size: 1.15rem;
    font-weight: 800;
    color: #1c1917;
    margin: 4px 0;
  }

  .stall-pill {
    display: inline-block;
    background: #fef3c7;
    border: 1px solid #d97706;
    color: #92400e;
    font-weight: 800;
    font-size: 0.95rem;
    padding: 3px 12px;
    border-radius: 9999px;
    margin-block-start: 4px;
  }

  .placard-body {
    display: grid;
    grid-template-columns: 1.2fr 1fr;
    gap: var(--k-space-4);
    align-items: center;
    text-align: start;
    padding: var(--k-space-2) 0;
  }

  .placard-artisan-info {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .placard-avatar {
    inline-size: 4.5rem;
    block-size: 4.5rem;
    border-radius: 50%;
    object-fit: cover;
    border: 3px solid #d97706;
  }

  .placard-avatar-fallback {
    inline-size: 4.5rem;
    block-size: 4.5rem;
    border-radius: 50%;
    background: #78350f;
    color: #fff;
    font-size: 1.8rem;
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .artisan-name {
    font-size: 1.3rem;
    font-weight: 800;
    color: #1c1917;
    margin: 0;
  }

  .craft-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: #b45309;
    margin: 0;
  }

  .artisan-tags {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 0.72rem;
    color: #44403c;
  }

  .placard-tag--gold {
    color: #92400e;
    font-weight: 700;
  }

  .placard-tag--gi {
    color: #15803d;
    font-weight: 700;
  }

  .placard-qr-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2);
    background: #f8fafc;
    border-radius: var(--k-radius-md);
    border: 1px dashed #cbd5e1;
  }

  .qr-box {
    inline-size: 10rem;
    block-size: 10rem;
    background: #fff;
    padding: 6px;
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }

  .qr-image {
    inline-size: 100%;
    block-size: 100%;
    object-fit: contain;
  }

  .qr-placeholder {
    inline-size: 100%;
    block-size: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    color: #94a3b8;
  }

  .qr-prompt {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .qr-prompt strong {
    font-size: 0.8rem;
    letter-spacing: 0.05em;
    color: #0f172a;
  }

  .qr-prompt span {
    font-size: 0.68rem;
    color: #64748b;
  }

  .placard-footer {
    border-block-start: 1px solid #e7e5e4;
    padding-block-start: var(--k-space-2);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .footer-seal {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.68rem;
    font-weight: 700;
    color: #78350f;
  }

  .footer-url {
    font-size: 0.6rem;
    color: #a8a29e;
    font-family: monospace;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
    border-block-start: 1px solid var(--k-border-hairline);
    padding-block-start: var(--k-space-3);
  }

  /* Print Specific Styles for Exhibition Standee / A4 */
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
    .controls-grid,
    .modal-actions {
      display: none !important;
    }
    #printable-stall-placard,
    #printable-stall-placard * {
      visibility: visible;
    }
    #printable-stall-placard {
      position: fixed;
      inset: 0;
      margin: auto;
      inline-size: 100%;
      background: #fff;
      border: none;
      padding: 0;
    }
    :global(.placard-card) {
      border: 4px solid #78350f !important;
      box-shadow: none;
    }
  }
</style>
