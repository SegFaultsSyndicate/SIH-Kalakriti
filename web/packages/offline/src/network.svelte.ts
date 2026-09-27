// packages/offline/src/network.svelte.ts
//
// Connectivity, as a rune.
//
// navigator.onLine reports network-interface state, not whether the internet
// or app API can be reached. Keep its instant offline event and verify both
// with tiny health responses while the page is visible.

const isBrowser = typeof window !== 'undefined';
const REACHABILITY_INTERVAL_MS = 15_000;
const REACHABILITY_TIMEOUT_MS = 5_000;
const INTERNET_CHECK_URL = 'https://connectivitycheck.gstatic.com/generate_204';

export class NetworkStatus {
  online = $state(isBrowser ? navigator.onLine : true);
  #startCount = 0;
  #stop: (() => void) | undefined;
  #probe: AbortController | undefined;
  #probeTimeout: number | undefined;

  /** Subscribe to connectivity changes. Returns a disposer. */
  start(): () => void {
    if (!isBrowser) return () => {};

    this.#startCount++;
    if (this.#startCount === 1) {
      const update = () => {
        if (!navigator.onLine) {
          this.online = false;
          this.#probe?.abort();
          return;
        }
        void this.#checkReachability();
      };
      const onVisibilityChange = () => {
        if (document.visibilityState === 'visible') void this.#checkReachability();
      };
      const interval = window.setInterval(() => {
        if (document.visibilityState === 'visible') void this.#checkReachability();
      }, REACHABILITY_INTERVAL_MS);

      window.addEventListener('online', update);
      window.addEventListener('offline', update);
      document.addEventListener('visibilitychange', onVisibilityChange);
      this.#stop = () => {
        window.removeEventListener('online', update);
        window.removeEventListener('offline', update);
        document.removeEventListener('visibilitychange', onVisibilityChange);
        window.clearInterval(interval);
        this.#probe?.abort();
        this.#probe = undefined;
        if (this.#probeTimeout !== undefined) window.clearTimeout(this.#probeTimeout);
        this.#probeTimeout = undefined;
        this.#stop = undefined;
      };

      update();
    }

    let stopped = false;
    return () => {
      if (stopped) return;
      stopped = true;
      this.#startCount--;
      if (this.#startCount === 0) this.#stop?.();
    };
  }

  async #checkReachability(): Promise<void> {
    if (!navigator.onLine || this.#probe) return;

    const controller = new AbortController();
    this.#probe = controller;
    const timeout = window.setTimeout(() => controller.abort(), REACHABILITY_TIMEOUT_MS);
    this.#probeTimeout = timeout;
    try {
      let appReachable = false;
      let internetReachable = false;

      try {
        const appResponse = await fetch('/healthz?_=' + Date.now(), {
          cache: 'no-store',
          signal: controller.signal,
        });
        appReachable = appResponse.ok || appResponse.status === 404;
      } catch {
        appReachable = false;
      }

      try {
        const internetResponse = await fetch(INTERNET_CHECK_URL + '?_=' + Date.now(), {
          cache: 'no-store',
          mode: 'no-cors',
          signal: controller.signal,
        });
        internetReachable = internetResponse.ok || internetResponse.type === 'opaque';
      } catch {
        internetReachable = false;
      }

      if (this.#probe === controller) {
        // When running locally, the app server (localhost) is always reachable even if the user 
        // turns off their Wi-Fi to test offline mode. Require internet reachability to accurately 
        // reflect offline testing state.
        const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
        this.online = navigator.onLine && (isLocalhost ? internetReachable : (appReachable || internetReachable));
      }
    } catch {
      if (this.#probe === controller) this.online = false;
    } finally {
      if (this.#probeTimeout === timeout) {
        window.clearTimeout(timeout);
        this.#probeTimeout = undefined;
      }
      if (this.#probe === controller) this.#probe = undefined;
    }
  }
}

/** One per app; connectivity is a property of the device, not of a screen. */
export const network = new NetworkStatus();
