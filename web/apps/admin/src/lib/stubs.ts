// apps/admin/src/lib/stubs.ts
//
// Sample data used to fill pages when the backend is unreachable or returns
// empty results, following the app's VITE_USE_MOCKS=1 mock-fallback pattern
// (see companies/+page.svelte). When VITE_USE_MOCKS=1 is set, mergeWithStubs
// also appends sample rows to real results so a page never looks empty.
// Each call site warns via console.warn('[mock fallback] ...') when it falls
// back after an error. Real API data always wins for rows it already covers.
import {
  getImpactSummary,
  type AgentProductivity,
  type ArtisanBadge,
  type Badge,
  type FinanceCoverageRow,
  type FinanceLinkForReview,
  type GovernmentScheme,
  type ImpactGroupRow,
  type LiteracyFunnelRow,
  type LinkForReview,
  type SalesMixMonth,
  type StaffAccount,
} from '@kalakriti/api';

export type ImpactSummary = Awaited<ReturnType<typeof getImpactSummary>>;

/** Real rows first, then any stub rows whose key isn't already present. */
export function mergeWithStubs<T>(items: T[], stubs: readonly T[], key: (item: T) => string): T[] {
  const merged = [...items];
  if (import.meta.env.VITE_USE_MOCKS !== '1') return merged;
  const seen = new Set<string>(items.map(key));
  for (const stub of stubs) {
    if (!seen.has(key(stub))) {
      merged.push(stub);
      seen.add(key(stub));
    }
  }
  return merged;
}

export const IMPACT_SUMMARY: ImpactSummary = {
  suppressed: false,
  beneficiaries: 48231,
  active_sellers_90d: 12240,
  uplift_sample: 8642,
  median_uplift_pct: 38.4,
  digital_share_pct: 32.6,
  certificates_issued: 15297,
  finance_linked: 2218,
  finance_verified: 1416,
  refreshed_at: '2026-09-20T06:00:00.000Z',
};

export const IMPACT_GROUPS: ImpactGroupRow[] = [
  { group: 'Kutch', state_code: 'IN-GJ', suppressed: false, artisan_count: 3280, active_sellers_90d: 1410, with_baseline_count: 1180, median_baseline_monthly_paise: 950000, median_current_monthly_paise: 1380000, median_uplift_pct: 45.3, platform_income_paise_90d: 152000000, offline_income_paise_90d: 224000000, fair_income_paise_90d: 98000000 },
  { group: 'Jaipur', state_code: 'IN-RJ', suppressed: false, artisan_count: 2960, active_sellers_90d: 1320, with_baseline_count: 1040, median_baseline_monthly_paise: 1100000, median_current_monthly_paise: 1560000, median_uplift_pct: 41.8, platform_income_paise_90d: 141000000, offline_income_paise_90d: 198000000, fair_income_paise_90d: 87000000 },
  { group: 'Varanasi', state_code: 'IN-UP', suppressed: false, artisan_count: 2650, active_sellers_90d: 1140, with_baseline_count: 920, median_baseline_monthly_paise: 890000, median_current_monthly_paise: 1270000, median_uplift_pct: 42.7, platform_income_paise_90d: 128000000, offline_income_paise_90d: 176000000, fair_income_paise_90d: 74000000 },
  { group: 'Bhadohi', state_code: 'IN-UP', suppressed: false, artisan_count: 2310, active_sellers_90d: 980, with_baseline_count: 810, median_baseline_monthly_paise: 760000, median_current_monthly_paise: 1010000, median_uplift_pct: 32.9, platform_income_paise_90d: 104000000, offline_income_paise_90d: 151000000, fair_income_paise_90d: 61000000 },
  { group: 'Salem', state_code: 'IN-TN', suppressed: false, artisan_count: 2140, active_sellers_90d: 930, with_baseline_count: 700, median_baseline_monthly_paise: 830000, median_current_monthly_paise: 1210000, median_uplift_pct: 45.8, platform_income_paise_90d: 98000000, offline_income_paise_90d: 139000000, fair_income_paise_90d: 52000000 },
  { group: 'Moradabad', state_code: 'IN-UP', suppressed: false, artisan_count: 1980, active_sellers_90d: 850, with_baseline_count: 640, median_baseline_monthly_paise: 940000, median_current_monthly_paise: 1330000, median_uplift_pct: 41.5, platform_income_paise_90d: 88000000, offline_income_paise_90d: 126000000, fair_income_paise_90d: 47000000 },
  { group: 'Channapatna', state_code: 'IN-KA', suppressed: false, artisan_count: 1760, active_sellers_90d: 790, with_baseline_count: 590, median_baseline_monthly_paise: 1000000, median_current_monthly_paise: 1450000, median_uplift_pct: 45.0, platform_income_paise_90d: 83000000, offline_income_paise_90d: 112000000, fair_income_paise_90d: 43000000 },
  { group: 'Kullu', state_code: 'IN-HP', suppressed: false, artisan_count: 1510, active_sellers_90d: 680, with_baseline_count: 500, median_baseline_monthly_paise: 820000, median_current_monthly_paise: 1180000, median_uplift_pct: 43.9, platform_income_paise_90d: 69000000, offline_income_paise_90d: 94000000, fair_income_paise_90d: 31000000 },
];

