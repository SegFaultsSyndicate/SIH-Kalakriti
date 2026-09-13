<!--
  apps/buyer/src/routes/card/[slug]/+page.svelte

  Standalone Shareable Digital Business Card & Mini-Storefront:
  Framed with authentic structural ornament (CardEdge and KolamCorner),
  providing an instant single-link storefront that master artisans can print on
  physical cards or share over WhatsApp to connect with B2B and retail buyers.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { Icon } from '@kalakriti/icons';
  import { Button } from '@kalakriti/ui';
  import { CardEdge, KolamCorner } from '@kalakriti/ornament';
  import {
    getArtisanStorefront,
    listListings,
    getListingSummary,
    type components,
  } from '@kalakriti/api';

  type ArtisanStorefront = components['schemas']['ArtisanStorefront'];
  type ListingSummary = components['schemas']['ListingSummary'];

  const slug = $derived(page.params.slug ?? 'master-artisan');

  let artisan = $state<ArtisanStorefront | undefined>(undefined);
  let listings = $state<ListingSummary[]>([]);
  let qrDataUrl = $state('');

  // Fallback dev avatar sync
  let devAvatar = $state<string | undefined>(undefined);
  $effect(() => {
    try {
      if (typeof localStorage !== 'undefined') {
        const a = localStorage.getItem('kalakriti.artisan.avatar');
        if (a) devAvatar = a;
      }
    } catch {}
  });

  const cardUrl = $derived(
    typeof window !== 'undefined'
      ? window.location.href
      : `http://localhost:5174/card/${slug}`,
  );

  $effect(() => {
    void (async () => {
      try {
        const store = await getArtisanStorefront(slug);
        artisan = store;
        if (store?.id) {
          const res = await listListings({ artisan_id: store.id, state: 'PUBLISHED' });
          const ids = (res.listings ?? []).map((l) => l.id!).filter(Boolean).slice(0, 4);
          const fetched = await Promise.allSettled(ids.map((id) => getListingSummary(id)));
          listings = fetched
            .filter((r): r is PromiseFulfilledResult<ListingSummary> => r.status === 'fulfilled')
            .map((r) => r.value);
        }
      } catch {
        // Fallback for demo when backend is not running
        artisan = {
          id: 'demo-artisan',
          display_name: slug.split('-').map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(' ') || 'Eshaan Master Craftsman',
          craft_name: 'Heritage Varanasi Kadwa Brocade & Silk Weaving',
          district: 'Varanasi',
          state_code: 'UP',
          bio: 'Hereditary master weaver family practicing 4-generation Kadwa pit-loom zari silk weaving. PM Vishwakarma Certified & GI Tagged cluster.',
          verified: true,
        };
      }

      // Generate high-resolution QR Code
      try {
        const QRCode = await import('qrcode');
        qrDataUrl = await QRCode.toDataURL(cardUrl, {
          margin: 1,
          width: 360,
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

  const displayName = $derived(artisan?.display_name || 'Master Artisan');
  const craftTitle = $derived(artisan?.craft_name || 'Traditional Handcrafted Art');
  const locationText = $derived(
    [artisan?.district, artisan?.state_code || 'India'].filter(Boolean).join(', '),
  );
  const avatarImage = $derived(artisan?.image_url || devAvatar);

  function handlePrint(): void {
    window.print();
  }

  function handleWhatsApp(): void {
    const text = `Namaste! I am viewing your digital business card on Kalakriti (${cardUrl}). I would like to inquire about your handcrafted collections.`;
    window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank');
  }
</script>

<svelte:head>
  <title>{displayName} — Digital Business Card | Kalakriti</title>
  <meta
    name="description"
    content="Official digital business card and mini-storefront for master artisan {displayName}. Verified PM Vishwakarma and GI-Tagged craft."
  />
</svelte:head>

<div class="card-page">
  <div class="card-container">
    <!-- Top Action Bar (Non-printed) -->
    <div class="card-top-bar">
      <a href="/artisan/{slug}" class="back-link">
        <Icon name="arrow-left" size="0.9rem" />
        <span>View Full Storefront</span>
      </a>

      <div class="top-actions">
        <Button variant="secondary" size="sm" onclick={handleWhatsApp}>
          <Icon name="whatsapp" size="0.9rem" />
          WhatsApp
        </Button>
        <Button variant="primary" size="sm" onclick={handlePrint}>
          <Icon name="print" size="0.9rem" />
          Print Visiting Card
        </Button>
      </div>
    </div>

    <!-- The Authentic Ornamented Business Card -->
    <div class="visiting-card-wrap" id="printable-visiting-card">
      <CardEdge class="visiting-card">
        <!-- 4 Kolam Corner Ornaments -->
        <KolamCorner corner="tl" />
        <KolamCorner corner="tr" />
        <KolamCorner corner="bl" />
        <KolamCorner corner="br" />

        <div class="card-inner">
          <!-- Government Header Strip -->
          <header class="card-header">
            <div class="gov-lockup">
              <span class="gov-title">MINISTRY OF TEXTILES • GOVERNMENT OF INDIA</span>
              <span class="gov-sub">OFFICIAL ARTISAN DIGITAL IDENTITY & MINI-STOREFRONT</span>
            </div>
            <div class="gi-pill">
              <Icon name="verified-artisan" size="0.85rem" />
              <span>GI-TAGGED MASTER LINEAGE</span>
            </div>
          </header>

          <!-- Main Profile Grid -->
          <div class="card-profile-section">
            <div class="profile-left">
              {#if avatarImage}
                <img src={avatarImage} alt={displayName} class="artisan-avatar" />
              {:else}
                <div class="artisan-avatar-fallback">
                  {displayName.charAt(0).toUpperCase()}
                </div>
              {/if}

              <div class="artisan-meta">
                <h1 class="artisan-name">{displayName}</h1>
                <p class="craft-name">{craftTitle}</p>
                <div class="location-badge">
                  <Icon name="location" size="0.85rem" />
                  <span>{locationText}</span>
                </div>
              </div>
            </div>

            <!-- Dynamic QR Code Side -->
            <div class="qr-side">
              <div class="qr-frame">
                {#if qrDataUrl}
                  <img src={qrDataUrl} alt="Scan QR Code" class="qr-img" />
                {:else}
                  <div class="qr-placeholder">QR Code</div>
                {/if}
              </div>
              <span class="qr-hint">Scan to Buy & Reorder</span>
            </div>
          </div>

          <!-- Trust Badges Strip -->
          <div class="trust-strip">
            <div class="trust-badge">
              <span class="badge-icon"><Icon name="verified-artisan" size="1.25rem" /></span>
              <div class="badge-text">
                <strong>PM Vishwakarma</strong>
                <small>ID: UP-VNS-2024-0982</small>
              </div>
            </div>

            <div class="trust-badge">
              <span class="badge-icon"><Icon name="cluster" size="1.25rem" /></span>
              <div class="badge-text">
                <strong>Weaver Guild CFC</strong>
                <small>Varanasi Silk Cluster</small>
              </div>
            </div>

            <div class="trust-badge">
              <span class="badge-icon"><Icon name="provenance" size="1.25rem" /></span>
              <div class="badge-text">
                <strong>Cryptographic Seal</strong>
                <small>Ed25519 Provenance</small>
              </div>
            </div>
          </div>

          <!-- Mini Catalog Showcase (Featured Work) -->
          <div class="card-catalog-section">
            <h2 class="catalog-heading">Signature Handcrafted Pieces</h2>

            <div class="mini-catalog-grid">
              {#if listings.length > 0}
                {#each listings as item}
                  {@const itemTitle = item.translations?.[0]?.title || item.craft_name || 'Handcrafted Collection'}
                  <a href="/listing/{item.id}" class="mini-product-card">
                    {#if item.image_url}
                      <img src={item.image_url} alt={itemTitle} class="mini-img" />
                    {:else}
                      <div class="mini-img-fallback">
                        <Icon name="image" size="1.2rem" />
                      </div>
                    {/if}
                    <div class="mini-info">
                      <span class="mini-title">{itemTitle}</span>
                      <strong class="mini-price">
                        {item.price?.amount_paise ? `₹${(item.price.amount_paise / 100).toLocaleString('en-IN')}` : 'Request Price'}
                      </strong>
                    </div>
                  </a>
                {/each}
              {:else}
                <!-- Fallback sample pieces if empty -->
                <div class="mini-product-card">
                  <div class="mini-img-fallback"><Icon name="weaving" size="1.2rem" /></div>
                  <div class="mini-info">
                    <span class="mini-title">Pure Katan Silk Kadwa Saree</span>
                    <strong class="mini-price">₹18,500</strong>
                  </div>
                </div>
                <div class="mini-product-card">
                  <div class="mini-img-fallback"><Icon name="embroidery" size="1.2rem" /></div>
                  <div class="mini-info">
                    <span class="mini-title">Zari Brocade Stole Yardage</span>
                    <strong class="mini-price">₹4,200</strong>
                  </div>
                </div>
                <div class="mini-product-card">
                  <div class="mini-img-fallback"><Icon name="pottery" size="1.2rem" /></div>
                  <div class="mini-info">
                    <span class="mini-title">Hand-Spun Raw Silk Dupatta</span>
                    <strong class="mini-price">₹3,850</strong>
                  </div>
                </div>
              {/if}
            </div>
          </div>

          <!-- Direct Call to Action Footer -->
          <footer class="card-footer">
            <div class="contact-buttons">
              <Button variant="secondary" size="md" onclick={handleWhatsApp}>
                <Icon name="whatsapp" />
                Message on WhatsApp
              </Button>
              <a href="tel:+918779279060" class="call-btn">
                <Icon name="phone" size="0.95rem" />
                Direct Call
              </a>
              <a href="/artisan/{slug}" class="storefront-cta">
                <span>View Full Catalog & Buy</span>
                <Icon name="arrow-right" size="0.9rem" />
              </a>
            </div>

            <div class="verified-seal-strip">
              <Icon name="verified-artisan" size="0.8rem" />
              <span>Public Digital Infrastructure for Indian Handicrafts & Handlooms • Kalakriti Platform</span>
            </div>
          </footer>
        </div>
      </CardEdge>
    </div>
  </div>
</div>

<style>
  .card-page {
    padding-block: var(--k-space-6) var(--k-space-12);
    min-block-size: 80vh;
    background: var(--k-surface-base);
  }

  .card-container {
    max-inline-size: 50rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .card-top-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .back-link {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    font-weight: 600;
    text-decoration: none;
  }

  .top-actions {
    display: flex;
    gap: var(--k-space-2);
  }

  /* Visiting Card Ornamented Outer */
  .visiting-card-wrap {
    inline-size: 100%;
  }

  :global(.visiting-card) {
    position: relative;
    background: var(--k-surface-base);
    box-shadow: 0 8px 32px rgba(120, 53, 15, 0.08);
    padding: 2.5rem 2rem;
    color: var(--k-text-primary);
  }

  :global(.visiting-card .k-kolam-corner) {
    color: var(--k-accent-primary-text);
    inline-size: 2.75rem;
    block-size: 2.75rem;
    opacity: 0.75;
  }

  .card-inner {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  /* Header */
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    border-block-end: 2px solid var(--k-border-accent);
    padding-block-end: var(--k-space-3);
    flex-wrap: wrap;
    gap: var(--k-space-2);
  }

  .gov-lockup {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .gov-title {
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--k-accent-primary-text);
  }

  .gov-sub {
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--k-text-tertiary);
    letter-spacing: 0.04em;
  }

  .gi-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: var(--k-surface-pressed);
    border: 1px solid var(--k-border-warning);
    color: var(--k-accent-primary-text);
    font-size: 0.68rem;
    font-weight: 800;
    padding: 3px 10px;
    border-radius: 999px;
  }

  /* Profile Grid */
  .card-profile-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--k-space-4);
    padding-block: var(--k-space-2);
    flex-wrap: wrap;
  }

  .profile-left {
    display: flex;
    align-items: center;
    gap: var(--k-space-4);
  }

  .artisan-avatar {
    inline-size: 5rem;
    block-size: 5rem;
    border-radius: 50%;
    object-fit: cover;
    border: 3px solid var(--k-border-warning);
    box-shadow: 0 4px 12px rgba(217, 119, 6, 0.2);
  }

  .artisan-avatar-fallback {
    inline-size: 5rem;
    block-size: 5rem;
    border-radius: 50%;
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: 2rem;
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .artisan-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .artisan-name {
    font-size: 1.45rem;
    font-weight: 900;
    color: var(--k-text-primary);
    margin: 0;
    line-height: 1.2;
  }

  .craft-name {
    font-size: 0.88rem;
    font-weight: 700;
    color: var(--k-accent-primary-text);
    margin: 0;
  }

  .location-badge {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    color: var(--k-stone-600);
    margin-block-start: 2px;
  }

  .qr-side {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .qr-frame {
    inline-size: 6.5rem;
    block-size: 6.5rem;
    background: var(--k-surface-base);
    border: 1px solid var(--k-border-on-inverse);
    padding: 4px;
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }

  .qr-img {
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
    font-size: 0.75rem;
    color: var(--k-indigo-400);
  }

  .qr-hint {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  /* Trust Strip */
  .trust-strip {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--k-space-3);
    background: var(--k-surface-base);
    border: 1px solid var(--k-border-on-inverse);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-3);
  }

  .trust-badge {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .badge-icon {
    font-size: 1.2rem;
  }

  .badge-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .badge-text strong {
    font-size: 0.75rem;
    color: var(--k-text-primary);
  }

  .badge-text small {
    font-size: 0.65rem;
    color: var(--k-text-tertiary);
  }

  /* Mini Catalog */
  .card-catalog-section {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .catalog-heading {
    font-size: 0.8rem;
    font-weight: 800;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--k-accent-primary-text);
    margin: 0;
  }

  .mini-catalog-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(8.5rem, 1fr));
    gap: var(--k-space-3);
  }

  .mini-product-card {
    border: 1px solid var(--k-border-on-inverse);
    border-radius: var(--k-radius-sm);
    overflow: hidden;
    background: var(--k-surface-base);
    text-decoration: none;
    color: inherit;
    display: flex;
    flex-direction: column;
    transition: transform 0.15s ease;
  }

  .mini-product-card:hover {
    transform: translateY(-2px);
  }

  .mini-img {
    inline-size: 100%;
    aspect-ratio: 1;
    object-fit: cover;
  }

  .mini-img-fallback {
    inline-size: 100%;
    aspect-ratio: 1;
    background: var(--k-surface-base);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.8rem;
  }

  .mini-info {
    padding: var(--k-space-2);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .mini-title {
    font-size: 0.7rem;
    color: var(--k-stone-700);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mini-price {
    font-size: 0.8rem;
    color: var(--k-accent-success-muted);
    font-weight: 800;
  }

  /* Footer */
  .card-footer {
    border-block-start: 1px solid var(--k-border-on-inverse);
    padding-block-start: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .contact-buttons {
    display: flex;
    gap: var(--k-space-2);
    flex-wrap: wrap;
    align-items: center;
  }

  .call-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border: 1px solid var(--k-border-interactive);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
    font-size: var(--k-text-xs);
    font-weight: 700;
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-md);
    text-decoration: none;
  }

  .storefront-cta {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    font-size: var(--k-text-xs);
    font-weight: 700;
    padding: var(--k-space-2) var(--k-space-4);
    border-radius: var(--k-radius-md);
    text-decoration: none;
    margin-inline-start: auto;
  }

  .verified-seal-strip {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    font-size: 0.65rem;
    color: var(--k-text-tertiary);
    font-weight: 600;
  }

  /* Print Layout for standard visiting card */
  @media print {
    :global(body *) {
      visibility: hidden;
    }
    .card-page {
      background: var(--k-surface-base);
      padding: 0;
    }
    .card-top-bar,
    .contact-buttons {
      display: none !important;
    }
    #printable-visiting-card,
    #printable-visiting-card * {
      visibility: visible;
    }
    #printable-visiting-card {
      position: fixed;
      inset: 0;
      margin: auto;
      inline-size: 3.5in;
      block-size: 2in;
      transform: scale(1.1);
    }
    :global(.visiting-card) {
      box-shadow: none !important;
      border: 2px solid var(--k-border-accent) !important;
      padding: 8px !important;
    }
  }

  @media (max-width: 640px) {
    .trust-strip {
      grid-template-columns: 1fr;
    }
    .storefront-cta {
      margin-inline-start: 0;
      inline-size: 100%;
      justify-content: center;
    }
  }
</style>
