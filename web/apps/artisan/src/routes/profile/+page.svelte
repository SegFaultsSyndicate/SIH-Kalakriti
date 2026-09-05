<!--
  apps/artisan/src/routes/profile/+page.svelte

  Profile bottom-nav target. Shows what was entered at registration -- there
  is no GET /artisans/me in services/bff/openapi.json to fetch a canonical
  copy back from, so the local draft this app already wrote to Dexie during
  registration is the only source of truth available, same reasoning as
  registration.ts's buildRegisterBody.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { getFollowerCount } from '@kalakriti/api';
  import { getDraft, getArtisanId } from '$lib/registration';
  import { network } from '$lib/orders';

  const t = $derived(locale.t);

  let name = $state('');
  let followerCount = $state<number | undefined>(undefined);

  $effect(() => {
    void getDraft().then((draft) => {
      name = draft.name ?? '';
    });
  });

  $effect(() => {
    void (async () => {
      if (!network.online) return;
      const artisanId = await getArtisanId();
      if (!artisanId || artisanId.startsWith('local:')) return;
      try {
        const res = await getFollowerCount(artisanId);
        followerCount = res.count;
      } catch {
        /* recognition, not critical -- a failed fetch just shows nothing. */
      }
    })();
  });
</script>

<svelte:head>
  <title>{t('profile.heading')} — {t('app.name')}</title>
</svelte:head>

<h1>{t('profile.heading')}</h1>
{#if name}
  <p class="profile__name">{name}</p>
{/if}

{#if followerCount !== undefined && followerCount > 0}
  <p class="profile__followers">
    <Icon name="users" />
    {t('profile.followers', { count: String(followerCount) })}
  </p>
{/if}

<a class="profile__link" href="/accessibility">{t('a11y.statement.linkLabel')}</a>

<style>
  .profile__name {
    margin-block-start: var(--k-space-3);
    font-size: var(--k-text-lg);
  }

  .profile__followers {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    margin-block-start: var(--k-space-2);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .profile__link {
    display: inline-block;
    margin-block-start: var(--k-space-5);
    color: var(--k-accent-secondary);
  }
</style>
