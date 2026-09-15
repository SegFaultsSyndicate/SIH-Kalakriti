// packages/api/src/operations.ts
//
// One function per endpoint in services/bff/openapi.json. Every request and
// response type is pulled from src/generated/schema.d.ts -- nothing here is
// hand-typed, so a spec change that breaks a call site fails to compile.
// Endpoints the spec does not define yet (orders/bulk, follows, statements,
// insights, /auth/refresh, /artisans/me, /search/suggest) have no function
// here until they exist in the spec.
//
// generateUploadUrl/confirmUpload are two of the media flow's three steps.
// The third -- PUT the raw bytes to the upload_url they return -- is
// deliberately NOT here: that URL points at object storage, not the bff, so
// it carries no bearer token or idempotency key and must not go through
// call()/retry.ts. See apps/artisan/src/lib/outbox-send.ts.

import { call, type CallOptions } from './retry';
import { API_BASE } from './transport';
import type { paths, components } from './generated/schema';

type Json<T> = T extends { content: { 'application/json': infer J } } ? J : never;

function toQueryString(
  query: Record<string, string | string[] | undefined> | undefined,
): string {
  if (query === undefined) return '';
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined) continue;
    if (Array.isArray(value)) {
      for (const v of value) params.append(key, v);
    } else {
      params.set(key, value);
    }
  }
  const search = params.toString();
  return search === '' ? '' : `?${search}`;
}

type OtpRequestBody = Json<paths['/auth/otp/request']['post']['requestBody']>;
type OtpRequestResponse = Json<paths['/auth/otp/request']['post']['responses'][200]>;

export function requestOtp(body: OtpRequestBody, options?: CallOptions): Promise<OtpRequestResponse> {
  return call('/auth/otp/request', { ...options, method: 'POST', body }) as Promise<OtpRequestResponse>;
}

type OtpVerifyBody = Json<paths['/auth/otp/verify']['post']['requestBody']>;
type OtpVerifyResponse = Json<paths['/auth/otp/verify']['post']['responses'][200]>;

export function verifyOtp(body: OtpVerifyBody, options?: CallOptions): Promise<OtpVerifyResponse> {
  return call('/auth/otp/verify', { ...options, method: 'POST', body }) as Promise<OtpVerifyResponse>;
}

type RefreshTokenBody = Json<paths['/auth/refresh']['post']['requestBody']>;
type RefreshTokenResponse = Json<paths['/auth/refresh']['post']['responses'][200]>;

export function refreshToken(
  body: RefreshTokenBody,
  options?: CallOptions,
): Promise<RefreshTokenResponse> {
  return call('/auth/refresh', { ...options, method: 'POST', body }) as Promise<RefreshTokenResponse>;
}

type RegisterArtisanBody = Json<paths['/artisans']['post']['requestBody']>;
type RegisterArtisanResponse = Json<paths['/artisans']['post']['responses'][201]>;

export function registerArtisan(
  body: RegisterArtisanBody,
  options?: CallOptions,
): Promise<RegisterArtisanResponse> {
  return call('/artisans', { ...options, method: 'POST', body }) as Promise<RegisterArtisanResponse>;
}

type GetArtisanProfileResponse = Json<paths['/artisans/me']['get']['responses'][200]>;

export function getArtisanProfile(options?: CallOptions): Promise<GetArtisanProfileResponse> {
  return call('/artisans/me', { ...options, method: 'GET' }) as Promise<GetArtisanProfileResponse>;
}

type UpdateArtisanProfileBody = Json<paths['/artisans/me']['patch']['requestBody']>;

export function updateArtisanProfile(
  body: UpdateArtisanProfileBody,
  options?: CallOptions,
): Promise<void> {
  return call('/artisans/me', { ...options, method: 'PATCH', body }) as Promise<void>;
}

type RequestPhoneChangeBody = Json<paths['/auth/phone/change/request']['post']['requestBody']>;
type RequestPhoneChangeResponse = Json<paths['/auth/phone/change/request']['post']['responses'][200]>;

export function requestPhoneChangeOtp(
  body: RequestPhoneChangeBody,
  options?: CallOptions,
): Promise<RequestPhoneChangeResponse> {
  return call('/auth/phone/change/request', {
    ...options,
    method: 'POST',
    body,
  }) as Promise<RequestPhoneChangeResponse>;
}

