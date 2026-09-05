// packages/observability/src/report.ts
//
// Sentry-compatible error reporting, behind a config flag.
//
// No @sentry/* dependency: with the DSN unset (the default -- no app sets
// PUBLIC_SENTRY_DSN today), pulling in the SDK would ship dead weight
// against the artisan app's 150KB budget for a feature that does nothing.
// A DSN's own URL already carries the ingest endpoint and public key, so
// posting a minimal envelope to it needs nothing beyond fetch -- see
// https://develop.sentry.dev/sdk/envelopes/ and
// https://develop.sentry.dev/sdk/data-model/envelope-items/#event. If a real
// deployment wants breadcrumbs, session replay, etc., swap this module's
// body for the real SDK; the call site (reportError) does not change.

interface ParsedDsn {
  endpoint: string;
  publicKey: string;
}

function parseDsn(dsn: string): ParsedDsn | null {
  try {
    const url = new URL(dsn);
    const projectId = url.pathname.replace(/^\//, '');
    if (!projectId || !url.username) return null;
    return {
      endpoint: `${url.protocol}//${url.host}/api/${projectId}/envelope/`,
      publicKey: url.username,
    };
  } catch {
    return null;
  }
}

export interface ReportContext {
  /** Where the error surfaced -- e.g. 'error-boundary', route id, or a component name. */
  source: string;
  [key: string]: unknown;
}

/**
 * Report an error to Sentry if PUBLIC_SENTRY_DSN is set, always log to the
 * console. Never throws -- a broken reporter must not mask the original
 * error or crash the boundary that is trying to recover from one.
 */
export function reportError(error: unknown, context: ReportContext, dsn?: string): void {
  const message = error instanceof Error ? error.message : String(error);
  const stack = error instanceof Error ? error.stack : undefined;
  console.error(`[${context.source}]`, error);

  if (!dsn || typeof fetch === 'undefined') return;
  const parsed = parseDsn(dsn);
  if (!parsed) return;

  const eventId = crypto.randomUUID().replace(/-/g, '');
  const event = {
    event_id: eventId,
    timestamp: new Date().toISOString(),
    platform: 'javascript',
    exception: {
      values: [{ type: error instanceof Error ? error.name : 'Error', value: message, stacktrace: stack ? { frames: [] } : undefined }],
    },
    extra: context,
  };
  const envelope = [
    JSON.stringify({ event_id: eventId, sent_at: event.timestamp, dsn }),
    JSON.stringify({ type: 'event' }),
    JSON.stringify(event),
  ].join('\n');

  void fetch(`${parsed.endpoint}?sentry_key=${parsed.publicKey}&sentry_version=7`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-sentry-envelope' },
    body: envelope,
    keepalive: true,
  }).catch(() => {
    // Reporting failure is not the artisan's problem.
  });
}
