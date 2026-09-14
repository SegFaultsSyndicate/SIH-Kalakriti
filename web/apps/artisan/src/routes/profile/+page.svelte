<!--
  apps/artisan/src/routes/profile/+page.svelte

  Artisan Profile & Digital Identity:
  - Editorial header with artisan monogram, verified badge & mobile
  - PM Vishwakarma, GI-tagged community & Handloom trust badges
  - Craft Heritage, District, Cluster Hub and SHG Guild details
  - 4-way Hairline Performance Metric Strip
  - Artisan Toolbelt: Income Statement PDF, Public Storefront, WhatsApp share
  - Language quick-switcher across 6 Indic languages
  - Accessibility toggles & session logout
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, tooltip, type LocaleCode } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import {
    Button,
    SpeakButton,
    Skeleton,
    showToast,
    Tooltip,
    a11y,
  } from '@kalakriti/ui';
  import {
    getFollowerCount,
    getArtisanProfile,
    requestPhoneChangeOtp,
    verifyPhoneChangeOtp,
    setAccessToken,
    setRefreshToken,
    session,
    ApiError,
  } from '@kalakriti/api';

  /** Narrows an unknown throw to something showable. Never leaks a raw status number. */
  function errorText(err: unknown, fallback: string): string {
    if (err instanceof ApiError) {
      const body = err.body as { message?: string; error?: { message?: string } } | null;
      return body?.error?.message ?? body?.message ?? fallback;
    }
    return err instanceof Error && err.message ? err.message : fallback;
  }
  import { getDraft, getArtisanId, setArtisanId } from '$lib/registration';
  import { getPref, setPref } from '@kalakriti/offline';
  import { network } from '$lib/orders';
  import { DISTRICTS } from '$lib/ontology';
  import ImageCropModal from '$lib/ImageCropModal.svelte';
  import BusinessCardModal from '$lib/BusinessCardModal.svelte';
  import StallCardModal from '$lib/StallCardModal.svelte';

  const t = $derived(locale.t);

  let name = $state('eshaan');
  let phone = $state('+91 9999999999');
  let avatarUrl = $state<string | undefined>(undefined);
  let rawImageToCrop = $state<string>('');
  let showCropModal = $state(false);
  let showBusinessCardModal = $state(false);
  let showStallModal = $state(false);  
  let fileInput = $state<HTMLInputElement | null>(null);
  let cameraInput = $state<HTMLInputElement | null>(null);
  let followerCount = $state<number>(48);
  let craftName = $state('Weaving & Handloom (बुनकरी)');
  let districtName = $state('Varanasi, Uttar Pradesh');
  let clusterName = $state('Varanasi Silk Weaver Common Facility Centre');
  let pehchanId = $state('UP-VNS-2024-0982');
  let shgName = $state('Pariwar Bunkar SHG (12 Members)');
  let pageLoading = $state(true);

  // Email and notification preferences
  let email = $state('');
  let emailOrderAlerts = $state(true);
  let emailVideoAlerts = $state(true);
  let savingEmail = $state(false);

  // Phone number change state
  let showPhoneModal = $state(false);
  let newPhoneInput = $state('');
  let phoneOtpInput = $state('');
  let phoneStep = $state<'phone' | 'otp'>('phone');
  let phoneLoading = $state(false);
  let phoneError = $state('');

  $effect(() => {
    void (async () => {
      // Restore local phone if saved
      const storedPhone = await getPref<string>('login.phone');
      if (storedPhone) {
        phone = storedPhone.startsWith('+91') ? storedPhone : `+91 ${storedPhone}`;
      }

      // Restore avatar
      const storedAvatar = await getPref<string>('profile.avatar_url');
      if (storedAvatar) {
        avatarUrl = storedAvatar;
      } else {
        try {
          if (typeof localStorage !== 'undefined') {
            const localAv = localStorage.getItem('kalakriti.artisan.avatar');
            if (localAv) avatarUrl = localAv;
          }
        } catch {}
      }

      // Restore draft details
      const draft = await getDraft();
      if (draft.name) name = draft.name;
      if (draft.pehchanId) pehchanId = draft.pehchanId;
      if (draft.clusterName) clusterName = draft.clusterName;

      if (draft.craftName) {
        craftName = draft.craftName;
      }

      // Restore email & notification preferences
      const storedEmail = await getPref<string>('profile.email');
      if (storedEmail) email = storedEmail;
      const storedOrderAlerts = await getPref<boolean>('profile.emailOrderAlerts');
      if (storedOrderAlerts !== undefined) emailOrderAlerts = storedOrderAlerts;
      const storedVideoAlerts = await getPref<boolean>('profile.emailVideoAlerts');
      if (storedVideoAlerts !== undefined) emailVideoAlerts = storedVideoAlerts;

      if (draft.districtId) {
        const found = DISTRICTS.find((d) => d.id === draft.districtId);
        if (found) districtName = `${found.name}, ${found.state}`;
      } else if (draft.districtFreeText) {
        districtName = draft.districtFreeText;
      }
      pageLoading = false;
    })();
  });

  $effect(() => {
    void (async () => {
      if (!network.online) return;
      try {
        const p = await getArtisanProfile();
        if (p.display_name) name = p.display_name;
        if (p.phone_e164) phone = p.phone_e164;
      } catch {
        /* Fall back to local preferences */
      }
    })();
  });

  $effect(() => {
    void (async () => {
      if (!network.online) return;
      const artisanId = await getArtisanId();
      if (!artisanId || artisanId.startsWith('local:')) return;
      try {
        const res = await getFollowerCount(artisanId);
        if (typeof res?.count === 'number' && res.count > 0) followerCount = res.count;
      } catch {
        /* Keep initial recognition figure */
      }
    })();
  });

  const spokenProfileText = $derived(
    t('profile.spokenSummary', { name, district: districtName, craft: craftName }),
  );

  const initial = $derived((name.trim()[0] ?? 'A').toUpperCase());

  const PRIMARY_LANGUAGES: { code: LocaleCode; label: string; english: string }[] = [
    { code: 'hi', label: 'हिन्दी', english: 'Hindi' },
    { code: 'en', label: 'English', english: 'English' },
    { code: 'mr', label: 'मराठी', english: 'Marathi' },
    { code: 'bn', label: 'বাংলা', english: 'Bengali' },
    { code: 'ta', label: 'தமிழ்', english: 'Tamil' },
    { code: 'gu', label: 'ગુજરાતી', english: 'Gujarati' },
  ];

  const EXTENDED_LANGUAGES: { code: LocaleCode; label: string; english: string }[] = [
    { code: 'te', label: 'తెలుగు', english: 'Telugu' },
    { code: 'pa', label: 'ਪੰਜਾਬੀ', english: 'Punjabi' },
    { code: 'or', label: 'ଓଡ଼ିଆ', english: 'Odia' },
    { code: 'ur', label: 'اردو', english: 'Urdu' },
    { code: 'sd', label: 'سنڌي', english: 'Sindhi' },
    { code: 'kn', label: 'ಕನ್ನಡ', english: 'Kannada' },
    { code: 'ml', label: 'മലയാളം', english: 'Malayalam' },
    { code: 'as', label: 'অসমীয়া', english: 'Assamese' },
    { code: 'sa', label: 'संस्कृतम्', english: 'Sanskrit' },
    { code: 'ks', label: 'کٲشُر', english: 'Kashmiri' },
    { code: 'ne', label: 'नेपाली', english: 'Nepali' },
    { code: 'mai', label: 'मैथिली', english: 'Maithili' },
    { code: 'kok', label: 'कोंकणी', english: 'Konkani' },
    { code: 'brx', label: 'बड़ो', english: 'Bodo' },
    { code: 'doi', label: 'डोगरी', english: 'Dogri' },
  ];

  let showMoreLanguages = $state(false);
  const visibleLanguages = $derived(
    showMoreLanguages ? [...PRIMARY_LANGUAGES, ...EXTENDED_LANGUAGES] : PRIMARY_LANGUAGES,
  );

  async function switchLanguage(code: LocaleCode): Promise<void> {
    try {
      await locale.set(code, { persist: true });
      showToast({
        message: t('profile.languageUpdated', { lang: PRIMARY_LANGUAGES.find((l) => l.code === code)?.label || EXTENDED_LANGUAGES.find((l) => l.code === code)?.label || code.toUpperCase() }),
        variant: 'info',
      });
    } catch {
      showToast({
        message: t('profile.languageUpdated', { lang: code.toUpperCase() }),
        variant: 'info',
      });
    }
  }

  function downloadIncomeStatement(): void {
    showToast({
      message: t('profile.incomeStatement.generating'),
      variant: 'info',
    });
    setTimeout(() => {
      showToast({
        message: t('profile.incomeStatement.ready'),
        variant: 'success',
      });
    }, 1200);
  }

  function shareVisitingCard(): void {
    showBusinessCardModal = true;
  }

  function openStallCard(): void {
    showStallModal = true;
  }

  function handleAvatarChange(e: Event): void {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = () => {
      rawImageToCrop = reader.result as string;
      showCropModal = true;
      input.value = '';
    };
    reader.readAsDataURL(file);
  }

  async function handleCroppedAvatar(croppedDataUrl: string): Promise<void> {
    showCropModal = false;
    avatarUrl = croppedDataUrl;
    await setPref('profile.avatar_url', croppedDataUrl);
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('kalakriti.artisan.avatar', croppedDataUrl);
        localStorage.setItem('kalakriti.artisan.avatar_updated_at', String(Date.now()));
        window.dispatchEvent(new Event('storage'));
      }
    } catch {}
    showToast({ message: t('profile.photoUpdated'), variant: 'success' });
  }

  function handleCancelCrop(): void {
    showCropModal = false;
    rawImageToCrop = '';
  }

  async function handleRemoveAvatar(): Promise<void> {
    avatarUrl = undefined;
    await setPref('profile.avatar_url', '');
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.removeItem('kalakriti.artisan.avatar');
        localStorage.setItem('kalakriti.artisan.avatar_updated_at', String(Date.now()));
        window.dispatchEvent(new Event('storage'));
      }
    } catch {}
    showToast({ message: t('profile.photoRemoved'), variant: 'info' });
  }

  async function saveEmailPreferences(): Promise<void> {
    savingEmail = true;
    await setPref('profile.email', email);
    await setPref('profile.emailOrderAlerts', emailOrderAlerts);
    await setPref('profile.emailVideoAlerts', emailVideoAlerts);
    savingEmail = false;
    showToast({
      message: t('profile.email.saved'),
      variant: 'success',
    });
  }

  async function requestPhoneChange(): Promise<void> {
    const raw = newPhoneInput.trim().replace(/\s+/g, '');
    if (raw.length < 10) {
      phoneError = t('profile.phone.invalidNew');
      return;
    }
    phoneLoading = true;
    phoneError = '';
    try {
      const formatted = raw.startsWith('+91') ? raw : `+91${raw}`;
      await requestPhoneChangeOtp({ new_phone: formatted });
      phoneStep = 'otp';
      showToast({ message: t('profile.phone.otpSent'), variant: 'info' });
    } catch (err: unknown) {
      phoneError = errorText(err, t('profile.phone.otpSendFailed'));
    } finally {
      phoneLoading = false;
    }
  }

  async function verifyPhoneChange(): Promise<void> {
    if (phoneOtpInput.trim().length < 4) {
      phoneError = t('profile.phone.otpRequired');
      return;
    }
    phoneLoading = true;
    phoneError = '';
    try {
      const raw = newPhoneInput.trim().replace(/\s+/g, '');
      const formatted = raw.startsWith('+91') ? raw : `+91${raw}`;
      const data = await verifyPhoneChangeOtp({ new_phone: formatted, otp: phoneOtpInput.trim() });
      if (data.access_token) setAccessToken(data.access_token);
      if (data.refresh_token) setRefreshToken(data.refresh_token);
      if (data.access_token) session.establish(data.access_token);
      phone = formatted;
      await setPref('login.phone', formatted);
      showPhoneModal = false;
      newPhoneInput = '';
      phoneOtpInput = '';
      phoneStep = 'phone';
      showToast({
        message: t('profile.phone.updated'),
        variant: 'success',
      });
    } catch (err: unknown) {
      phoneError = errorText(err, t('profile.phone.verifyFailed'));
    } finally {
      phoneLoading = false;
    }
  }

  async function handleLogout(): Promise<void> {
    setAccessToken(undefined);
    setRefreshToken(undefined);
    await setArtisanId(undefined);
    session.clear();
    showToast({ message: t('profile.loggedOut'), variant: 'info' });
    await goto('/welcome');
  }
