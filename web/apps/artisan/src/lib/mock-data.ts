/**
 * apps/artisan/src/lib/mock-data.ts
 *
 * Rich offline/mock data fixtures for Artisan App:
 * - Government Scheme Matches categorized caste-wise (SC, ST, OBC, General)
 * - Loan & EMI Repayment Coverage and Finance Links (MoSJE corporations)
 * - Artisan Income Summary & Economic Uplift calculations
 */

import type { components } from '@kalakriti/api';
import { ESHAAN_MOCK_ORDERS } from './sih-demo-store';

type IncomeSummary = components['schemas']['IncomeSummary'];
type FinanceLink = components['schemas']['FinanceLink'];
type RepaymentCoverage = components['schemas']['RepaymentCoverage'];
type SchemeMatch = components['schemas']['SchemeMatch'];

/* ==========================================================================
   1. INCOME SUMMARY (Artisan Economic Uplift)
   ========================================================================== */

export const MOCK_INCOME_SUMMARY: IncomeSummary = {
  baseline_monthly_paise: 1200000, // ₹12,000 baseline before Kalakriti
  months: [
    {
      month: '2026-06',
      platform_paise: 2450000,
      offline_paise: 800000,
      fair_paise: 600000,
      total_paise: 3250000,
    },
    {
      month: '2026-07',
      platform_paise: 2900000,
      offline_paise: 750000,
      fair_paise: 500000,
      total_paise: 3650000,
    },
    {
      month: '2026-08',
      platform_paise: 3850000,
      offline_paise: 1100000,
      fair_paise: 800000,
      total_paise: 4950000,
    },
    {
      month: '2026-09',
      platform_paise: 4400000,
      offline_paise: 950000,
      fair_paise: 650000,
      total_paise: 5350000,
    },
  ],
  platform_paise_90d: 11150000,
  offline_paise_90d: 2800000,
  fair_paise_90d: 1950000,
  platform_pending_paise: 350000,
  current_monthly_paise: 5350000, // ₹53,500 current month
  uplift_pct: 142, // +142% uplift over baseline
  insufficient_data: '',
  digital_share_pct: 82,
};

const eshaanCompletedSalesByMonth = new Map<string, number>();
for (const order of ESHAAN_MOCK_ORDERS) {
  if (order.state !== 'COMPLETED') continue;
  const month = order.placedAt.slice(0, 7);
  eshaanCompletedSalesByMonth.set(
    month,
    (eshaanCompletedSalesByMonth.get(month) ?? 0) + order.totalPaise,
  );
}

const eshaanIncomeMonths = ['2026-06', '2026-07', '2026-08', '2026-09'].map((month) => {
  const platformPaise = eshaanCompletedSalesByMonth.get(month) ?? 0;
  return {
    month,
    platform_paise: platformPaise,
    offline_paise: 0,
    fair_paise: 0,
    total_paise: platformPaise,
  };
});

const eshaanRecentSalesPaise = eshaanIncomeMonths
  .slice(-3)
  .reduce((total, month) => total + month.platform_paise, 0);

export const MOCK_ESHAAN_INCOME_SUMMARY: IncomeSummary = {
  baseline_monthly_paise: 1200000,
  months: eshaanIncomeMonths,
  platform_paise_90d: eshaanRecentSalesPaise,
  offline_paise_90d: 0,
  fair_paise_90d: 0,
  platform_pending_paise: ESHAAN_MOCK_ORDERS
    .filter((order) => order.state === 'IN_PROGRESS')
    .reduce((total, order) => total + order.totalPaise, 0),
  current_monthly_paise: eshaanIncomeMonths[eshaanIncomeMonths.length - 1].platform_paise,
  uplift_pct: Math.round(
    ((eshaanIncomeMonths[eshaanIncomeMonths.length - 1].platform_paise - 1200000) / 1200000) * 100,
  ),
  insufficient_data: '',
  digital_share_pct: 100,
};

/* ==========================================================================
   2. FINANCE & EMI REPAYMENT COVERAGE
   ========================================================================== */

export const MOCK_FINANCE_LINKS: FinanceLink[] = [
  {
    id: 'link-nsfdc-01',
    corporation: 'NSFDC',
    channelizing_agency: 'State Scheduled Castes Development Corporation',
    reference_last4: '8821',
    sanctioned_paise: 20000000, // ₹2,00,000 loan
    emi_paise: 385000, // ₹3,850 / month
    emi_day_of_month: 10,
    repayment_start: '2026-01-10',
    status: 'VERIFIED',
    verified_at: '2026-01-15T10:00:00Z',
    consent_at: '2026-01-05T08:00:00Z',
    consent_version: 'finance-consent-2026-09',
    created_at: '2026-01-05T08:00:00Z',
  },
  {
    id: 'link-nbcfdc-02',
    corporation: 'NBCFDC',
    channelizing_agency: 'Backward Classes Welfare Corporation',
    reference_last4: '4192',
    sanctioned_paise: 15000000, // ₹1,50,000 loan
    emi_paise: 275000, // ₹2,750 / month
    emi_day_of_month: 15,
    repayment_start: '2026-03-15',
    status: 'SELF_REPORTED',
    consent_at: '2026-03-01T08:00:00Z',
    consent_version: 'finance-consent-2026-09',
    created_at: '2026-03-01T08:00:00Z',
  },
];

