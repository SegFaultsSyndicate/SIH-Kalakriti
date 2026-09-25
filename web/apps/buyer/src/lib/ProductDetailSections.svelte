<!--
  apps/buyer/src/lib/ProductDetailSections.svelte

  The below-the-fold half of the product page, laid out after Amazon's
  product detail page: frequently bought together, product details table,
  more from this craft, customer reviews (average + 5..1 histogram + review
  cards), questions & answers, and customers also viewed.

  `social` (ratings/reviews/Q&A) is only ever passed for demo pieces --
  see demo-catalog.ts's demoSocial(). A real backend listing gets the
  structural sections and an honest "no reviews yet" state, never invented
  reviews.

  All copy is pdp.* catalogue keys.
-->
<script lang="ts">
  import { Money, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { locale, matchesLocale } from '@kalakriti/i18n';
  import type { components } from '@kalakriti/api';
  import ListingCard from './ListingCard.svelte';
  import { wishlist } from './wishlist.svelte';
  import type { DemoSocial } from './demo-catalog';

  type ListingSummary = components['schemas']['ListingSummary'];

  interface Props {
    listing: ListingSummary;
    details: { bullets: string[]; specs: [string, string][] };
    social?: DemoSocial;
    related: { sameCraft: ListingSummary[]; alsoViewed: ListingSummary[] };
  }

  let { listing, details, social, related }: Props = $props();

  const t = $derived(locale.t);

  function titleOf(l: ListingSummary): string {
    return (
      l.translations?.find((tr) => matchesLocale(tr.language, locale.code))?.title ??
      l.translations?.[0]?.title ??
      l.craft_name ??
      ''
    );
  }

  // Frequently bought together: this piece + the next two from the same craft
  // (or from "also viewed" when the craft has nothing else).
  // `bundleCompanions` are the 2 other picks shown alongside the current listing;
  // they are excluded from "More from this craft" so nothing repeats.
  const bundleCompanions = $derived(related.alsoViewed.filter((x) => x.id !== listing.id).slice(0, 2));
  const bundle = $derived([listing, ...bundleCompanions]);
  let bundleOn = $state<Record<string, boolean>>({});
  const bundleTotal = $derived(
    bundle.reduce((sum, l) => sum + (bundleOn[l.id ?? ''] === false ? 0 : (l.price?.amount_paise ?? 0)), 0),
  );
  const bundleCount = $derived(bundle.filter((l) => bundleOn[l.id ?? ''] !== false).length);
  // IDs already shown in FBT — excluded from the "More from craft" row below.
  const bundleIds = $derived(new Set(bundle.map((l) => l.id)));
  // sameCraft items with FBT picks removed so neither section repeats the other.
  const sameCraftFiltered = $derived(related.sameCraft.filter((l) => !bundleIds.has(l.id)));

  // ponytail: no cart exists yet -- "add all" saves to the wishlist; orders are
  // still placed per piece via PurchaseForm.
  function addBundle(): void {
    for (const l of bundle) {
      const id = l.id ?? '';
      if (bundleOn[id] !== false && !wishlist.has(id)) wishlist.toggle(id, titleOf(l));
    }
  }

  let helpfulVoted = $state<Record<number, boolean>>({});
  let question = $state('');

  function askQuestion(e: SubmitEvent): void {
    e.preventDefault();
    if (!question.trim()) return;
    showToast({ variant: 'success', message: t('pdp.qa.sent', { artisan: listing.artisan_name ?? t('pdp.theArtisan') }) });
    question = '';
  }

  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleDateString(locale.code, { day: 'numeric', month: 'long', year: 'numeric' });
  }
</script>

