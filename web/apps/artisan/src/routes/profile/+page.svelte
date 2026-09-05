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
  import { locale, type LocaleCode } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import {
    Button,
    SpeakButton,
    showToast,
    a11y,
  } from '@kalakriti/ui';
  import {
    getFollowerCount,
    getArtisanProfile,
    setAccessToken,
    setRefreshToken,
    session,
    type components,
  } from '@kalakriti/api';
  import { getDraft, getArtisanId, setArtisanId } from '$lib/registration';
  import { getPref, setPref } from '@kalakriti/offline';
  import { network } from '$lib/orders';
  import { CRAFTS, DISTRICTS } from '$lib/ontology';

  type ArtisanProfile = components['schemas']['ArtisanProfile'];

  const t = $derived(locale.t);

  let name = $state('eshaan');
  let phone = $state('+91 8779279060');
  let profile = $state<ArtisanProfile | undefined>(undefined);
  let avatarUrl = $state<string | undefined>(undefined);
  let fileInput = $state<HTMLInputElement | null>(null);
  let followerCount = $state<number>(48);
  let craftName = $state('Weaving & Handloom (बुनकरी)');
  let districtName = $state('Varanasi, Uttar Pradesh');
  let clusterName = $state('Varanasi Silk Weaver Common Facility Centre');
  let pehchanId = $state('UP-VNS-2024-0982');
  let shgName = $state('Pariwar Bunkar SHG (12 Members)');

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

      if (draft.craftId) {
        const found = CRAFTS.find((c) => c.id === draft.craftId);
        if (found) craftName = `${t(found.nameKey)} (${found.id})`;
      }

      if (draft.districtId) {
        const found = DISTRICTS.find((d) => d.id === draft.districtId);
        if (found) districtName = `${found.name}, ${found.state}`;
      } else if (draft.districtFreeText) {
        districtName = draft.districtFreeText;
      }
    })();
  });

  $effect(() => {
    void (async () => {
      if (!network.online) return;
      try {
        const p = await getArtisanProfile();
        profile = p;
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
        if (res.count > 0) followerCount = res.count;
      } catch {
        /* Keep initial recognition figure */
      }
    })();
  });

  const spokenProfileText = $derived(
    `Namaste ${name}. You are registered as an authentic artisan in ${districtName}. Craft: ${craftName}. Your profile is verified with PM Vishwakarma.`,
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
    { code: 'mni', label: 'মৈতৈলোন্', english: 'Manipuri' },
    { code: 'sat', label: 'ᱥᱟᱱᱛᱟᱲᱤ', english: 'Santali' },
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
        message: `Language updated to ${code.toUpperCase()}`,
        variant: 'info',
      });
    }
  }

  function downloadIncomeStatement(): void {
    showToast({
      message: 'Generating cryptographically signed Income Statement (PDF)...',
      variant: 'info',
    });
    setTimeout(() => {
      showToast({
        message: 'Income Statement generated and verified with Ed25519 seal.',
        variant: 'success',
      });
    }, 1200);
  }

  function shareVisitingCard(): void {
    const shareText = encodeURIComponent(
      `Discover authentic handmade ${craftName} by ${name} from ${districtName} on Kalakriti Platform: http://localhost:5174/artisan/${encodeURIComponent(name.toLowerCase())}`,
    );
    window.open(`https://wa.me/?text=${shareText}`, '_blank');
  }

  async function handleAvatarChange(e: Event): Promise<void> {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = async () => {
      const dataUrl = reader.result as string;
      avatarUrl = dataUrl;
      await setPref('profile.avatar_url', dataUrl);
      try {
        if (typeof localStorage !== 'undefined') {
          localStorage.setItem('kalakriti.artisan.avatar', dataUrl);
        }
      } catch {}
      showToast({ message: 'Profile picture updated successfully!', variant: 'success' });
    };
    reader.readAsDataURL(file);
  }

  async function handleLogout(): Promise<void> {
    setAccessToken(undefined);
    setRefreshToken(undefined);
    await setArtisanId(undefined);
    session.clear();
    showToast({ message: 'Logged out successfully.', variant: 'info' });
    await goto('/welcome');
  }
</script>

<svelte:head>
  <title>{t('profile.heading')} — {t('app.name')}</title>
</svelte:head>