type VerifyPhoneChangeBody = Json<paths['/auth/phone/change/verify']['post']['requestBody']>;
type VerifyPhoneChangeResponse = Json<paths['/auth/phone/change/verify']['post']['responses'][200]>;

export function verifyPhoneChangeOtp(
  body: VerifyPhoneChangeBody,
  options?: CallOptions,
): Promise<VerifyPhoneChangeResponse> {
  return call('/auth/phone/change/verify', {
    ...options,
    method: 'POST',
    body,
  }) as Promise<VerifyPhoneChangeResponse>;
}

type ListListingsQuery = paths['/listings']['get']['parameters']['query'];
type ListListingsResponse = Json<paths['/listings']['get']['responses'][200]>;

export function listListings(
  query?: ListListingsQuery,
  options?: CallOptions,
): Promise<ListListingsResponse> {
  return call(`/listings${toQueryString(query)}`, { ...options, method: 'GET' }) as Promise<ListListingsResponse>;
}

type CreateListingBody = Json<paths['/listings']['post']['requestBody']>;
type CreateListingResponse = Json<paths['/listings']['post']['responses'][201]>;

export function createListing(
  body: CreateListingBody,
  options?: CallOptions,
): Promise<CreateListingResponse> {
  return call('/listings', { ...options, method: 'POST', body }) as Promise<CreateListingResponse>;
}

type SearchQuery = paths['/search']['get']['parameters']['query'];
type SearchResponse = Json<paths['/search']['get']['responses'][200]>;

export function search(query: SearchQuery, options?: CallOptions): Promise<SearchResponse> {
  return call(`/search${toQueryString(query)}`, { ...options, method: 'GET' }) as Promise<SearchResponse>;
}

type OrderEventsParams = paths['/orders/{id}/events']['get']['parameters']['path'];

/**
 * SSE stream, not JSON -- callers open this with their own EventSource.
 * since (RFC3339Nano) backfills from that instant; pass the occurred_at of
 * the last event seen when reconnecting manually. A native EventSource
 * reconnect sends the same value automatically via Last-Event-ID, so this
 * is only needed for a caller managing its own retry loop.
 */
export function orderEventsUrl(params: OrderEventsParams, since?: string): string {
  const base = `${API_BASE}/orders/${encodeURIComponent(params.id)}/events`;
  return since ? `${base}?since=${encodeURIComponent(since)}` : base;
}

type CreateBulkOrderBody = Json<paths['/orders/bulk']['post']['requestBody']>;
type CreateBulkOrderResponse = Json<paths['/orders/bulk']['post']['responses'][201]>;

export function createBulkOrder(
  body: CreateBulkOrderBody,
  options?: CallOptions,
): Promise<CreateBulkOrderResponse> {
  return call('/orders/bulk', { ...options, method: 'POST', body }) as Promise<CreateBulkOrderResponse>;
}

type GetListingResponse = Json<paths['/listings/{id}']['get']['responses'][200]>;

export function getListing(id: string, options?: CallOptions): Promise<GetListingResponse> {
  return call(`/listings/${encodeURIComponent(id)}`, {
    ...options,
    method: 'GET',
  }) as Promise<GetListingResponse>;
}

type UpdateListingBody = Json<paths['/listings/{id}']['patch']['requestBody']>;

export function updateListing(
  id: string,
  body: UpdateListingBody,
  options?: CallOptions,
): Promise<void> {
  return call(`/listings/${encodeURIComponent(id)}`, {
    ...options,
    method: 'PATCH',
    body,
  }) as Promise<void>;
}

export function submitListing(id: string, options?: CallOptions): Promise<void> {
  return call(`/listings/${encodeURIComponent(id)}/submit`, {
    ...options,
    method: 'POST',
  }) as Promise<void>;
}

type ApproveListingBody = Json<paths['/listings/{id}/approve']['post']['requestBody']>;

