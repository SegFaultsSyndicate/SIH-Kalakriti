// apps/artisan/src/lib/registration.ts
//
// The registration wizard's Dexie-backed draft, and its submission through
// the outbox rather than a direct call -- same offline-first path every
// other write in this app takes (see @kalakriti/offline's optimistic.ts).

import { liveQuery } from 'dexie';
import { db, getPref, setPref, enqueue, applyOptimistic } from '@kalakriti/offline';

const DRAFT_PREF_KEY = 'registration.draft';
const ARTISAN_ID_PREF_KEY = 'registration.artisanId';

export interface RegistrationDraft {
  name?: string;
  craftId?: string;
  districtId?: string;
  districtFreeText?: string;
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

export async function getArtisanId(): Promise<string | undefined> {
  return getPref<string>(ARTISAN_ID_PREF_KEY);
}

export async function setArtisanId(id: string | undefined): Promise<void> {
  await setPref(ARTISAN_ID_PREF_KEY, id);
}

/** Live updates to the artisan id -- the root layout's route guard watches this so a registration completing (or, on terminal failure, rolling back) is reflected without a reload. */
export function watchArtisanId(onChange: (id: string | undefined) => void): () => void {
  const sub = liveQuery(() => db.prefs.get(ARTISAN_ID_PREF_KEY)).subscribe((row) => {
    onChange(row?.value as string | undefined);
  });
  return () => sub.unsubscribe();
}

/**
 * Only display_name and language reach POST /artisans -- craft, district,
 * the PM Vishwakarma/Pehchan ID and cluster/SHG membership stay in the local
 * draft only. services/bff/openapi.json's requestBody for this endpoint has
 * no field for any of the four, and there is no PATCH /artisans/me to send
 * them to later. Per the ABSOLUTE RULE they are not invented; this is the
 * one place that fact is enforced, so every caller goes through it rather
 * than building its own request body.
 */
export function buildRegisterBody(
  draft: RegistrationDraft,
  language: string,
): { display_name: string; language: string } {
  return { display_name: draft.name?.trim() || '', language };
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