export const MOCK_REPAYMENT_COVERAGE: RepaymentCoverage = {
  month: '2026-09',
  emi_paise: 385000, // ₹3,850 EMI
  earned_platform_paise: 4400000, // ₹44,000 platform sales
  earned_offline_paise: 950000, // ₹9,500 offline sales
  pending_paise: 0,
  coverage_ratio: 13.8, // 13.8x coverage
  days_to_emi: 8,
  status: 'COVERED',
  any_verified: true,
};

/* ==========================================================================
   3. GOVERNMENT SCHEME MATCHES (Caste-wise & Ministry-aligned)
   ========================================================================== */

export interface CasteCategorizedSchemeMatch extends SchemeMatch {
  caste_category: 'SC' | 'ST' | 'OBC' | 'ALL';
  caste_label: string;
}

export const MOCK_SCHEME_MATCHES: CasteCategorizedSchemeMatch[] = [
  // ---- SC (Scheduled Caste) Schemes ----
  {
    caste_category: 'SC',
    caste_label: 'Scheduled Caste (SC)',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-nsfdc-term',
      code: 'nsfdc_term_loan',
      authority: 'CENTRAL',
      ministry: 'Ministry of Social Justice & Empowerment',
      official_url: 'https://nsfdc.nic.in',
      name_text: 'NSFDC Term Loan & Mahila Samriddhi Yojana',
      summary_text: 'Concessional loans up to ₹5,00,000 at 4% p.a. for Scheduled Caste traditional artisans, handloom weavers, and craftswomen.',
      sort_order: 10,
    },
    matched_criteria_labels: [
      'Social Category: Scheduled Caste (SC)',
      'Registered Traditional Artisan',
      'Valid Pehchan / Vishwakarma ID',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Family income certificate from State Channelising Agency (within limit)' },
      { check_text: 'Aadhaar-linked bank account ready for direct benefit transfer' },
    ],
  },
  {
    caste_category: 'SC',
    caste_label: 'Scheduled Caste (SC)',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-pm-ajay',
      code: 'pm_ajay',
      authority: 'CENTRAL',
      ministry: 'Ministry of Social Justice & Empowerment',
      summary_i18n_key: 'scheme.pm_ajay.summary',
      name_i18n_key: 'scheme.pm_ajay.name',
      name_text: 'Pradhan Mantri Anusuchit Jaati Abhyuday Yojana (PM-AJAY)',
      summary_text: 'Income-generating grants and cluster skill development for Scheduled Caste artisan settlements and cooperative societies.',
      official_url: 'https://pmajay.dosje.gov.in',
      sort_order: 15,
    },
    matched_criteria_labels: [
      'Social Category: Scheduled Caste (SC)',
      'Artisan Cluster Member',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Belongs to an SC-majority habitation or village cluster' },
    ],
  },
  {
    caste_category: 'SC',
    caste_label: 'Scheduled Caste (SC)',
    status: 'CHECK_REQUIRED',
    scheme: {
      id: 'scheme-ahvy-sc',
      code: 'ambedkar_hastshilp_vikas_yojana',
      authority: 'CENTRAL',
      ministry: 'Ministry of Textiles (DC Handicrafts)',
      name_text: 'Ambedkar Hastshilp Vikas Yojana (AHVY)',
      summary_text: 'Cluster empowerment scheme offering technical design workshops, common facility centers, and market linkages for SC artisan clusters.',
      official_url: 'https://www.handicrafts.nic.in',
      sort_order: 20,
    },
    matched_criteria_labels: [
      'Practicing Traditional Handicraft',
      'Artisan SHG / Cluster Registered',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Confirm active Self-Help Group (SHG) or Cooperative membership' },
      { check_text: 'Pehchan artisan card issued by DC Handicrafts' },
    ],
  },

  // ---- OBC (Other Backward Classes) Schemes ----
  {
    caste_category: 'OBC',
    caste_label: 'Other Backward Classes (OBC)',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-nbcfdc-swarnima',
      code: 'nbcfdc_swarnima',
      authority: 'CENTRAL',
      ministry: 'Ministry of Social Justice & Empowerment',
      summary_i18n_key: 'scheme.nbcfdc.summary',
      name_text: 'NBCFDC Swarnima Scheme for Women & Artisans',
      summary_text: 'Term loans up to ₹2,00,000 at 5% per annum for backward class women entrepreneurs and craftspersons with no collateral required.',
      official_url: 'https://nbcfdc.gov.in',
      sort_order: 30,
    },
    matched_criteria_labels: [
      'Social Category: Backward Classes (OBC)',
      'Verified Micro-Craft Enterprise',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'OBC Certificate issued by competent revenue authority' },
      { check_text: 'Family income under ₹3.00 Lakh per annum' },
    ],
  },
  {
    caste_category: 'OBC',
    caste_label: 'Other Backward Classes (OBC)',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-pm-daksh-obc',
      code: 'pm_daksh',
      authority: 'CENTRAL',
      ministry: 'Ministry of Social Justice & Empowerment',
      summary_i18n_key: 'scheme.pm_daksh.summary',
      name_i18n_key: 'scheme.pm_daksh.name',
      name_text: 'PM-DAKSH (Pradhan Mantri Dakshta Aur Kushalta Sampann Hitgrahi)',
      summary_text: 'Free upskilling, master craftsmanship training, and financial stipend (₹1,500/month) for OBC, SC, and nomadic craftspeople.',
      official_url: 'https://pmdaksh.dosje.gov.in',
      sort_order: 35,
    },
    matched_criteria_labels: [
      'OBC / SC / DNT Artisan Category',
      'Age between 18 and 45 years',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Valid caste category certificate or self-declaration' },
    ],
  },

  // ---- ST (Scheduled Tribe) Schemes ----
  {
    caste_category: 'ST',
    caste_label: 'Scheduled Tribe (ST)',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-nstfdc-term',
      code: 'nstfdc_term_loan',
      authority: 'CENTRAL',
      ministry: 'Ministry of Tribal Affairs',
      name_text: 'NSTFDC Adivasi Mahila Sashaktikaran Yojana & Term Loan',
      summary_text: 'Concessional finance up to ₹2,00,000 at 4% p.a. for Scheduled Tribe artisans, bamboo weavers, terracotta and metal casters.',
      official_url: 'https://nstfdc.tribal.gov.in',
      sort_order: 40,
    },
    matched_criteria_labels: [
      'Social Category: Scheduled Tribe (ST)',
      'Traditional Forest / Tribal Craft Artisan',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Scheduled Tribe (ST) caste verification certificate' },
      { check_text: 'Recommendation through State Tribal Development Corporation' },
    ],
  },
  {
    caste_category: 'ST',
    caste_label: 'Scheduled Tribe (ST)',
    status: 'CHECK_REQUIRED',
    scheme: {
      id: 'scheme-trifed-vvdk',
      code: 'trifed_van_dhan',
      authority: 'CENTRAL',
      ministry: 'Ministry of Tribal Affairs (TRIFED)',
      name_text: 'TRIFED Pradhan Mantri Van Dhan Yojana',
      summary_text: 'Value addition, common facility tooling and direct retail market access for tribal craftsperson self-help groups.',
      official_url: 'https://trifed.tribal.gov.in',
      sort_order: 45,
    },
    matched_criteria_labels: [
      'Tribal Gathering & Craft Cluster Member',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Member of a registered Van Dhan Vikas Kendra SHG' },
    ],
  },

  // ---- All Artisans / Ministry / Central Schemes ----
  {
    caste_category: 'ALL',
    caste_label: 'All Artisans & Crafts',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-pmv',
      code: 'pm_vishwakarma',
      authority: 'CENTRAL',
      ministry: 'Ministry of Micro, Small & Medium Enterprises',
      official_url: 'https://pmvishwakarma.gov.in',
      name_text: 'PM Vishwakarma Scheme',
      summary_text: 'Skilling verification, ₹15,000 digital toolkit incentive, and collateral-free credit up to ₹3,00,000 at 5% interest rate.',
      sort_order: 5,
    },
    matched_criteria_labels: [
      'Traditional Artisan Category',
      'Hands-on Family Craft Lineage',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Gram Panchayat or Urban Local Body verification complete' },
      { check_text: 'Enrolled under one of the 18 eligible traditional trades' },
    ],
  },
  {
    caste_category: 'ALL',
    caste_label: 'All Artisans & Crafts',
    status: 'MAY_QUALIFY',
    scheme: {
      id: 'scheme-pmegp',
      code: 'pmegp',
      authority: 'CENTRAL',
      ministry: 'Ministry of MSME / KVIC',
      official_url: 'https://www.kviconline.gov.in',
      name_text: "Prime Minister's Employment Generation Programme (PMEGP)",
      summary_text: 'Credit-linked subsidy program providing 25% to 35% margin money subsidy for setting up new micro craft enterprises (higher subsidy for SC/ST/OBC/Women).',
      sort_order: 50,
    },
    matched_criteria_labels: [
      'Age 18+ Years',
      'New Craft Enterprise Project',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Project report prepared with District Industries Centre (DIC) or KVIC' },
    ],
  },
  {
    caste_category: 'ALL',
    caste_label: 'All Artisans & Crafts',
    status: 'CHECK_REQUIRED',
    scheme: {
      id: 'scheme-mudra',
      code: 'pradhan_mantri_mudra_yojana',
      authority: 'CENTRAL',
      ministry: 'Ministry of Finance',
      official_url: 'https://www.mudra.org.in',
      name_text: 'Pradhan Mantri MUDRA Yojana (Shishu & Kishore)',
      summary_text: 'Collateral-free working capital loans up to ₹5,00,000 for raw material purchase, loom upgrade, and exhibition stock building.',
      sort_order: 60,
    },
    matched_criteria_labels: [
      'Operational Micro Craft Business',
    ],
    unmet_criteria_labels: [],
    manual_checks: [
      { check_text: 'Current or savings bank account with active sales record' },
    ],
  },
];
