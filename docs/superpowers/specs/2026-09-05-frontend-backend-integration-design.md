# Frontend–Backend Integration Design

## Goal

Make the three SvelteKit applications use the Go BFF as their only runtime API boundary, while keeping the BFF, protobuf clients, OpenAPI document, generated TypeScript types, and backend services aligned.

## Current findings

- The BFF already exposes most required REST routes under `/api/v1` and translates requests to gRPC clients.
- `web/packages/api` contains a transport, retry, auth, SSE, and operation layer, but app routes still contain local/mock runtime paths and the generated-contract workflow needs verification.
- The backend prompt's service seams, migrations, protobufs, and BFF are present, but acceptance coverage varies by subsystem and must be verified rather than assumed.
- Browser code must not call gRPC or service ports directly.

## Architecture

The browser calls typed functions from `@kalakriti/api`. Those functions use the BFF's versioned REST API, attach locale/auth/idempotency headers, and normalize errors. The BFF remains a translation and aggregation layer; business rules stay in Go services. Offline persistence is limited to drafts and safe, idempotent mutations.

Authentication uses OTP verification and refresh-token rotation through the BFF. A request that receives a 401 attempts one refresh and retries once; only a failed refresh redirects to login. Public reads remain available without a session.

## Integration rules

1. `services/bff/openapi.json` is the browser contract.
2. `web/packages/api/src/generated/schema.d.ts` is generated from that contract and is not hand-edited.
3. Every frontend request/response shape comes from generated OpenAPI types.
4. Mutations use `Idempotency-Key`; retries never replay a mutation without the same key.
5. Network failures are explicit and use the existing offline/outbox seams where supported.
6. A successful-looking local fallback is forbidden in production.
7. Server errors expose only the BFF's consistent error body.
8. SEO and verification pages may use the BFF's public server-rendered routes, but interactive browser data still uses `/api/v1`.

## Data flows

### Artisan

OTP → session → profile/registration → upload URL → MinIO upload → media confirm → listing create/update → pricing advisory → submit → approval → provenance seal → orders/events/statements.

### Buyer

Search/text/voice → listing summary → artisan storefront/follow → bulk order → order detail/SSE → provenance verification.

### Admin

Session/role → clusters and members → SHGs → moderation → craft index refresh → insight aggregates.

## Error and offline behavior

- 401: refresh once, then clear session and navigate to login.
- 403/422/409: return typed API error details to the current screen; do not retry automatically.
- 502/503/504/network failure: retry reads with bounded backoff; queue only explicitly supported idempotent mutations.
- Upload failures keep the local draft and expose a retry action.
- SSE reconnects using the last event timestamp and does not duplicate events.

## Verification

The implementation must pass:

- generated-contract freshness and route parity checks;
- focused TypeScript/Vitest tests for API/session/mutation behavior;
- focused Go tests for BFF handlers and clients;
- existing frontend and backend unit suites;
- integration tests when Docker dependencies are available;
- a documented matrix separating automated results from physical-device, print, and external-provider validation.

## Scope boundary

This work fixes backend/frontend contract and runtime wiring that is directly coupled to the requested integration. It does not replace external provider credentials, certify real-device accessibility, or claim physical QR scanning without those environments.
