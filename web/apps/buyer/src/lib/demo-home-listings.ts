/**
 * apps/buyer/src/lib/demo-home-listings.ts
 *
 * Home page fallback listings (shown when the backend has no published
 * stock). Lives here, not inline in routes/+page.svelte, so the product page
 * can resolve these ids too -- see demo-catalog.ts.
 */
import type { components } from "@kalakriti/api";
import type { MessageKey } from "@kalakriti/i18n";

type ListingSummary = components["schemas"]["ListingSummary"];

export interface RawFallbackListing {
  id: string;
  product_id: string;
  artisan_id: string;
  artisanNameKey: MessageKey;
  craftNameKey: MessageKey;
  craft_slug: string;
  craft_gi_registration_no: string;
  gi_certified: boolean;
  artisan_verified: boolean;
  artisan_district: string;
  artisan_state_code: string;
  type: string;
  price: { amount_paise: number; currency_code: string };
  image_url: string;
  titleKey: MessageKey;
  descriptionKey: MessageKey;
}

export function toFallbackListing(
  raw: RawFallbackListing,
  tFn: (k: MessageKey, params?: Record<string, string | number>) => string,
  currentCode: string
): ListingSummary {
  return {
    id: raw.id,
    product_id: raw.product_id,
    artisan_id: raw.artisan_id,
    artisan_name: tFn(raw.artisanNameKey),
    craft_name: tFn(raw.craftNameKey),
    craft_slug: raw.craft_slug,
    craft_gi_registration_no: raw.craft_gi_registration_no,
    gi_certified: raw.gi_certified,
    artisan_verified: raw.artisan_verified,
    artisan_district: raw.artisan_district,
    artisan_state_code: raw.artisan_state_code,
    type: raw.type,
    price: raw.price,
    image_url: raw.image_url,
    translations: [
      {
        language: currentCode,
        title: tFn(raw.titleKey),
        description: tFn(raw.descriptionKey),
      },
    ],
  };
}

export const RAW_FALLBACK_GI_LISTINGS: RawFallbackListing[] = [
  {
    id: 'listing-gi-2',
    product_id: 'prod-pashmina-kani',
    artisan_id: 'artisan-mir',
    artisanNameKey: 'home.fallbackListing.gi2.artisanName',
    craftNameKey: 'home.fallbackListing.gi2.craftName',
    craft_slug: 'pashmina-weaving',
    craft_gi_registration_no: 'GI-46',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Srinagar',
    artisan_state_code: 'JK',
    type: 'READY_STOCK',
    price: { amount_paise: 3800000, currency_code: 'INR' },
    image_url: '/craft-images/embroidery/kashmir_pashmina_sozni_02.jpeg',
    titleKey: 'home.fallbackListing.gi2.title',
    descriptionKey: 'home.fallbackListing.gi2.description',
  },
  {
    id: 'listing-gi-3',
    product_id: 'prod-patola-shikargah',
    artisan_id: 'artisan-salvi',
    artisanNameKey: 'home.fallbackListing.gi3.artisanName',
    craftNameKey: 'home.fallbackListing.gi3.craftName',
    craft_slug: 'patan-patola',
    craft_gi_registration_no: 'GI-232',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Patan',
    artisan_state_code: 'GJ',
    type: 'MADE_TO_ORDER',
    price: { amount_paise: 12000000, currency_code: 'INR' },
    image_url: '/craft-images/weaving_and_looms/zari-work-saree-beige.jpeg',
    titleKey: 'home.fallbackListing.gi3.title',
    descriptionKey: 'home.fallbackListing.gi3.description',
  },
  {
    id: 'listing-gi-4',
    product_id: 'prod-dhokra-nandi',
    artisan_id: 'artisan-budheshwar',
    artisanNameKey: 'home.fallbackListing.gi4.artisanName',
    craftNameKey: 'home.fallbackListing.gi4.craftName',
    craft_slug: 'dhokra-casting',
    craft_gi_registration_no: 'GI-117',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Bastar',
    artisan_state_code: 'CT',
    type: 'READY_STOCK',
    price: { amount_paise: 850000, currency_code: 'INR' },
    image_url: '/craft-images/metalwork/dhokra-casting.jpg',
    titleKey: 'home.fallbackListing.gi4.title',
    descriptionKey: 'home.fallbackListing.gi4.description',
  },
];

export const RAW_FALLBACK_NEW_ARRIVALS: RawFallbackListing[] = [
  {
    id: 'listing-arr-1',
    product_id: 'prod-ajrakh-stole',
    artisan_id: 'artisan-khatri',
    artisanNameKey: 'home.fallbackListing.arr1.artisanName',
    craftNameKey: 'home.fallbackListing.arr1.craftName',
    craft_slug: 'ajrakh-printing',
    craft_gi_registration_no: 'GI-312',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Kutch',
    artisan_state_code: 'GJ',
    type: 'READY_STOCK',
    price: { amount_paise: 420000, currency_code: 'INR' },
    image_url: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
    titleKey: 'home.fallbackListing.arr1.title',
    descriptionKey: 'home.fallbackListing.arr1.description',
  },
  {
    id: 'listing-arr-2',
    product_id: 'prod-black-pottery-handi',
    artisan_id: 'artisan-prajapati',
    artisanNameKey: 'home.fallbackListing.arr2.artisanName',
    craftNameKey: 'home.fallbackListing.arr2.craftName',
    craft_slug: 'nizamabad-black-pottery',
    craft_gi_registration_no: 'GI-264',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Azamgarh',
    artisan_state_code: 'UP',
    type: 'READY_STOCK',
    price: { amount_paise: 320000, currency_code: 'INR' },
    image_url: '/craft-images/pottery/nizamabad-black-pottery.jpg',
    titleKey: 'home.fallbackListing.arr2.title',
    descriptionKey: 'home.fallbackListing.arr2.description',
  },
  {
    id: 'listing-arr-3',
    product_id: 'prod-bidri-vase',
    artisan_id: 'artisan-rashid',
    artisanNameKey: 'home.fallbackListing.arr3.artisanName',
    craftNameKey: 'home.fallbackListing.arr3.craftName',
    craft_slug: 'bidriware',
    craft_gi_registration_no: 'GI-19',
    gi_certified: true,
    artisan_verified: true,
    artisan_district: 'Bidar',
    artisan_state_code: 'KA',
    type: 'READY_STOCK',
    price: { amount_paise: 650000, currency_code: 'INR' },
    image_url: '/craft-images/metalwork/bidriware.jpg',
    titleKey: 'home.fallbackListing.arr3.title',
    descriptionKey: 'home.fallbackListing.arr3.description',
  },
];
