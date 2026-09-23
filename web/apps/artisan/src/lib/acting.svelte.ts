// apps/artisan/src/lib/acting.svelte.ts
//
// Assisted mode (F14): which artisan a signed-in field agent is helping.
// The header itself is added by @kalakriti/api's call() (only on the routes
// the bff honours it on); this module is the app-side state the layout's
// banner and route guard read. sessionStorage, not IndexedDB: helping
// someone is a per-visit act, and a closed tab should end it.

import { setActingFor } from '@kalakriti/api';

const KEY = 'kalakriti.actingFor';

export interface ActingFor {
  id: string;
  name: string;
}

function restore(): ActingFor | null {
  try {
    const raw = sessionStorage.getItem(KEY);
    const v = raw ? (JSON.parse(raw) as ActingFor) : null;
    if (v?.id) setActingFor(v.id);
    return v?.id ? v : null;
  } catch {
    return null;
  }
}

class Acting {
  current = $state<ActingFor | null>(typeof sessionStorage === 'undefined' ? null : restore());

  start(who: ActingFor): void {
    this.current = who;
    setActingFor(who.id);
    try {
      sessionStorage.setItem(KEY, JSON.stringify(who));
    } catch {}
  }

  stop(): void {
    this.current = null;
    setActingFor(undefined);
    try {
      sessionStorage.removeItem(KEY);
    } catch {}
  }
}

export const acting = new Acting();