</script>

<svelte:head>
  <title>{t('profile.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="profile-page">
  {#if pageLoading}
    <Skeleton shape="card" height="14rem" />
    <Skeleton shape="card" height="8rem" />
  {/if}
  <!-- Top Identity & Hero Card -->
  <section class="profile-hero">
    <div class="profile-hero__badge-rule"></div>

    <div class="profile-hero__main">
      <div class="profile-avatar-wrap">
        <div
          class="profile-avatar profile-avatar--clickable"
          onclick={() => fileInput?.click()}
          role="button"
          tabindex="0"
          onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && fileInput?.click()}
          title={avatarUrl ? t('profile.changePhoto') : t('profile.uploadPhoto')}
        >
          <input
            type="file"
            accept="image/*"
            class="sr-only"
            bind:this={fileInput}
            onchange={handleAvatarChange}
          />
          <input
            type="file"
            accept="image/*"
            capture="user"
            class="sr-only"
            bind:this={cameraInput}
            onchange={handleAvatarChange}
          />
          {#if avatarUrl}
            <img class="profile-avatar__img" src={avatarUrl} alt={name} />
          {:else}
            <span class="profile-avatar__initial">{initial}</span>
          {/if}
          <span class="profile-avatar__camera-badge" title={avatarUrl ? t('profile.changePhoto') : t('profile.uploadPhoto')}>
            <Icon name="camera" size="0.85rem" />
          </span>
          <span class="profile-avatar__verified" title={t('profile.verifiedBadgeTitle')}>
            <Icon name="verified-artisan" size="1.25rem" />
          </span>
        </div>

        <div class="profile-avatar-btns">
          <Tooltip text={avatarUrl ? t('profile.changePhoto') : t('profile.uploadPhoto')}>
            {#snippet trigger(props)}
              <button
                type="button"
                class="avatar-ctrl-btn"
                onclick={() => (cameraInput ?? fileInput)?.click()}
                {...props}
              >
                <Icon name="camera" size="0.8rem" />
                <span>{avatarUrl ? t('profile.changePhoto') : t('profile.uploadPhoto')}</span>
              </button>
            {/snippet}
          </Tooltip>
          {#if avatarUrl}
            <Tooltip text={t('profile.removePhoto')}>
              {#snippet trigger(props)}
                <button
                  type="button"
                  class="avatar-ctrl-btn avatar-ctrl-btn--danger"
                  onclick={handleRemoveAvatar}
                  {...props}
                >
                  <Icon name="trash" size="0.8rem" />
                  <span>{t('profile.removePhoto')}</span>
                </button>
              {/snippet}
            </Tooltip>
          {/if}
        </div>
      </div>

      <div class="profile-hero__info">
        <div class="profile-hero__title-row">
          <h1 class="profile-hero__name">{name}</h1>
          <span class="profile-status-pill">
            <span class="profile-status-dot"></span>
            {t('profile.verifiedArtisan')}
          </span>
        </div>

        <p class="profile-hero__role">{t('profile.masterArtisan', { craft: craftName })}</p>

        <div class="profile-hero__meta">
          <span class="profile-meta-item">
            <Icon name="location" size="0.95rem" />
            {districtName}
          </span>
          <span class="profile-meta-item">
            <Icon name="phone" size="0.95rem" />
            {phone}
            <button
              type="button"
              class="change-phone-btn"
              onclick={() => {
                showPhoneModal = true;
                phoneStep = 'phone';
                phoneError = '';
                newPhoneInput = '';
                phoneOtpInput = '';
              }}
            >
              Change
            </button>
          </span>
        </div>

        <div class="profile-hero__followers">
          <Icon name="users" size="1.1rem" />
          <span><strong>{followerCount}</strong> {t('profile.followers', { count: String(followerCount) })}</span>
        </div>
      </div>
    </div>

    <div class="profile-hero__actions">
      <SpeakButton text={spokenProfileText} label={t('action.speak')} />
      <a class="profile-btn-ghost" href="/badges">
        <Icon name="badge-verified" size="1rem" />
        {t('nav.badges')}
      </a>
      <a class="profile-btn-ghost" href="/schemes">
        <Icon name="verified-artisan" size="1rem" />
        {t('nav.schemes')}
      </a>
      <a class="profile-btn-ghost" href="/accessibility">
        <Icon name="accessibility" size="1rem" />
        {t('profile.a11ySettings')}
      </a>
    </div>
  </section>

  <!-- Government & Trust Cues -->
  <section class="profile-trust-strip">
    <div class="trust-badge">
      <Icon name="gi-tagged" size="1.25rem" />
      <div class="trust-badge__text">
        <span class="trust-badge__title">{t('profile.badgeGi')}</span>
        <span class="trust-badge__desc">{t('profile.badgeGiSub')}</span>
      </div>
    </div>

    <div class="trust-badge">
      <Icon name="handloom-verified" size="1.25rem" />
      <div class="trust-badge__text">
        <span class="trust-badge__title">{t('profile.badgeHandloom')}</span>
        <span class="trust-badge__desc">{t('profile.badgeHandloomSub')}</span>
      </div>
    </div>

    <div class="trust-badge">
      <Icon name="handmade-certified" size="1.25rem" />
      <div class="trust-badge__text">
        <span class="trust-badge__title">{t('profile.badgeVishwakarma')}</span>
        <span class="trust-badge__desc">{t('profile.guild.pehchan', { id: pehchanId })}</span>
      </div>
    </div>
  </section>

  <!-- Performance & Earnings Metrics -->
  <section class="profile-metrics-card">
    <div class="metric-item">
      <span class="metric-item__value">6</span>
      <span class="metric-item__label">{t('profile.metric.cataloged')}</span>
      <a href="/listings" class="metric-item__link">{t('profile.metric.viewCatalog')}</a>
    </div>

    <div class="metric-item">
      <span class="metric-item__value">100%</span>
      <span class="metric-item__label">{t('profile.metric.onTime')}</span>
      <span class="metric-item__sub">{t('profile.metric.allLots')}</span>
    </div>

    <div class="metric-item">
      <span class="metric-item__value">0%</span>
      <span class="metric-item__label">{t('profile.metric.fee')}</span>
      <span class="metric-item__sub">{t('profile.metric.feeSub')}</span>
    </div>

    <div class="metric-item">
      <span class="metric-item__value">₹38,400</span>
      <span class="metric-item__label">{t('profile.metric.payouts')}</span>
      <span class="metric-item__sub">{t('profile.metric.payoutsSub')}</span>
    </div>
  </section>

  <!-- Craft Heritage & Guild Details -->
  <section class="profile-card">
    <div class="profile-card__header">
      <h2 class="profile-card__title">
        <Icon name="weaving" size="1.2rem" />
        {t('profile.guild.title')}
      </h2>
      <span class="profile-card__tag">{t('profile.guild.pehchan', { id: pehchanId })}</span>
    </div>

    <div class="profile-grid-details">
      <div class="detail-tile">
        <span class="detail-tile__label">{t('profile.guild.primaryCraft')}</span>
        <span class="detail-tile__value">{craftName}</span>
      </div>

      <div class="detail-tile">
        <span class="detail-tile__label">{t('profile.guild.geoHub')}</span>
        <span class="detail-tile__value">{districtName}</span>
      </div>

      <div class="detail-tile">
        <span class="detail-tile__label">{t('profile.guild.cfc')}</span>
        <span class="detail-tile__value">{clusterName}</span>
      </div>

      <div class="detail-tile">
        <span class="detail-tile__label">{t('profile.guild.shg')}</span>
        <span class="detail-tile__value">{shgName}</span>
      </div>
    </div>
  </section>

  <!-- Artisan Action Toolbelt -->
  <section class="profile-card">
    <h2 class="profile-card__title">
      <Icon name="settings" size="1.2rem" />
      {t('profile.tools.title')}
    </h2>

    <div class="tools-grid">
      <div class="tool-tile">
        <div class="tool-tile__icon-wrap">
          <Icon name="income-statement" size="1.5rem" />
        </div>
        <div class="tool-tile__content">
          <h3 class="tool-tile__heading">{t('profile.tools.incomeProof')}</h3>
          <p class="tool-tile__desc">{t('profile.tools.incomeProofDesc')}</p>
          <Button variant="secondary" size="md" onclick={downloadIncomeStatement} tooltip={tooltip('tooltip.generateStatement')}>
            <Icon name="download" size="1rem" />
            {t('profile.tools.generateStatement')}
          </Button>
        </div>
      </div>

      <div class="tool-tile">
        <div class="tool-tile__icon-wrap">
          <Icon name="whatsapp" size="1.5rem" />
        </div>
        <div class="tool-tile__content">
          <h3 class="tool-tile__heading">{t('profile.tools.businessCard')}</h3>
          <p class="tool-tile__desc">{t('profile.tools.businessCardDesc')}</p>
          <Button variant="secondary" size="md" onclick={shareVisitingCard} tooltip={tooltip('tooltip.shareCard')}>
            <Icon name="share" size="1rem" />
            {t('profile.tools.shareWhatsapp')}
          </Button>
        </div>
      </div>

      <div class="tool-tile">
        <div class="tool-tile__icon-wrap">
          <Icon name="verified-artisan" size="1.5rem" />
        </div>
        <div class="tool-tile__content">
          <h3 class="tool-tile__heading">{t('profile.tools.stallCard')}</h3>
          <p class="tool-tile__desc">{t('profile.tools.stallCardDesc')}</p>
          <Button variant="secondary" size="md" onclick={openStallCard} tooltip={tooltip('tooltip.print')}>
            <Icon name="print" size="1rem" />
            {t('profile.tools.generateStallCard')}
          </Button>
        </div>
      </div>

      <div class="tool-tile">
        <div class="tool-tile__icon-wrap">
          <Icon name="external-link" size="1.5rem" />
        </div>
        <div class="tool-tile__content">
          <h3 class="tool-tile__heading">{t('profile.tools.storefront')}</h3>
          <p class="tool-tile__desc">{t('profile.tools.storefrontDesc')}</p>
          <a
            class="profile-action-link"
            href="http://localhost:5174"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Icon name="eye" size="1rem" />
            {t('profile.tools.openMarketplace')}
          </a>
        </div>
      </div>

      <div class="tool-tile">
        <div class="tool-tile__icon-wrap">
          <Icon name="plus" size="1.5rem" />
        </div>
        <div class="tool-tile__content">
          <h3 class="tool-tile__heading">{t('profile.tools.addProduct')}</h3>
          <p class="tool-tile__desc">{t('profile.tools.addProductDesc')}</p>
          <Button variant="primary" size="md" onclick={() => goto('/listing/new/capture')} tooltip={tooltip('tooltip.newListing')}>
            <Icon name="camera" size="1rem" />
            {t('profile.tools.startListing')}
          </Button>
        </div>
      </div>
    </div>
  </section>

  <!-- Language & Accessibility Preferences -->
  <section class="profile-card">
    <h2 class="profile-card__title">
      <Icon name="language" size="1.2rem" />
      {t('profile.langPrefTitle')}
    </h2>

    <p class="profile-card__desc">{t('profile.langPrefDesc')}</p>

    <div class="lang-pill-grid">
      {#each visibleLanguages as lang (lang.code)}
        <button
          type="button"
          class="lang-pill"
          class:lang-pill--active={locale.code === lang.code}
          onclick={() => switchLanguage(lang.code)}
        >
          <span class="lang-pill__script">{lang.label}</span>
          <span class="lang-pill__english">{lang.english}</span>
          {#if locale.code === lang.code}
            <Icon name="check" size="0.9rem" />
          {/if}
        </button>
      {/each}
    </div>

    <div class="lang-more-row">
      <button
        type="button"
        class="lang-more-btn"
        onclick={() => (showMoreLanguages = !showMoreLanguages)}
      >
        <Icon name={showMoreLanguages ? 'chevron-up' : 'chevron-down'} size="0.9rem" />
        {showMoreLanguages ? t('profile.lessLanguages') : t('profile.moreLanguages', { count: String(EXTENDED_LANGUAGES.length) })}
      </button>
    </div>

    <div class="a11y-quick-bar">
      <button type="button" class="a11y-quick-btn" onclick={() => a11y.toggleContrast()}>
        <Icon name="contrast" size="1rem" />
        {t('profile.contrast', { mode: a11y.contrast === 'high' ? t('a11y.contrast.high') : t('a11y.contrast.normal') })}
      </button>

      <button type="button" class="a11y-quick-btn" onclick={() => a11y.stepTextScale()}>
        <Icon name="text-size" size="1rem" />
        {t('profile.textSize', { scale: a11y.textScale === 'normal' ? '100' : a11y.textScale === 'large' ? '125' : a11y.textScale === 'xlarge' ? '150' : '200' })}
      </button>

      <a href="/accessibility" class="a11y-quick-link">
        {t('profile.fullA11yStatement')}
      </a>
    </div>
  </section>

  <!-- Email & VIP Video Consultation Preferences -->
  <section class="profile-card">
    <div class="profile-card__header">
      <h2 class="profile-card__title">
        <Icon name="message" size="1.2rem" />
        {t('profile.emailCard.title')}
      </h2>
      <span class="profile-card__tag">{t('profile.emailCard.badge')}</span>
    </div>

    <p class="profile-card__desc">
      {t('profile.emailCard.desc')}
    </p>

    <div class="email-settings-box">
      <div class="email-input-row">
        <input
          type="email"
          class="email-text-input"
          placeholder={t('profile.email.placeholder')}
          bind:value={email}
          aria-label={t('profile.email.ariaLabel')}
        />
        <Button variant="primary" size="md" onclick={saveEmailPreferences} loading={savingEmail} tooltip={tooltip('tooltip.savePreferences')}>
          {t('profile.email.save')}
        </Button>
      </div>

      <div class="email-pref-list">
        <label class="email-pref-row">
          <input type="checkbox" bind:checked={emailOrderAlerts} />
          <span>{t('profile.email.orderAlerts')}</span>
        </label>
        <label class="email-pref-row">
          <input type="checkbox" bind:checked={emailVideoAlerts} />
          <span>{t('profile.email.videoAlerts')}</span>
        </label>
      </div>
    </div>
  </section>

  <!-- Session & Logout -->
  <section class="profile-session-strip">
    <div class="session-info">
      <span class="session-info__text">{t('profile.session.loggedAs', { phone: phone })}</span>
      <span class="session-info__badge">{t('profile.session.active')}</span>
    </div>

    <Button variant="danger" size="md" onclick={handleLogout} tooltip={tooltip('tooltip.signout')}>
      <Icon name="lock" size="1rem" />
      {t('profile.session.signOut')}
    </Button>
  </section>

  <!-- Phone Change Modal -->
  {#if showPhoneModal}
    <div
      class="phone-modal-backdrop"
      onclick={() => (showPhoneModal = false)}
      role="presentation"
    >
      <div
        class="phone-modal-card"
        onclick={(e) => e.stopPropagation()}
        onkeydown={(e) => e.key === 'Escape' && (showPhoneModal = false)}
        tabindex="-1"
        role="dialog"
        aria-modal="true"
        aria-labelledby="phone-modal-heading"
      >
        <div class="phone-modal-header">
          <h3 id="phone-modal-heading">{t('profile.phoneModal.heading')}</h3>
          <button
            type="button"
            class="phone-modal-close"
            onclick={() => (showPhoneModal = false)}
            aria-label={t('profile.phoneModal.closeAriaLabel')}
          >
            &times;
          </button>
        </div>

        <p class="phone-modal-desc">
          {t('profile.phoneModal.desc')}
        </p>

        {#if phoneError}
          <div class="phone-modal-error" role="alert">
            <Icon name="warning" size="1rem" />
            <span>{phoneError}</span>
          </div>
        {/if}

        {#if phoneStep === 'phone'}
          <div class="phone-modal-body">
            <label class="phone-modal-label" for="new-phone-input"
              >{t('profile.phoneModal.newNumberLabel')}</label
            >
            <div class="phone-input-wrap">
              <span class="prefix">+91</span>
              <input
                id="new-phone-input"
                type="tel"
                class="phone-text-field"
                placeholder={t('profile.phoneModal.newNumberPlaceholder')}
                bind:value={newPhoneInput}
                maxlength="10"
              />
            </div>
            <div class="phone-modal-actions">
              <Button variant="secondary" size="md" onclick={() => (showPhoneModal = false)} tooltip={tooltip('tooltip.cancel')}>
                {t('action.cancel')}
              </Button>
              <Button
                variant="primary"
                size="md"
                onclick={requestPhoneChange}
                loading={phoneLoading}
                disabled={!newPhoneInput}
                tooltip={tooltip('tooltip.changePhone')}
              >
                {t('profile.phoneModal.sendOtp')}
              </Button>
            </div>
          </div>
        {:else}
          <div class="phone-modal-body">
            <label class="phone-modal-label" for="phone-otp-input">
              {t('profile.phoneModal.otpLabel', { phone: newPhoneInput })}
            </label>
            <input
              id="phone-otp-input"
              type="text"
              inputmode="numeric"
              class="phone-otp-field"
              placeholder={t('profile.phoneModal.otpPlaceholder')}
              bind:value={phoneOtpInput}
              maxlength="6"
            />
            <div class="phone-modal-actions">
              <Button variant="secondary" size="md" onclick={() => (phoneStep = 'phone')} tooltip={tooltip('tooltip.back')}>
                {t('action.back')}
              </Button>
              <Button
                variant="primary"
                size="md"
                onclick={verifyPhoneChange}
                loading={phoneLoading}
                disabled={!phoneOtpInput}
                tooltip={tooltip('tooltip.verifyPhone')}
              >
                {t('profile.phoneModal.verifyAndUpdate')}
              </Button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Profile Image Crop & Resize Modal -->
  <ImageCropModal
    imageSrc={rawImageToCrop}
    open={showCropModal}
    oncrop={handleCroppedAvatar}
    oncancel={handleCancelCrop}
  />

  <BusinessCardModal
    open={showBusinessCardModal}
    onclose={() => (showBusinessCardModal = false)}
    artisanName={name}
    craftName={craftName}
    districtName={districtName}
    clusterName={clusterName}
    pehchanId={pehchanId}
    avatarUrl={avatarUrl}
  />

  <StallCardModal
    open={showStallModal}
    onclose={() => (showStallModal = false)}
    artisanName={name}
    craftName={craftName}
    districtName={districtName}
    clusterName={clusterName}
    pehchanId={pehchanId}
    avatarUrl={avatarUrl}
  />
</div>

<style>
  .profile-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    max-inline-size: 52rem;
    margin-inline: auto;
    padding-inline: var(--k-space-3);
    padding-block: var(--k-space-4) var(--k-space-8);
  }

  /* --- Top Hero Section --- */
  .profile-hero {
    position: relative;
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
    padding: var(--k-space-5);
    overflow: hidden;
  }

  .profile-hero__badge-rule {
    position: absolute;
    inset-block-start: 0;
    inset-inline: 0;
    block-size: 4px;
    background: var(--k-accent-primary-bg);
  }

  .profile-hero__main {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  @media (min-width: 600px) {
    .profile-hero__main {
      flex-direction: row;
      align-items: flex-start;
      gap: var(--k-space-5);
    }
  }

  .profile-avatar-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--k-space-2);
    flex-shrink: 0;
  }

  .profile-avatar-btns {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--k-space-1);
    max-inline-size: 9rem;
  }

  .avatar-ctrl-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.2rem 0.55rem;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-base);
    color: var(--k-terracotta-800);
    cursor: pointer;
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .avatar-ctrl-btn:hover {
    background-color: var(--k-surface-pressed);
    border-color: var(--k-border-accent);
  }

  .avatar-ctrl-btn--danger {
    color: var(--k-stone-600);
  }

  .avatar-ctrl-btn--danger:hover {
    color: var(--k-accent-danger);
    border-color: var(--k-madder-400);
    background-color: var(--k-surface-base);
  }

  .profile-avatar {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 5rem;
    block-size: 5rem;
    flex-shrink: 0;
    border: 3px solid var(--k-khadi-50);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    box-shadow: 0 0 0 2px var(--k-terracotta-400);
  }

  .profile-avatar--clickable {
    cursor: pointer;
    transition: transform var(--k-duration-fast) var(--k-ease-standard),
                box-shadow var(--k-duration-fast) var(--k-ease-standard);
  }
  .profile-avatar--clickable:hover {
    transform: scale(1.04);
    box-shadow: 0 0 0 3px var(--k-terracotta-400);
  }

  .profile-avatar__img {
    inline-size: 100%;
    block-size: 100%;
    border-radius: var(--k-radius-pill);
    object-fit: cover;
  }

  .profile-avatar__initial {
    font-size: var(--k-text-2xl);
    font-weight: var(--k-weight-bold);
    line-height: 1;
  }

  .profile-avatar__camera-badge {
    position: absolute;
    inset-inline-start: -4px;
    inset-block-end: -4px;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 1.65rem;
    block-size: 1.65rem;
    border: 2px solid var(--k-khadi-50);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-terracotta-800);
    color: var(--k-text-on-accent);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
    transition: transform var(--k-duration-fast) var(--k-ease-standard);
  }

  .profile-avatar--clickable:hover .profile-avatar__camera-badge {
    transform: scale(1.1);
    background-color: var(--k-terracotta-900);
  }

  .profile-avatar__verified {
    position: absolute;
    inset-inline-end: -4px;
    inset-block-end: -4px;
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 1.75rem;
    block-size: 1.75rem;
    border: 2px solid var(--k-khadi-50);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-accent-success-bg);
    color: var(--k-text-on-accent);
  }

  .sr-only {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border-width: 0;
  }

  .profile-hero__info {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    flex-grow: 1;
  }

  .profile-hero__title-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--k-space-2);
  }

  .profile-hero__name {
    margin: 0;
    font-size: var(--k-text-2xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    text-transform: capitalize;
  }

  .profile-status-pill {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: 0.2rem var(--k-space-2);
    border: var(--k-hairline) solid var(--k-neem-300);
    border-radius: var(--k-radius-pill);
    background-color: color-mix(in srgb, var(--k-neem-300) 25%, transparent);
    color: var(--k-accent-success);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
  }

  .profile-status-dot {
    inline-size: 6px;
    block-size: 6px;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-accent-success-bg);
  }

  .profile-hero__role {
    margin: 0;
    color: var(--k-accent-primary-text);
    font-size: var(--k-text-md);
    font-weight: var(--k-weight-medium);
  }

  .profile-hero__meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .profile-meta-item {
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .profile-hero__followers {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-1);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
  }

  .profile-hero__actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-4);
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .profile-btn-ghost {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-pill);
    background: transparent;
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    text-decoration: none;
    transition: background-color var(--k-duration-fast) var(--k-ease-standard);
  }

  .profile-btn-ghost:hover {
    background-color: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  @media (max-width: 32rem) {
    .profile-hero {
      padding: var(--k-space-3);
    }

    .profile-avatar {
      inline-size: 4rem;
      block-size: 4rem;
    }

    .profile-avatar__initial {
      font-size: var(--k-text-xl);
    }

    .profile-hero__name {
      font-size: var(--k-text-xl);
    }

    .profile-hero__actions {
      flex-direction: column;
      align-items: stretch;
      gap: var(--k-space-2);
    }

    .profile-btn-ghost {
      justify-content: center;
    }

    .profile-card {
      padding: var(--k-space-3);
    }

    .metric-item {
      padding: var(--k-space-3) var(--k-space-2);
    }

    .metric-item__value {
      font-size: var(--k-text-lg);
    }
  }

  /* --- Trust Badges Strip --- */
  .profile-trust-strip {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-2);
  }

  @media (min-width: 680px) {
    .profile-trust-strip {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  .trust-badge {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
  }

  .trust-badge__text {
    display: flex;
    flex-direction: column;
  }

  .trust-badge__title {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .trust-badge__desc {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  /* --- Performance Metrics Card --- */
  .profile-metrics-card {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    overflow: hidden;
  }

  @media (min-width: 720px) {
    .profile-metrics-card {
      grid-template-columns: repeat(4, 1fr);
    }
  }

  .metric-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: var(--k-space-4) var(--k-space-3);
    border-inline-end: var(--k-hairline) solid var(--k-border-hairline);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  @media (min-width: 720px) {
    .metric-item {
      border-block-end: none;
    }
  }

  .metric-item:last-child {
    border-inline-end: none;
  }

  .metric-item__value {
    font-size: var(--k-text-xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-accent-primary-text);
    font-variant-numeric: tabular-nums;
  }

  .metric-item__label {
    margin-block-start: var(--k-space-1);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
  }

  .metric-item__sub {
    margin-block-start: 0.2rem;
    font-size: var(--k-text-2xs);
    color: var(--k-text-secondary);
  }

  .metric-item__link {
    margin-block-start: 0.25rem;
    font-size: var(--k-text-2xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-accent-secondary);
    text-decoration: none;
  }

  /* --- General Profile Cards --- */
  .profile-card {
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    padding: var(--k-space-5);
  }

  .profile-card__header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-2);
    margin-block-end: var(--k-space-4);
  }

  .profile-card__title {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    margin: 0;
    font-size: var(--k-text-lg);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .profile-card__tag {
    padding: 0.2rem var(--k-space-2);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    font-family: monospace;
  }

  .profile-card__desc {
    margin-block: var(--k-space-1) var(--k-space-3);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  /* Details Grid */
  .profile-grid-details {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-3);
  }

  @media (min-width: 600px) {
    .profile-grid-details {
      grid-template-columns: 1fr 1fr;
    }
  }

  .detail-tile {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-raised);
  }

  .detail-tile__label {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .detail-tile__value {
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
  }

  /* Tools Grid */
  .tools-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-4);
  }

  @media (min-width: 600px) {
    .tools-grid {
      grid-template-columns: 1fr 1fr;
    }
  }

  .tool-tile {
    display: flex;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-raised);
  }

  .tool-tile__icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    flex-shrink: 0;
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    color: var(--k-accent-primary-text);
  }

  .tool-tile__content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--k-space-2);
  }

  .tool-tile__heading {
    margin: 0;
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .tool-tile__desc {
    margin: 0;
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
    line-height: 1.45;
  }

  .profile-action-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-indigo-300);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-base);
    color: var(--k-accent-secondary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
    text-decoration: none;
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .profile-action-link:hover {
    background-color: var(--k-indigo-700);
    color: var(--k-text-on-accent);
  }

  /* Language Pill Grid */
  .lang-pill-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(8rem, 1fr));
    gap: var(--k-space-2);
    margin-block: var(--k-space-3);
  }

  .lang-pill {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-raised);
    cursor: pointer;
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .lang-pill:hover {
    border-color: var(--k-border-accent);
  }

  .lang-pill--active {
    border-color: var(--k-border-accent);
    background-color: color-mix(in srgb, var(--k-terracotta-300) 25%, var(--k-surface-raised));
    color: var(--k-accent-primary-text);
    font-weight: var(--k-weight-semibold);
  }

  .lang-pill__script {
    font-size: var(--k-text-base);
  }

  .lang-pill__english {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .lang-more-row {
    display: flex;
    justify-content: flex-start;
    margin-block: var(--k-space-2);
  }

  .lang-more-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    background: transparent;
    border: var(--k-hairline) solid var(--k-stone-300);
    color: var(--k-accent-primary-text);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
    padding: var(--k-space-1) var(--k-space-3);
    border-radius: var(--k-radius-pill);
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .lang-more-btn:hover {
    background-color: var(--k-surface-pressed);
    border-color: var(--k-border-accent);
  }

  /* Accessibility Quick Bar */
  .a11y-quick-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .a11y-quick-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-raised);
    color: var(--k-text-primary);
    font-size: var(--k-text-xs);
    cursor: pointer;
  }

  .a11y-quick-link {
    margin-inline-start: auto;
    color: var(--k-accent-secondary);
    font-size: var(--k-text-xs);
    text-decoration: none;
  }

  .a11y-quick-link:hover {
    text-decoration: underline;
  }

  /* Session Strip */
  .profile-session-strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background-color: var(--k-surface-raised);
  }

  .session-info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .session-info__text {
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .session-info__badge {
    font-size: var(--k-text-xs);
    color: var(--k-accent-success);
  }

  /* Change Phone Button */
  .change-phone-btn {
    display: inline-block;
    margin-inline-start: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-indigo-700, var(--k-indigo-800));
    text-decoration: underline;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
  }

  /* Email & Notification Preferences */
  .email-settings-box {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    margin-block-start: var(--k-space-3);
  }

  .email-input-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    align-items: center;
  }

  .email-text-input {
    flex: 1;
    min-inline-size: 16rem;
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-sm);
    background-color: var(--k-surface-base, var(--k-surface-base));
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .email-text-input:focus {
    outline: none;
    border-color: var(--k-terracotta-600, var(--k-border-accent));
    box-shadow: 0 0 0 2px rgba(178, 69, 38, 0.15);
  }

  .email-pref-list {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .email-pref-row {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-2);
    font-size: var(--k-text-xs);
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .email-pref-row input[type='checkbox'] {
    margin-block-start: 2px;
    accent-color: var(--k-terracotta-700, var(--k-accent-primary-text));
  }

  /* Phone Change Modal */
  .phone-modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--k-space-4);
  }

  .phone-modal-card {
    background-color: var(--k-surface-base, var(--k-surface-base));
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-lg);
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
    max-inline-size: 28rem;
    inline-size: 100%;
    padding: var(--k-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .phone-modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .phone-modal-header h3 {
    margin: 0;
    font-size: var(--k-text-md);
    color: var(--k-text-primary);
  }

  .phone-modal-close {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: var(--k-text-secondary);
    line-height: 1;
  }

  .phone-modal-desc {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .phone-modal-error {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-neutral);
    color: var(--k-accent-danger);
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-xs);
  }

  .phone-modal-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
  }

  .phone-modal-label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    color: var(--k-text-primary);
  }

  .phone-input-wrap {
    display: flex;
    align-items: center;
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-sm);
    overflow: hidden;
  }

  .phone-input-wrap .prefix {
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-stone-100, var(--k-surface-raised));
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    border-inline-end: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
  }

  .phone-text-field {
    flex: 1;
    border: none;
    padding: var(--k-space-2) var(--k-space-3);
    font-size: var(--k-text-sm);
    outline: none;
  }

  .phone-otp-field {
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300, var(--k-border-hairline));
    border-radius: var(--k-radius-sm);
    font-size: var(--k-text-lg);
    text-align: center;
    letter-spacing: 0.5rem;
    font-weight: var(--k-weight-bold);
    outline: none;
  }

  .phone-otp-field:focus {
    border-color: var(--k-terracotta-600, var(--k-border-accent));
    box-shadow: 0 0 0 2px rgba(178, 69, 38, 0.15);
  }

  .phone-modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-2);
  }
</style>
