<!--
  apps/buyer/src/routes/login/+page.svelte

  Kalakriti Multi-Portal Sign In & Gateway.
  Enables routing and access for:
  1. Buyers / Patrons (access marketplace, orders, and Amazon-style user details)
  2. Master Artisans (direct access to voice-first Artisan PWA)
  3. Ministry Officials & Cluster Admins (direct access to Admin Console)

  Strictly follows GIGW 3.0 government accessibility and Svelte 5 runes ($state, $derived).
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { session, setAccessToken, setRefreshToken } from '@kalakriti/api';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { showToast } from '@kalakriti/ui';

  const t = $derived(locale.t);

  // Active persona tab: 'buyer' | 'artisan' | 'admin'
  let activePortal = $state<'buyer' | 'artisan' | 'admin'>('buyer');
  let authMode = $state<'signin' | 'register'>('signin');

  // Form states
  let identifier = $state(''); // email or mobile
  let password = $state('');
  let fullName = $state('');
  let otpCode = $state('');
  let otpSent = $state(false);
  let isSubmitting = $state(false);
  let rememberMe = $state(true);

  // Check query param for initial tab
  $effect(() => {
    const tabParam = $page.url.searchParams.get('tab');
    if (tabParam === 'register') {
      authMode = 'register';
    }
    const portalParam = $page.url.searchParams.get('portal');
    if (portalParam === 'artisan' || portalParam === 'admin') {
      activePortal = portalParam;
    }
  });

  function handleRequestOtp(e: Event): void {
    e.preventDefault();
    if (!identifier.trim()) {
      showToast({ message: 'Please enter your mobile number or email', variant: 'error' });
      return;
    }
    isSubmitting = true;
    setTimeout(() => {
      isSubmitting = false;
      otpSent = true;
      showToast({ message: 'One-Time Password (OTP) sent to your mobile/email', variant: 'success' });
    }, 600);
  }

  function handleBuyerSubmit(e: Event): void {
    e.preventDefault();
    if (!identifier.trim()) {
      showToast({ message: 'Please enter your mobile number or email', variant: 'error' });
      return;
    }

    isSubmitting = true;
    setTimeout(() => {
      isSubmitting = false;
      // Simulate JWT establishment with mock token for demo
      const mockToken = 'mock.jwt.token.aarav';
      setAccessToken(mockToken);
      setRefreshToken('mock.refresh.token');
      session.establish(mockToken);

      showToast({
        message: authMode === 'signin' ? 'Welcome back to Kalakriti!' : 'Account registered successfully!',
        variant: 'success',
      });
      void goto('/account');
    }, 700);
  }
</script>

<svelte:head>
  <title>Sign In & Institutional Gateway — {t('app.name')}</title>
  <meta name="description" content="Sign in to your Kalakriti Buyer Account or access the Artisan PWA and Ministry Admin consoles." />
</svelte:head>