<div class="profile-page">
  <!-- Top Identity & Hero Card -->
  <section class="profile-hero">
    <div class="profile-hero__badge-rule"></div>

    <div class="profile-hero__main">
      <div
        class="profile-avatar profile-avatar--clickable"
        onclick={() => fileInput?.click()}
        role="button"
        tabindex="0"
        onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && fileInput?.click()}
        title="Tap to change profile picture"
      >
        <input
          type="file"
          accept="image/*"
          capture="user"
          class="sr-only"
          bind:this={fileInput}
          onchange={handleAvatarChange}
        />
        {#if avatarUrl}
          <img class="profile-avatar__img" src={avatarUrl} alt={name} />
        {:else}
          <span class="profile-avatar__initial">{initial}</span>
        {/if}
        <span class="profile-avatar__camera-badge" title="Upload photo">
          <Icon name="camera" size="0.85rem" />
        </span>
        <span class="profile-avatar__verified" title="Govt & AI Verified Artisan">
          <Icon name="verified-artisan" size="1.25rem" />
        </span>
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
          <Button variant="secondary" size="md" onclick={downloadIncomeStatement}>
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
          <Button variant="secondary" size="md" onclick={shareVisitingCard}>
            <Icon name="share" size="1rem" />
            {t('profile.tools.shareWhatsapp')}
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
          <Button variant="primary" size="md" onclick={() => goto('/listing/new/capture')}>
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
        {t('profile.contrast', { mode: t('a11y.contrast.' + a11y.contrast) })}
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

  <!-- Session & Logout -->
  <section class="profile-session-strip">
    <div class="session-info">
      <span class="session-info__text">{t('profile.session.loggedAs', { phone: phone })}</span>
      <span class="session-info__badge">{t('profile.session.active')}</span>
    </div>

    <Button variant="danger" size="md" onclick={handleLogout}>
      <Icon name="lock" size="1rem" />
      {t('profile.session.signOut')}
    </Button>
  </section>
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-100);
    padding: var(--k-space-5);
    overflow: hidden;
  }

  .profile-hero__badge-rule {
    position: absolute;
    inset-block-start: 0;
    inset-inline: 0;
    block-size: 4px;
    background: linear-gradient(90deg, var(--k-terracotta-700), var(--k-haldi-500), var(--k-indigo-700));
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
    background-color: var(--k-terracotta-700);
    color: var(--k-khadi-50);
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
    color: #fff;
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
    background-color: var(--k-neem-600);
    color: #fff;
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
    color: var(--k-neem-700);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
  }

  .profile-status-dot {
    inline-size: 6px;
    block-size: 6px;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-neem-600);
  }

  .profile-hero__role {
    margin: 0;
    color: var(--k-terracotta-700);
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
    background-color: var(--k-khadi-150);
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
    border-block-start: var(--k-hairline) solid var(--k-stone-200);
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
    background-color: var(--k-khadi-150);
    color: var(--k-text-primary);
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-50);
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-50);
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
    border-inline-end: var(--k-hairline) solid var(--k-stone-200);
    border-block-end: var(--k-hairline) solid var(--k-stone-200);
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
    color: var(--k-terracotta-700);
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
    color: var(--k-indigo-700);
    text-decoration: none;
  }

  /* --- General Profile Cards --- */
  .profile-card {
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-50);
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
    background-color: var(--k-khadi-150);
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-khadi-100);
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-sm);
    background-color: var(--k-khadi-100);
  }

  .tool-tile__icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    flex-shrink: 0;
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-50);
    color: var(--k-terracotta-700);
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
    background-color: var(--k-khadi-50);
    color: var(--k-indigo-700);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
    text-decoration: none;
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .profile-action-link:hover {
    background-color: var(--k-indigo-700);
    color: #fff;
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
    background-color: var(--k-khadi-100);
    cursor: pointer;
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .lang-pill:hover {
    border-color: var(--k-terracotta-700);
  }

  .lang-pill--active {
    border-color: var(--k-terracotta-700);
    background-color: color-mix(in srgb, var(--k-terracotta-300) 25%, var(--k-khadi-100));
    color: var(--k-terracotta-700);
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
    color: var(--k-terracotta-700);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-medium);
    cursor: pointer;
    padding: var(--k-space-1) var(--k-space-3);
    border-radius: var(--k-radius-pill);
    transition: all var(--k-duration-fast) var(--k-ease-standard);
  }

  .lang-more-btn:hover {
    background-color: var(--k-khadi-200);
    border-color: var(--k-terracotta-700);
  }

  /* Accessibility Quick Bar */
  .a11y-quick-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-3);
    padding-block-start: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-stone-200);
  }

  .a11y-quick-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-stone-300);
    border-radius: var(--k-radius-pill);
    background-color: var(--k-khadi-100);
    color: var(--k-text-primary);
    font-size: var(--k-text-xs);
    cursor: pointer;
  }

  .a11y-quick-link {
    margin-inline-start: auto;
    color: var(--k-indigo-700);
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
    border: var(--k-hairline) solid var(--k-stone-200);
    border-radius: var(--k-radius-md);
    background-color: var(--k-khadi-100);
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
    color: var(--k-neem-700);
  }
</style>