export function approveListing(
  id: string,
  body?: ApproveListingBody,
  options?: CallOptions,
): Promise<void> {
  return call(`/listings/${encodeURIComponent(id)}/approve`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<void>;
}

type UploadUrlBody = Json<paths['/media/upload-url']['post']['requestBody']>;
type UploadUrlResponse = Json<paths['/media/upload-url']['post']['responses'][200]>;

export function generateUploadUrl(
  body: UploadUrlBody,
  options?: CallOptions,
): Promise<UploadUrlResponse> {
  return call('/media/upload-url', { ...options, method: 'POST', body }) as Promise<UploadUrlResponse>;
}

export function confirmUpload(mediaId: string, options?: CallOptions): Promise<void> {
  return call(`/media/${encodeURIComponent(mediaId)}/confirm`, {
    ...options,
    method: 'POST',
  }) as Promise<void>;
}

type AdvisePricingBody = Json<paths['/pricing/advise']['post']['requestBody']>;
type AdvisePricingResponse = Json<paths['/pricing/advise']['post']['responses'][200]>;

export function advisePricing(
  body: AdvisePricingBody,
  options?: CallOptions,
): Promise<AdvisePricingResponse> {
  return call('/pricing/advise', { ...options, method: 'POST', body }) as Promise<AdvisePricingResponse>;
}

type SealProvenanceBody = Json<paths['/listings/{id}/seal-provenance']['post']['requestBody']>;
type SealProvenanceResponse = Json<paths['/listings/{id}/seal-provenance']['post']['responses'][200]>;

export function sealProvenance(
  id: string,
  body: SealProvenanceBody,
  options?: CallOptions,
): Promise<SealProvenanceResponse> {
  return call(`/listings/${encodeURIComponent(id)}/seal-provenance`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<SealProvenanceResponse>;
}

type SearchVoiceResponse = Json<paths['/search/voice']['post']['responses'][200]>;

/**
 * Body is the raw recorded audio (Blob), not JSON -- see transport.ts's
 * isRawBody. language is a BCP-47 tag (e.g. "hi-IN"); the BFF reads it off
 * Accept-Language, not the body, so it overrides the app's own locale header
 * for just this call.
 */
export function searchVoice(
  audio: Blob,
  language: string,
  options?: CallOptions,
): Promise<SearchVoiceResponse> {
  return call('/search/voice', {
    ...options,
    method: 'POST',
    body: audio,
    headers: { 'Accept-Language': language, ...options?.headers },
  }) as Promise<SearchVoiceResponse>;
}

type GetOrderResponse = Json<paths['/orders/{id}']['get']['responses'][200]>;

export function getOrder(id: string, options?: CallOptions): Promise<GetOrderResponse> {
  return call(`/orders/${encodeURIComponent(id)}`, { ...options, method: 'GET' }) as Promise<GetOrderResponse>;
}

type RespondToLotBody = Json<paths['/orders/lots/{id}/respond']['post']['requestBody']>;

export function respondToLot(lotId: string, body: RespondToLotBody, options?: CallOptions): Promise<void> {
  return call(`/orders/lots/${encodeURIComponent(lotId)}/respond`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<void>;
}

type ReportProgressBody = Json<paths['/orders/lots/{id}/progress']['post']['requestBody']>;
type ReportProgressResponse = Json<paths['/orders/lots/{id}/progress']['post']['responses'][200]>;

export function reportProgress(
  lotId: string,
  body: ReportProgressBody,
  options?: CallOptions,
): Promise<ReportProgressResponse> {
  return call(`/orders/lots/${encodeURIComponent(lotId)}/progress`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<ReportProgressResponse>;
}

type RequestReallocationBody = Json<paths['/orders/lots/{id}/reallocate']['post']['requestBody']>;
type RequestReallocationResponse = Json<paths['/orders/lots/{id}/reallocate']['post']['responses'][200]>;

export function requestReallocation(
  lotId: string,
  body: RequestReallocationBody,
  options?: CallOptions,
): Promise<RequestReallocationResponse> {
  return call(`/orders/lots/${encodeURIComponent(lotId)}/reallocate`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<RequestReallocationResponse>;
}

type GetFeedResponse = Json<paths['/feed']['get']['responses'][200]>;

export function getFeed(options?: CallOptions): Promise<GetFeedResponse> {
  return call('/feed', { ...options, method: 'GET' }) as Promise<GetFeedResponse>;
}

export function markFeedItemRead(notificationId: string, options?: CallOptions): Promise<void> {
  return call(`/feed/${encodeURIComponent(notificationId)}/read`, {
    ...options,
    method: 'POST',
  }) as Promise<void>;
}

type FollowerCountResponse = Json<paths['/artisans/{id}/follower-count']['get']['responses'][200]>;

export function getFollowerCount(artisanId: string, options?: CallOptions): Promise<FollowerCountResponse> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/follower-count`, {
    ...options,
    method: 'GET',
  }) as Promise<FollowerCountResponse>;
}

type GenerateStatementBody = Json<paths['/statements']['post']['requestBody']>;
type GenerateStatementResponse = Json<paths['/statements']['post']['responses'][201]>;

export function generateStatement(
  body: GenerateStatementBody,
  options?: CallOptions,
): Promise<GenerateStatementResponse> {
  return call('/statements', { ...options, method: 'POST', body }) as Promise<GenerateStatementResponse>;
}

type ListIncomeStatementsResponse = Json<paths['/statements']['get']['responses'][200]>;

export function listIncomeStatements(options?: CallOptions): Promise<ListIncomeStatementsResponse> {
  return call('/statements', { ...options, method: 'GET' }) as Promise<ListIncomeStatementsResponse>;
}

type GetStatementResponse = Json<paths['/statements/{id}']['get']['responses'][200]>;

export function getStatement(id: string, options?: CallOptions): Promise<GetStatementResponse> {
  return call(`/statements/${encodeURIComponent(id)}`, {
    ...options,
    method: 'GET',
  }) as Promise<GetStatementResponse>;
}

type SuggestResponse = Json<paths['/search/suggest']['get']['responses'][200]>;

export function suggest(q: string, options?: CallOptions): Promise<SuggestResponse> {
  return call(`/search/suggest${toQueryString({ q })}`, { ...options, method: 'GET' }) as Promise<SuggestResponse>;
}

type ListingSummaryResponse = Json<paths['/listings/{id}/summary']['get']['responses'][200]>;

export function getListingSummary(id: string, options?: CallOptions): Promise<ListingSummaryResponse> {
  return call(`/listings/${encodeURIComponent(id)}/summary`, {
    ...options,
    method: 'GET',
  }) as Promise<ListingSummaryResponse>;
}

type BatchListingSummariesResponse = Json<paths['/listings/summaries']['get']['responses'][200]>;

export function batchGetListingSummaries(
  ids: string[],
  options?: CallOptions,
): Promise<BatchListingSummariesResponse> {
  return call(`/listings/summaries${toQueryString({ ids: ids.join(',') })}`, {
    ...options,
    method: 'GET',
  }) as Promise<BatchListingSummariesResponse>;
}

type ListCraftsResponse = Json<paths['/crafts']['get']['responses'][200]>;

export function listCrafts(options?: CallOptions): Promise<ListCraftsResponse> {
  return call('/crafts', { ...options, method: 'GET' }) as Promise<ListCraftsResponse>;
}

type CraftDetailResponse = Json<paths['/crafts/{slug}']['get']['responses'][200]>;

export function getCraft(slug: string, options?: CallOptions): Promise<CraftDetailResponse> {
  return call(`/crafts/${encodeURIComponent(slug)}`, {
    ...options,
    method: 'GET',
  }) as Promise<CraftDetailResponse>;
}

type ArtisanStorefrontResponse = Json<paths['/artisans/{id}/storefront']['get']['responses'][200]>;

export function getArtisanStorefront(
  id: string,
  options?: CallOptions,
): Promise<ArtisanStorefrontResponse> {
  return call(`/artisans/${encodeURIComponent(id)}/storefront`, {
    ...options,
    method: 'GET',
  }) as Promise<ArtisanStorefrontResponse>;
}

export function followArtisan(id: string, options?: CallOptions): Promise<void> {
  return call(`/artisans/${encodeURIComponent(id)}/follow`, {
    ...options,
    method: 'POST',
  }) as Promise<void>;
}

export function unfollowArtisan(id: string, options?: CallOptions): Promise<void> {
  return call(`/artisans/${encodeURIComponent(id)}/follow`, {
    ...options,
    method: 'DELETE',
  }) as Promise<void>;
}

type ProcessFeedResponse = Json<paths['/feed/process']['get']['responses'][200]>;

export function getProcessFeed(options?: CallOptions): Promise<ProcessFeedResponse> {
  return call('/feed/process', { ...options, method: 'GET' }) as Promise<ProcessFeedResponse>;
}

// Batch 13 -- ministry insights, cluster/SHG administration, moderation and
// craft-index refresh. All require an authenticated CLUSTER_OFFICER or
// MINISTRY caller; the BFF enforces the exact role per endpoint.

type ArtisansByCategoryQuery = paths['/insights/artisans-by-category']['get']['parameters']['query'];
type ArtisansByCategoryResponse = Json<paths['/insights/artisans-by-category']['get']['responses'][200]>;

export function getArtisansByCategory(
  query?: ArtisansByCategoryQuery,
  options?: CallOptions,
): Promise<ArtisansByCategoryResponse> {
  return call(`/insights/artisans-by-category${toQueryString(query)}`, {
    ...options,
    method: 'GET',
  }) as Promise<ArtisansByCategoryResponse>;
}

type ListingsByCraftMonthQuery = paths['/insights/listings-by-craft-month']['get']['parameters']['query'];
type ListingsByCraftMonthResponse = Json<paths['/insights/listings-by-craft-month']['get']['responses'][200]>;

export function getListingsByCraftMonth(
  query?: ListingsByCraftMonthQuery,
  options?: CallOptions,
): Promise<ListingsByCraftMonthResponse> {
  return call(`/insights/listings-by-craft-month${toQueryString(query)}`, {
    ...options,
    method: 'GET',
  }) as Promise<ListingsByCraftMonthResponse>;
}

type EarningsByDistrictQuery = paths['/insights/earnings-by-district']['get']['parameters']['query'];
type EarningsByDistrictResponse = Json<paths['/insights/earnings-by-district']['get']['responses'][200]>;

export function getEarningsByDistrict(
  query?: EarningsByDistrictQuery,
  options?: CallOptions,
): Promise<EarningsByDistrictResponse> {
  return call(`/insights/earnings-by-district${toQueryString(query)}`, {
    ...options,
    method: 'GET',
  }) as Promise<EarningsByDistrictResponse>;
}

type IncomeComparisonQuery = paths['/insights/income-comparison']['get']['parameters']['query'];
type IncomeComparisonResponse = Json<paths['/insights/income-comparison']['get']['responses'][200]>;

export function getIncomeComparison(
  query?: IncomeComparisonQuery,
  options?: CallOptions,
): Promise<IncomeComparisonResponse> {
  return call(`/insights/income-comparison${toQueryString(query)}`, {
    ...options,
    method: 'GET',
  }) as Promise<IncomeComparisonResponse>;
}

type DyingCraftsResponse = Json<paths['/insights/dying-crafts']['get']['responses'][200]>;

export function getDyingCrafts(options?: CallOptions): Promise<DyingCraftsResponse> {
  return call('/insights/dying-crafts', { ...options, method: 'GET' }) as Promise<DyingCraftsResponse>;
}

type RefreshInsightsResponse = Json<paths['/insights/refresh']['post']['responses'][200]>;

export function refreshInsights(options?: CallOptions): Promise<RefreshInsightsResponse> {
  return call('/insights/refresh', { ...options, method: 'POST' }) as Promise<RefreshInsightsResponse>;
}

type CreateClusterBody = Json<paths['/clusters']['post']['requestBody']>;
type ClusterResponse = Json<paths['/clusters']['post']['responses'][201]>;

export function createCluster(body: CreateClusterBody, options?: CallOptions): Promise<ClusterResponse> {
  return call('/clusters', { ...options, method: 'POST', body }) as Promise<ClusterResponse>;
}

type GetClusterResponse = Json<paths['/clusters/{id}']['get']['responses'][200]>;

export function getCluster(id: string, options?: CallOptions): Promise<GetClusterResponse> {
  return call(`/clusters/${encodeURIComponent(id)}`, { ...options, method: 'GET' }) as Promise<GetClusterResponse>;
}

type ClusterMembersResponse = Json<paths['/clusters/{id}/members']['get']['responses'][200]>;

export function listClusterMembers(id: string, options?: CallOptions): Promise<ClusterMembersResponse> {
  return call(`/clusters/${encodeURIComponent(id)}/members`, {
    ...options,
    method: 'GET',
  }) as Promise<ClusterMembersResponse>;
}

type AddClusterMemberBody = Json<paths['/clusters/{id}/members']['post']['requestBody']>;
type AddClusterMemberResponse = Json<paths['/clusters/{id}/members']['post']['responses'][200]>;

export function addClusterMember(
  id: string,
  body: AddClusterMemberBody,
  options?: CallOptions,
): Promise<AddClusterMemberResponse> {
  return call(`/clusters/${encodeURIComponent(id)}/members`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<AddClusterMemberResponse>;
}

export function removeClusterMember(id: string, artisanId: string, options?: CallOptions): Promise<void> {
  return call(`/clusters/${encodeURIComponent(id)}/members/${encodeURIComponent(artisanId)}`, {
    ...options,
    method: 'DELETE',
  }) as Promise<void>;
}

type OnboardClusterArtisanBody = Json<paths['/clusters/{id}/onboard']['post']['requestBody']>;
type OnboardClusterArtisanResponse = Json<paths['/clusters/{id}/onboard']['post']['responses'][201]>;

export function onboardClusterArtisan(
  id: string,
  body: OnboardClusterArtisanBody,
  options?: CallOptions,
): Promise<OnboardClusterArtisanResponse> {
  return call(`/clusters/${encodeURIComponent(id)}/onboard`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<OnboardClusterArtisanResponse>;
}

type CreateSHGBody = Json<paths['/self-help-groups']['post']['requestBody']>;
type SHGResponse = Json<paths['/self-help-groups']['post']['responses'][201]>;

export function createSelfHelpGroup(body: CreateSHGBody, options?: CallOptions): Promise<SHGResponse> {
  return call('/self-help-groups', { ...options, method: 'POST', body }) as Promise<SHGResponse>;
}

type GetSHGResponse = Json<paths['/self-help-groups/{id}']['get']['responses'][200]>;

export function getSelfHelpGroup(id: string, options?: CallOptions): Promise<GetSHGResponse> {
  return call(`/self-help-groups/${encodeURIComponent(id)}`, {
    ...options,
    method: 'GET',
  }) as Promise<GetSHGResponse>;
}

type SetSHGMembersBody = Json<paths['/self-help-groups/{id}/members']['put']['requestBody']>;
type SetSHGMembersResponse = Json<paths['/self-help-groups/{id}/members']['put']['responses'][200]>;

export function setSelfHelpGroupMembers(
  id: string,
  body: SetSHGMembersBody,
  options?: CallOptions,
): Promise<SetSHGMembersResponse> {
  return call(`/self-help-groups/${encodeURIComponent(id)}/members`, {
    ...options,
    method: 'PUT',
    body,
  }) as Promise<SetSHGMembersResponse>;
}

type SuspendListingBody = Json<paths['/listings/{id}/suspend']['post']['requestBody']>;
type SuspendListingResponse = Json<paths['/listings/{id}/suspend']['post']['responses'][200]>;

export function suspendListing(
  id: string,
  body: SuspendListingBody,
  options?: CallOptions,
): Promise<SuspendListingResponse> {
  return call(`/listings/${encodeURIComponent(id)}/suspend`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<SuspendListingResponse>;
}

type ReinstateListingResponse = Json<paths['/listings/{id}/reinstate']['post']['responses'][200]>;

export function reinstateListing(id: string, options?: CallOptions): Promise<ReinstateListingResponse> {
  return call(`/listings/${encodeURIComponent(id)}/reinstate`, {
    ...options,
    method: 'POST',
  }) as Promise<ReinstateListingResponse>;
}

type RefreshCraftIndexResponse = Json<paths['/crafts/refresh-index']['post']['responses'][200]>;

export function refreshCraftIndex(options?: CallOptions): Promise<RefreshCraftIndexResponse> {
  return call('/crafts/refresh-index', { ...options, method: 'POST' }) as Promise<RefreshCraftIndexResponse>;
}

// --- B2B & Boutique Operations ---

export type Company = Json<paths['/companies']['post']['responses'][201]>;
export type RegisterCompanyBody = Json<paths['/companies']['post']['requestBody']>;
export type CompanyListResponse = Json<paths['/companies']['get']['responses'][200]>;
export type VerifyCompanyBody = Json<paths['/companies/{id}/verify']['post']['requestBody']>;
export type CommissionStatsResponse = Json<paths['/companies/commission-stats']['get']['responses'][200]>;
export type RecordCompanySaleBody = Json<paths['/companies/sales/settle']['post']['requestBody']>;
export type RecordCompanySaleResponse = Json<paths['/companies/sales/settle']['post']['responses'][200]>;
export type CompanySalesListResponse = Json<paths['/companies/{id}/sales']['get']['responses'][200]>;
export type ExpressInterestBody = Json<paths['/companies/{id}/interest']['post']['requestBody']>;
export type ExpressInterestResponse = Json<paths['/companies/{id}/interest']['post']['responses'][201]>;
export type RespondToInterestBody = Json<paths['/leads/{id}/respond']['post']['requestBody']>;
export type RespondToInterestResponse = Json<paths['/leads/{id}/respond']['post']['responses'][200]>;
export type ArtisanLeadsResponse = Json<paths['/artisans/me/leads']['get']['responses'][200]>;
export type BoutiqueMatchesResponse = Json<paths['/artisans/me/boutique-matches']['get']['responses'][200]>;
export type NearbyBoutiquesResponse = Json<paths['/boutiques/nearby']['get']['responses'][200]>;

export function registerCompany(body: RegisterCompanyBody, options?: CallOptions): Promise<Company> {
  return call('/companies', { ...options, method: 'POST', body }) as Promise<Company>;
}

export function listCompanies(
  query?: { state_code?: string; type?: string; verification_status?: string; verified_only?: boolean; page_token?: string },
  options?: CallOptions,
): Promise<CompanyListResponse> {
  const q = new URLSearchParams();
  if (query?.state_code) q.set('state_code', query.state_code);
  if (query?.type) q.set('type', query.type);
  if (query?.verification_status) q.set('verification_status', query.verification_status);
  if (query?.verified_only !== undefined) q.set('verified_only', String(query.verified_only));
  if (query?.page_token) q.set('page_token', query.page_token);
  const qs = q.toString();
  return call(`/companies${qs ? `?${qs}` : ''}`, { ...options, method: 'GET' }) as Promise<CompanyListResponse>;
}

export function getCompany(id: string, options?: CallOptions): Promise<Company> {
  return call(`/companies/${encodeURIComponent(id)}`, { ...options, method: 'GET' }) as Promise<Company>;
}

export function getMyCompany(options?: CallOptions): Promise<Company> {
  return call('/companies/me', { ...options, method: 'GET' }) as Promise<Company>;
}

export function verifyCompany(id: string, body: VerifyCompanyBody, options?: CallOptions): Promise<Company> {
  return call(`/companies/${encodeURIComponent(id)}/verify`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<Company>;
}

export function getCommissionStats(options?: CallOptions): Promise<CommissionStatsResponse> {
  return call('/companies/commission-stats', { ...options, method: 'GET' }) as Promise<CommissionStatsResponse>;
}

export function recordCompanySale(body: RecordCompanySaleBody, options?: CallOptions): Promise<RecordCompanySaleResponse> {
  return call('/companies/sales/settle', { ...options, method: 'POST', body }) as Promise<RecordCompanySaleResponse>;
}

export function listCompanySales(id: string, query?: { page_token?: string }, options?: CallOptions): Promise<CompanySalesListResponse> {
  const qs = query?.page_token ? `?page_token=${encodeURIComponent(query.page_token)}` : '';
  return call(`/companies/${encodeURIComponent(id)}/sales${qs}`, {
    ...options,
    method: 'GET',
  }) as Promise<CompanySalesListResponse>;
}

export function expressCompanyInterest(companyId: string, body: ExpressInterestBody, options?: CallOptions): Promise<ExpressInterestResponse> {
  return call(`/companies/${encodeURIComponent(companyId)}/interest`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<ExpressInterestResponse>;
}

export function respondToInterest(interestId: string, body: RespondToInterestBody, options?: CallOptions): Promise<RespondToInterestResponse> {
  return call(`/leads/${encodeURIComponent(interestId)}/respond`, {
    ...options,
    method: 'POST',
    body,
  }) as Promise<RespondToInterestResponse>;
}

export function listArtisanLeads(query?: { status?: string }, options?: CallOptions): Promise<ArtisanLeadsResponse> {
  const qs = query?.status ? `?status=${encodeURIComponent(query.status)}` : '';
  return call(`/artisans/me/leads${qs}`, { ...options, method: 'GET' }) as Promise<ArtisanLeadsResponse>;
}

export function listBoutiqueMatches(options?: CallOptions): Promise<BoutiqueMatchesResponse> {
  return call('/artisans/me/boutique-matches', { ...options, method: 'GET' }) as Promise<BoutiqueMatchesResponse>;
}

export function listNearbyBoutiques(
  query: { latitude: number; longitude: number; radius_km?: number; craft_id?: string },
  options?: CallOptions,
): Promise<NearbyBoutiquesResponse> {
  const q = new URLSearchParams({
    latitude: String(query.latitude),
    longitude: String(query.longitude),
  });
  if (query.radius_km) q.set('radius_km', String(query.radius_km));
  if (query.craft_id) q.set('craft_id', query.craft_id);
  return call(`/boutiques/nearby?${q.toString()}`, { ...options, method: 'GET' }) as Promise<NearbyBoutiquesResponse>;
}

// --- Market Trends Operations ---

export type TrendLink = Json<paths['/trends']['post']['responses'][201]>;
export type CreateTrendLinkBody = Json<paths['/trends']['post']['requestBody']>;
export type TrendLinksResponse = Json<paths['/trends']['get']['responses'][200]>;

export function listTrendLinks(
  query?: { craft_id?: string; source_type?: string; exclude_expired?: boolean },
  options?: CallOptions,
): Promise<TrendLinksResponse> {
  const q = new URLSearchParams();
  if (query?.craft_id) q.set('craft_id', query.craft_id);
  if (query?.source_type) q.set('source_type', query.source_type);
  if (query?.exclude_expired !== undefined) q.set('exclude_expired', String(query.exclude_expired));
  const qs = q.toString();
  return call(`/trends${qs ? `?${qs}` : ''}`, { ...options, method: 'GET' }) as Promise<TrendLinksResponse>;
}

export function createTrendLink(body: CreateTrendLinkBody, options?: CallOptions): Promise<TrendLink> {
  return call('/trends', { ...options, method: 'POST', body }) as Promise<TrendLink>;
}

export function deleteTrendLink(id: string, options?: CallOptions): Promise<{ status?: string }> {
  return call(`/trends/${encodeURIComponent(id)}`, { ...options, method: 'DELETE' }) as Promise<{ status?: string }>;
}

export function pinTrendLink(id: string, pinned: boolean, options?: CallOptions): Promise<TrendLink> {
  return call(`/trends/${encodeURIComponent(id)}/pin`, {
    ...options,
    method: 'POST',
    body: { pinned },
  }) as Promise<TrendLink>;
}

// --- Badge Operations ---

export type Badge = NonNullable<Json<paths['/badges']['get']['responses'][200]>['badges']>[number];
export type ArtisanBadge = components['schemas']['ArtisanBadge'];
export type ArtisanBadgesResponse = Json<paths['/artisans/{id}/badges']['get']['responses'][200]>;
export type BadgeProgressResponse = Json<paths['/badges/me/progress']['get']['responses'][200]>;
export type GrantBadgeBody = Json<paths['/artisans/{id}/badges']['post']['requestBody']>;

export function listBadgeCatalog(options?: CallOptions): Promise<{ badges: Badge[] }> {
  return call('/badges', { ...options, method: 'GET' }) as Promise<{ badges: Badge[] }>;
}

export function listArtisanBadges(artisanId: string, options?: CallOptions): Promise<ArtisanBadgesResponse> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges`, { ...options, method: 'GET' }) as Promise<ArtisanBadgesResponse>;
}

export function getBadgeProgress(options?: CallOptions): Promise<BadgeProgressResponse> {
  return call('/badges/me/progress', { ...options, method: 'GET' }) as Promise<BadgeProgressResponse>;
}

export function grantBadge(artisanId: string, body: GrantBadgeBody, options?: CallOptions): Promise<ArtisanBadge> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges`, { ...options, method: 'POST', body }) as Promise<ArtisanBadge>;
}

export function revokeBadge(artisanId: string, code: string, body: { reason: string }, options?: CallOptions): Promise<{ status?: string }> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges/${encodeURIComponent(code)}`, { ...options, method: 'DELETE', body }) as Promise<{ status?: string }>;
}

// --- Government Scheme Operations ---

export type GovernmentScheme = NonNullable<Json<paths['/schemes']['get']['responses'][200]>['schemes']>[number];
export type SchemeMatchesResponse = Json<paths['/schemes/match']['get']['responses'][200]>;
export type UpsertSchemeBody = Json<paths['/schemes']['post']['requestBody']>;

export function listSchemes(options?: CallOptions): Promise<{ schemes: GovernmentScheme[] }> {
  return call('/schemes', { ...options, method: 'GET' }) as Promise<{ schemes: GovernmentScheme[] }>;
}

export function matchSchemes(options?: CallOptions): Promise<SchemeMatchesResponse> {
  return call('/schemes/match', { ...options, method: 'GET' }) as Promise<SchemeMatchesResponse>;
}

export function upsertScheme(body: UpsertSchemeBody, id?: string, options?: CallOptions): Promise<GovernmentScheme> {
  const path = id ? `/schemes/${encodeURIComponent(id)}` : '/schemes';
  return call(path, { ...options, method: id ? 'PATCH' : 'POST', body }) as Promise<GovernmentScheme>;
}

export function deleteScheme(id: string, options?: CallOptions): Promise<{ status?: string }> {
  return call(`/schemes/${encodeURIComponent(id)}`, { ...options, method: 'DELETE' }) as Promise<{ status?: string }>;
}