<div class="pdp-below">
  {#if bundleCompanions.length > 0}
    <section class="pdp-section" aria-labelledby="fbt-heading">
      <h2 id="fbt-heading">{t('pdp.fbt.heading')}</h2>
      <div class="fbt">
        <!-- Thumbnails: companion picks only (current listing is already the hero image above) -->
        <ul class="fbt__items">
          {#each bundleCompanions as item, i (item.id)}
            {#if i > 0}<li class="fbt__plus" aria-hidden="true">+</li>{/if}
            <li class="fbt__item">
              <a href="/listing/{item.id}"><img src={item.image_url} alt={titleOf(item)} loading="lazy" /></a>
            </li>
          {/each}
        </ul>
        <div class="fbt__summary">
          <p>{t('pdp.fbt.total')} <strong><Money paise={bundleTotal} /></strong></p>
          <button type="button" class="pdp-btn pdp-btn--primary" onclick={addBundle} disabled={bundleCount === 0}>
            {t('pdp.fbt.saveAll', { count: bundleCount })}
          </button>
        </div>
        <!-- Checklist: all 3 items including "This item" current listing -->
        <ul class="fbt__list">
          {#each bundle as item, i (item.id)}
            <li>
              <label>
                <input
                  type="checkbox"
                  checked={bundleOn[item.id ?? ''] !== false}
                  onchange={(e) => (bundleOn[item.id ?? ''] = e.currentTarget.checked)}
                />
                <span>{i === 0 ? `${t('pdp.fbt.thisItem')} ` : ''}{titleOf(item)}</span>
                <strong><Money paise={item.price?.amount_paise ?? 0} /></strong>
              </label>
            </li>
          {/each}
        </ul>
      </div>
    </section>
  {/if}

  <section class="pdp-section" aria-labelledby="details-heading">
    <h2 id="details-heading">{t('pdp.details.heading')}</h2>
    <table class="pdp-specs">
      <tbody>
        {#each details.specs as [k, v] (k)}
          <tr><th scope="row">{k}</th><td>{v}</td></tr>
        {/each}
      </tbody>
    </table>
  </section>

  {#if sameCraftFiltered.length > 0}
    <section class="pdp-section" aria-labelledby="same-craft-heading">
      <h2 id="same-craft-heading">{t('pdp.sameCraft.heading', { craft: listing.craft_name ?? '' })}</h2>
      <div class="pdp-row">
        {#each sameCraftFiltered as l (l.id)}
          <div class="pdp-row__cell"><ListingCard listing={l} href={`/listing/${l.id}`} /></div>
        {/each}
      </div>
    </section>
  {/if}

  <section class="pdp-section pdp-reviews" id="reviews" aria-labelledby="reviews-heading">
    <h2 id="reviews-heading">{t('pdp.reviews.heading')}</h2>
    {#if social}
      <div class="pdp-reviews__grid">
        <div class="pdp-reviews__summary">
          <p class="pdp-reviews__avg">
            <span class="pdp-stars pdp-stars--lg" style="--pct: {(social.rating / 5) * 100}%" role="img" aria-label={t('pdp.starsAria', { rating: social.rating })}></span>
            <span>{t('pdp.reviews.outOf5', { rating: social.rating.toFixed(1) })}</span>
          </p>
          <p class="pdp-reviews__count">{t('pdp.reviews.global', { count: social.ratingCount.toLocaleString(locale.code) })}</p>
          <ul class="pdp-histogram">
            {#each social.histogram as pct, i (i)}
              <li>
                <span>{t('pdp.reviews.star', { n: 5 - i })}</span>
                <span class="pdp-histogram__bar"><span style="inline-size: {pct}%"></span></span>
                <span>{pct}%</span>
              </li>
            {/each}
          </ul>
        </div>
        <ul class="pdp-reviews__list">
          {#each social.reviews as rv, i (i)}
            <li class="review">
              <p class="review__who"><span class="review__avatar" aria-hidden="true">{rv.name.charAt(0)}</span>{rv.name}</p>
              <p class="review__head">
                <span class="pdp-stars" style="--pct: {rv.rating * 20}%" role="img" aria-label={t('pdp.starsAria', { rating: rv.rating })}></span>
                <strong>{rv.title}</strong>
              </p>
              <p class="review__meta">{t('pdp.reviews.meta', { location: rv.location, date: fmtDate(rv.date) })}</p>
              <p class="review__verified"><Icon name="check" size="0.8rem" /> {t('pdp.reviews.verified')}</p>
              <p class="review__body">{rv.body}</p>
              <p class="review__helpful">
                <span>{t('pdp.reviews.helpfulCount', { count: rv.helpful + (helpfulVoted[i] ? 1 : 0) })}</span>
                <button type="button" class="pdp-btn" disabled={helpfulVoted[i]} onclick={() => (helpfulVoted[i] = true)}>
                  {helpfulVoted[i] ? t('pdp.reviews.thanks') : t('pdp.reviews.helpful')}
                </button>
              </p>
            </li>
          {/each}
        </ul>
      </div>
    {:else}
      <p class="pdp-empty">{t('pdp.reviews.empty')}</p>
    {/if}
  </section>

  <section class="pdp-section" aria-labelledby="qa-heading">
    <h2 id="qa-heading">{t('pdp.qa.heading')}</h2>
    {#if social}
      <dl class="pdp-qa">
        {#each social.qa as item (item.q)}
          <div>
            <dt><span>{t('pdp.qa.q')}</span>{item.q}</dt>
            <dd><span>{t('pdp.qa.a')}</span>{item.a} <small>{t('pdp.qa.by', { name: item.by })}</small></dd>
          </div>
        {/each}
      </dl>
    {/if}
    <form class="pdp-ask" onsubmit={askQuestion}>
      <label class="visually-hidden" for="pdp-question">{t('pdp.qa.askLabel')}</label>
      <input id="pdp-question" placeholder={t('pdp.qa.askPlaceholder')} bind:value={question} />
      <button type="submit" class="pdp-btn">{t('pdp.qa.ask')}</button>
    </form>
  </section>

  {#if related.alsoViewed.length > 0}
    <section class="pdp-section" aria-labelledby="also-viewed-heading">
      <h2 id="also-viewed-heading">{t('pdp.alsoViewed.heading')}</h2>
      <div class="pdp-row">
        {#each related.alsoViewed as l (l.id)}
          <div class="pdp-row__cell"><ListingCard listing={l} href={`/listing/${l.id}`} /></div>
        {/each}
      </div>
    </section>
  {/if}
</div>

<style>
  .pdp-below {
    display: grid;
    /* minmax(0, …) so the scrolling card rows can't stretch the column past the viewport. */
    grid-template-columns: minmax(0, 1fr);
    gap: var(--k-space-7, 2.5rem);
    margin-block-start: var(--k-space-7, 2.5rem);
  }

  .pdp-section {
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-5);
  }

  .pdp-section h2 {
    font-size: var(--k-text-lg);
    margin: 0 0 var(--k-space-4);
  }

  /* Frequently bought together */
  .fbt {
    display: grid;
    gap: var(--k-space-4);
  }

  .fbt__items {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    list-style: none;
    padding: 0;
    margin: 0;
    flex-wrap: wrap;
  }

  .fbt__item img {
    inline-size: 7rem;
    block-size: 7rem;
    object-fit: cover;
    border-radius: var(--k-radius-md);
    border: var(--k-hairline) solid var(--k-border-hairline);
  }

  .fbt__plus {
    font-size: var(--k-text-xl);
    color: var(--k-text-secondary);
  }

  .fbt__summary p {
    margin: 0 0 var(--k-space-2);
  }

  .fbt__list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
  }

  .fbt__list label {
    display: flex;
    gap: var(--k-space-2);
    align-items: baseline;
  }

  .fbt__list span {
    flex: 1;
  }

  @media (min-width: 60rem) {
    .fbt {
      grid-template-columns: auto 1fr;
      align-items: center;
    }

    .fbt__list {
      grid-column: 1 / -1;
    }
  }

  /* Specs table */
  .pdp-specs {
    inline-size: 100%;
    max-inline-size: 48rem;
    border-collapse: collapse;
    font-size: var(--k-text-sm);
  }

  .pdp-specs th,
  .pdp-specs td {
    padding: var(--k-space-2) var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    text-align: start;
    vertical-align: top;
  }

  .pdp-specs th {
    inline-size: 35%;
    background-color: var(--k-surface-sunken);
    font-weight: var(--k-weight-semibold);
  }

  /* Horizontal card rows */
  .pdp-row {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(12rem, 14rem);
    gap: var(--k-space-4);
    overflow-x: auto;
    padding-block-end: var(--k-space-2);
    scroll-snap-type: x mandatory;
  }

  .pdp-row__cell {
    scroll-snap-align: start;
  }

  /* Reviews */
  .pdp-reviews__grid {
    display: grid;
    gap: var(--k-space-6);
  }

  @media (min-width: 60rem) {
    .pdp-reviews__grid {
      grid-template-columns: 18rem 1fr;
    }
  }

  .pdp-reviews__avg {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-md);
    margin: 0;
  }

  .pdp-reviews__count {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .pdp-histogram {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
  }

  .pdp-histogram li {
    display: grid;
    grid-template-columns: 3.5rem 1fr 2.5rem;
    align-items: center;
    gap: var(--k-space-2);
  }

  .pdp-histogram__bar {
    block-size: 1rem;
    border-radius: var(--k-radius-sm);
    border: var(--k-hairline) solid var(--k-border-hairline);
    background-color: var(--k-surface-sunken);
    overflow: hidden;
  }

  .pdp-histogram__bar span {
    display: block;
    block-size: 100%;
    background-color: var(--k-accent-primary-bg);
  }

  .pdp-reviews__list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: var(--k-space-5);
  }

  .review p {
    margin: 0 0 var(--k-space-1);
  }

  .review__who {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
  }

  .review__avatar {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 1.75rem;
    block-size: 1.75rem;
    border-radius: var(--k-radius-pill);
    background-color: var(--k-surface-sunken);
    font-weight: var(--k-weight-bold);
  }

  .review__head {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }

  .review__meta,
  .review__helpful {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .review__verified {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    color: var(--k-accent-primary-text);
  }

  .review__body {
    line-height: 1.55;
  }

  .review__helpful {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-2);
  }

  /* Q&A */
  .pdp-qa {
    display: grid;
    gap: var(--k-space-4);
    margin: 0 0 var(--k-space-4);
  }

  .pdp-qa dt {
    font-weight: var(--k-weight-semibold);
  }

  .pdp-qa dd {
    margin: var(--k-space-1) 0 0;
    color: var(--k-text-secondary);
  }

  .pdp-qa span {
    display: inline-block;
    min-inline-size: 1.5rem;
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
  }

  .pdp-ask {
    display: flex;
    gap: var(--k-space-2);
    max-inline-size: 36rem;
  }

  .pdp-ask input {
    flex: 1;
    min-inline-size: 0;
    padding: var(--k-space-2) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    font: inherit;
    background: var(--k-surface-base);
    color: var(--k-text-primary);
  }

  .pdp-empty {
    color: var(--k-text-secondary);
  }

  /* Shared buttons */
  .pdp-btn {
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: var(--k-surface-raised);
    color: var(--k-text-primary);
    font: inherit;
    font-size: var(--k-text-sm);
    cursor: pointer;
  }

  .pdp-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .pdp-btn--primary {
    background: var(--k-accent-primary-bg);
    border-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    padding: var(--k-space-2) var(--k-space-4);
  }

  /* Stars: shared visual, painted with a gradient clip so any fraction renders. */
  :global(.pdp-stars) {
    --pct: 0%;
    display: inline-block;
    font-size: 1rem;
    line-height: 1;
    letter-spacing: 0.05em;
    background: linear-gradient(90deg, #e8a33a var(--pct), var(--k-border-hairline) var(--pct));
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }

  :global(.pdp-stars)::before {
    content: '★★★★★';
  }

  :global(.pdp-stars--lg) {
    font-size: 1.35rem;
  }
</style>