export const IMPACT_GROUPS_BY_CORPORATION: ImpactGroupRow[] = [
  { group: 'NSFDC', state_code: 'IN-RJ', suppressed: false, artisan_count: 820, active_sellers_90d: 610, with_baseline_count: 460, median_baseline_monthly_paise: 880000, median_current_monthly_paise: 1240000, median_uplift_pct: 40.9, platform_income_paise_90d: 42000000, offline_income_paise_90d: 58000000, fair_income_paise_90d: 21000000 },
  { group: 'NBCFDC', state_code: 'IN-RJ', suppressed: false, artisan_count: 640, active_sellers_90d: 470, with_baseline_count: 380, median_baseline_monthly_paise: 900000, median_current_monthly_paise: 1280000, median_uplift_pct: 42.2, platform_income_paise_90d: 34000000, offline_income_paise_90d: 46000000, fair_income_paise_90d: 18000000 },
  { group: 'NSKFDC', state_code: 'IN-RJ', suppressed: false, artisan_count: 480, active_sellers_90d: 350, with_baseline_count: 270, median_baseline_monthly_paise: 850000, median_current_monthly_paise: 1160000, median_uplift_pct: 36.5, platform_income_paise_90d: 25000000, offline_income_paise_90d: 33000000, fair_income_paise_90d: 12000000 },
  { group: 'NDFDC', state_code: 'IN-RJ', suppressed: false, artisan_count: 350, active_sellers_90d: 240, with_baseline_count: 190, median_baseline_monthly_paise: 860000, median_current_monthly_paise: 1180000, median_uplift_pct: 37.2, platform_income_paise_90d: 18000000, offline_income_paise_90d: 24000000, fair_income_paise_90d: 9000000 },
  { group: 'PM_DAKSH', state_code: 'IN-RJ', suppressed: false, artisan_count: 420, active_sellers_90d: 300, with_baseline_count: 230, median_baseline_monthly_paise: 910000, median_current_monthly_paise: 1310000, median_uplift_pct: 44.0, platform_income_paise_90d: 22000000, offline_income_paise_90d: 29000000, fair_income_paise_90d: 11000000 },
  { group: 'PM_AJAY', state_code: 'IN-RJ', suppressed: false, artisan_count: 230, active_sellers_90d: 160, with_baseline_count: 120, median_baseline_monthly_paise: 870000, median_current_monthly_paise: 1220000, median_uplift_pct: 40.2, platform_income_paise_90d: 12000000, offline_income_paise_90d: 15000000, fair_income_paise_90d: 5000000 },
  { group: 'OTHER', state_code: 'IN-RJ', suppressed: false, artisan_count: 128, active_sellers_90d: 90, with_baseline_count: 64, median_baseline_monthly_paise: 960000, median_current_monthly_paise: 1390000, median_uplift_pct: 44.8, platform_income_paise_90d: 7000000, offline_income_paise_90d: 9000000, fair_income_paise_90d: 3000000 },
];

