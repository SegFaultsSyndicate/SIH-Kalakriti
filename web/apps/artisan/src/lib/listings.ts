// apps/artisan/src/lib/listings.ts
//
// The artisan's own catalog: fetching, grouping, and the actions available
// once a listing already exists (batch 9), as distinct from $lib/listing-draft.ts
// (batch 8), which owns *creating* one through the offline wizard.
//
// Unlike the wizard, everything here is online-first: GET /listings, PATCH,
// and seal-provenance are called directly, not queued through the outbox.
// The wizard's offline guarantee is about capture never being blocked by a
// connection; managing an already-published catalog has no such requirement
// in this batch's brief, and routing it through the draft/outbox machinery
// would need a local draft for every remote listing, which most won't have.
// Screens using this module show a plain "you're offline" state instead
// (network.online, from @kalakriti/offline) rather than a spinner with
// nothing behind it.

import { getCached, setCached, network, getPref, setPref } from '@kalakriti/offline';
import { db } from '@kalakriti/offline';
import { listListings, getListing, updateListing, type components } from '@kalakriti/api';
import { matchesLocale, type LocaleCode, type MessageKey } from '@kalakriti/i18n';
import { getArtisanId } from './registration';
import { createDraft, patchFields } from './listing-draft';

export type Listing = components['schemas']['Listing'];
export type ListingState = NonNullable<Listing['state']>;

const CACHE_KEY = 'listings:mine';

export const STATE_GROUPS = ['needsAttention', 'draft', 'pending', 'published', 'suspended'] as const;
export type StateGroup = (typeof STATE_GROUPS)[number];

export const STATE_GROUP_LABEL: Record<StateGroup, MessageKey> = {
  needsAttention: 'listings.group.needsAttention',
  draft: 'listings.group.draft',
  pending: 'listings.group.pending',
  published: 'listings.group.published',
  suspended: 'listings.group.suspended',
};

/**
 * GET /listings never returns translations-empty-but-DRAFT any differently
 * than a fully-worded draft -- there's no server-side "needs attention" flag
 * (core-svc has NeedsDescription on the domain type, but it isn't in the
 * Listing schema this BFF returns; see ml_wiring.md). A DRAFT with no
 * translations at all is the one case this module can honestly compute
 * itself: it has nothing an artisan or buyer could read.
 */
export function groupFor(listing: Listing): StateGroup {
  if (listing.state === 'DRAFT' && (listing.translations?.length ?? 0) === 0) return 'needsAttention';
  switch (listing.state) {
    case 'DRAFT':
      return 'draft';
    case 'PENDING_ARTISAN_APPROVAL':
      return 'pending';
    case 'PUBLISHED':
      return 'published';
    case 'SUSPENDED':
      return 'suspended';
    default:
      return 'draft';
  }
}

/** Picks the artisan's active-language title, falling back to whichever translation exists. */
export function titleFor(listing: Listing, locale: LocaleCode): string {
  const translations = listing.translations ?? [];
  return (
    translations.find((t) => matchesLocale(t.language, locale))?.title ??
    translations[0]?.title ??
    ''
  );
}

const HAS_PUBLISHED_PREF = 'listings:hasPublishedOnce';

/** Whether this artisan has ever had a listing reach PUBLISHED, on this device -- gates the PWA install prompt (see $lib/InstallPrompt.svelte): never on first load, only after there is something real to keep coming back for. */
export async function hasPublishedOnce(): Promise<boolean> {
  return (await getPref<boolean>(HAS_PUBLISHED_PREF)) ?? false;
}

/** Fetches the artisan's own listings across every state and caches the result for offline re-display. */
export async function fetchMyListings(): Promise<Listing[]> {
  const artisanId = await getArtisanId();
  if (!artisanId || artisanId.startsWith('local:')) return [];
  const res = await listListings({ artisan_id: artisanId });
  const listings = res.listings ?? [];
  await setCached(CACHE_KEY, listings);
  if (listings.some((l) => l.state === 'PUBLISHED')) {
    await setPref(HAS_PUBLISHED_PREF, true);
  }
  return listings;
}

