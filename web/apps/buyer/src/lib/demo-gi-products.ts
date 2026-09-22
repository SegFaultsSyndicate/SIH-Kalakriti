/**
 * apps/buyer/src/lib/demo-gi-products.ts
 *
 * GI directory demo products. Shared with the product page (demo-catalog.ts)
 * so /listing/gi-N resolves instead of 404ing.
 */
import type { MessageKey } from "@kalakriti/i18n";

export interface GIProduct {
  id: string;
  titleKey: MessageKey;
  craftName: string;
  craftNameKey: MessageKey;
  giRegNo: string;
  state: string;
  category: string;
  price: number;
  mrp: number;
  discountPct: number;
  image: string;
  colorHex: string;
  artisanNameKey: MessageKey;
  weavingStyle?: string;
  pattern?: string;
  fabric?: string;
}

export const GI_PRODUCTS: GIProduct[] = [
  {
    id: 'gi-1',
    titleKey: 'giTagged.product.gi-1.title',
    craftName: 'Uttarakhand Aipan Art',
    craftNameKey: 'giTagged.productCraft.aipan',
    giRegNo: 'GI-696',
    state: 'Uttarakhand',
    category: 'Paintings',
    price: 350000,
    mrp: 450000,
    discountPct: 22,
    image: '/craft-images/paintings/category_cover.jpg',
    colorHex: '#8c2323',
    artisanNameKey: 'giTagged.artisan.bhawanaBhatt',
  },
  {
    id: 'gi-2',
    titleKey: 'giTagged.product.gi-2.title',
    craftName: 'Uttarakhand Aipan Art',
    craftNameKey: 'giTagged.productCraft.aipan',
    giRegNo: 'GI-696',
    state: 'Uttarakhand',
    category: 'Paintings',
    price: 250000,
    mrp: 350000,
    discountPct: 29,
    image: '/craft-images/paintings/category_cover.jpg',
    colorHex: '#8c2323',
    artisanNameKey: 'giTagged.artisan.kamlaDevi',
  },
  {
    id: 'gi-3',
    titleKey: 'giTagged.product.gi-3.title',
    craftName: 'Uttarakhand Aipan Art',
    craftNameKey: 'giTagged.productCraft.aipan',
    giRegNo: 'GI-696',
    state: 'Uttarakhand',
    category: 'Paintings',
    price: 600000,
    mrp: 700000,
    discountPct: 14,
    image: '/craft-images/paintings/category_cover.jpg',
    colorHex: '#c68237',
    artisanNameKey: 'giTagged.artisan.deepaJoshi',
  },
  {
    id: 'gi-4',
    titleKey: 'giTagged.product.gi-4.title',
    craftName: 'Uttarakhand Aipan Art',
    craftNameKey: 'giTagged.productCraft.aipan',
    giRegNo: 'GI-696',
    state: 'Uttarakhand',
    category: 'Paintings',
    price: 600000,
    mrp: 700000,
    discountPct: 14,
    image: '/craft-images/paintings/category_cover.jpg',
    colorHex: '#c68237',
    artisanNameKey: 'giTagged.artisan.deepaJoshi',
  },
  {
    id: 'gi-5',
    titleKey: 'giTagged.product.gi-5.title',
    craftName: 'Ajrakh Block Print',
    craftNameKey: 'giTagged.productCraft.ajrakh',
    giRegNo: 'GI-384',
    state: 'Gujarat',
    category: 'Home and living',
    price: 420000,
    mrp: 550000,
    discountPct: 24,
    image: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
    colorHex: '#0033cc',
    artisanNameKey: 'giTagged.artisan.ismailKhatri',
  },
  {
    id: 'gi-6',
    titleKey: 'giTagged.product.gi-6.title',
    craftName: 'Banarasi Brocade',
    craftNameKey: 'giTagged.productCraft.banarasi',
    giRegNo: 'GI-99',
    state: 'Uttar Pradesh',
    category: 'Women',
    price: 1850000,
    mrp: 2400000,
    discountPct: 23,
    image: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
    colorHex: '#e60000',
    artisanNameKey: 'giTagged.artisan.sitaSharma',
  },
  {
    id: 'gi-7',
    titleKey: 'giTagged.product.gi-7.title',
    craftName: 'Kashmir Pashmina',
    craftNameKey: 'giTagged.productCraft.pashmina',
    giRegNo: 'GI-46',
    state: 'Jammu & Kashmir',
    category: 'Women',
    price: 1450000,
    mrp: 1800000,
    discountPct: 19,
    image: '/craft-images/embroidery/kashmir_pashmina_sozni_01.jpeg',
    colorHex: '#fdfcf0',
    artisanNameKey: 'giTagged.artisan.ghulamHassan',
  },
  {
    id: 'gi-8',
    titleKey: 'giTagged.product.gi-8.title',
    craftName: 'Bastar Dhokra',
    craftNameKey: 'giTagged.productCraft.dhokra',
    giRegNo: 'GI-83',
    state: 'Chhattisgarh',
    category: 'Home and living',
    price: 480000,
    mrp: 600000,
    discountPct: 20,
    image: '/craft-images/metalwork/dhokra-casting.jpg',
    colorHex: '#f4c430',
    artisanNameKey: 'giTagged.artisan.mangluGhadwa',
  },
  {
    id: 'gi-9',
    titleKey: 'giTagged.product.gi-9.title',
    craftName: 'Blue Pottery',
    craftNameKey: 'giTagged.productCraft.bluePottery',
    giRegNo: 'GI-180',
    state: 'Rajasthan',
    category: 'Home and living',
    price: 180000,
    mrp: 240000,
    discountPct: 25,
    image: '/craft-images/pottery/nizamabad-black-pottery.jpg',
    colorHex: '#007a78',
    artisanNameKey: 'giTagged.artisan.kripalKumbhar',
  },
  {
    id: 'gi-10',
    titleKey: 'giTagged.product.gi-10.title',
    craftName: 'Madhubani Painting',
    craftNameKey: 'giTagged.productCraft.madhubani',
    giRegNo: 'GI-105',
    state: 'Bihar',
    category: 'Paintings',
    price: 320000,
    mrp: 400000,
    discountPct: 20,
    image: '/craft-images/paintings/madhubani-painting.jpg',
    colorHex: '#ffd200',
    artisanNameKey: 'giTagged.artisan.lakshmiDevi',
  },
  {
    id: 'gi-12',
    titleKey: 'giTagged.product.gi-12.title',
    craftName: 'Patan Patola',
    craftNameKey: 'giTagged.productCraft.patola',
    giRegNo: 'GI-232',
    state: 'Gujarat',
    category: 'Women',
    price: 2600000,
    mrp: 3200000,
    discountPct: 19,
    image: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
    colorHex: '#730000',
    artisanNameKey: 'giTagged.artisan.rohitSalvi',
  },
];