export const IMPACT_GROUPS_BY_CATEGORY: ImpactGroupRow[] = [
  { group: 'GENERAL', state_code: 'IN-RJ', suppressed: false, artisan_count: 18420, active_sellers_90d: 6230, with_baseline_count: 3980, median_baseline_monthly_paise: 1000000, median_current_monthly_paise: 1430000, median_uplift_pct: 43.0, platform_income_paise_90d: 640000000, offline_income_paise_90d: 910000000, fair_income_paise_90d: 290000000 },
  { group: 'OBC', state_code: 'IN-RJ', suppressed: false, artisan_count: 15060, active_sellers_90d: 4820, with_baseline_count: 3180, median_baseline_monthly_paise: 900000, median_current_monthly_paise: 1270000, median_uplift_pct: 41.1, platform_income_paise_90d: 500000000, offline_income_paise_90d: 720000000, fair_income_paise_90d: 230000000 },
  { group: 'SC', state_code: 'IN-RJ', suppressed: false, artisan_count: 9210, active_sellers_90d: 3010, with_baseline_count: 1980, median_baseline_monthly_paise: 780000, median_current_monthly_paise: 1080000, median_uplift_pct: 38.5, platform_income_paise_90d: 300000000, offline_income_paise_90d: 430000000, fair_income_paise_90d: 140000000 },
  { group: 'ST', state_code: 'IN-RJ', suppressed: false, artisan_count: 4210, active_sellers_90d: 1280, with_baseline_count: 820, median_baseline_monthly_paise: 720000, median_current_monthly_paise: 990000, median_uplift_pct: 37.5, platform_income_paise_90d: 120000000, offline_income_paise_90d: 180000000, fair_income_paise_90d: 60000000 },
  { group: 'EWS', state_code: 'IN-RJ', suppressed: false, artisan_count: 1240, active_sellers_90d: 380, with_baseline_count: 240, median_baseline_monthly_paise: 830000, median_current_monthly_paise: 1140000, median_uplift_pct: 37.3, platform_income_paise_90d: 38000000, offline_income_paise_90d: 52000000, fair_income_paise_90d: 18000000 },
];

export const SALES_MIX: SalesMixMonth[] = [
  { month: '2026-03', platform_paise: 82000000, fair_paise: 46000000, other_offline_paise: 12000000, suppressed: false },
  { month: '2026-04', platform_paise: 87000000, fair_paise: 51000000, other_offline_paise: 13000000, suppressed: false },
  { month: '2026-05', platform_paise: 91000000, fair_paise: 56000000, other_offline_paise: 14000000, suppressed: false },
  { month: '2026-06', platform_paise: 79000000, fair_paise: 61000000, other_offline_paise: 12000000, suppressed: false },
  { month: '2026-07', platform_paise: 96000000, fair_paise: 48000000, other_offline_paise: 17000000, suppressed: false },
  { month: '2026-08', platform_paise: 99000000, fair_paise: 53000000, other_offline_paise: 15000000, suppressed: false },
];

export const FINANCE_COVERAGE: FinanceCoverageRow[] = [
  { corporation: 'NSFDC', suppressed: false, beneficiaries: 620, verified: 388, self_reported: 214, active_sellers_90d: 501, median_coverage_ratio: 0.62 },
  { corporation: 'NBCFDC', suppressed: false, beneficiaries: 512, verified: 302, self_reported: 187, active_sellers_90d: 409, median_coverage_ratio: 0.58 },
  { corporation: 'NSKFDC', suppressed: false, beneficiaries: 348, verified: 211, self_reported: 119, active_sellers_90d: 274, median_coverage_ratio: 0.61 },
  { corporation: 'NDFDC', suppressed: false, beneficiaries: 256, verified: 146, self_reported: 93, active_sellers_90d: 198, median_coverage_ratio: 0.57 },
  { corporation: 'PM_DAKSH', suppressed: false, beneficiaries: 332, verified: 189, self_reported: 121, active_sellers_90d: 261, median_coverage_ratio: 0.6 },
  { corporation: 'PM_AJAY', suppressed: false, beneficiaries: 184, verified: 108, self_reported: 64, active_sellers_90d: 139, median_coverage_ratio: 0.55 },
  { corporation: 'OTHER', suppressed: false, beneficiaries: 96, verified: 72, self_reported: 24, active_sellers_90d: 80, median_coverage_ratio: 0.74 },
];

