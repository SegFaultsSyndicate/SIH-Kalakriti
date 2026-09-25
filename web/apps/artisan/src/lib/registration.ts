// apps/artisan/src/lib/registration.ts
//
// The registration wizard's Dexie-backed draft, and its submission through
// the outbox rather than a direct call -- same offline-first path every
// other write in this app takes (see @kalakriti/offline's optimistic.ts).

import { liveQuery } from 'dexie';
import { db, getPref, setPref, enqueue, applyOptimistic } from '@kalakriti/offline';
import { LOCALES } from '@kalakriti/i18n';
import { DISTRICTS, STATE_CODES } from './ontology';

const DRAFT_PREF_KEY = 'registration.draft';
const ARTISAN_ID_PREF_KEY = 'registration.artisanId';

export interface RegistrationDraft {
  name?: string;
  /** Real craft ontology UUID from GET /crafts, per $lib/ontology's loadCrafts(). */
  craftId?: string;
  /** Display name captured alongside craftId, so profile display never needs a network round-trip. */
  craftName?: string;
  districtId?: string;
  districtFreeText?: string;
  /** State name for a free-text district, from $lib/ontology's STATES -- unused when districtId is set (state is derived from it instead). */
  stateFreeText?: string;
  pehchanId?: string;
  clusterName?: string;
  socialCategory?: string;
  /** F13 income baseline: income_bracket enum value (pkg/impact's Bracket*). */
  incomeBracket?: string;
}

const STORAGE_DRAFT_KEY = 'kalakriti.registration.draft';

export function getCachedDraftSync(): RegistrationDraft {
  try {
    if (typeof localStorage !== 'undefined') {
      const raw = localStorage.getItem(STORAGE_DRAFT_KEY);
      if (raw) return JSON.parse(raw);
    }
  } catch {}
  return {};
}

export async function getDraft(): Promise<RegistrationDraft> {
  const pref = await getPref<RegistrationDraft>(DRAFT_PREF_KEY);
  if (pref !== undefined) {
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_DRAFT_KEY, JSON.stringify(pref));
      }
    } catch {}
    return pref;
  }
  return getCachedDraftSync();
}

/** Merges into the stored draft immediately -- "every answer persists to Dexie immediately" is this call, made from each wizard step's onchange. */
export async function patchDraft(patch: Partial<RegistrationDraft>): Promise<void> {
  const current = await getDraft();
  const next = { ...current, ...patch };
  await setPref(DRAFT_PREF_KEY, next);
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_DRAFT_KEY, JSON.stringify(next));
    }
  } catch {}
}

const STORAGE_ARTISAN_ID_KEY = 'kalakriti.artisan.id';

export async function getArtisanId(): Promise<string | undefined> {
  const pref = await getPref<string>(ARTISAN_ID_PREF_KEY);
  if (pref !== undefined) return pref;
  try {
    if (typeof localStorage !== 'undefined') {
      return localStorage.getItem(STORAGE_ARTISAN_ID_KEY) ?? undefined;
    }
  } catch {}
  return undefined;
}

export async function setArtisanId(id: string | undefined): Promise<void> {
  await setPref(ARTISAN_ID_PREF_KEY, id);
  try {
    if (typeof localStorage !== 'undefined') {
      if (id === undefined) localStorage.removeItem(STORAGE_ARTISAN_ID_KEY);
      else localStorage.setItem(STORAGE_ARTISAN_ID_KEY, id);
    }
  } catch {}
}

/** Live updates to the artisan id -- the root layout's route guard watches this so a registration completing (or, on terminal failure, rolling back) is reflected without a reload. */
export function watchArtisanId(onChange: (id: string | undefined) => void): () => void {
  const sub = liveQuery(() => db.prefs.get(ARTISAN_ID_PREF_KEY)).subscribe((row) => {
    onChange(row?.value as string | undefined);
  });
  return () => sub.unsubscribe();
}

export interface RegisterBody {
  display_name: string;
  craft_ids: string[];
  languages: string[];
  region: { state_code: string; district?: string };
  cluster_id?: string;
  pehchan_id?: string;
  social_category?: string;
}

/**
 * commonv1.Language's enum names are the bare English name uppercased
 * (LANGUAGE_HINDI, LANGUAGE_ENGLISH, ...) -- see
 * services/bff/internal/bff/client/search.go's languageToProto, which the
 * bff's Register call runs every entry of `languages` through. LOCALES'
 * englishName already matches that convention for every locale this app
 * ships; this only needs the uppercase.
 */
function toProtoLanguageName(code: string): string {
  const meta = (LOCALES as Record<string, { englishName: string }>)[code];
  return (meta?.englishName ?? 'English').toUpperCase();
}

/**
 * Builds POST /artisans' real request body (services/bff/openapi.json,
 * regenerated from the actual handler requirements -- core-svc rejects a
 * register call with no craft_ids or no region.state_code). craft/district
 * selection lives in the local draft the whole wizard writes to; this is the
 * one place that draft shape is turned into the wire shape, so every caller
 * goes through it rather than building its own request body.
 */
export function buildRegisterBody(draft: RegistrationDraft, language: string): RegisterBody {
  const district = draft.districtId ? DISTRICTS.find((d) => d.id === draft.districtId) : undefined;
  const stateName = district?.state ?? draft.stateFreeText;
  const body: RegisterBody = {
    display_name: draft.name?.trim() || '',
    craft_ids: draft.craftId ? [draft.craftId] : [],
    languages: [toProtoLanguageName(language)],
    region: {
      state_code: (stateName && STATE_CODES[stateName]) || '',
      district: district?.name ?? draft.districtFreeText,
    },
  };
  if (draft.pehchanId) body.pehchan_id = draft.pehchanId;
  if (draft.socialCategory) body.social_category = draft.socialCategory;
  return body;
}

/**
 * Submits through the outbox: a local placeholder artisan id is assigned and
 * durably written *before* this resolves -- the caller navigates to `/`
 * immediately after, and the root layout's guard reads that id via
 * watchArtisanId to decide the artisan is registered, so the write must have
 * landed first or the guard bounces them straight back to /register. The
 * real artisan_id (or a rollback to unregistered) lands when the queued
 * entry eventually drains -- see $lib/outbox-send.ts.
 */
export async function submitRegistration(
  draft: RegistrationDraft,
  language: string,
): Promise<void> {
  const body = buildRegisterBody(draft, language);
  const localId = `local:${crypto.randomUUID()}`;
  await setArtisanId(localId);

  await applyOptimistic({
    apply: () => {},
    rollback: () => {
      void setArtisanId(undefined);
    },
    enqueue: async () => {
      const profile = await enqueue({ kind: 'profile.update', payload: body });
      // The baseline needs the artisan-scoped token registration returns, so
      // it waits on the profile entry rather than racing it.
      if (draft.incomeBracket) {
        await enqueue({
          kind: 'income.baseline',
          payload: { monthly_bracket: draft.incomeBracket },
          dependsOn: [profile.id],
        });
      }
      return profile;
    },
  });
}

