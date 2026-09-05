// packages/offline/src/network.svelte.ts
//
// Connectivity, as a rune.
//
// navigator.onLine is famously weak -- it reports "online" for a phone that is
// associated with a tower and getting nothing through, which is the normal
// state of a 2G connection in the field. It is still worth reading, because
// the offline transition it *does* report is instant and free. Anything
// stronger (a reachability probe against the BFF) belongs with the sync
// engine, which is already making requests and can observe their outcome
// rather than spending an extra one.

const isBrowser = typeof window !== 'undefined';

export class NetworkStatus {
  online = $state(isBrowser ? navigator.onLine : true);

  /** Subscribe to connectivity changes. Returns a disposer. */
  start(): () => void {
    if (!isBrowser) return () => {};
    const update = () => {
      this.online = navigator.onLine;
    };
    window.addEventListener('online', update);
    window.addEventListener('offline', update);
    update();
    return () => {
      window.removeEventListener('online', update);
      window.removeEventListener('offline', update);
    };
  }
}

/** One per app; connectivity is a property of the device, not of a screen. */
export const network = new NetworkStatus();
