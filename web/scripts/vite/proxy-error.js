// web/scripts/vite/proxy-error.js
//
// Shared vite dev-proxy error handling for all three apps.
//
// When the bff isn't running (or crashes), vite's http-proxy middleware
// fails the request itself and writes a plain-text/HTML 500 with a stack
// trace as the body -- never something @kalakriti/api's transport.ts can
// parse as JSON. transport.ts's catch only reaches its "never reached the
// server" branch (ApiError status 0, mapped to the friendly
// 'api.error.network' message) when fetch() itself throws; a *received*
// non-JSON 500 instead becomes ApiError(500, null), which messageKeyFor
// can't map to anything better than 'api.error.unknown' ("Something went
// wrong") -- indistinguishable from a real backend crash. Since this is
// overwhelmingly "bff isn't up" during local dev, answer it in the same
// {error, message} shape the real bff uses for its own upstream failures
// (see components.schemas.Error in services/bff/openapi.json) so the app
// shows its existing 'api.error.unavailable' copy instead.
export function respondUnavailableOnProxyError(proxy) {
  proxy.options.proxyTimeout = 1500;
  proxy.options.timeout = 1500;
  proxy.on('error', (_err, _req, res) => {
    if (res.headersSent || res.writableEnded) return;
    res.writeHead(503, { 'Content-Type': 'application/json', 'X-Vite-Proxy-Error': '1' });
    res.end(JSON.stringify({ error: 'unavailable', message: 'bff is not reachable' }));
  });
}