export const LITERACY_FUNNEL: LiteracyFunnelRow[] = [
  { state_code: 'IN-GJ', district: 'Kutch', suppressed: false, artisans: 2140, started: 1560, half_way: 980, certified: 720 },
  { state_code: 'IN-RJ', district: 'Jaipur', suppressed: false, artisans: 1860, started: 1340, half_way: 840, certified: 610 },
  { state_code: 'IN-UP', district: 'Varanasi', suppressed: false, artisans: 1620, started: 1150, half_way: 700, certified: 520 },
  { state_code: 'IN-RJ', district: 'Bhilwara', suppressed: false, artisans: 980, started: 720, half_way: 430, certified: 310 },
  { state_code: 'IN-KA', district: 'Channapatna', suppressed: false, artisans: 1130, started: 810, half_way: 510, certified: 380 },
];

export const FINANCE_LINKS_FOR_REVIEW: FinanceLinkForReview[] = [
  { id: 'fl-demo-001', artisan_id: 'art-demo-1001', artisan_name: 'Sunita Devi', district: 'Varanasi', state_code: 'IN-UP', corporation: 'NSFDC', channelizing_agency: 'Varanasi Zila Gramin Bank', reference_last4: '4821', sanctioned_paise: 1200000, emi_paise: 126000, emi_day_of_month: 7, status: 'SELF_REPORTED', created_at: '2026-09-18T09:30:00.000Z' },
  { id: 'fl-demo-002', artisan_id: 'art-demo-1002', artisan_name: 'Rashida Bibi', district: 'Kutch', state_code: 'IN-GJ', corporation: 'NBCFDC', channelizing_agency: 'State Bank of India', reference_last4: '7740', sanctioned_paise: 2000000, emi_paise: 208000, emi_day_of_month: 12, status: 'SELF_REPORTED', created_at: '2026-09-19T11:45:00.000Z' },
  { id: 'fl-demo-003', artisan_id: 'art-demo-1003', artisan_name: 'Ganesh Prajapati', district: 'Salem', state_code: 'IN-TN', corporation: 'NDFDC', channelizing_agency: 'District Co-operative Bank', reference_last4: '3192', sanctioned_paise: 800000, emi_paise: 72000, emi_day_of_month: 3, status: 'SELF_REPORTED', created_at: '2026-09-20T08:15:00.000Z' },
  { id: 'fl-demo-004', artisan_id: 'art-demo-1004', artisan_name: 'Meena Bai Sahu', district: 'Jaipur', state_code: 'IN-RJ', corporation: 'PM_DAKSH', channelizing_agency: 'Punjab National Bank', reference_last4: '9058', sanctioned_paise: 1500000, emi_paise: 148000, emi_day_of_month: 15, status: 'SELF_REPORTED', created_at: '2026-09-21T14:20:00.000Z' },
  { id: 'fl-demo-005', artisan_id: 'art-demo-1005', artisan_name: 'Kuldeep Singh Rathore', district: 'Bhadohi', state_code: 'IN-UP', corporation: 'NSKFDC', channelizing_agency: 'Bank of Baroda', reference_last4: '2217', sanctioned_paise: 2500000, emi_paise: 264000, emi_day_of_month: 20, status: 'SELF_REPORTED', created_at: '2026-09-22T10:05:00.000Z' },
];