<div class="auth-wrapper">
  <div class="auth-card">
    <!-- Brand Header -->
    <header class="auth-header">
      <a href="/" class="auth-brand" aria-label="Kalakriti Home">
        <img src="/favicon.svg" alt="" width="36" height="36" class="auth-emblem" />
        <div class="brand-text">
          <span class="brand-title">Kalakriti</span>
          <span class="brand-sub">Ministry of Social Justice & Empowerment</span>
        </div>
      </a>
      <h1 class="auth-heading">Institutional Access Gateway</h1>
      <p class="auth-desc">Choose your portal or sign in to your verified customer account</p>
    </header>

    <!-- Portal Switcher Tabs -->
    <div class="portal-tabs" role="tablist" aria-label="Portal Selection">
      <button
        type="button"
        role="tab"
        aria-selected={activePortal === 'buyer'}
        class="portal-tab {activePortal === 'buyer' ? 'is-active' : ''}"
        onclick={() => (activePortal = 'buyer')}
      >
        <span class="portal-icon"><Icon name="package" size="1.5rem" /></span>
        <span class="portal-label">
          <strong>Buyer Account</strong>
          <small>Orders & Settings</small>
        </span>
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activePortal === 'artisan'}
        class="portal-tab {activePortal === 'artisan' ? 'is-active' : ''}"
        onclick={() => (activePortal = 'artisan')}
      >
        <span class="portal-icon"><Icon name="weaving" size="1.5rem" /></span>
        <span class="portal-label">
          <strong>Artisan Loom</strong>
          <small>Voice PWA</small>
        </span>
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activePortal === 'admin'}
        class="portal-tab {activePortal === 'admin' ? 'is-active' : ''}"
        onclick={() => (activePortal = 'admin')}
      >
        <span class="portal-icon"><Icon name="cluster" size="1.5rem" /></span>
        <span class="portal-label">
          <strong>Ministry Admin</strong>
          <small>Cluster Console</small>
        </span>
      </button>
    </div>

    <!-- Tab 1: Buyer Account Flow -->
    {#if activePortal === 'buyer'}
      <div class="auth-panel" role="tabpanel">
        <div class="auth-mode-toggle">
          <button
            type="button"
            class="mode-btn {authMode === 'signin' ? 'is-selected' : ''}"
            onclick={() => (authMode = 'signin')}
          >
            Sign In
          </button>
          <button
            type="button"
            class="mode-btn {authMode === 'register' ? 'is-selected' : ''}"
            onclick={() => (authMode = 'register')}
          >
            Create Account
          </button>
        </div>

        <form class="auth-form" onsubmit={handleBuyerSubmit}>
          {#if authMode === 'register'}
            <div class="form-field">
              <label for="reg-name" class="field-label">Full Name</label>
              <input
                id="reg-name"
                type="text"
                class="field-input"
                placeholder="e.g. Aarav Sharma"
                bind:value={fullName}
                required
              />
            </div>
          {/if}

          <div class="form-field">
            <label for="buyer-id" class="field-label">
              Mobile Number or Government Email
            </label>
            <div class="input-with-action">
              <input
                id="buyer-id"
                type="text"
                class="field-input"
                placeholder="e.g. +91 98765 43210 or name@gov.in"
                bind:value={identifier}
                required
              />
              <button
                type="button"
                class="otp-send-btn"
                onclick={handleRequestOtp}
                disabled={isSubmitting}
              >
                {otpSent ? 'Resend OTP' : 'Send OTP'}
              </button>
            </div>
            <span class="field-hint">
              Secured with GIGW 3.0 & anti-enumeration protection
            </span>
          </div>

          {#if otpSent}
            <div class="form-field">
              <label for="buyer-otp" class="field-label">Enter 6-Digit OTP</label>
              <input
                id="buyer-otp"
                type="text"
                class="field-input otp-input"
                maxlength="6"
                placeholder="• • • • • •"
                bind:value={otpCode}
                required
              />
            </div>
          {:else}
            <div class="form-field">
              <div class="label-row">
                <label for="buyer-pw" class="field-label">Password</label>
                {#if authMode === 'signin'}
                  <a href="/login?reset=1" class="forgot-link">Forgot password?</a>
                {/if}
              </div>
              <input
                id="buyer-pw"
                type="password"
                class="field-input"
                placeholder="••••••••••••"
                bind:value={password}
                required
              />
            </div>
          {/if}

          <div class="form-actions-row">
            <label class="remember-label">
              <input type="checkbox" bind:checked={rememberMe} />
              <span>Keep me signed in</span>
            </label>
          </div>

          <button
            type="submit"
            class="submit-primary-btn"
            disabled={isSubmitting}
          >
            {#if isSubmitting}
              <Icon name="refresh" size="1.1rem" />
              <span>Verifying credentials...</span>
            {:else if authMode === 'signin'}
              <span>Sign In to Your Account</span>
              <Icon name="arrow-right" size="1rem" />
            {:else}
              <span>Register & Continue to Marketplace</span>
              <Icon name="arrow-right" size="1rem" />
            {/if}
          </button>
        </form>

        <footer class="panel-footer">
          <p class="terms-notice">
            By continuing, you agree to Kalakriti's 
            <a href="/terms">Terms of Service</a> and 
            <a href="/privacy">DPDP Act 2023 Privacy Policy</a>.
          </p>
        </footer>
      </div>

    <!-- Tab 2: Artisan Studio Portal Link -->
    {:else if activePortal === 'artisan'}
      <div class="auth-panel portal-redirect-panel" role="tabpanel">
        <div class="portal-illustration">
          <Icon name="weaving" size="3rem" />
        </div>
        <h2 class="portal-heading">Artisan Loom & Guild Studio</h2>
        <p class="portal-subtext">
          Designed specifically for India's 74 master artisan corridors and craft collectives.
          Features voice-first narration, offline-first sync, and multilingual assistance.
        </p>

        <div class="portal-features-list">
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Voice-first craft cataloging in 12 Indian languages</span>
          </div>
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Ed25519 cryptographic GI provenance stamping</span>
          </div>
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Direct payments into verified Jan Dhan bank accounts</span>
          </div>
        </div>

        <a
          href="http://localhost:5173"
          target="_blank"
          rel="noopener noreferrer"
          class="portal-launch-btn artisan-launch"
        >
          <span>Launch Artisan PWA (Port 5173)</span>
          <Icon name="external-link" size="1.1rem" />
        </a>
      </div>

    <!-- Tab 3: Admin Console Portal Link -->
    {:else if activePortal === 'admin'}
      <div class="auth-panel portal-redirect-panel" role="tabpanel">
        <div class="portal-illustration">
          <Icon name="cluster" size="3rem" />
        </div>
        <h2 class="portal-heading">Ministry & Cluster Development Admin</h2>
        <p class="portal-subtext">
          Restricted to authorized officials from the Ministry of Social Justice & Empowerment,
          Cluster Development Officers, and GI verification registrars.
        </p>

        <div class="portal-features-list">
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Live telemetry across 74 national craft belts</span>
          </div>
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Cryptographic catalog moderation and GI integrity checks</span>
          </div>
          <div class="feat-item">
            <Icon name="check" size="1rem" />
            <span>Institutional escrow settlement and fair-wage audits</span>
          </div>
        </div>

        <a
          href="http://localhost:5175"
          target="_blank"
          rel="noopener noreferrer"
          class="portal-launch-btn admin-launch"
        >
          <span>Launch Ministry Admin Console (Port 5175)</span>
          <Icon name="external-link" size="1.1rem" />
        </a>
      </div>
    {/if}
  </div>
</div>

<style>
  .auth-wrapper {
    min-block-size: 80vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem 1rem;
  }

  .auth-card {
    inline-size: 100%;
    max-inline-size: 32rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline, var(--k-khadi-200));
    border-radius: 16px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.06);
    overflow: hidden;
  }

  .auth-header {
    padding: 1.75rem 2rem 1.25rem;
    text-align: center;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-khadi-150);
  }

  .auth-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    text-decoration: none;
    margin-block-end: 0.75rem;
  }

  .auth-emblem {
    inline-size: 2.25rem;
    block-size: 2.25rem;
  }

  .brand-text {
    display: flex;
    flex-direction: column;
    text-align: start;
    line-height: 1.2;
  }

  .brand-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .brand-sub {
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--k-stone-500);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .auth-heading {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.35rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0.25rem 0;
  }

  .auth-desc {
    font-size: 0.85rem;
    color: var(--k-stone-500);
    margin: 0;
  }

  /* Portal Switcher Tabs */
  .portal-tabs {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    background-color: var(--k-surface-raised);
    padding: 0.35rem;
    gap: 0.35rem;
    border-block-end: 1px solid var(--k-khadi-150);
  }

  .portal-tab {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 0.6rem 0.25rem;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: inherit;
  }

  /* Keep the three portal tabs usable on a 320px screen: tighten padding
     and drop the secondary descriptor so labels never truncate. */
  @media (max-width: 28rem) {
    .portal-tabs {
      gap: 0.2rem;
    }

    .portal-tab {
      padding: 0.5rem 0.1rem;
    }

    .portal-label small {
      display: none;
    }
  }

  .portal-tab.is-active {
    background-color: var(--k-surface-base);
    border-color: var(--k-khadi-200);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.06);
  }

  .portal-icon {
    font-size: 1.2rem;
    margin-block-end: 0.2rem;
  }

  .portal-label {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }

  .portal-label strong {
    font-size: 0.775rem;
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .portal-label small {
    font-size: 0.65rem;
    color: var(--k-stone-500);
  }

  /* Form Panels */
  .auth-panel {
    padding: 1.75rem 2rem;
  }

  .auth-mode-toggle {
    display: flex;
    background-color: var(--k-surface-raised);
    border-radius: 8px;
    padding: 0.25rem;
    margin-block-end: 1.5rem;
  }

  .mode-btn {
    flex: 1;
    padding: 0.45rem;
    background: transparent;
    border: none;
    border-radius: 6px;
    font-weight: 600;
    font-size: 0.85rem;
    color: var(--k-stone-500);
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: inherit;
  }

  .mode-btn.is-selected {
    background-color: var(--k-surface-base);
    color: var(--k-text-primary);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  }

  .auth-form {
    display: flex;
    flex-direction: column;
    gap: 1.15rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .field-label {
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .forgot-link {
    font-size: 0.75rem;
    color: var(--k-madder-600);
    text-decoration: underline;
  }

  .field-input {
    padding: 0.65rem 0.85rem;
    border: 1px solid var(--k-border-hairline);
    border-radius: 8px;
    font-size: 0.9rem;
    color: var(--k-text-primary);
    background-color: var(--k-surface-base);
    transition: border-color 0.15s ease;
  }

  .field-input:focus {
    outline: none;
    border-color: var(--k-madder-600);
    box-shadow: 0 0 0 3px rgba(184, 74, 57, 0.12);
  }

  .input-with-action {
    display: flex;
    gap: 0.4rem;
  }

  .input-with-action .field-input {
    flex: 1;
  }

  .otp-send-btn {
    padding: 0.65rem 0.9rem;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-khadi-200);
    border-radius: 8px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--k-text-secondary);
    cursor: pointer;
    white-space: nowrap;
    transition: background-color 0.15s ease;
  }

  .otp-send-btn:hover {
    background-color: var(--k-surface-pressed);
  }

  .otp-input {
    font-size: 1.25rem;
    letter-spacing: 0.4em;
    text-align: center;
  }

  .field-hint {
    font-size: 0.7rem;
    color: var(--k-stone-400);
  }

  .form-actions-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .remember-label {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8rem;
    color: var(--k-stone-600);
    cursor: pointer;
  }

  .submit-primary-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.8rem 1.25rem;
    background-color: var(--k-madder-600);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: 8px;
    font-size: 0.925rem;
    font-weight: 700;
    cursor: pointer;
    font-family: inherit;
    transition: background-color 0.15s ease;
  }

  .submit-primary-btn:hover {
    background-color: var(--k-madder-600);
  }

  .submit-primary-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .panel-footer {
    margin-block-start: 1.5rem;
    padding-block-start: 1.25rem;
    border-block-start: 1px solid var(--k-khadi-150);
    text-align: center;
  }

  .terms-notice {
    font-size: 0.725rem;
    color: var(--k-stone-400);
    margin: 0;
    line-height: 1.4;
  }

  .terms-notice a {
    color: var(--k-madder-600);
    text-decoration: underline;
  }

  /* Portal Redirect Panels */
  .portal-redirect-panel {
    text-align: center;
  }

  .portal-illustration {
    margin-block-end: 0.75rem;
  }

  .portal-heading {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.2rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0.25rem 0 0.5rem;
  }

  .portal-subtext {
    font-size: 0.825rem;
    color: var(--k-stone-500);
    line-height: 1.45;
    margin-block-end: 1.25rem;
  }

  .portal-features-list {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    text-align: start;
    background-color: var(--k-surface-base);
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid var(--k-khadi-150);
    margin-block-end: 1.5rem;
  }

  .feat-item {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    font-size: 0.8rem;
    color: var(--k-stone-700);
  }

  .feat-item :global(svg) {
    color: var(--k-neem-600);
    flex: none;
  }

  .portal-launch-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.85rem 1.25rem;
    border-radius: 8px;
    font-weight: 700;
    font-size: 0.9rem;
    text-decoration: none;
    transition: opacity 0.15s ease;
  }

  .portal-launch-btn:hover {
    opacity: 0.9;
  }

  .artisan-launch {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
  }

  .admin-launch {
    background-color: var(--k-indigo-900);
    color: var(--k-text-on-accent);
  }
</style>
