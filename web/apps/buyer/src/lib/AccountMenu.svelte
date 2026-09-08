<!--
  apps/buyer/src/lib/AccountMenu.svelte

  Amazon-style Header Account Popover & Multi-Portal Gateway Trigger.
  Accessible (WCAG 2.1 AA / GIGW 3.0) with full keyboard navigation and escape-to-close.
  Adheres to Svelte 5 runes ($state, $derived, $effect).
-->
<script lang="ts">
  import { session, getAccessToken, setAccessToken, setRefreshToken } from '@kalakriti/api';
  import { Icon } from '@kalakriti/icons';
  import { goto } from '$app/navigation';
  import { showToast } from '@kalakriti/ui';

  let isOpen = $state(false);
  let menuContainer: HTMLDivElement | null = $state(null);

  const isAuthenticated = $derived(session.status === 'authenticated' || !!getAccessToken());
  const userName = $derived(
    (session.claims?.name as string) || 
    (session.claims?.sub as string) || 
    'Aarav Sharma'
  );
  const userEmail = $derived(
    (session.claims?.email as string) || 
    'aarav.sharma@example.gov.in'
  );

  function toggleMenu(): void {
    isOpen = !isOpen;
  }

  function closeMenu(): void {
    isOpen = false;
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape' && isOpen) {
      isOpen = false;
    }
  }

  function handleSignOut(): void {
    setAccessToken(undefined);
    setRefreshToken(undefined);
    session.clear();
    isOpen = false;
    showToast({ message: 'Signed out successfully', variant: 'info' });
    void goto('/');
  }

  // Click outside listener
  $effect(() => {
    if (!isOpen) return;

    function handleClickOutside(event: MouseEvent): void {
      if (menuContainer && !menuContainer.contains(event.target as Node)) {
        isOpen = false;
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="account-menu" bind:this={menuContainer}>
  <button
    type="button"
    class="account-trigger"
    onclick={toggleMenu}
    aria-expanded={isOpen}
    aria-haspopup="true"
    aria-label="Account and multi-portal menu"
  >
    <div class="account-avatar">
      {#if isAuthenticated}
        <span class="avatar-initials">AS</span>
      {:else}
        <Icon name="user" size="1.1rem" />
      {/if}
    </div>
    <div class="account-label">
      <span class="account-greeting">
        {isAuthenticated ? 'Hello, Aarav' : 'Sign In'}
      </span>
      <span class="account-title">
        Account & Lists
        <Icon name="chevron-down" size="0.75rem" />
      </span>
    </div>
  </button>

  {#if isOpen}
    <div
      class="account-popover"
      role="menu"
      tabindex="-1"
    >
      <!-- Popover Header -->
      <div class="popover-header">
        {#if isAuthenticated}
          <div class="user-badge-row">
            <span class="user-name">{userName}</span>
            <span class="patron-badge">Verified Patron</span>
          </div>
          <span class="user-email">{userEmail}</span>
        {:else}
          <a href="/login" class="signin-primary-btn" onclick={closeMenu}>
            Sign In
          </a>
          <p class="signup-prompt">
            New customer? <a href="/login?tab=register" class="signup-link" onclick={closeMenu}>Start here.</a>
          </p>
        {/if}
      </div>

      <div class="popover-grid" role="none">
        <!-- Row 1: Headings -->
        <h3 class="col-heading col-left">Your Account</h3>
        <h3 class="col-heading col-right">Institutional Portals</h3>

        <!-- Row 2: Account vs Artisan Studio -->
        <div class="grid-cell cell-left">
          <a href="/account" class="menu-link" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="user" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Your Account</strong>
              <small>Profile, contact & security</small>
            </span>
          </a>
        </div>
        <div class="grid-cell cell-right">
          <a
            href="http://localhost:5173"
            target="_blank"
            rel="noopener noreferrer"
            class="menu-link portal-link"
            role="menuitem"
            onclick={closeMenu}
          >
            <span class="portal-badge artisan-badge">Artisan</span>
            <span class="menu-link__text">
              <strong>Artisan Studio (PWA)</strong>
              <small>Voice-first loom portal</small>
            </span>
            <span class="portal-link__ext"><Icon name="external-link" size="0.75rem" /></span>
          </a>
        </div>

        <!-- Row 3: Orders vs Ministry Admin -->
        <div class="grid-cell cell-left">
          <a href="/orders" class="menu-link" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="collective-order" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Your Orders</strong>
              <small>Track, cancel & invoices</small>
            </span>
          </a>
        </div>
        <div class="grid-cell cell-right">
          <a
            href="http://localhost:5175"
            target="_blank"
            rel="noopener noreferrer"
            class="menu-link portal-link"
            role="menuitem"
            onclick={closeMenu}
          >
            <span class="portal-badge admin-badge">Ministry</span>
            <span class="menu-link__text">
              <strong>Ministry Admin Console</strong>
              <small>Cluster governance & audits</small>
            </span>
            <span class="portal-link__ext"><Icon name="external-link" size="0.75rem" /></span>
          </a>
        </div>

        <!-- Row 4: Provenance vs Institutional Procurement -->
        <div class="grid-cell cell-left">
          <a href="/verify" class="menu-link" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="provenance" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Craft Provenance</strong>
              <small>Ed25519 digital seals</small>
            </span>
          </a>
        </div>
        <div class="grid-cell cell-right">
          <a href="/bulk-order" class="menu-link" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="cluster" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Institutional Procurement</strong>
              <small>Tenders & bulk guild orders</small>
            </span>
          </a>
        </div>

        <!-- Row 5: Your Addresses vs Contact Us & Support (Aligned) -->
        <div class="grid-cell cell-left">
          <a href="/account#addresses" class="menu-link" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="location" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Your Addresses</strong>
              <small>Shipping & institutional GST</small>
            </span>
          </a>
        </div>
        <div class="grid-cell cell-right">
          <a href="/contact" class="menu-link contact-menu-item" role="menuitem" onclick={closeMenu}>
            <span class="menu-link__icon"><Icon name="phone" size="1rem" /></span>
            <span class="menu-link__text">
              <strong>Contact Us & Support</strong>
              <small>Tollfree helpline & grievance</small>
            </span>
          </a>
        </div>
      </div>

      {#if isAuthenticated}
        <div class="popover-footer">
          <button type="button" class="signout-btn" onclick={handleSignOut}>
            <Icon name="lock" size="0.9rem" />
            Sign Out of Kalakriti
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .account-menu {
    position: relative;
    display: inline-block;
  }

  .account-trigger {
    display: flex;
    align-items: center;
    gap: var(--k-space-2, 0.5rem);
    padding: 0.35rem 0.65rem;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--k-radius-md, 8px);
    color: var(--k-premium-header-text, #F4F0EA);
    cursor: pointer;
    font-family: inherit;
    text-align: start;
    transition: background-color 0.15s ease, border-color 0.15s ease;
  }

  .account-trigger:hover,
  .account-trigger[aria-expanded='true'] {
    background-color: rgba(244, 240, 234, 0.15);
    border-color: rgba(244, 240, 234, 0.2);
  }

  .account-avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2rem;
    block-size: 2rem;
    border-radius: 50%;
    background-color: var(--k-premium-warm-cream, #F4F0EA);
    color: var(--k-premium-header-bg, #873032);
    border: 1px solid rgba(244, 240, 234, 0.3);
    font-size: 0.75rem;
    font-weight: 700;
  }

  .avatar-initials {
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.05em;
  }

  .account-label {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }

  .account-greeting {
    font-size: 0.7rem;
    color: var(--k-premium-header-text, #F4F0EA);
    font-weight: 500;
    opacity: 0.9;
  }

  .account-title {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.825rem;
    font-weight: 700;
    color: var(--k-premium-header-text, #F4F0EA);
  }

  /* Popover */
  .account-popover {
    position: absolute;
    inset-inline-end: 0;
    inset-block-start: calc(100% + 0.5rem);
    inline-size: 36rem;
    max-inline-size: 94vw;
    background-color: #ffffff;
    border: 1px solid var(--k-border-hairline, #e2dcd2);
    border-radius: 12px;
    box-shadow: 0 12px 36px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.04);
    z-index: 1000;
    overflow: hidden;
    animation: popoverFade 0.15s ease-out;
  }

  @keyframes popoverFade {
    from {
      opacity: 0;
      transform: translateY(-6px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .popover-header {
    padding: 1rem 1.25rem;
    background-color: #faf7f2;
    border-block-end: 1px solid #eee8df;
  }

  .user-badge-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .user-name {
    font-size: 0.95rem;
    font-weight: 700;
    color: #1e1915;
  }

  .patron-badge {
    font-size: 0.65rem;
    font-weight: 700;
    padding: 0.15rem 0.5rem;
    border-radius: 999px;
    background-color: #e8f5e9;
    color: #2e7d32;
    border: 1px solid #c8e6c9;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .user-email {
    display: block;
    font-size: 0.75rem;
    color: #756d65;
    margin-block-start: 0.15rem;
  }

  .signin-primary-btn {
    display: block;
    inline-size: 100%;
    text-align: center;
    padding: 0.6rem 1rem;
    background-color: #b84a39;
    color: #ffffff;
    font-weight: 700;
    font-size: 0.85rem;
    border-radius: 8px;
    text-decoration: none;
    transition: background-color 0.15s ease;
  }

  .signin-primary-btn:hover {
    background-color: #993b2d;
  }

  .signup-prompt {
    margin-block-start: 0.5rem;
    font-size: 0.75rem;
    text-align: center;
    color: #756d65;
    margin-block-end: 0;
  }

  .signup-link {
    color: #1d4ed8;
    font-weight: 600;
    text-decoration: underline;
  }

  .popover-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    padding: 0.5rem 0;
  }

  @media (max-width: 600px) {
    .popover-grid {
      grid-template-columns: 1fr;
    }
  }

  .col-heading {
    font-size: 0.725rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #8c8278;
    margin: 0;
    padding: 0.4rem 1rem 0.4rem 1rem;
    display: flex;
    align-items: center;
  }

  .col-left {
    padding-inline-start: 1.25rem;
  }

  .col-right {
    padding-inline-start: 1.25rem;
    border-inline-start: 1px solid #eee8df;
    background-color: #fdfbf7;
  }

  .grid-cell {
    display: flex;
    align-items: stretch;
    padding: 0.15rem 0.65rem;
  }

  .cell-left {
    padding-inline-start: 0.85rem;
    background-color: #ffffff;
  }

  .cell-right {
    padding-inline-start: 0.85rem;
    border-inline-start: 1px solid #eee8df;
    background-color: #fdfbf7;
  }

  .menu-link {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    padding: 0.45rem 0.5rem;
    border-radius: 6px;
    text-decoration: none;
    color: #2b2520;
    inline-size: 100%;
    min-block-size: 2.75rem;
    box-sizing: border-box;
    transition: background-color 0.12s ease;
  }

  .menu-link:hover {
    background-color: #f3efe6;
  }

  .menu-link__icon {
    inline-size: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: center;
    flex: none;
    color: #6a6259;
  }

  .menu-link__text {
    display: flex;
    flex-direction: column;
    line-height: 1.25;
    flex: 1;
    min-inline-size: 0;
  }

  .menu-link__text strong {
    font-size: 0.825rem;
    font-weight: 600;
    color: #1e1915;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .menu-link__text small {
    font-size: 0.7rem;
    color: #7a7269;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .portal-link {
    justify-content: flex-start;
  }

  .portal-link__ext {
    flex: none;
    color: #9c9288;
    display: flex;
    align-items: center;
    margin-inline-start: 0.25rem;
  }

  .portal-badge {
    font-size: 0.58rem;
    font-weight: 800;
    text-transform: uppercase;
    padding: 0.15rem 0.35rem;
    border-radius: 4px;
    letter-spacing: 0.04em;
    flex: none;
    inline-size: 3.4rem;
    text-align: center;
    box-sizing: border-box;
  }

  .artisan-badge {
    background-color: #fff3e0;
    color: #e65100;
    border: 1px solid #ffe0b2;
  }

  .admin-badge {
    background-color: #ede7f6;
    color: #4a148c;
    border: 1px solid #d1c4e9;
  }

  .popover-footer {
    padding: 0.65rem 1.25rem;
    background-color: #faf7f2;
    border-block-start: 1px solid #eee8df;
    display: flex;
    justify-content: flex-end;
  }

  .signout-btn {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    background: transparent;
    border: none;
    color: #b84a39;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0.3rem 0.5rem;
    border-radius: 4px;
    font-family: inherit;
  }

  .signout-btn:hover {
    background-color: #fee2e2;
  }

  /* On a very narrow phone the header action row is already saturated:
     collapse the account trigger to avatar-only rather than letting it
     push the row past the viewport edge. */
  @media (max-width: 30rem) {
    .account-label {
      display: none;
    }

    .account-trigger {
      padding: 0.35rem;
    }
  }
</style>
