<!--
  apps/buyer/src/lib/ProvenanceTerminal.svelte

  Government Provenance Verification Terminal.
  Built to UIDAI & GIGW 3.0 guidelines: high-contrast authority, hairline
  separation, monospace ledger hashing, and zero sci-fi neon gloss.
-->
<script lang="ts">
  import { locale, tooltip, type MessageKey } from '@kalakriti/i18n';
  import { SectionHeader, Tooltip } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  interface RawRecord {
    code: string;
    titleKey: MessageKey;
    artisanKey: MessageKey;
    locationKey: MessageKey;
    techniqueKey: MessageKey;
    shaHash: string;
    publicKey: string;
    timestamp: string;
    fftMatchKey: MessageKey;
  }

  const RECORDS: Record<'vns' | 'ktc' | 'srn', RawRecord> = {
    vns: {
      code: 'UP-VNS-2024-0982',
      titleKey: 'home.provenanceTerminal.record.vns.title',
      artisanKey: 'home.provenanceTerminal.record.vns.artisan',
      locationKey: 'home.provenanceTerminal.record.vns.location',
      techniqueKey: 'home.provenanceTerminal.record.vns.technique',
      shaHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      publicKey: 'ed25519:v1:7f4a8b...92d1',
      timestamp: '2024-08-14 11:22:04 IST',
      fftMatchKey: 'home.provenanceTerminal.record.vns.fftMatch',
    },
    ktc: {
      code: 'GJ-KTC-2024-0417',
      titleKey: 'home.provenanceTerminal.record.ktc.title',
      artisanKey: 'home.provenanceTerminal.record.ktc.artisan',
      locationKey: 'home.provenanceTerminal.record.ktc.location',
      techniqueKey: 'home.provenanceTerminal.record.ktc.technique',
      shaHash: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
      publicKey: 'ed25519:v1:4c81a2...3f90',
      timestamp: '2024-07-29 16:40:12 IST',
      fftMatchKey: 'home.provenanceTerminal.record.ktc.fftMatch',
    },
    srn: {
      code: 'JK-SRN-2024-1105',
      titleKey: 'home.provenanceTerminal.record.srn.title',
      artisanKey: 'home.provenanceTerminal.record.srn.artisan',
      locationKey: 'home.provenanceTerminal.record.srn.location',
      techniqueKey: 'home.provenanceTerminal.record.srn.technique',
      shaHash: '5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8',
      publicKey: 'ed25519:v1:8b13c7...61a4',
      timestamp: '2024-09-02 09:15:30 IST',
      fftMatchKey: 'home.provenanceTerminal.record.srn.fftMatch',
    },
  };

  function toDisplayRecord(raw: RawRecord) {
    return {
      code: raw.code,
      title: t(raw.titleKey),
      artisan: t(raw.artisanKey),
      location: t(raw.locationKey),
      technique: t(raw.techniqueKey),
      shaHash: raw.shaHash,
      publicKey: raw.publicKey,
      timestamp: raw.timestamp,
      fftMatch: t(raw.fftMatchKey),
    };
  }

  let searchCode = $state('UP-VNS-2024-0982');
  let isChecking = $state(false);
  let selectedRecordKey = $state<'vns' | 'ktc' | 'srn' | undefined>('vns');
  let notFoundCode = $state<string | undefined>(undefined);

  const verifiedRecord = $derived(selectedRecordKey ? toDisplayRecord(RECORDS[selectedRecordKey]) : undefined);

  const CODE_TO_KEY: Record<string, 'vns' | 'ktc' | 'srn'> = {
    [RECORDS.vns.code]: 'vns',
    [RECORDS.ktc.code]: 'ktc',
    [RECORDS.srn.code]: 'srn',
  };

  const SAMPLE_CODES = [
    { code: 'UP-VNS-2024-0982', labelKey: 'home.provenanceTerminal.sample.vns.label' as MessageKey },
    { code: 'GJ-KTC-2024-0417', labelKey: 'home.provenanceTerminal.sample.ktc.label' as MessageKey },
    { code: 'JK-SRN-2024-1105', labelKey: 'home.provenanceTerminal.sample.srn.label' as MessageKey },
  ];

  function pickSample(code: string) {
    searchCode = code;
    verifyCode();
  }

  function verifyCode() {
    const trimmed = searchCode.trim();
    if (!trimmed) return;
    isChecking = true;

    setTimeout(() => {
      const matchedKey = CODE_TO_KEY[trimmed.toUpperCase()];
      if (matchedKey) {
        selectedRecordKey = matchedKey;
        notFoundCode = undefined;
      } else {
        selectedRecordKey = undefined;
        notFoundCode = trimmed.toUpperCase();
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
            <span>{isChecking ? t('home.provenanceTerminal.inspecting') : t('home.provenanceTerminal.verifyButton')}</span>
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
                <span>{t(sample.labelKey)}</span>
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
          <span class="cert-registry-number">{t('home.provenanceTerminal.registryIdLabel', { code: verifiedRecord.code })}</span>
        </div>

        <h3 class="cert-piece-name">{verifiedRecord.title}</h3>

        <div class="cert-table">
          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.certLabel.artisanGuild')}</span>
            <span class="cert-val">{verifiedRecord.artisan}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.loomCoordinates')}</span>
            <span class="cert-val">{verifiedRecord.location}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.certLabel.craftedTechnique')}</span>
            <span class="cert-val">{verifiedRecord.technique}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.certLabel.spectralVerification')}</span>
            <span class="cert-val cert-highlight">{verifiedRecord.fftMatch}</span>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.certLabel.shaLedger')}</span>
            <code class="cert-hash">{verifiedRecord.shaHash}</code>
          </div>

          <div class="cert-row">
            <span class="cert-key">{t('home.provenanceTerminal.artisanSignature')}</span>
            <code class="cert-key-val">{verifiedRecord.publicKey} • {t('home.provenanceTerminal.sealedAt', { timestamp: verifiedRecord.timestamp })}</code>
          </div>
        </div>

        <div class="cert-footer">
          <a href={`/verify/${verifiedRecord.code}`} class="public-proof-link">
            <span>{t('home.provenanceTerminal.viewRecord')}</span>
            <Icon name="arrow-right" size="0.9rem" />
          </a>
        </div>
      </div>
    {:else if notFoundCode}
      <div class="certificate-sheet certificate-sheet--invalid" role="alert">
        <div class="cert-header">
          <div class="cert-status cert-status--invalid">
            <Icon name="error" size="1.25rem" />
            <strong class="cert-verdict">{t('buyer.verify.invalid.title')}</strong>
          </div>
        </div>
        <p class="cert-invalid-body">{t('buyer.verify.invalid.body', { code: notFoundCode })}</p>
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

  .cert-status--invalid {
    color: var(--k-accent-danger);
  }

  .certificate-sheet--invalid {
    border-color: var(--k-accent-danger);
  }

  .cert-invalid-body {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0;
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
