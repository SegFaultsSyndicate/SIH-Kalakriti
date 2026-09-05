// packages/api/src/index.ts
//
// Public surface of the BFF client. transport.ts is the bare fetch wrapper;
// retry.ts's call() adds retry-with-backoff, the bearer token, and the
// Idempotency-Key on top; operations.ts is spec-typed and is what app code
// should actually call.
//
// THE TYPES ARE GENERATED, NOT WRITTEN. `pnpm -F @kalakriti/api api:gen`
// runs openapi-typescript against services/bff/openapi.json and writes
// src/generated/schema.d.ts (committed, so builds are reproducible without a
// live BFF). No request or response interface may be typed by hand in this
// package or in any app: a hand-written shape that drifts from the spec
// fails at runtime, in the field, on a phone. If a field is needed and the
// spec does not have it, the spec changes first.
//
// The real services/bff/openapi.json has 6 paths today. Per-domain wrappers
// for media, pricing, orders/bulk, follows, statements and insights don't
// exist here because those endpoints don't exist in the spec yet -- adding
// them would mean hand-typing a shape, which is exactly what this package
// exists to refuse to do. Same for silent token refresh: there is no
// /auth/refresh path, so a 401 is reported, not repaired -- wire
// setUnauthorizedHandler(createLoginRedirectHandler(goto)) once at app start
// (goto from '$app/navigation', injected rather than imported here since
// that alias only resolves inside a real SvelteKit app).

export { API_BASE, ApiError, request, type RequestOptions } from './transport';
export { call, type CallOptions } from './retry';
export {
  setAcceptLanguage,
  getAcceptLanguage,
  setUnauthorizedHandler,
  getUnauthorizedHandler,
  type UnauthorizedHandler,
  DEFAULT_TIMEOUT_MS,
} from './config';
export { setAccessToken, getAccessToken, restoreAccessToken } from './auth';
export { decodeJwtClaims } from './jwt';
export { session, type SessionStatus } from './session.svelte';
export { requireRole, type RequireRoleOptions } from './requireRole';
export { createLoginRedirectHandler, type LoginRedirectOptions } from './unauthorized-redirect';
export { completeOtpVerification } from './auth-flow';
export { messageKeyFor } from './errors';
export {
  requestOtp,
  verifyOtp,
  registerArtisan,
  listListings,
  createListing,
  getListing,
  updateListing,
  submitListing,
  approveListing,
  generateUploadUrl,
  confirmUpload,
  advisePricing,
  sealProvenance,
  searchVoice,
  search,
  orderEventsUrl,
  getOrder,
  respondToLot,
  reportProgress,
  requestReallocation,
  getFeed,
  markFeedItemRead,
  getFollowerCount,
  generateStatement,
  listIncomeStatements,
  suggest,
  getListingSummary,
  listCrafts,
  getCraft,
  getArtisanStorefront,
  followArtisan,
  unfollowArtisan,
  getProcessFeed,
  createBulkOrder,
  getArtisansByCategory,
  getListingsByCraftMonth,
  getEarningsByDistrict,
  getIncomeComparison,
  getDyingCrafts,
  refreshInsights,
  createCluster,
  getCluster,
  onboardClusterArtisan,
  listClusterMembers,
  addClusterMember,
  removeClusterMember,
  createSelfHelpGroup,
  getSelfHelpGroup,
  setSelfHelpGroupMembers,
  suspendListing,
  reinstateListing,
  refreshCraftIndex,
} from './operations';
export { watchOrderEvents, type SseStatus } from './sse.svelte';
export type { SseEvent } from './sse-parse';
export type { paths, components } from './generated/schema';
