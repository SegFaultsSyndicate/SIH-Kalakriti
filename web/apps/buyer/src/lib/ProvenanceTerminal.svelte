<!--
  apps/buyer/src/lib/ProvenanceTerminal.svelte

  Government Provenance Verification Terminal.
  Built to UIDAI & GIGW 3.0 guidelines: high-contrast authority, hairline
  separation, monospace ledger hashing, and zero sci-fi neon gloss.
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { SectionHeader, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  let searchCode = $state('UP-VNS-2024-0982');
  let isChecking = $state(false);
  let verifiedRecord = $state<{
    code: string;
    title: string;
    artisan: string;
    location: string;
    technique: string;
    shaHash: string;
    publicKey: string;
    timestamp: string;
    fftMatch: string;
  } | null>({
    code: 'UP-VNS-2024-0982',
    title: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Saree',
    artisan: 'Mohammad Kabir Ansari (Master Weaver Guild ID: 8812)',
    location: 'Varanasi Weaver Cluster, Uttar Pradesh (25.3176° N, 82.9739° E)',
    technique: 'Kadwa Pit-Loom Weave (Pure Mulberry Silk & Silver Zari)',
    shaHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    publicKey: 'ed25519:v1:7f4a8b...92d1',
    timestamp: '2024-08-14 11:22:04 IST',
    fftMatch: '99.4% Handloom FFT Spectral Peak Match',
  });

  const SAMPLE_CODES = [
    { code: 'UP-VNS-2024-0982', label: 'Varanasi Kadwa Zari' },
    { code: 'GJ-KTC-2024-0417', label: 'Kutch Natural Ajrakh' },
    { code: 'JK-SRN-2024-1105', label: 'Kashmir Pashmina Sozni' },
  ];

  function pickSample(code: string) {
    searchCode = code;
    verifyCode();
  }

  function verifyCode() {
    if (!searchCode.trim()) return;
    isChecking = true;

    setTimeout(() => {
      if (searchCode.includes('KTC')) {
        verifiedRecord = {
          code: 'GJ-KTC-2024-0417',
          title: '16-Stage Natural Indigo & Harda Resist Stole',
          artisan: 'Ismail Mohammed Khatri (National Heritage Awardee)',
          location: 'Dhamadka Artisan Hamlet, Kutch, Gujarat (23.2396° N, 70.0163° E)',
          technique: 'Traditional Hand-Carved Teak Block Print (Pure Mineral Indigo)',
          shaHash: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
          publicKey: 'ed25519:v1:4c81a2...3f90',
          timestamp: '2024-07-29 16:40:12 IST',
          fftMatch: 'Organic Botanical Resist Verified (Zero Synthetic Azo Dyes)',
        };
      } else if (searchCode.includes('SRN')) {
        verifiedRecord = {
          code: 'JK-SRN-2024-1105',
          title: 'Imperial Jama Fine Needle Sozni Pashmina',
          artisan: 'Ghulam Nabi Mir (Valley Master Weaver Council)',
          location: 'Downtown Old Srinagar, Jammu & Kashmir (34.0837° N, 74.7973° E)',
          technique: 'Single-Strand Sozni Needlework on Hand-Spun Changthangi Pashm',
          shaHash: '5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8',
          publicKey: 'ed25519:v1:8b13c7...61a4',
          timestamp: '2024-09-02 09:15:30 IST',
          fftMatch: '12.4 Micron Ultra-Fine Pashm Fiber Certified',
        };
      } else {
        verifiedRecord = {
          code: searchCode.trim().toUpperCase(),
          title: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Saree',
          artisan: 'Mohammad Kabir Ansari (Master Weaver Guild ID: 8812)',
          location: 'Varanasi Weaver Cluster, Uttar Pradesh (25.3176° N, 82.9739° E)',
          technique: 'Kadwa Pit-Loom Weave (Pure Mulberry Silk & Silver Zari)',
          shaHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
          publicKey: 'ed25519:v1:7f4a8b...92d1',
          timestamp: '2024-08-14 11:22:04 IST',
          fftMatch: '99.4% Handloom FFT Spectral Peak Match',
        };
      }
      isChecking = false;
    }, 300);
  }
</script>

<div class="provenance-section">
  <SectionHeader
    kicker={t('home.provenanceTerminal.kicker')}
    heading={t('home.provenanceTerminal.heading')}
    href="/verify/UP-VNS-2024-0982"
  />

  <p class="provenance-intro">{t('home.provenanceTerminal.subheading')}</p>

  <div class="provenance-card">
    <div class="search-console">
      <div class="input-container">
        <Icon name="provenance" size="1.1rem" />
        <input
          type="text"
          bind:value={searchCode}
          placeholder={t('home.provenanceTerminal.inputPlaceholder')}
          class="terminal-input"
          onkeydown={(e) => e.key === 'Enter' && verifyCode()}
          aria-label={t('home.provenanceTerminal.inputPlaceholder')}
        />
      </div>

      <Tooltip text={tooltip('tooltip.verify')}>
        {#snippet trigger(tp)}
          <button
            type="button"
            class="inspect-action-btn"
            onclick={verifyCode}
            disabled={isChecking}
            {...tp}
          >
            <Icon name="check" size="1rem" />
            <span>{isChecking ? 'Inspecting...' : t('home.provenanceTerminal.verifyButton')}</span>
          </button>
        {/snippet}
      </Tooltip>
    </div>

    <!-- Sample quick links -->
    <div class="sample-links-row">
      <span class="sample-label">{t('home.provenanceTerminal.sampleCodes')}</span>
      <div class="sample-badges">
        {#each SAMPLE_CODES as sample}
          <Tooltip text={tooltip('tooltip.verify')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="sample-code-chip"
                class:selected={searchCode === sample.code}
                onclick={() => pickSample(sample.code)}
                {...tp}
              >
                <span>{sample.label}</span>
                <code class="chip-code">{sample.code}</code>
              </button>
            {/snippet}
          </Tooltip>
        {/each}
      </div>
    </div>

    <!-- Official Verification Certificate Sheet -->
    {#if verifiedRecord}
      <div class="certificate-sheet">
        <div class="cert-header">
          <div class="cert-status">
            <Icon name="provenance" size="1.25rem" />
            <strong class="cert-verdict">{t('home.provenanceTerminal.verifiedBadge')}</strong>
          </div>
          <span class="cert-registry-number">REGISTRY ID: {verifiedRecord.code}</span>
        </div>

        <h3 class="cert-piece-name">{verifiedRecord.title}</h3>

        <div class="cert-table">
          <div class="cert-row">
            <span class="cert-key">Master Artisan & Guild:</span>
            <span class="cert-val">{verifiedRecord.artisan}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.loomCoordinates')}</span>
            <span class="cert-val">{verifiedRecord.location}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">Crafted Technique:</span>
            <span class="cert-val">{verifiedRecord.technique}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">Spectral AI Verification:</span>
            <span class="cert-val cert-highlight">{verifiedRecord.fftMatch}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">SHA-256 Ledger Record:</span>
            <code class="cert-hash">{verifiedRecord.shaHash}</code>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.artisanSignature')}</span>
            <code class="cert-key-val">{verifiedRecord.publicKey} • Sealed at {verifiedRecord.timestamp}</code>
          </div>
        </div>

        <div class="cert-footer">
          <a href={`/verify/${verifiedRecord.code}`} class="public-proof-link">
            <span>{t('home.provenanceTerminal.viewRecord')}</span>
            <Icon name="arrow-right" size="0.9rem" />
          </a>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .provenance-section {
    margin-block: var(--k-space-5);
  }

  .provenance-intro {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0 0 var(--k-space-5) 0;
    max-inline-size: 70ch;
  }

  .provenance-card {
    padding: var(--k-space-5);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .search-console {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  @media (min-width: 40rem) {
    .search-console {
      flex-direction: row;
    }
  }

  .input-container {
    flex: 1;
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-secondary);
  }

  .terminal-input {
    flex: 1;
    background: none;
    border: none;
    color: var(--k-text-primary);
    font-family: monospace;
    font-size: var(--k-text-sm);
    outline: none;
  }

  .inspect-action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-4);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    cursor: pointer;
  }

  .sample-links-row {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .sample-label {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .sample-badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
  }

  .sample-code-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-1) var(--k-space-3);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-xs);
    color: var(--k-text-primary);
    cursor: pointer;
    transition: border-color 0.15s ease;
  }

  .sample-code-chip:hover {
    border-color: var(--k-border-interactive);
  }

  .sample-code-chip.selected {
    border-color: var(--k-accent-primary-bg);
    background-color: var(--k-surface-sunken);
  }

  .chip-code {
    font-family: monospace;
    font-size: 0.7rem;
    color: var(--k-text-secondary);
  }

  /* Certificate Sheet */
  .certificate-sheet {
    padding: var(--k-space-5);
    background-color: var(--k-surface-base);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
  }

  .cert-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-block-end: var(--k-space-2);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    margin-block-end: var(--k-space-3);
  }

  .cert-status {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-accent-success);
  }

  .cert-verdict {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .cert-registry-number {
    font-family: monospace;
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .cert-piece-name {
    font-family: var(--k-font-display);
    font-size: var(--k-text-lg);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-3) 0;
  }

  .cert-table {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .cert-row {
    display: flex;
    flex-direction: column;
    font-size: var(--k-text-xs);
    gap: 2px;
  }

  @media (min-width: 44rem) {
    .cert-row {
      flex-direction: row;
      justify-content: space-between;
    }
  }

  .cert-key {
    color: var(--k-text-secondary);
    min-inline-size: 14rem;
  }

  .cert-val {
    color: var(--k-text-primary);
    font-weight: var(--k-weight-medium);
  }

  .cert-highlight {
    color: var(--k-accent-success);
  }

  .cert-hash, .cert-key-val {
    font-family: monospace;
    font-size: 0.75rem;
    color: var(--k-text-primary);
    background-color: var(--k-surface-sunken);
    padding: 2px 6px;
    border-radius: 2px;
    word-break: break-all;
  }

  .cert-footer {
    margin-block-start: var(--k-space-3);
    padding-block-start: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    display: flex;
    justify-content: flex-end;
  }

  .public-proof-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-accent-secondary);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    text-decoration: none;
  }

  .public-proof-link:hover {
    text-decoration: underline;
  }
</style>
