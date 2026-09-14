<!--
  apps/buyer/src/routes/account/+page.svelte

  Amazon-Style Buyer "Your Account" & Personal Details Hub for Kalakriti.
  Features:
  - User profile header with Patron tier badges & telemetry
  - Amazon-style 6-tile navigation hub grid
  - Interactive tabs:
    1. Personal Details (Name, Email, Mobile with OTP change modal)
    2. Saved Addresses (Home, Office with GSTIN, Add Address modal)
    3. Login & Security (Password, 2FA, Active Sessions with "Log out other devices", DPDP controls)
    4. Recent Orders Quick Rail
    5. Loom Consultations (1-on-1 video calls with master artisans)

  Strictly adheres to Svelte 5 runes ($state, $derived) and GIGW 3.0 accessibility.
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Breadcrumbs, type BreadcrumbItem, showToast, Tooltip } from '@kalakriti/ui';
  import { session, setAccessToken, setRefreshToken } from '@kalakriti/api';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';

  const t = $derived(locale.t);

  const breadcrumbs = $derived<BreadcrumbItem[]>([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: t('account.breadcrumb') },
  ]);

  // Active section tab: 'personal' | 'addresses' | 'security' | 'orders' | 'consultations'
  let activeTab = $state<'personal' | 'addresses' | 'security' | 'orders' | 'consultations'>('personal');

  // User details state
  let fullName = $state('Aarav Sharma');
  let displayName = $state('Aarav');
  let email = $state('aarav.sharma@example.gov.in');
  let phone = $state('+91 98765 43210');
  let preferredLanguage = $state('en');
  let emailAlerts = $state(true);
  let whatsappAlerts = $state(true);

  // Phone change OTP modal state
  let isPhoneModalOpen = $state(false);
  let newPhone = $state('');
  let phoneOtp = $state('');
  let phoneOtpSent = $state(false);
  let isVerifyingPhone = $state(false);

  // Address modal state
  let isAddAddressOpen = $state(false);
  let newAddressFullName = $state('');
  let newAddressLine = $state('');
  let newAddressCity = $state('');
  let newAddressState = $state('Delhi');
  let newAddressPin = $state('');
  let newAddressGstin = $state('');
  let newAddressIsDefault = $state(false);

  let savedAddresses = $state([
    {
      id: 'addr-1',
      type: 'Home',
      isDefault: true,
      name: 'Aarav Sharma',
      lines: 'B-42 Defence Colony, Near Flyover',
      city: 'New Delhi',
      state: 'Delhi',
      pin: '110024',
      phone: '+91 98765 43210',
      gstin: '',
    },
    {
      id: 'addr-2',
      type: 'Institutional / Office',
      isDefault: false,
      name: 'Aarav Sharma (MSJE)',
      lines: 'Shastri Bhawan, Dr. Rajendra Prasad Road',
      city: 'New Delhi',
      state: 'Delhi',
      pin: '110001',
      phone: '+91 98765 43210',
      gstin: '07AAACM1234F1Z5',
    },
  ]);

  // Security states
  let twoFactorEnabled = $state(true);
  let activeSessions = $state([
    {
      id: 'sess-1',
      device: 'Chrome 124 on Windows 11',
      location: 'New Delhi, India',
      ip: '103.21.***.***',
      isCurrent: true,
      lastActive: 'Active now',
    },
    {
      id: 'sess-2',
      device: 'Kalakriti PWA on Pixel 8 Pro',
      location: 'Varanasi, Uttar Pradesh',
      ip: '103.45.***.***',
      isCurrent: false,
      lastActive: '2 hours ago',
    },
  ]);

  // Handle hash navigation
  $effect(() => {
    const hash = $page.url.hash.replace('#', '');
    if (hash === 'addresses' || hash === 'security' || hash === 'orders' || hash === 'consultations') {
      activeTab = hash;
    }
  });

  function handleSaveProfile(e: Event): void {
    e.preventDefault();
    showToast({ message: t('account.page.toast.profileSaved'), variant: 'success' });
  }

  function handleRequestPhoneOtp(): void {
    if (!newPhone.trim()) {
      showToast({ message: t('account.page.toast.invalidPhone'), variant: 'error' });
      return;
    }
    phoneOtpSent = true;
    showToast({ message: t('account.page.toast.otpSent', { phone: newPhone }), variant: 'info' });
  }

  function handleVerifyPhone(): void {
    if (!phoneOtp.trim()) {
      showToast({ message: t('account.page.toast.invalidOtp'), variant: 'error' });
      return;
    }
    isVerifyingPhone = true;
    setTimeout(() => {
      isVerifyingPhone = false;
      phone = newPhone.startsWith('+91') ? newPhone : `+91 ${newPhone}`;
      isPhoneModalOpen = false;
      phoneOtpSent = false;
      newPhone = '';
      phoneOtp = '';
      showToast({ message: t('account.page.toast.phoneUpdated'), variant: 'success' });
    }, 600);
  }

  function handleAddAddress(e: Event): void {
    e.preventDefault();
    if (!newAddressLine.trim() || !newAddressCity.trim() || !newAddressPin.trim()) {
      showToast({ message: t('account.page.toast.addressFieldsRequired'), variant: 'error' });
      return;
    }

    const newAddr = {
      id: `addr-${Date.now()}`,
      type: newAddressGstin ? t('account.page.addresses.typeInstitutional') : t('account.page.addresses.typeHome'),
      isDefault: newAddressIsDefault,
      name: newAddressFullName || fullName,
      lines: newAddressLine,
      city: newAddressCity,
      state: newAddressState,
      pin: newAddressPin,
      phone: phone,
      gstin: newAddressGstin,
    };

    if (newAddressIsDefault) {
      savedAddresses = savedAddresses.map((a) => ({ ...a, isDefault: false }));
    }

    savedAddresses = [...savedAddresses, newAddr];
    isAddAddressOpen = false;
    newAddressFullName = '';
    newAddressLine = '';
    newAddressCity = '';
    newAddressPin = '';
    newAddressGstin = '';
    newAddressIsDefault = false;
    showToast({ message: t('account.page.toast.addressSaved'), variant: 'success' });
  }

  function handleSetDefaultAddress(id: string): void {
    savedAddresses = savedAddresses.map((a) => ({
      ...a,
      isDefault: a.id === id,
    }));
    showToast({ message: t('account.page.toast.defaultAddressUpdated'), variant: 'success' });
  }

  function handleDeleteAddress(id: string): void {
    savedAddresses = savedAddresses.filter((a) => a.id !== id);
    showToast({ message: t('account.page.toast.addressRemoved'), variant: 'info' });
  }

  function handleRevokeOtherSessions(): void {
    activeSessions = activeSessions.filter((s) => s.isCurrent);
    showToast({ message: t('account.page.toast.sessionsRevoked'), variant: 'success' });
  }

  function handleSignOut(): void {
    setAccessToken(undefined);
    setRefreshToken(undefined);
    session.clear();
    showToast({ message: t('account.page.toast.signedOut'), variant: 'info' });
    void goto('/');
  }
</script>

<svelte:head>
  <title>{t('account.page.title')}</title>
  <meta name="description" content={t('account.page.metaDescription')} />
</svelte:head>