/** Last successful fetch, for an offline first paint -- real data, just not fresh. */
export async function cachedListings(): Promise<Listing[]> {
  return (await getCached<Listing[]>(CACHE_KEY)) ?? [];
}

/**
 * A listing's primary photo, when this device happens to know one: the
 * local draft it was created from, if this is the device that created it.
 * GET /listings has no media field at all (proto gap, see ml_wiring.md) --
 * there is no server path to a thumbnail for a listing from another device.
 */
export async function localPrimaryImageUrl(listingId: string): Promise<string | undefined> {
  const draft = await db.drafts.where('remoteId').equals(listingId).first();
  const photoId = draft?.mediaIds?.[0];
  if (!photoId) return undefined;
  const media = await db.media.get(photoId);
  return media ? URL.createObjectURL(media.blob) : undefined;
}

/** The made_to_order_terms this device last knows for a listing, if it created it -- see pauseSelling's own note on why this matters. */
async function localMadeToOrderTerms(listingId: string): Promise<Record<string, unknown> | undefined> {
  const draft = await db.drafts.where('remoteId').equals(listingId).first();
  const fields = draft?.fields as { madeToOrderTerms?: Record<string, unknown> } | undefined;
  return fields?.madeToOrderTerms;
}

/**
 * Pauses selling. This is deliberately NOT the SUSPENDED state -- that is an
 * operator-only action core-svc reserves for cluster officers and the
 * ministry ("an artisan who wants to stop selling pauses orders instead",
 * service/catalog.go). READY_STOCK pauses by zeroing stock, a standalone
 * field safe to PATCH alone. MADE_TO_ORDER pauses via accepting_orders, but
 * that lives inside made_to_order_terms, which PATCH replaces wholesale --
 * sending it without lead_time_days/capacity_per_month would silently zero
 * them. Only offered for MADE_TO_ORDER when this device has those cached
 * locally (see canPauseOrResume); canPauseOrResume gates the UI.
 */
export async function canPauseOrResume(listing: Listing): Promise<boolean> {
  if (listing.type === 'READY_STOCK') return true;
  return (await localMadeToOrderTerms(listing.id!)) !== undefined;
}

export async function pauseSelling(listing: Listing): Promise<void> {
  if (listing.type === 'READY_STOCK') {
    await updateListing(listing.id!, { stock_quantity: 0 });
    return;
  }
  const terms = await localMadeToOrderTerms(listing.id!);
  if (!terms) throw new Error('made_to_order_terms unknown on this device');
  await updateListing(listing.id!, { made_to_order_terms: { ...terms, accepting_orders: false } });
}

export async function resumeSelling(listing: Listing, readyStockQuantity?: number): Promise<void> {
  if (listing.type === 'READY_STOCK') {
    await updateListing(listing.id!, { stock_quantity: readyStockQuantity ?? 0 });
    return;
  }
  const terms = await localMadeToOrderTerms(listing.id!);
  if (!terms) throw new Error('made_to_order_terms unknown on this device');
  await updateListing(listing.id!, { made_to_order_terms: { ...terms, accepting_orders: true } });
}

/**
 * Starts a new local draft pre-filled with whatever a fetched Listing
 * actually carries -- type, price, quantities, and the existing translations
 * as a starting point. Craft and photos cannot be copied: Listing has
 * neither a craft_id nor any media field (see ml_wiring.md), so the new
 * draft still needs both re-entered through the normal wizard steps.
 */
export async function duplicateAsDraft(listing: Listing): Promise<string> {
  const draft = await createDraft();
  await patchFields(draft.id, {
    type: listing.type,
    priceAmountPaise: listing.price?.amount_paise,
    stockQuantity: listing.stock_quantity,
    minOrderQuantity: listing.min_order_quantity,
    workingTitle: listing.translations?.[0]?.title,
  });
  return draft.id;
}

/** Fetches one listing fresh, bypassing whatever is cached. */
export async function fetchListing(id: string): Promise<Listing> {
  return getListing(id);
}

export { network };
