/**
 * apps/artisan/src/lib/business-tips.ts
 *
 * Daily curated, actionable business, artistic craftsmanship, revenue growth,
 * and financial management advice tailored for Indian artisans.
 */
import type { IconName } from '@kalakriti/icons';

export type TipCategory = 'business' | 'craft' | 'revenue' | 'finance';

export interface BusinessTip {
  id: string;
  category: TipCategory;
  categoryLabel: string;
  title: string;
  text: string;
  icon: IconName;
}

export const BUSINESS_TIPS: readonly BusinessTip[] = [
  // -------------------------------------------------------------
  // 1. Business Strategy & Operations
  // -------------------------------------------------------------
  {
    id: 'biz-packaging',
    category: 'business',
    categoryLabel: 'Business & Orders',
    title: 'Secure Packaging Protects Your Reputation',
    text: 'Wrap delicate crafts with double layers of butter paper in sturdy corrugated boxes. Placing a small handwritten thank-you card inside creates an emotional bond that turns first-time buyers into loyal repeat collectors.',
    icon: 'package',
  },
  {
    id: 'biz-lead-time',
    category: 'business',
    categoryLabel: 'Business & Orders',
    title: 'Quote Safe Delivery Timelines',
    text: 'If a handloom weave or carving takes 6 days, quote 8 days to the buyer. Delivering 2 days early delights customers and earns glowing 5-star ratings, whereas a 1-day delay damages credibility.',
    icon: 'clock',
  },
  {
    id: 'biz-material-pooling',
    category: 'business',
    categoryLabel: 'Business & Orders',
    title: 'Pool Raw Material Purchases',
    text: 'Band together with 3–5 artisan peers in your cluster or SHG to buy brass, silk yarn, organic clay, or dyes in bulk. Ordering together unlocks wholesale distributor rates and saves 15% to 25% on input costs.',
    icon: 'shg',
  },
  {
    id: 'biz-repeat-customers',
    category: 'business',
    categoryLabel: 'Business & Orders',
    title: 'Maintain a Customer Contact List',
    text: 'Keep a record of previous satisfied buyers. Whenever you finish a fresh seasonal batch or introduce a new colorway, share a direct preview. Repeat buyers require zero marketing spend to convert.',
    icon: 'whatsapp',
  },
  {
    id: 'biz-seasonal-prep',
    category: 'business',
    categoryLabel: 'Business & Orders',
    title: 'Prepare for Festivals 60 Days Early',
    text: 'Diwali, Durga Puja, and wedding gifting demand surges rapidly. Stock up on raw materials and start making popular motifs 2 months ahead so you do not lose high-margin festival orders to stockouts.',
    icon: 'calendar',
  },

  // -------------------------------------------------------------
  // 2. Artistic Products & Craft Mastery
  // -------------------------------------------------------------
  {
    id: 'craft-authenticity',
    category: 'craft',
    categoryLabel: 'Artistic Products',
    title: 'Subtle Variations Prove Authenticity',
    text: 'Gentle variations in handloom texture, natural dye shades, or wood grain are not defects—they are the hallmark of authentic human craftsmanship. Highlight these natural characteristics so buyers know your work is genuinely handmade.',
    icon: 'provenance',
  },
  {
    id: 'craft-process-photos',
    category: 'craft',
    categoryLabel: 'Artistic Products',
    title: 'Document the Process of Creation',
    text: 'Include 1–2 photos or a short 10-second video of your hands at the loom, wheel, or carving bench. Customers value and pay significantly more for a craft piece when they see the human skill and time invested in making it.',
    icon: 'process-video',
  },
  {
    id: 'craft-finishing-quality',
    category: 'craft',
    categoryLabel: 'Artistic Products',
    title: 'Flawless Finishing Commands Top Prices',
    text: 'Discerning art collectors always inspect the reverse side, inner seams, and base finish of a craft. Clean edge trimming, smooth bases, and even polishing instantly elevate your piece from casual craft to gallery quality.',
    icon: 'handmade-certified',
  },
  {
    id: 'craft-gi-heritage',
    category: 'craft',
    categoryLabel: 'Artistic Products',
    title: 'Proudly Display Your GI Tag',
    text: 'If your craft has Geographical Indication (GI) heritage, always showcase your certification. Corporate buyers and international collectors actively seek verified GI origin and pay premium rates for authentic cultural lineage.',
    icon: 'gi-tagged',
  },
  {
    id: 'craft-signature-mark',
    category: 'craft',
    categoryLabel: 'Artistic Products',
    title: "Develop a Signature Maker's Mark",
    text: "Include a distinctive stamped mark, border weave, or subtle maker's seal on your finished items. A recognizable signature style builds brand identity and encourages collectors to seek out your creations year after year.",
    icon: 'verified-artisan',
  },

  // -------------------------------------------------------------
  // 3. How to Grow Your Revenue
  // -------------------------------------------------------------
  {
    id: 'rev-value-pricing',
    category: 'revenue',
    categoryLabel: 'Revenue Growth',
    title: 'Always Price Above Your Cost Floor',
    text: 'Calculate: Raw Materials + Hours Worked × Fair Wage + Packaging & Platform Fees. Never discount below this baseline during slow weeks—undervaluing your craft erodes your brand and trains buyers to wait for discounts.',
    icon: 'fair-price',
  },
  {
    id: 'rev-product-bundles',
    category: 'revenue',
    categoryLabel: 'Revenue Growth',
    title: 'Bundle Matching Items into Sets',
    text: 'Pair small complementary items together—like a brass diya with a handcrafted incense holder, or a silk stole with matching fabric pouches. Curated gift sets increase your average order value by 30% to 50%.',
    icon: 'collective-order',
  },
  {
    id: 'rev-limited-editions',
    category: 'revenue',
    categoryLabel: 'Revenue Growth',
    title: 'Introduce Numbered Limited Runs',
    text: 'Create an exclusive batch of 15–20 pieces featuring a rare natural dye or intricate carving, marked "Edition of 20". Scarcity creates genuine buyer excitement and allows you to charge a 25%–40% premium for exclusivity.',
    icon: 'badge-award',
  },
  {
    id: 'rev-customization',
    category: 'revenue',
    categoryLabel: 'Revenue Growth',
    title: 'Offer Custom Names or Sizing',
    text: 'Allow buyers to request custom monogramming, bespoke dimensions, or specific color palettes for home decor and weddings. Customized artisanal pieces easily command a 20% to 40% personalization premium.',
    icon: 'made-to-order',
  },
  {
    id: 'rev-three-tier',
    category: 'revenue',
    categoryLabel: 'Revenue Growth',
    title: 'Maintain an Entry, Core, and Masterpiece Tier',
    text: 'Offer quick impulse items (₹300–₹600) for high sales volume, core regular items (₹1,200–₹3,500) for steady income, and 1–2 flagship masterpieces (₹10,000+) to establish high artistic authority and luxury positioning.',
    icon: 'ready-stock',
  },

  // -------------------------------------------------------------
  // 4. Finance & Capital Management
  // -------------------------------------------------------------
  {
    id: 'fin-separate-accounts',
    category: 'finance',
    categoryLabel: 'Finance & Money',
    title: 'Keep Business and Personal Money Separate',
    text: 'Open a dedicated bank account and UPI QR code strictly for workshop sales and raw material costs. Mixing household grocery spending with craft earnings makes it impossible to know your real profit margin.',
    icon: 'income-statement',
  },
  {
    id: 'fin-order-advance',
    category: 'finance',
    categoryLabel: 'Finance & Money',
    title: 'Secure 40%–50% Advance for Large Orders',
    text: 'Never purchase materials or begin production on custom or bulk orders without an advance payment of at least 40%. This covers your upfront input costs and completely protects you against last-minute cancellations.',
    icon: 'lock',
  },
  {
    id: 'fin-digital-ledger',
    category: 'finance',
    categoryLabel: 'Finance & Money',
    title: 'Record Every Sale Digitally',
    text: 'Accepting UPI and recording offline cash sales in the app creates a verifiable transaction trail. Public banks and NBFCs require 6 months of digital turnover proof to sanction low-interest working capital loans.',
    icon: 'dollar-sign',
  },
  {
    id: 'fin-emergency-fund',
    category: 'finance',
    categoryLabel: 'Finance & Money',
    title: 'Save 10% During Peak Seasons for Lean Months',
    text: 'During high-sales festival months, deposit 10% of your earnings into an emergency reserve. This buffer pays your workshop rent and material deposits during monsoon or post-festival slowdowns without taking high-interest debt.',
    icon: 'income-statement',
  },
  {
    id: 'fin-govt-schemes',
    category: 'finance',
    categoryLabel: 'Finance & Money',
    title: 'Leverage PM Vishwakarma & Mudra Loans',
    text: 'Government initiatives like PM Vishwakarma and Pradhan Mantri Mudra Yojana provide subsidized loans up to ₹3 Lakhs with interest subvention (5%) and toolkits. Avoid local moneylenders who charge predatory daily interest rates.',
    icon: 'verified-artisan',
  },
];

/**
 * Returns a randomized tip from the collection.
 * Passing excludeId ensures a different tip is chosen upon refresh.
 */
export function getRandomTip(excludeId?: string): BusinessTip {
  const pool = excludeId
    ? BUSINESS_TIPS.filter((t) => t.id !== excludeId)
    : BUSINESS_TIPS;
  const randomIndex = Math.floor(Math.random() * pool.length);
  return pool[randomIndex] ?? BUSINESS_TIPS[0];
}
