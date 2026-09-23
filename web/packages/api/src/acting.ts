// packages/api/src/acting.ts
//
// Assisted mode (F14): a field agent acting for an artisan. call() adds
// X-On-Behalf-Of ONLY on the routes the bff honours it on -- the list below
// mirrors services/bff/internal/bff/mosje/onbehalf.go's onBehalfAllowed. Any
// other route carrying the header is a 403, so the agent's own calls
// (/staff/me, /assisted/*, /helpers, /admin/*, /impact/*) must go out
// without it, and do: they are simply not on this list.
//
// An outbox entry freezes its target at enqueue time and passes it as
// CallOptions.onBehalfOf, so a sale queued for artisan A never replays as
// artisan B (or as the agent) because the agent switched artisans before the
// phone came back online.

let actingFor: string | undefined;

/** The artisan the signed-in agent is currently helping, if any. */
export function getActingFor(): string | undefined {
  return actingFor;
}

export function setActingFor(artisanId: string | undefined): void {
  actingFor = artisanId || undefined;
}

const ALLOWED: ReadonlyArray<readonly [string, RegExp]> = [
  ['GET', /^\/artisans\/me$/],
  ['PATCH', /^\/artisans\/me$/],
  ['GET', /^\/listings$/],
  ['GET', /^\/listings\/[^/]+$/],
  ['GET', /^\/listings\/[^/]+\/attributes$/],
  ['POST', /^\/listings$/],
  ['PATCH', /^\/listings\/[^/]+$/],
  ['POST', /^\/listings\/[^/]+\/media$/],
  ['POST', /^\/listings\/[^/]+\/submit$/],
  ['POST', /^\/media\/upload-url$/],
  ['POST', /^\/media\/[^/]+\/confirm$/],
  ['POST', /^\/pricing\/advise$/],
  ['GET', /^\/badges\/me\/progress$/],
  ['GET', /^\/schemes\/match$/],
  ['POST', /^\/statements$/],
  ['GET', /^\/statements(\/[^/]+)?$/],
  ['GET', /^\/income\/(baseline|sales|summary)$/],
  ['PUT', /^\/income\/baseline$/],
  ['POST', /^\/income\/sales$/],
  ['DELETE', /^\/income\/sales\/[^/]+$/],
  ['GET', /^\/finance\/(links|coverage)$/],
  ['POST', /^\/finance\/links$/],
  ['PATCH', /^\/finance\/links\/[^/]+$/],
  ['GET', /^\/learn\/(lessons|certificate)$/],
  ['POST', /^\/learn\/lessons\/[^/]+\/progress$/],
  ['POST', /^\/learn\/certificate$/],
];

/** Whether the bff honours X-On-Behalf-Of on this route. */
export function onBehalfAllowed(method: string, path: string): boolean {
  const bare = path.split('?')[0];
  return ALLOWED.some(([m, re]) => m === method && re.test(bare));
}

/**
 * The X-On-Behalf-Of header for one call: `explicit` (from a frozen outbox
 * entry; null = explicitly nobody) wins over the live acting state.
 */
export function onBehalfHeader(method: string, path: string, explicit?: string | null): Record<string, string> {
  const target = explicit === undefined ? actingFor : (explicit ?? undefined);
  return target !== undefined && onBehalfAllowed(method, path) ? { 'X-On-Behalf-Of': target } : {};
}