<div class="account-page">
  <div class="container">
    <Breadcrumbs items={breadcrumbs} />

    <!-- Amazon-Style User Profile Header Banner -->
    <section class="profile-banner">
      <div class="profile-avatar-wrap">
        <div class="profile-avatar">AS</div>
        <Tooltip text={tooltip('tooltip.addPhoto')}>
          {#snippet trigger(tp)}
            <button type="button" class="avatar-edit-btn" aria-label={t('account.changePhotoLabel')} {...tp}>
              <Icon name="camera" size="0.85rem" />
            </button>
          {/snippet}
        </Tooltip>
      </div>

      <div class="profile-meta">
        <div class="name-badge-row">
          <h1 class="profile-name">{fullName}</h1>
          <span class="patron-tier-pill">{t('account.patronTier')}</span>
          <span class="member-since-pill">{t('account.memberSince')}</span>
        </div>

        <div class="profile-contact-strip">
          <span class="contact-item">
            <Icon name="user" size="0.9rem" />
            {email}
          </span>
          <span class="contact-divider">•</span>
          <span class="contact-item">
            <Icon name="phone" size="0.9rem" />
            {phone}
            <Tooltip text={t('account.phoneVerifiedTitle')}>
              {#snippet trigger(tp)}
                <span class="verified-dot" {...tp}>{t('account.phoneVerifiedBadge')}</span>
              {/snippet}
            </Tooltip>
          </span>
        </div>
      </div>

      <div class="profile-telemetry">
        <div class="telemetry-card">
          <span class="telemetry-value">4</span>
          <span class="telemetry-label">{t('account.telemetry.orders')}</span>
        </div>
        <div class="telemetry-card">
          <span class="telemetry-value">6</span>
          <span class="telemetry-label">{t('account.telemetry.giPassports')}</span>
        </div>
        <div class="telemetry-card">
          <span class="telemetry-value">1</span>
          <span class="telemetry-label">{t('account.telemetry.loomCommission')}</span>
        </div>
      </div>

      <div class="profile-actions-col">
        <Tooltip text={tooltip('tooltip.signout')}>
        {#snippet trigger(tp)}
          <button type="button" class="banner-signout-btn" onclick={handleSignOut} {...tp}>
            <Icon name="lock" size="0.85rem" />
            <span>{t('account.signOut')}</span>
          </button>
        {/snippet}
      </Tooltip>
      </div>
    </section>

    <!-- Amazon's Signature 6-Card Navigation Hub -->
    <section class="amazon-hub-section" aria-label={t('account.shortcutsAriaLabel')}>
      <div class="hub-grid">
        <!-- Card 1: Your Orders -->
        <a href="/orders" class="hub-card">
          <div class="card-icon-box">
            <Icon name="collective-order" size="1.75rem" />
          </div>
          <div class="card-content">
            <h2 class="card-title">{t('account.hub.orders.title')}</h2>
            <p class="card-desc">{t('account.hub.orders.desc')}</p>
          </div>
        </a>

        <!-- Card 2: Login & Security -->
        <Tooltip text={tooltip('tooltip.selectTab')}>
        {#snippet trigger(tp)}
          <button
            type="button"
            class="hub-card {activeTab === 'security' ? 'card-active' : ''}"
            onclick={() => (activeTab = 'security')}
            {...tp}
          >
            <div class="card-icon-box">
              <Icon name="lock" size="1.75rem" />
            </div>
            <div class="card-content">
              <h2 class="card-title">{t('account.hub.security.title')}</h2>
              <p class="card-desc">{t('account.hub.security.desc')}</p>
            </div>
          </button>
        {/snippet}
      </Tooltip>

        <!-- Card 3: Your Addresses -->
        <Tooltip text={tooltip('tooltip.selectTab')}>
        {#snippet trigger(tp)}
          <button
            type="button"
            class="hub-card {activeTab === 'addresses' ? 'card-active' : ''}"
            onclick={() => (activeTab = 'addresses')}
            {...tp}
          >
            <div class="card-icon-box">
              <Icon name="location" size="1.75rem" />
            </div>
            <div class="card-content">
              <h2 class="card-title">{t('account.hub.addresses.title')}</h2>
              <p class="card-desc">{t('account.hub.addresses.desc')}</p>
            </div>
          </button>
        {/snippet}
      </Tooltip>

        <!-- Card 4: Craft Provenance Passports -->
        <a href="/verify" class="hub-card">
          <div class="card-icon-box">
            <Icon name="provenance" size="1.75rem" />
          </div>
          <div class="card-content">
            <h2 class="card-title">{t('account.hub.provenance.title')}</h2>
            <p class="card-desc">{t('account.hub.provenance.desc')}</p>
          </div>
        </a>

        <!-- Card 5: Loom Consultations -->
        <Tooltip text={tooltip('tooltip.selectTab')}>
        {#snippet trigger(tp)}
          <button
            type="button"
            class="hub-card {activeTab === 'consultations' ? 'card-active' : ''}"
            onclick={() => (activeTab = 'consultations')}
            {...tp}
          >
            <div class="card-icon-box">
              <Icon name="video" size="1.75rem" />
            </div>
            <div class="card-content">
              <h2 class="card-title">{t('account.hub.consultations.title')}</h2>
              <p class="card-desc">{t('account.hub.consultations.desc')}</p>
            </div>
          </button>
        {/snippet}
      </Tooltip>

        <!-- Card 6: Bulk Procurement -->
        <a href="/bulk-order" class="hub-card">
          <div class="card-icon-box">
            <Icon name="cluster" size="1.75rem" />
          </div>
          <div class="card-content">
            <h2 class="card-title">{t('account.hub.procurement.title')}</h2>
            <p class="card-desc">{t('account.hub.procurement.desc')}</p>
          </div>
        </a>
      </div>
    </section>

    <!-- Sub-Panels Tabbed Details Section -->
    <div class="account-details-container">
      <!-- Section Navigation Pills -->
      <nav class="details-nav-pills" aria-label={t('account.page.tabsAriaLabel')}>
        <Tooltip text={tooltip('tooltip.selectTab')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="pill-btn {activeTab === 'personal' ? 'is-active' : ''}"
              onclick={() => (activeTab = 'personal')}
              {...tp}
            >
              <Icon name="user" size="0.95rem" />
              {t('account.page.tab.personal')}
            </button>
          {/snippet}
        </Tooltip>
        <Tooltip text={tooltip('tooltip.selectTab')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="pill-btn {activeTab === 'addresses' ? 'is-active' : ''}"
              onclick={() => (activeTab = 'addresses')}
              {...tp}
            >
              <Icon name="location" size="0.95rem" />
              {t('account.page.tab.addresses', { count: String(savedAddresses.length) })}
            </button>
          {/snippet}
        </Tooltip>
        <Tooltip text={tooltip('tooltip.selectTab')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="pill-btn {activeTab === 'security' ? 'is-active' : ''}"
              onclick={() => (activeTab = 'security')}
              {...tp}
            >
              <Icon name="lock" size="0.95rem" />
              {t('account.page.tab.security')}
            </button>
          {/snippet}
        </Tooltip>
        <Tooltip text={tooltip('tooltip.selectTab')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="pill-btn {activeTab === 'orders' ? 'is-active' : ''}"
              onclick={() => (activeTab = 'orders')}
              {...tp}
            >
              <Icon name="collective-order" size="0.95rem" />
              {t('account.page.tab.orders', { count: '2' })}
            </button>
          {/snippet}
        </Tooltip>
        <Tooltip text={tooltip('tooltip.selectTab')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="pill-btn {activeTab === 'consultations' ? 'is-active' : ''}"
              onclick={() => (activeTab = 'consultations')}
              {...tp}
            >
              <Icon name="video" size="0.95rem" />
              {t('account.page.tab.consultations', { count: '1' })}
            </button>
          {/snippet}
        </Tooltip>
      </nav>

      <!-- Panel 1: Personal Details & Contact -->
      {#if activeTab === 'personal'}
        <div class="detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{t('account.page.personal.title')}</h2>
              <p class="panel-desc">{t('account.page.personal.desc')}</p>
            </div>
          </div>

          <form class="details-form" onsubmit={handleSaveProfile}>
            <div class="form-grid-2">
              <div class="form-field">
                <label for="prof-fullname" class="form-label">{t('account.page.personal.fullNameLabel')}</label>
                <input
                  id="prof-fullname"
                  type="text"
                  class="form-input"
                  bind:value={fullName}
                  required
                />
              </div>

              <div class="form-field">
                <label for="prof-dispname" class="form-label">{t('account.page.personal.displayNameLabel')}</label>
                <input
                  id="prof-dispname"
                  type="text"
                  class="form-input"
                  bind:value={displayName}
                />
              </div>
            </div>

            <div class="form-grid-2">
              <div class="form-field">
                <label for="prof-email" class="form-label">{t('account.page.personal.emailLabel')}</label>
                <input
                  id="prof-email"
                  type="email"
                  class="form-input"
                  bind:value={email}
                  required
                />
                <span class="field-tip">{t('account.page.personal.emailTip')}</span>
              </div>

              <div class="form-field">
                <label for="prof-phone" class="form-label">{t('account.page.personal.phoneLabel')}</label>
                <div class="input-with-action">
                  <input
                    id="prof-phone"
                    type="text"
                    class="form-input"
                    value={phone}
                    readonly
                  />
                  <Tooltip text={tooltip('tooltip.changePhone')}>
                    {#snippet trigger(tp)}
                      <button
                        type="button"
                        class="action-inline-btn"
                        onclick={() => (isPhoneModalOpen = true)}
                        {...tp}
                      >
                        {t('account.page.personal.changeViaOtp')}
                      </button>
                    {/snippet}
                  </Tooltip>
                </div>
                <span class="field-tip">{t('account.page.personal.phoneTip')}</span>
              </div>
            </div>

            <div class="form-grid-2">
              <div class="form-field">
                <label for="prof-lang" class="form-label">{t('account.page.personal.languageLabel')}</label>
                <select id="prof-lang" class="form-input" bind:value={preferredLanguage}>
                  <option value="en">English (India)</option>
                  <option value="hi">हिन्दी (Hindi)</option>
                  <option value="mr">मराठी (Marathi)</option>
                  <option value="bn">বাংলা (Bengali)</option>
                  <option value="ta">தமிழ் (Tamil)</option>
                  <option value="te">తెలుగు (Telugu)</option>
                </select>
              </div>

              <div class="form-field">
                <span class="form-label">{t('account.page.personal.notificationChannels')}</span>
                <div class="toggle-row">
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={emailAlerts} />
                    <span>{t('account.page.personal.emailAlertsLabel')}</span>
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={whatsappAlerts} />
                    <span>{t('account.page.personal.whatsappAlertsLabel')}</span>
                  </label>
                </div>
              </div>
            </div>

            <div class="form-submit-row">
              <Tooltip text={tooltip('tooltip.save')}>
              {#snippet trigger(tp)}
                <button type="submit" class="primary-save-btn" {...tp}>
                  {t('account.page.personal.saveButton')}
                </button>
              {/snippet}
            </Tooltip>
            </div>
          </form>
        </div>

      <!-- Panel 2: Saved Delivery Addresses (Amazon Style) -->
      {:else if activeTab === 'addresses'}
        <div class="detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{t('account.page.addresses.title')}</h2>
              <p class="panel-desc">{t('account.page.addresses.desc')}</p>
            </div>
            <Tooltip text={tooltip('tooltip.addAddress')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="add-addr-btn"
                onclick={() => (isAddAddressOpen = true)}
                {...tp}
              >
                <Icon name="plus" size="0.95rem" />
                {t('account.page.addresses.addButton')}
              </button>
            {/snippet}
          </Tooltip>
          </div>

          <div class="addresses-grid">
            {#each savedAddresses as addr (addr.id)}
              <div class="address-card {addr.isDefault ? 'is-default-card' : ''}">
                <div class="addr-header-row">
                  <span class="addr-type-tag">{addr.type}</span>
                  {#if addr.isDefault}
                    <span class="default-badge">{t('account.page.addresses.defaultBadge')}</span>
                  {/if}
                </div>

                <h3 class="addr-name">{addr.name}</h3>
                <p class="addr-text">
                  {addr.lines}<br />
                  {addr.city}, {addr.state} - <strong>{addr.pin}</strong><br />
                  {t('account.page.addresses.country')}
                </p>
                <p class="addr-phone">{t('account.page.addresses.phoneLabel', { phone: addr.phone })}</p>
                {#if addr.gstin}
                  <p class="addr-gstin">{t('account.page.addresses.gstinLabel')} <code>{addr.gstin}</code></p>
                {/if}

                <div class="addr-actions">
                  {#if !addr.isDefault}
                    <Tooltip text={tooltip('tooltip.setDefault')}>
                      {#snippet trigger(tp)}
                        <button
                          type="button"
                          class="addr-text-btn"
                          onclick={() => handleSetDefaultAddress(addr.id)}
                          {...tp}
                        >
                          {t('account.page.addresses.setDefault')}
                        </button>
                      {/snippet}
                    </Tooltip>
                    <span class="btn-divider">|</span>
                  {/if}
                  <Tooltip text={tooltip('tooltip.removeAddress')}>
                    {#snippet trigger(tp)}
                      <button
                        type="button"
                        class="addr-text-btn remove-btn"
                        onclick={() => handleDeleteAddress(addr.id)}
                        {...tp}
                      >
                        {t('action.remove')}
                      </button>
                    {/snippet}
                  </Tooltip>
                </div>
              </div>
            {/each}

            <!-- Add Address Card Placeholder -->
            <Tooltip text={tooltip('tooltip.addAddress')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="new-address-placeholder-card"
                onclick={() => (isAddAddressOpen = true)}
                {...tp}
              >
                <div class="plus-circle">
                  <Icon name="plus" size="1.5rem" />
                </div>
                <strong>{t('account.page.addresses.placeholderTitle')}</strong>
                <small>{t('account.page.addresses.placeholderSub')}</small>
              </button>
            {/snippet}
          </Tooltip>
          </div>
        </div>

      <!-- Panel 3: Login & Security (Amazon Style) -->
      {:else if activeTab === 'security'}
        <div class="detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{t('account.page.security.title')}</h2>
              <p class="panel-desc">{t('account.page.security.desc')}</p>
            </div>
          </div>

          <div class="security-sections">
            <!-- Row 1: Name & Email -->
            <div class="security-row">
              <div class="sec-meta">
                <strong>{t('account.page.security.nameLabel')}</strong>
                <span>{fullName}</span>
              </div>
              <Tooltip text={tooltip('tooltip.edit')}>
                {#snippet trigger(tp)}
                  <button type="button" class="sec-edit-btn" onclick={() => (activeTab = 'personal')} {...tp}>
                    {t('account.page.security.editButton')}
                  </button>
                {/snippet}
              </Tooltip>
            </div>

            <!-- Row 2: Mobile Phone Number -->
            <div class="security-row">
              <div class="sec-meta">
                <strong>{t('account.page.personal.phoneLabel')}</strong>
                <span>{t('account.page.security.phoneVerifiedSuffix', { phone })}</span>
              </div>
              <Tooltip text={tooltip('tooltip.changePhone')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="sec-edit-btn"
                    onclick={() => (isPhoneModalOpen = true)}
                    {...tp}
                  >
                    {t('account.page.security.changeButton')}
                  </button>
                {/snippet}
              </Tooltip>
            </div>

            <!-- Row 3: Password -->
            <div class="security-row">
              <div class="sec-meta">
                <strong>{t('account.page.security.passwordLabel')}</strong>
                <span>{t('account.page.security.passwordValue')}</span>
              </div>
              <Tooltip text={tooltip('tooltip.edit')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="sec-edit-btn"
                    onclick={() => showToast({ message: t('account.page.toast.passwordResetSent'), variant: 'info' })}
                    {...tp}
                  >
                    {t('account.page.security.changeButton')}
                  </button>
                {/snippet}
              </Tooltip>
            </div>

            <!-- Row 4: Two-Factor Authentication -->
            <div class="security-row">
              <div class="sec-meta">
                <strong>{t('account.page.security.twoFaLabel')}</strong>
                <span>{t('account.page.security.twoFaDesc')}</span>
              </div>
              <Tooltip text={tooltip('tooltip.toggle2fa')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="sec-toggle-btn {twoFactorEnabled ? 'is-enabled' : ''}"
                    onclick={() => {
                      twoFactorEnabled = !twoFactorEnabled;
                      showToast({
                        message: twoFactorEnabled ? t('account.page.toast.twoFaEnabled') : t('account.page.toast.twoFaDisabled'),
                        variant: 'info',
                      });
                    }}
                    {...tp}
                  >
                    {twoFactorEnabled ? t('account.page.security.enabled') : t('account.page.security.disabled')}
                  </button>
                {/snippet}
              </Tooltip>
            </div>
          </div>

          <!-- Active Device Sessions (Go BFF Invalidation feature) -->
          <div class="active-sessions-block">
            <div class="sessions-header">
              <div>
                <h3 class="block-title">{t('account.page.security.sessionsHeading')}</h3>
                <p class="block-sub">{t('account.page.security.sessionsDesc')}</p>
              </div>
              <Tooltip text={tooltip('tooltip.revokeSessions')}>
              {#snippet trigger(tp)}
                <button
                  type="button"
                  class="revoke-all-btn"
                  onclick={handleRevokeOtherSessions}
                  {...tp}
                >
                  <Icon name="lock" size="0.85rem" />
                  {t('account.page.security.revokeAll')}
                </button>
              {/snippet}
            </Tooltip>
            </div>

            <div class="sessions-list">
              {#each activeSessions as sess (sess.id)}
                <div class="session-item {sess.isCurrent ? 'is-current-sess' : ''}">
                  <div class="sess-icon">
                    <Icon name="user" size="1.2rem" />
                  </div>
                  <div class="sess-info">
                    <div class="sess-title-row">
                      <strong>{sess.device}</strong>
                      {#if sess.isCurrent}
                        <span class="current-badge">{t('account.page.security.thisDevice')}</span>
                      {/if}
                    </div>
                    <span class="sess-loc">{t('account.page.security.sessionLocationLine', { location: sess.location, ip: sess.ip })}</span>
                    <span class="sess-time">{sess.lastActive}</span>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- DPDP Act 2023 Data Controls -->
          <div class="dpdp-privacy-block">
            <h3 class="block-title">{t('account.page.security.dpdpHeading')}</h3>
            <p class="block-sub">{t('account.page.security.dpdpDesc')}</p>
            <div class="dpdp-actions">
              <Tooltip text={tooltip('tooltip.exportData')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="dpdp-btn"
                    onclick={() => showToast({ message: t('account.page.toast.dataArchiveStarted'), variant: 'success' })}
                    {...tp}
                  >
                    <Icon name="download" size="0.9rem" />
                    {t('account.page.security.downloadArchive')}
                  </button>
                {/snippet}
              </Tooltip>
              <Tooltip text={tooltip('tooltip.withdrawConsent')}>
                {#snippet trigger(tp)}
                  <button
                    type="button"
                    class="dpdp-btn danger"
                    onclick={() => showToast({ message: t('account.page.toast.consentWithdrawalLogged'), variant: 'info' })}
                    {...tp}
                  >
                    {t('account.page.security.manageConsent')}
                  </button>
                {/snippet}
              </Tooltip>
            </div>
          </div>
        </div>

      <!-- Panel 4: Recent Orders Quick Rail -->
      {:else if activeTab === 'orders'}
        <div class="detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{t('account.page.orders.title')}</h2>
              <p class="panel-desc">{t('account.page.orders.desc')}</p>
            </div>
            <a href="/orders" class="view-all-orders-btn">
              {t('account.page.orders.viewAll')}
              <Icon name="arrow-right" size="0.85rem" />
            </a>
          </div>

          <div class="orders-rail-list">
            <!-- Order Item 1 -->
            <div class="order-rail-card">
              <div class="rail-card-head">
                <div>
                  <span class="order-id">Order #KLK-79402</span>
                  <span class="order-date">Placed on 01 Sep 2026</span>
                </div>
                <div class="rail-price-wrap">
                  <span class="rail-price">₹18,500.00</span>
                  <span class="rail-status-pill status-ship">Dispatched via India Post</span>
                </div>
              </div>

              <div class="rail-card-body">
                <img
                  src="/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg"
                  alt="Dhamadka Ajrakh Saree"
                  class="rail-thumb"
                />
                <div class="rail-prod-info">
                  <h3 class="rail-prod-title">Dhamadka Natural Indigo Block-Printed Silk Saree</h3>
                  <p class="rail-artisan">{t('account.page.orders.craftedByLabel')} <strong>Master Artisan Ismail Khatri</strong> • Ajrakhpur, Gujarat</p>
                  <span class="rail-gi-tag">Certified GI Registered Craft (GI-184)</span>

                  <!-- 4-Stage Loom Progress Bar -->
                  <div class="rail-progress-track">
                    <div class="track-step step-done"><span>{t('account.page.orders.step.ordered')}</span></div>
                    <div class="track-step step-done"><span>{t('account.page.orders.step.loomWoven')}</span></div>
                    <div class="track-step step-done"><span>{t('account.page.orders.step.giSealed')}</span></div>
                    <div class="track-step step-active"><span>{t('account.page.orders.step.enRoute')}</span></div>
                  </div>
                </div>
              </div>

              <div class="rail-card-foot">
                <a href="/orders/ord-1" class="rail-action-link">{t('account.page.orders.trackPackage')}</a>
                <span class="dot-sep">•</span>
                <a href="/verify" class="rail-action-link">{t('account.page.orders.viewSeal')}</a>
                <span class="dot-sep">•</span>
                <Tooltip text={tooltip('tooltip.downloadInvoice')}>
                    {#snippet trigger(tp)}
                      <button
                        type="button"
                        class="rail-invoice-btn"
                        onclick={() => showToast({ message: t('account.page.toast.invoiceDownloaded'), variant: 'success' })}
                        {...tp}
                      >
                        {t('account.page.orders.downloadInvoice')}
                      </button>
                    {/snippet}
                  </Tooltip>
              </div>
            </div>

            <!-- Order Item 2 -->
            <div class="order-rail-card">
              <div class="rail-card-head">
                <div>
                  <span class="order-id">Order #KLK-68190</span>
                  <span class="order-date">Placed on 18 Aug 2026</span>
                </div>
                <div class="rail-price-wrap">
                  <span class="rail-price">₹12,200.00</span>
                  <span class="rail-status-pill status-delivered">Delivered & Verified</span>
                </div>
              </div>

              <div class="rail-card-body">
                <img
                  src="/craft-images/metalwork/dhokra-casting.jpg"
                  alt="Bastar Brass Dhokra Nandi"
                  class="rail-thumb"
                />
                <div class="rail-prod-info">
                  <h3 class="rail-prod-title">Bastar Hand-Cast Lost-Wax Bell Metal Dhokra Nandi</h3>
                  <p class="rail-artisan">{t('account.page.orders.craftedByLabel')} <strong>Sukchand Ghadwa</strong> • Bastar Tribal Collective, Chhattisgarh</p>
                  <span class="rail-gi-tag">Certified GI Registered Craft (GI-83)</span>

                  <div class="rail-progress-track">
                    <div class="track-step step-done"><span>{t('account.page.orders.step.ordered')}</span></div>
                    <div class="track-step step-done"><span>{t('account.page.orders.step.castInWax')}</span></div>
                    <div class="track-step step-done"><span>{t('account.page.orders.step.giSealed')}</span></div>
                    <div class="track-step step-done"><span>{t('account.page.orders.step.delivered')}</span></div>
                  </div>
                </div>
              </div>

              <div class="rail-card-foot">
                <a href="/orders/ord-2" class="rail-action-link">{t('account.page.orders.orderDetails')}</a>
                <span class="dot-sep">•</span>
                <a href="/verify" class="rail-action-link">{t('account.page.orders.provenanceCertificate')}</a>
                <span class="dot-sep">•</span>
                <Tooltip text={tooltip('tooltip.downloadInvoice')}>
                    {#snippet trigger(tp)}
                      <button
                        type="button"
                        class="rail-invoice-btn"
                        onclick={() => showToast({ message: t('account.page.toast.invoiceDownloaded'), variant: 'success' })}
                        {...tp}
                      >
                        {t('account.page.orders.downloadInvoice')}
                      </button>
                    {/snippet}
                  </Tooltip>
              </div>
            </div>
          </div>
        </div>

      <!-- Panel 5: Loom Consultations & Video Calls -->
      {:else if activeTab === 'consultations'}
        <div class="detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{t('account.page.consultations.title')}</h2>
              <p class="panel-desc">{t('account.page.consultations.desc')}</p>
            </div>
          </div>

          <div class="consultation-card">
            <div class="consult-badge-row">
              <span class="consult-status-badge">Confirmed • Tomorrow at 04:30 PM IST</span>
              <span class="consult-type">Bespoke Loom Commission</span>
            </div>

            <div class="consult-main">
              <div class="artisan-preview">
                <div class="artisan-avatar-circle">RM</div>
                <div>
                  <h3 class="artisan-head-name">Master Weaver Ramdas Maurya</h3>
                  <p class="artisan-guild">Varanasi Brocade Guild, Uttar Pradesh (National Awardee)</p>
                </div>
              </div>

              <p class="consult-topic">
                <strong>{t('account.page.consultations.topicLabel')}</strong> Custom Katan Silk weaving motif selection and gold zari thread selection for upcoming ceremonial attire.
              </p>
            </div>

            <div class="consult-actions">
              <Tooltip text={tooltip('tooltip.joinRoom')}>
              {#snippet trigger(tp)}
                <button
                  type="button"
                  class="join-room-btn"
                  onclick={() => showToast({ message: t('account.page.toast.videoRoomActivates'), variant: 'info' })}
                  {...tp}
                >
                  <Icon name="video" size="1.1rem" />
                  {t('account.page.consultations.joinRoom')}
                </button>
              {/snippet}
            </Tooltip>
            <Tooltip text={tooltip('tooltip.reschedule')}>
              {#snippet trigger(tp)}
                <button
                  type="button"
                  class="reschedule-btn"
                  onclick={() => showToast({ message: t('account.page.toast.rescheduleSent'), variant: 'info' })}
                  {...tp}
                >
                  {t('account.page.consultations.reschedule')}
                </button>
              {/snippet}
            </Tooltip>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<!-- Modal: Change Phone Number via OTP (Go BFF Integration) -->
{#if isPhoneModalOpen}
  <div class="modal-backdrop" onclick={() => (isPhoneModalOpen = false)} role="presentation">
    <div
      class="modal-box"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => { if (e.key === 'Escape') isPhoneModalOpen = false; }}
      role="dialog"
      tabindex="-1"
      aria-label={t('account.page.phoneModal.ariaLabel')}
    >
      <div class="modal-head">
        <h3 class="modal-title">{t('account.page.phoneModal.title')}</h3>
        <Tooltip text={tooltip('tooltip.close')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="modal-close"
              onclick={() => (isPhoneModalOpen = false)}
              aria-label={t('ui.dialog.close')}
              {...tp}
            >
              ✕
            </button>
          {/snippet}
        </Tooltip>
      </div>

      <div class="modal-body">
        <p class="modal-desc">
          {t('account.page.phoneModal.desc')}
        </p>

        <div class="form-field">
          <label for="new-phone-input" class="form-label">{t('account.page.phoneModal.newPhoneLabel')}</label>
          <div class="input-with-action">
            <input
              id="new-phone-input"
              type="tel"
              class="form-input"
              placeholder={t('account.phonePlaceholder')}
              bind:value={newPhone}
              maxlength="13"
            />
            <Tooltip text={tooltip('tooltip.sendOtp')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="action-inline-btn"
                onclick={handleRequestPhoneOtp}
                {...tp}
              >
                {phoneOtpSent ? t('account.page.phoneModal.resend') : t('account.page.phoneModal.sendCode')}
              </button>
            {/snippet}
          </Tooltip>
          </div>
        </div>

        {#if phoneOtpSent}
          <div class="form-field">
            <label for="phone-otp-input" class="form-label">{t('account.page.phoneModal.otpLabel')}</label>
            <input
              id="phone-otp-input"
              type="text"
              class="form-input otp-text"
              placeholder="• • • • • •"
              maxlength="6"
              bind:value={phoneOtp}
            />
            <span class="field-tip">{t('account.page.phoneModal.verificationNote')}</span>
          </div>
        {/if}
      </div>

      <div class="modal-foot">
        <Tooltip text={tooltip('tooltip.cancel')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="modal-cancel-btn"
              onclick={() => (isPhoneModalOpen = false)}
              {...tp}
            >
              {t('action.cancel')}
            </button>
          {/snippet}
        </Tooltip>
        <Tooltip text={tooltip('tooltip.verifyOtp')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="modal-submit-btn"
              disabled={!phoneOtpSent || isVerifyingPhone}
              onclick={handleVerifyPhone}
              {...tp}
            >
              {isVerifyingPhone ? t('account.page.phoneModal.verifying') : t('account.page.phoneModal.verifyButton')}
            </button>
          {/snippet}
        </Tooltip>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Add Delivery Address -->
{#if isAddAddressOpen}
  <div class="modal-backdrop" onclick={() => (isAddAddressOpen = false)} role="presentation">
    <div
      class="modal-box modal-wide"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => { if (e.key === 'Escape') isAddAddressOpen = false; }}
      role="dialog"
      tabindex="-1"
      aria-label={t('account.page.addresses.modalAriaLabel')}
    >
      <div class="modal-head">
        <h3 class="modal-title">{t('account.page.addresses.modalTitle')}</h3>
        <Tooltip text={tooltip('tooltip.close')}>
          {#snippet trigger(tp)}
            <button
              type="button"
              class="modal-close"
              onclick={() => (isAddAddressOpen = false)}
              aria-label={t('ui.dialog.close')}
              {...tp}
            >
              ✕
            </button>
          {/snippet}
        </Tooltip>
      </div>

      <form onsubmit={handleAddAddress}>
        <div class="modal-body">
          <div class="form-field">
            <label for="addr-fullname" class="form-label">{t('account.page.addresses.fullNameLabel')}</label>
            <input
              id="addr-fullname"
              type="text"
              class="form-input"
              placeholder={t('account.page.addresses.fullNamePlaceholder')}
              bind:value={newAddressFullName}
              required
            />
          </div>

          <div class="form-field">
            <label for="addr-lines" class="form-label">{t('account.page.addresses.lineLabel')}</label>
            <input
              id="addr-lines"
              type="text"
              class="form-input"
              placeholder={t('account.page.addresses.linePlaceholder')}
              bind:value={newAddressLine}
              required
            />
          </div>

          <div class="form-grid-2">
            <div class="form-field">
              <label for="addr-city" class="form-label">{t('account.page.addresses.cityLabel')}</label>
              <input
                id="addr-city"
                type="text"
                class="form-input"
                placeholder={t('account.page.addresses.cityPlaceholder')}
                bind:value={newAddressCity}
                required
              />
            </div>

            <div class="form-field">
              <label for="addr-state" class="form-label">{t('account.page.addresses.stateLabel')}</label>
              <select id="addr-state" class="form-input" bind:value={newAddressState}>
                <option value="Delhi">Delhi</option>
                <option value="Maharashtra">Maharashtra</option>
                <option value="Uttar Pradesh">Uttar Pradesh</option>
                <option value="Gujarat">Gujarat</option>
                <option value="Rajasthan">Rajasthan</option>
                <option value="Karnataka">Karnataka</option>
                <option value="Tamil Nadu">Tamil Nadu</option>
                <option value="West Bengal">West Bengal</option>
              </select>
            </div>
          </div>

          <div class="form-grid-2">
            <div class="form-field">
              <label for="addr-pin" class="form-label">{t('account.page.addresses.pinLabel')}</label>
              <input
                id="addr-pin"
                type="text"
                class="form-input"
                placeholder="110024"
                maxlength="6"
                bind:value={newAddressPin}
                required
              />
            </div>

            <div class="form-field">
              <label for="addr-gstin" class="form-label">{t('account.page.addresses.gstinInputLabel')}</label>
              <input
                id="addr-gstin"
                type="text"
                class="form-input"
                placeholder={t('account.page.addresses.gstinPlaceholder')}
                bind:value={newAddressGstin}
              />
            </div>
          </div>

          <label class="checkbox-label" style="margin-block-start: 0.5rem;">
            <input type="checkbox" bind:checked={newAddressIsDefault} />
            <span>{t('account.page.addresses.setDefaultCheckbox')}</span>
          </label>
        </div>

        <div class="modal-foot">
          <Tooltip text={tooltip('tooltip.cancel')}>
            {#snippet trigger(tp)}
              <button
                type="button"
                class="modal-cancel-btn"
                onclick={() => (isAddAddressOpen = false)}
                {...tp}
              >
                {t('action.cancel')}
              </button>
            {/snippet}
          </Tooltip>
          <Tooltip text={tooltip('tooltip.addAddress')}>
            {#snippet trigger(tp)}
              <button type="submit" class="modal-submit-btn" {...tp}>
                {t('account.page.addresses.saveButton')}
              </button>
            {/snippet}
          </Tooltip>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .account-page {
    padding-block: 1rem 3rem;
  }

  .container {
    inline-size: min(100% - 2rem, 74rem);
    margin-inline: auto;
  }

  /* User Profile Banner (Amazon Style) */
  .profile-banner {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 1.5rem 1.75rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline, var(--k-border-muted));
    border-radius: 14px;
    margin-block: 1rem 1.5rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }

  @media (max-width: 860px) {
    .profile-banner {
      flex-direction: column;
      align-items: flex-start;
      gap: 1.25rem;
    }
  }

  .profile-avatar-wrap {
    position: relative;
    flex: none;
  }

  .profile-avatar {
    inline-size: 4.25rem;
    block-size: 4.25rem;
    border-radius: 50%;
    background-color: var(--k-surface-sunken);
    color: var(--k-accent-danger-muted);
    border: 2px solid var(--k-border-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.4rem;
    font-weight: 700;
    font-family: var(--k-font-display, Georgia, serif);
  }

  .avatar-edit-btn {
    position: absolute;
    inset-inline-end: -2px;
    inset-block-end: -2px;
    inline-size: 1.6rem;
    block-size: 1.6rem;
    border-radius: 50%;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
  }

  .profile-meta {
    flex: 1;
    min-inline-size: 0;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .name-badge-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.65rem;
  }

  .profile-name {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.45rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .patron-tier-pill {
    font-size: 0.7rem;
    font-weight: 700;
    padding: 0.18rem 0.6rem;
    border-radius: 999px;
    background-color: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
    border: 1px solid var(--k-neem-300);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .member-since-pill {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 0.18rem 0.6rem;
    border-radius: 999px;
    background-color: var(--k-surface-raised);
    color: var(--k-text-tertiary);
    border: 1px solid var(--k-border-muted);
  }

  .profile-contact-strip {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.5rem;
    font-size: 0.825rem;
    color: var(--k-text-tertiary);
  }

  .contact-item {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }

  .contact-divider {
    color: var(--k-khadi-200);
  }

  .verified-dot {
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--k-accent-success-muted);
    background-color: var(--k-surface-raised);
    padding: 0.1rem 0.35rem;
    border-radius: 4px;
  }

  .profile-telemetry {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  @media (max-width: 580px) {
    .profile-telemetry {
      inline-size: 100%;
      flex-wrap: wrap;
    }
    .telemetry-card {
      flex: 1 1 5rem;
      min-inline-size: 0;
      padding: 0.5rem;
    }
    .detail-panel {
      padding: 1.25rem 0.85rem;
    }
    .panel-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.75rem;
    }
    .profile-banner {
      padding: 1.25rem 1rem;
    }
  }

  .telemetry-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 0.65rem 1rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 10px;
    min-inline-size: 5.5rem;
  }

  .telemetry-value {
    font-size: 1.35rem;
    font-weight: 800;
    color: var(--k-accent-danger-muted);
    font-family: var(--k-font-display, Georgia, serif);
  }

  .telemetry-label {
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--k-text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin-block-start: 0.15rem;
  }

  /* Amazon 6-Card Hub Grid */
  .amazon-hub-section {
    margin-block-end: 2rem;
  }

  .hub-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1.25rem;
  }

  @media (max-width: 900px) {
    .hub-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 580px) {
    .hub-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .hub-card {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    padding: 1.25rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 12px;
    text-decoration: none;
    color: inherit;
    text-align: start;
    cursor: pointer;
    font-family: inherit;
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .hub-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
    border-color: var(--k-border-danger);
  }

  .hub-card.card-active {
    border-color: var(--k-border-danger);
    background-color: var(--k-surface-base);
  }

  .card-icon-box {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 3rem;
    block-size: 3rem;
    border-radius: 10px;
    background-color: var(--k-surface-base);
    color: var(--k-accent-danger-muted);
    border: 1px solid var(--k-border-subtle);
    flex: none;
  }

  .card-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .card-title {
    font-size: 1rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .card-desc {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
    line-height: 1.35;
    margin: 0;
  }

  /* Sub-Panels Section */
  .account-details-container {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 14px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }

  .details-nav-pills {
    display: flex;
    gap: 0.5rem;
    padding: 0.85rem 1.25rem;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-border-subtle);
    overflow-x: auto;
  }

  .pill-btn {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.5rem 0.95rem;
    border-radius: 999px;
    background: transparent;
    border: 1px solid transparent;
    font-weight: 600;
    font-size: 0.825rem;
    color: var(--k-text-tertiary);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s ease;
    font-family: inherit;
  }

  .pill-btn:hover {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  .pill-btn.is-active {
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-border-danger);
  }

  .detail-panel {
    padding: 2rem;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-block-end: 1.75rem;
    padding-block-end: 1rem;
    border-block-end: 1px solid var(--k-border-subtle);
  }

  .panel-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.35rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .panel-desc {
    font-size: 0.85rem;
    color: var(--k-text-tertiary);
    margin: 0.2rem 0 0;
  }

  .add-addr-btn,
  .view-all-orders-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.5rem 0.9rem;
    border-radius: 8px;
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border: none;
    font-size: 0.825rem;
    font-weight: 600;
    text-decoration: none;
    cursor: pointer;
    transition: background-color 0.15s ease;
  }

  .add-addr-btn:hover,
  .view-all-orders-btn:hover {
    background-color: var(--k-accent-danger-bg);
  }

  /* Form Styles */
  .details-form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .form-grid-2 {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1.25rem;
  }

  @media (max-width: 650px) {
    .form-grid-2 {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .form-label {
    font-size: 0.825rem;
    font-weight: 700;
    color: var(--k-text-primary);
  }

  .form-input {
    padding: 0.65rem 0.85rem;
    border: 1px solid var(--k-border-hairline);
    border-radius: 8px;
    font-size: 0.9rem;
    color: var(--k-text-primary);
    background-color: var(--k-surface-base);
    font-family: inherit;
  }

  .form-input:focus {
    outline: none;
    border-color: var(--k-border-danger);
    box-shadow: 0 0 0 3px rgba(184, 74, 57, 0.12);
  }

  .input-with-action {
    display: flex;
    gap: 0.4rem;
  }

  .input-with-action .form-input {
    flex: 1;
  }

  .action-inline-btn {
    padding: 0.65rem 0.85rem;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-border-muted);
    border-radius: 8px;
    font-size: 0.775rem;
    font-weight: 600;
    color: var(--k-text-secondary);
    cursor: pointer;
    white-space: nowrap;
    transition: background-color 0.15s ease;
  }

  .action-inline-btn:hover {
    background-color: var(--k-surface-pressed);
  }

  .field-tip {
    font-size: 0.7rem;
    color: var(--k-stone-400);
  }

  .toggle-row {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    padding-block-start: 0.35rem;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    font-size: 0.825rem;
    color: var(--k-stone-700);
    cursor: pointer;
  }

  .form-submit-row {
    padding-block-start: 0.5rem;
  }

  .primary-save-btn {
    padding: 0.75rem 1.5rem;
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 700;
    cursor: pointer;
    transition: background-color 0.15s ease;
  }

  .primary-save-btn:hover {
    background-color: var(--k-accent-danger-bg);
  }

  /* Addresses Grid (Amazon Style) */
  .addresses-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.25rem;
  }

  @media (max-width: 900px) {
    .addresses-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 600px) {
    .addresses-grid {
      grid-template-columns: 1fr;
    }
  }

  .address-card {
    display: flex;
    flex-direction: column;
    padding: 1.25rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    border-radius: 12px;
    position: relative;
  }

  .address-card.is-default-card {
    border-color: var(--k-border-danger);
    box-shadow: 0 0 0 1px var(--k-border-danger);
  }

  .addr-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-block-end: 0.65rem;
  }

  .addr-type-tag {
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--k-text-tertiary);
  }

  .default-badge {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--k-accent-danger-muted);
  }

  .addr-name {
    font-size: 0.95rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0 0 0.4rem;
  }

  .addr-text {
    font-size: 0.8rem;
    color: var(--k-stone-600);
    line-height: 1.45;
    margin: 0 0 0.5rem;
    flex: 1;
  }

  .addr-phone {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
    margin: 0 0 0.35rem;
  }

  .addr-gstin {
    font-size: 0.725rem;
    color: var(--k-text-secondary);
    background-color: var(--k-surface-base);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    margin: 0 0 0.75rem;
  }

  .addr-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-block-start: auto;
    padding-block-start: 0.75rem;
    border-block-start: 1px solid var(--k-border-subtle);
  }

  .addr-text-btn {
    background: transparent;
    border: none;
    padding: 0;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--k-indigo-900);
    cursor: pointer;
  }

  .addr-text-btn.remove-btn {
    color: var(--k-accent-danger-muted);
  }

  .btn-divider {
    color: var(--k-khadi-200);
    font-size: 0.75rem;
  }

  .new-address-placeholder-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem 1.25rem;
    background-color: var(--k-surface-base);
    border: 2px dashed var(--k-border-hairline);
    border-radius: 12px;
    cursor: pointer;
    text-align: center;
    transition: all 0.15s ease;
    font-family: inherit;
  }

  .new-address-placeholder-card:hover {
    border-color: var(--k-border-danger);
    background-color: var(--k-surface-base);
  }

  .plus-circle {
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border-radius: 50%;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--k-accent-danger-muted);
    margin-block-end: 0.65rem;
  }

  .new-address-placeholder-card strong {
    font-size: 0.9rem;
    color: var(--k-text-primary);
  }

  .new-address-placeholder-card small {
    font-size: 0.75rem;
    color: var(--k-text-tertiary);
    margin-block-start: 0.2rem;
  }

  /* Security Settings Rows */
  .security-sections {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--k-border-subtle);
    border-radius: 10px;
    margin-block-end: 2rem;
  }

  .security-sections > * + * {
    border-block-start: 1px solid var(--k-border-subtle);
  }

  .banner-signout-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.45rem 0.85rem;
    background: transparent;
    border: 1px solid var(--k-border-hairline);
    border-radius: 6px;
    color: var(--k-text-tertiary);
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .banner-signout-btn:hover {
    background-color: var(--k-surface-base);
    border-color: var(--k-terracotta-400);
    color: var(--k-accent-danger-muted);
  }

  .security-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    background-color: var(--k-surface-base);
  }

  .sec-meta {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .sec-meta strong {
    font-size: 0.85rem;
    color: var(--k-text-primary);
  }

  .sec-meta span {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
  }

  .sec-edit-btn {
    padding: 0.4rem 0.85rem;
    border: 1px solid var(--k-border-hairline);
    border-radius: 6px;
    background-color: var(--k-surface-base);
    color: var(--k-stone-700);
    font-size: 0.775rem;
    font-weight: 600;
    cursor: pointer;
  }

  .sec-toggle-btn {
    padding: 0.4rem 0.85rem;
    border-radius: 6px;
    font-size: 0.775rem;
    font-weight: 700;
    cursor: pointer;
    border: 1px solid var(--k-border-hairline);
    background-color: var(--k-surface-raised);
    color: var(--k-text-tertiary);
  }

  .sec-toggle-btn.is-enabled {
    background-color: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
    border-color: var(--k-neem-300);
  }

  /* Active Sessions */
  .active-sessions-block,
  .dpdp-privacy-block {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 10px;
    padding: 1.25rem;
    margin-block-end: 1.5rem;
  }

  .sessions-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-block-end: 1rem;
  }

  .block-title {
    font-size: 0.95rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .block-sub {
    font-size: 0.75rem;
    color: var(--k-text-tertiary);
    margin: 0.15rem 0 0;
  }

  .revoke-all-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.45rem 0.85rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    border-radius: 6px;
    color: var(--k-accent-danger-muted);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
  }

  .sessions-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .session-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 8px;
  }

  .session-item.is-current-sess {
    border-color: var(--k-border-danger);
  }

  .sess-icon {
    inline-size: 2.25rem;
    block-size: 2.25rem;
    border-radius: 50%;
    background-color: var(--k-surface-raised);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--k-accent-danger-muted);
  }

  .sess-info {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .sess-title-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .sess-title-row strong {
    font-size: 0.825rem;
    color: var(--k-text-primary);
  }

  .current-badge {
    font-size: 0.65rem;
    font-weight: 700;
    background-color: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
  }

  .sess-loc {
    font-size: 0.725rem;
    color: var(--k-text-tertiary);
  }

  .sess-time {
    font-size: 0.675rem;
    color: var(--k-stone-400);
  }

  .dpdp-actions {
    display: flex;
    gap: 0.75rem;
    margin-block-start: 1rem;
  }

  .dpdp-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.55rem 1rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-hairline);
    color: var(--k-text-primary);
  }

  .dpdp-btn.danger {
    color: var(--k-accent-danger-muted);
    border-color: var(--k-madder-400);
  }

  /* Recent Orders Quick Rail */
  .orders-rail-list {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .order-rail-card {
    border: 1px solid var(--k-border-muted);
    border-radius: 12px;
    background-color: var(--k-surface-base);
    overflow: hidden;
  }

  .rail-card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.85rem 1.25rem;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-border-subtle);
  }

  .order-id {
    font-weight: 700;
    font-size: 0.85rem;
    color: var(--k-text-primary);
    margin-inline-end: 0.65rem;
  }

  .order-date {
    font-size: 0.75rem;
    color: var(--k-text-tertiary);
  }

  .rail-price-wrap {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .rail-price {
    font-weight: 800;
    font-size: 0.95rem;
    color: var(--k-text-primary);
    font-family: var(--k-font-display, Georgia, serif);
  }

  .rail-status-pill {
    font-size: 0.7rem;
    font-weight: 700;
    padding: 0.2rem 0.55rem;
    border-radius: 999px;
  }

  .status-ship {
    background-color: var(--k-surface-neutral);
    color: var(--k-indigo-600);
  }

  .status-delivered {
    background-color: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
  }

  .rail-card-body {
    display: flex;
    gap: 1.25rem;
    padding: 1.25rem;
  }

  .rail-thumb {
    inline-size: 5.5rem;
    block-size: 5.5rem;
    object-fit: cover;
    border-radius: 8px;
    border: 1px solid var(--k-border-subtle);
    flex: none;
  }

  .rail-prod-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .rail-prod-title {
    font-size: 0.975rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .rail-artisan {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
    margin: 0;
  }

  .rail-gi-tag {
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--k-accent-danger-muted);
  }

  .rail-progress-track {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.35rem;
    margin-block-start: 0.65rem;
  }

  .track-step {
    text-align: center;
    padding: 0.35rem;
    border-radius: 4px;
    font-size: 0.675rem;
    font-weight: 700;
    background-color: var(--k-surface-raised);
    color: var(--k-stone-400);
  }

  .track-step.step-done {
    background-color: var(--k-surface-raised);
    color: var(--k-accent-success-muted);
  }

  .track-step.step-active {
    background-color: var(--k-surface-neutral);
    color: var(--k-indigo-600);
  }

  .rail-card-foot {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1.25rem;
    background-color: var(--k-surface-base);
    border-block-start: 1px solid var(--k-border-subtle);
  }

  .rail-action-link {
    font-size: 0.775rem;
    font-weight: 600;
    color: var(--k-indigo-900);
    text-decoration: underline;
  }

  .dot-sep {
    color: var(--k-stone-200);
  }

  .rail-invoice-btn {
    background: transparent;
    border: none;
    font-size: 0.775rem;
    font-weight: 600;
    color: var(--k-text-secondary);
    cursor: pointer;
  }

  .rail-invoice-btn:hover {
    text-decoration: underline;
  }

  /* Consultation Card */
  .consultation-card {
    border: 1px solid var(--k-border-muted);
    border-radius: 12px;
    padding: 1.5rem;
    background-color: var(--k-surface-base);
  }

  .consult-badge-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-block-end: 1rem;
  }

  .consult-status-badge {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.25rem 0.65rem;
    border-radius: 999px;
    background-color: var(--k-surface-pressed);
    color: var(--k-accent-primary-text);
    border: 1px solid var(--k-haldi-300);
  }

  .consult-type {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--k-text-tertiary);
  }

  .artisan-preview {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-block-end: 0.75rem;
  }

  .artisan-avatar-circle {
    inline-size: 2.75rem;
    block-size: 2.75rem;
    border-radius: 50%;
    background-color: var(--k-surface-sunken);
    color: var(--k-accent-danger-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    border: 1px solid var(--k-border-muted);
  }

  .artisan-head-name {
    font-size: 1rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .artisan-guild {
    font-size: 0.775rem;
    color: var(--k-text-tertiary);
    margin: 0;
  }

  .consult-topic {
    font-size: 0.825rem;
    color: var(--k-stone-700);
    background-color: var(--k-surface-base);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    border: 1px solid var(--k-border-subtle);
    line-height: 1.45;
  }

  .consult-actions {
    display: flex;
    gap: 0.75rem;
    margin-block-start: 1.25rem;
  }

  .join-room-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.65rem 1.25rem;
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border: none;
    border-radius: 8px;
    font-weight: 700;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .reschedule-btn {
    padding: 0.65rem 1.25rem;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-border-muted);
    border-radius: 8px;
    font-weight: 600;
    font-size: 0.85rem;
    color: var(--k-text-secondary);
    cursor: pointer;
  }

  /* Modals */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(2px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    z-index: 1000;
  }

  .modal-box {
    inline-size: 100%;
    max-inline-size: 28rem;
    background-color: var(--k-surface-base);
    border-radius: 14px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.18);
    overflow: hidden;
  }

  .modal-box.modal-wide {
    max-inline-size: 34rem;
  }

  .modal-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.15rem 1.5rem;
    background-color: var(--k-surface-base);
    border-block-end: 1px solid var(--k-border-subtle);
  }

  .modal-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .modal-close {
    background: transparent;
    border: none;
    font-size: 1.1rem;
    color: var(--k-text-tertiary);
    cursor: pointer;
  }

  .modal-body {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .modal-desc {
    font-size: 0.825rem;
    color: var(--k-text-tertiary);
    margin: 0;
    line-height: 1.4;
  }

  .otp-text {
    font-size: 1.25rem;
    letter-spacing: 0.35em;
    text-align: center;
  }

  .modal-foot {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.65rem;
    padding: 1rem 1.5rem;
    background-color: var(--k-surface-base);
    border-block-start: 1px solid var(--k-border-subtle);
  }

  .modal-cancel-btn {
    padding: 0.55rem 1rem;
    background: transparent;
    border: 1px solid var(--k-border-hairline);
    border-radius: 6px;
    font-weight: 600;
    font-size: 0.8rem;
    color: var(--k-stone-600);
    cursor: pointer;
  }

  .modal-submit-btn {
    padding: 0.55rem 1.25rem;
    background-color: var(--k-accent-danger-bg);
    border: none;
    border-radius: 6px;
    font-weight: 700;
    font-size: 0.825rem;
    color: var(--k-text-on-accent);
    cursor: pointer;
  }

  .modal-submit-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