export const AGENT_PRODUCTIVITY: AgentProductivity[] = [
  { agent_id: 'agent-demo-001', display_name: 'Sunita Chouhan', role: 'FIELD_AGENT', state_code: 'IN-RJ', district: 'Bhilwara', csc_id: 'RJ-BHW-014', active: true, artisans_onboarded: 142, listings_created: 96, last_active_at: '2026-09-22T11:00:00.000Z' },
  { agent_id: 'agent-demo-002', display_name: 'Prakash Meena', role: 'FIELD_AGENT', state_code: 'IN-RJ', district: 'Dausa', csc_id: 'RJ-DAU-021', active: true, artisans_onboarded: 118, listings_created: 74, last_active_at: '2026-09-21T09:40:00.000Z' },
  { agent_id: 'agent-demo-003', display_name: 'Arif Khan', role: 'FIELD_AGENT', state_code: 'IN-UP', district: 'Varanasi', csc_id: 'UP-VAR-009', active: true, artisans_onboarded: 97, listings_created: 61, last_active_at: '2026-09-22T15:30:00.000Z' },
  { agent_id: 'agent-demo-004', display_name: 'Kavita Rathore', role: 'FIELD_AGENT', state_code: 'IN-GJ', district: 'Kutch', csc_id: 'GJ-KUT-033', active: true, artisans_onboarded: 133, listings_created: 88, last_active_at: '2026-09-20T16:10:00.000Z' },
  { agent_id: 'agent-demo-005', display_name: 'Manoj Bishnoi', role: 'FIELD_AGENT', state_code: 'IN-RJ', district: 'Jodhpur', csc_id: 'RJ-JOD-018', active: true, artisans_onboarded: 76, listings_created: 45, last_active_at: '2026-09-19T12:50:00.000Z' },
];

export const LINKS_FOR_REVIEW: LinkForReview[] = [
  { link_id: 'link-demo-001', agent_id: 'agent-demo-001', agent_name: 'Sunita Chouhan', artisan_id: 'art-demo-2001', artisan_name: 'Ramesh Choudhary', district: 'Bhilwara', state_code: 'IN-RJ', voice_media_id: '', created_at: '2026-09-21T12:15:00.000Z' },
  { link_id: 'link-demo-002', agent_id: 'agent-demo-004', agent_name: 'Kavita Rathore', artisan_id: 'art-demo-2002', artisan_name: 'Gulab Jani', district: 'Kutch', state_code: 'IN-GJ', voice_media_id: '', created_at: '2026-09-21T13:45:00.000Z' },
  { link_id: 'link-demo-003', agent_id: 'agent-demo-003', agent_name: 'Arif Khan', artisan_id: 'art-demo-2003', artisan_name: 'Sabira Khatoon', district: 'Varanasi', state_code: 'IN-UP', voice_media_id: '', created_at: '2026-09-22T08:30:00.000Z' },
  { link_id: 'link-demo-004', agent_id: 'agent-demo-002', agent_name: 'Prakash Meena', artisan_id: 'art-demo-2004', artisan_name: 'Laxmi Devi', district: 'Dausa', state_code: 'IN-RJ', voice_media_id: '', created_at: '2026-09-22T10:20:00.000Z' },
];

export const STAFF_ACCOUNTS: StaffAccount[] = [
  { id: 'staff-demo-001', phone_e164: '+919876123401', display_name: 'Anita Sharma', role: 'FIELD_AGENT', state_code: 'IN-RJ', district: 'Bhilwara', csc_id: 'RJ-BHW-014', cluster_id: undefined, active: true, created_at: '2026-05-02T08:00:00.000Z' },
  { id: 'staff-demo-002', phone_e164: '+919876123402', display_name: 'Ramesh Yadav', role: 'CLUSTER_OFFICER', state_code: 'IN-UP', district: 'Varanasi', csc_id: undefined, cluster_id: 'cl-vns-01', active: true, created_at: '2026-05-11T08:30:00.000Z' },
  { id: 'staff-demo-003', phone_e164: '+919876123403', display_name: 'Naseem Bano', role: 'FIELD_AGENT', state_code: 'IN-GJ', district: 'Kutch', csc_id: 'GJ-KUT-033', cluster_id: undefined, active: true, created_at: '2026-06-03T09:00:00.000Z' },
  { id: 'staff-demo-004', phone_e164: '+919876123404', display_name: 'Deepak Kumar', role: 'FIELD_AGENT', state_code: 'IN-RJ', district: 'Jodhpur', csc_id: 'RJ-JOD-018', cluster_id: undefined, active: false, created_at: '2026-07-19T10:15:00.000Z' },
  { id: 'staff-demo-005', phone_e164: '+919876123405', display_name: 'Shobha Devi', role: 'MINISTRY', state_code: 'IN-DL', district: 'New Delhi', csc_id: undefined, cluster_id: undefined, active: true, created_at: '2026-08-01T09:45:00.000Z' },
];

