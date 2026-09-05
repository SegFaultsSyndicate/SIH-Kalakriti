// packages/offline/src/connection.ts
//
// Save-Data and effectiveType, read once at call time (not reactive: these
// change so rarely in practice that a $state wrapper would be one more thing
// to subscribe/unsubscribe for no real benefit -- callers that need a fresh
// read just call again).
//
// navigator.connection is Chromium-only (no Safari/Firefox), so both reads
// degrade to "assume a decent connection" when the API is absent -- that is
// the safer default for a demo running on a reviewer's laptop.

interface NetworkInformation {
  saveData?: boolean;
  effectiveType?: 'slow-2g' | '2g' | '3g' | '4g';
}

function connection(): NetworkInformation | undefined {
  if (typeof navigator === 'undefined') return undefined;
  return (navigator as Navigator & { connection?: NetworkInformation }).connection;
}

/** True if the user has Chrome's Data Saver on. */
export function saveDataEnabled(): boolean {
  return connection()?.saveData ?? false;
}

/** True on a 2G-class link (slow-2g or 2g) -- where a video should not autoplay and images should ship at lower resolution. */
export function isSlowConnection(): boolean {
  const effectiveType = connection()?.effectiveType;
  return effectiveType === 'slow-2g' || effectiveType === '2g';
}

/** Combined check callers actually want: honour either signal. */
export function shouldConserveData(): boolean {
  return saveDataEnabled() || isSlowConnection();
}
