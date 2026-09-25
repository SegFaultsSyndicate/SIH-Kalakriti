<!--
  apps/artisan/src/lib/BottomNav.svelte

  Persistent bottom navigation: Home, My Work, Orders, Profile. Icon plus
  label, always -- never icon-only, per the batch spec and the low-literacy
  audience an icon alone cannot serve. 56px targets exceed the 44px
  system-wide minimum on purpose: this is the artisan's primary means of
  moving around the app, tapped far more than any other control.

  Mounted only once the artisan is authenticated and registered -- see
  +layout.svelte's guard -- so it never appears on /language, /login, /verify
  or a /register/* step.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { Icon, type IconName } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const ITEMS: readonly { href: string; icon: IconName; labelKey: Parameters<typeof t>[0] }[] = [
    { href: '/', icon: 'home', labelKey: 'nav.home' },
    { href: '/listings', icon: 'edit', labelKey: 'nav.listings' },
    { href: '/orders', icon: 'collective-order', labelKey: 'nav.orders' },
    { href: '/profile', icon: 'user', labelKey: 'nav.profile' },
  ];

  const current = $derived(page.url.pathname);
</script>

<nav class="bottom-nav" aria-label={t('nav.home')}>
  {#each ITEMS as item (item.href)}
    <a
      href={item.href}
      class="bottom-nav__item"
      aria-current={current === item.href ? 'page' : undefined}
      data-sveltekit-preload-code="eager"
      data-sveltekit-preload-data="tap"
    >
      <Icon name={item.icon} class="bottom-nav__icon" />
      <span class="bottom-nav__label">{t(item.labelKey)}</span>
    </a>
  {/each}
</nav>

<style>
  .bottom-nav {
    position: fixed;
    inset-inline: 0;
    inset-block-end: 0;
    z-index: var(--k-z-sticky);
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface-raised);
    max-inline-size: 100%;
    overflow-x: hidden;
  }

  .bottom-nav__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-1);
    min-block-size: calc(var(--k-touch-min) * 1.27); /* 56px at the base 44px scale */
    padding-block: var(--k-space-2);
    color: var(--k-text-secondary);
    text-decoration: none;
    font-size: var(--k-text-xs);
    transition: color var(--k-duration-fast, 150ms) var(--k-ease-standard),
                background-color var(--k-duration-fast, 150ms) var(--k-ease-standard);
    -webkit-tap-highlight-color: transparent;
    user-select: none;
  }

  .bottom-nav__item:active {
    background-color: var(--k-surface-pressed);
  }

  .bottom-nav__item[aria-current='page'] {
    color: var(--k-accent-primary-text);
    font-weight: var(--k-weight-semibold);
    border-block-start: var(--k-rule) solid var(--k-accent-primary-bg);
    margin-block-start: calc(-1 * var(--k-hairline));
  }
</style>