export const BADGE_CATALOG: Badge[] = [
  { id: 'badge-va', code: 'verified_artisan', kind: 'CONFERRED', tier: 'BRONZE', icon_name: 'verified-artisan', sort_order: 10 },
  { id: 'badge-mc', code: 'master_craftsperson', kind: 'CONFERRED', tier: 'SILVER', icon_name: 'award', sort_order: 20 },
  { id: 'badge-na', code: 'national_awardee', kind: 'CONFERRED', tier: 'GOLD', icon_name: 'medal', sort_order: 30 },
  { id: 'badge-gi', code: 'gi_practitioner', kind: 'CONFERRED', icon_name: 'gi-tagged', sort_order: 40 },
  { id: 'badge-cc', code: 'cluster_coordinator', kind: 'CONFERRED', icon_name: 'cluster', sort_order: 50 },
  { id: 'badge-fl', code: 'first_listing', kind: 'EARNED', metric: 'LISTINGS_PUBLISHED', threshold: 1, icon_name: 'package', sort_order: 60 },
  { id: 'badge-cb', code: 'catalog_builder', kind: 'EARNED', metric: 'LISTINGS_PUBLISHED', threshold: 10, icon_name: 'stack', sort_order: 70 },
  { id: 'badge-pk', code: 'provenance_keeper', kind: 'EARNED', metric: 'PROVENANCE_SEALED', threshold: 5, icon_name: 'seal', sort_order: 80 },
];

export const GRANTED_BADGES: ArtisanBadge[] = [
  { badge: BADGE_CATALOG[0], granted_at: '2026-06-14T10:00:00.000Z', granted_by: 'MINISTRY_OFFICER_01' },
  { badge: BADGE_CATALOG[1], granted_at: '2026-08-02T09:00:00.000Z', granted_by: 'MINISTRY_OFFICER_01' },
  { badge: BADGE_CATALOG[5], granted_at: '2026-05-30T18:00:00.000Z', granted_by: 'system' },
];

export const GOVERNMENT_SCHEMES: GovernmentScheme[] = [
  { id: 'scheme-pmv', code: 'pm_vishwakarma', authority: 'CENTRAL', ministry: 'Ministry of Micro, Small & Medium Enterprises', official_url: 'https://pmvishwakarma.gov.in', name_text: 'PM Vishwakarma Scheme', summary_text: 'Skilling, toolkit loans and digital onboarding for traditional artisans and craftspeople.', sort_order: 10 },
  { id: 'scheme-pmegp', code: 'pmegp', authority: 'CENTRAL', ministry: 'Ministry of Micro, Small & Medium Enterprises', official_url: 'https://www.kviconline.gov.in/pmegpeportal/pmegphome/index.jsp', name_text: "Prime Minister's Employment Generation Programme (PMEGP)", summary_text: 'Margin money subsidy for new craft and micro-enterprises set up by artisans.', sort_order: 20 },
  { id: 'scheme-mudra', code: 'pradhan_mantri_mudra_yojana', authority: 'CENTRAL', ministry: 'Ministry of Finance', official_url: 'https://www.mudra.org.in', name_text: 'Pradhan Mantri MUDRA Yojana', summary_text: 'Collateral-free loans up to ₹10 lakh for non-corporate micro and small businesses, including craft enterprises.', sort_order: 30 },
  { id: 'scheme-nhd', code: 'national_handloom_dev_programme', authority: 'CENTRAL', ministry: 'Ministry of Textiles', official_url: 'https://handlooms.textiles.gov.in', name_text: 'National Handloom Development Programme', summary_text: 'Cluster development, infrastructure and welfare support for handloom weavers.', sort_order: 40 },
  { id: 'scheme-ahvy', code: 'ambedkar_hastshilp_vikas_yojana', authority: 'CENTRAL', ministry: 'Ministry of Textiles', official_url: 'https://www.handicrafts.nic.in', name_text: 'Ambedkar Hastshilp Vikas Yojana', summary_text: 'Comprehensive handicrafts cluster development covering infrastructure, design intervention and marketing support.', sort_order: 50 },
];