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
// All browser operations are generated from services/bff/openapi.json and
// wrapped in operations.ts. Wire the auth handlers once at app start:
// setUnauthorizedHandler(createLoginRedirectHandler(goto)) and
// setSessionRefreshHandler(...)
// (goto from '$app/navigation', injected rather than imported here since
// that alias only resolves inside a real SvelteKit app).

export { API_BASE, ApiError, request, type RequestOptions } from './transport';
export { call, type CallOptions } from './retry';
export {
  setAcceptLanguage,
  getAcceptLanguage,
  setUnauthorizedHandler,
  getUnauthorizedHandler,
  setSessionRefreshHandler,
  getSessionRefreshHandler,
  type SessionRefreshHandler,
  type UnauthorizedHandler,
  DEFAULT_TIMEOUT_MS,
} from './config';
export {
  setAccessToken,
  getAccessToken,
  setRefreshToken,
  getRefreshToken,
  restoreAccessToken,
} from './auth';
export { decodeJwtClaims } from './jwt';
export { session, type SessionStatus } from './session.svelte';
export { requireRole, type RequireRoleOptions } from './requireRole';
export { createLoginRedirectHandler, type LoginRedirectOptions } from './unauthorized-redirect';
export { completeOtpVerification } from './auth-flow';
export { establishMockSession } from './mock-session';
export { refreshSession } from './session-refresh';
export { messageKeyFor } from './errors';
export {
  requestOtp,
  verifyOtp,
  refreshToken,
  registerArtisan,
  getArtisanProfile,
  updateArtisanProfile,
  requestPhoneChangeOtp,
  verifyPhoneChangeOtp,
  listListings,
  createListing,
  getListing,
  updateListing,
  submitListing,
  approveListing,
  attachListingMedia,
  getListingAttributes,
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
  getStatement,
  listIncomeStatements,
  suggest,
  getListingSummary,
  batchGetListingSummaries,
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
  registerCompany,
  listCompanies,
  getCompany,
  getMyCompany,
  verifyCompany,
  getCommissionStats,
  recordCompanySale,
  listCompanySales,
  expressCompanyInterest,
  respondToInterest,
  listArtisanLeads,
  listBoutiqueMatches,
  listNearbyBoutiques,
  listTrendLinks,
  createTrendLink,
  deleteTrendLink,
  pinTrendLink,
  listBadgeCatalog,
  listArtisanBadges,
  getBadgeProgress,
  grantBadge,
  revokeBadge,
  listSchemes,
  matchSchemes,
  upsertScheme,
  deleteScheme,
  createPartnership,
  listPartnerships,
  createWebhookSubscription,
  listWebhookSubscriptions,
  deleteWebhookSubscription,
  type Company,
  type RegisterCompanyBody,
  type CompanyListResponse,
  type VerifyCompanyBody,
  type CommissionStatsResponse,
  type RecordCompanySaleBody,
  type RecordCompanySaleResponse,
  type CompanySalesListResponse,
  type ExpressInterestBody,
  type ExpressInterestResponse,
  type RespondToInterestBody,
  type RespondToInterestResponse,
  type ArtisanLeadsResponse,
  type BoutiqueMatchesResponse,
  type NearbyBoutiquesResponse,
  type TrendLink,
  type CreateTrendLinkBody,
  type TrendLinksResponse,
  type Badge,
  type ArtisanBadge,
  type ArtisanBadgesResponse,
  type BadgeProgressResponse,
  type GrantBadgeBody,
  type GovernmentScheme,
  type SchemeMatchesResponse,
  type UpsertSchemeBody,
} from './operations';
export { watchOrderEvents, type SseStatus } from './sse.svelte';
export type { SseEvent } from './sse-parse';
export type { paths, components } from './generated/schema';

