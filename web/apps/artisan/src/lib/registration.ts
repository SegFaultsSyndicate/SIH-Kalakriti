// apps/artisan/src/lib/registration.ts
//
// The registration wizard's Dexie-backed draft, and its submission through
// the outbox rather than a direct call -- same offline-first path every
// other write in this app takes (see @kalakriti/offline's optimistic.ts).

import { liveQuery } from 'dexie';
import { db, getPref, setPref, enqueue, applyOptimistic } from '@kalakriti/offline';
import { LOCALES, isLocaleCode } from '@kalakriti/i18n';
import { DISTRICTS } from './ontology';

const DRAFT_PREF_KEY = 'registration.draft';
const ARTISAN_ID_PREF_KEY = 'registration.artisanId';

export interface RegistrationDraft {
  name?: string;
  craftId?: string;
  districtId?: string;
  districtFreeText?: string;
  /** State picked alongside a free-text district -- region.state_code has no other source when districtId is unset. */
  districtStateCode?: string;
  pehchanId?: string;
  clusterName?: string;
}

export async function getDraft(): Promise<RegistrationDraft> {
  return (await getPref<RegistrationDraft>(DRAFT_PREF_KEY)) ?? {};
}

/** Merges into the stored draft immediately -- "every answer persists to Dexie immediately" is this call, made from each wizard step's onchange. */
export async function patchDraft(patch: Partial<RegistrationDraft>): Promise<void> {
  const current = await getDraft();
  await setPref(DRAFT_PREF_KEY, { ...current, ...patch });
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

/**
 * languageToProto (services/bff/internal/bff/client/search.go) matches
 * against LANGUAGE_<NAME>, so the wire form is the locale's English name
 * uppercased ('HINDI', not 'hi'). Every code in @kalakriti/i18n's LOCALES
 * has an englishName that matches one of proto's 23 named Language values
 * exactly (checked against common.proto), so this never actually drops a
 * language for any locale the app offers -- the undefined fallback only
 * guards a code isLocaleCode would already have rejected. core-svc requires
 * at least one language, so `language` here must always be a real locale
 * code (see submitRegistration's caller, which passes locale.code).
 */
function languageWireName(localeCode: string): string | undefined {
  return isLocaleCode(localeCode) ? LOCALES[localeCode].englishName.toUpperCase() : undefined;
}

/**
 * Builds the POST /artisans body. The PM Vishwakarma/Pehchan ID and
 * cluster/SHG membership still stay local only -- there is no field for
 * them on this endpoint and no PATCH /artisans/me to send them to later --
 * but craft and region now travel with the request: core-svc's
 * Artisan.Register requires at least one craft_id and a region.state_code,
 * so a registration that omitted them was never actually going to persist
 * server-side. Per the ABSOLUTE RULE nothing here is invented beyond what
 * services/bff/internal/bff/client/artisan.go actually reads.
 */
export function buildRegisterBody(
  draft: RegistrationDraft,
  language: string,
): {
  display_name: string;
  craft_ids: string[];
  languages: string[];
  region: { state_code: string; district?: string };
} {
  const knownDistrict = draft.districtId ? DISTRICTS.find((d) => d.id === draft.districtId) : undefined;
  const wireLanguage = languageWireName(language);
  const districtName = knownDistrict?.name ?? draft.districtFreeText;
  return {
    display_name: draft.name?.trim() || '',
    craft_ids: draft.craftId ? [draft.craftId] : [],
    languages: wireLanguage ? [wireLanguage] : [],
    region: {
      state_code: knownDistrict?.stateCode ?? draft.districtStateCode ?? '',
      ...(districtName ? { district: districtName } : {}),
    },
  };
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
    enqueue: () => enqueue({ kind: 'profile.update', payload: body }),
  });
}
