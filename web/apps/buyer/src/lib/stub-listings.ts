/**
 * apps/buyer/src/lib/stub-listings.ts
 *
 * Real photographed craft pieces (artisan-submitted image batch, sorted by
 * category, one .txt caption per photo) standing in for backend listings
 * until real sellers publish real stock -- replaces the old Unsplash stock
 * photos used across the home page, search fallback, and GI directory.
 *
 * `category` matches ARTISAN_CRAFT_CATEGORIES[].name exactly (see
 * craft-categories.ts) because ArtisanCraftGrid links to
 * `/search?category=${craft.name}` -- stubListingsForQuery() below is what
 * makes that click show real pieces instead of an empty-results page.
 *
 * Per I18N_PLAN.md §4.1(b), each item's `titleKey` is a catalogue key (not a
 * hardcoded string) so these fixtures render correctly in all 20 languages
 * even with the backend down. Titles are resolved lazily by the caller's
 * own reactive `t`, not baked in at module load, so a language switch
 * updates them like any other UI text.
 */

import type { components } from '@kalakriti/api';
import { locale, type MessageKey, type MessageValues } from '@kalakriti/i18n';

type ListingSummary = components['schemas']['ListingSummary'];
type TFn = (key: MessageKey, values?: MessageValues) => string;

interface StubItem {
  category: string;
  craftSlug: string;
  craftName: string;
  craftNameKey: MessageKey;
  giNo: string;
  artisan: string;
  artisanNameKey: MessageKey;
  district: string;
  state: string;
  titleKey: MessageKey;
  image: string;
  pricePaise: number;
  madeToOrder?: boolean;
}

function img(dir: string, file: string): string {
  return `/craft-images/${dir}/${file}`;
}

const ITEMS: StubItem[] = [
  // ---- Bamboo Craft ----
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', craftNameKey: 'stub.craftName.assamBambooCraft', giNo: 'GI-401', artisan: 'Bhaben Kalita', artisanNameKey: 'stub.artisanName.assamBambooCraft', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.01.title', image: img('bamboo_craft', 'assam-bamboo-craft.jpg'), pricePaise: 285000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', craftNameKey: 'stub.craftName.tripuraBambooCraft', giNo: 'GI-402', artisan: 'Sudip Debbarma', artisanNameKey: 'stub.artisanName.tripuraBambooCraft', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.02.title', image: img('bamboo_craft', 'tripura-bamboo-craft.jpg'), pricePaise: 320000, madeToOrder: true },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', craftNameKey: 'stub.craftName.assamBambooCraft', giNo: 'GI-401', artisan: 'Bhaben Kalita', artisanNameKey: 'stub.artisanName.assamBambooCraft', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.03.title', image: img('bamboo_craft', 'assam_jaapi_structural_weaving_01.jpeg'), pricePaise: 195000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', craftNameKey: 'stub.craftName.assamBambooCraft', giNo: 'GI-401', artisan: 'Bhaben Kalita', artisanNameKey: 'stub.artisanName.assamBambooCraft', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.04.title', image: img('bamboo_craft', 'bamboo_craft_mug_01.jpeg'), pricePaise: 65000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', craftNameKey: 'stub.craftName.assamBambooCraft', giNo: 'GI-401', artisan: 'Bhaben Kalita', artisanNameKey: 'stub.artisanName.assamBambooCraft', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.05.title', image: img('bamboo_craft', 'kerala_bamboo_reed_ply_mat_craft_01.jpeg'), pricePaise: 220000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', craftNameKey: 'stub.craftName.tripuraBambooCraft', giNo: 'GI-402', artisan: 'Sudip Debbarma', artisanNameKey: 'stub.artisanName.tripuraBambooCraft', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.06.title', image: img('bamboo_craft', 'meghalaya_knup_rain_shield_craft_01.jpeg'), pricePaise: 145000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', craftNameKey: 'stub.craftName.tripuraBambooCraft', giNo: 'GI-402', artisan: 'Sudip Debbarma', artisanNameKey: 'stub.artisanName.tripuraBambooCraft', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.07.title', image: img('bamboo_craft', 'tripura_cane_split_bamboo_furniture_01.jpeg'), pricePaise: 480000, madeToOrder: true },

  // ---- Basketry ----
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', craftNameKey: 'stub.craftName.sikkiGrassBasketry', giNo: 'GI-403', artisan: 'Droupadi Devi', artisanNameKey: 'stub.artisanName.sikkiGrassBasketry', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.08.title', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 145000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', craftNameKey: 'stub.craftName.sabaiGrassBasketry', giNo: 'GI-404', artisan: 'Shanti Soren', artisanNameKey: 'stub.artisanName.sabaiGrassBasketry', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.09.title', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 95000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', craftNameKey: 'stub.craftName.sikkiGrassBasketry', giNo: 'GI-403', artisan: 'Droupadi Devi', artisanNameKey: 'stub.artisanName.sikkiGrassBasketry', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.10.title', image: img('basketry', 'assamese_cane_bamboo_basketry_01.jpeg'), pricePaise: 78000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', craftNameKey: 'stub.craftName.sikkiGrassBasketry', giNo: 'GI-403', artisan: 'Droupadi Devi', artisanNameKey: 'stub.artisanName.sikkiGrassBasketry', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.11.title', image: img('basketry', 'kottan_palm_leaf_basketry_01.jpeg'), pricePaise: 62000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', craftNameKey: 'stub.craftName.sabaiGrassBasketry', giNo: 'GI-404', artisan: 'Shanti Soren', artisanNameKey: 'stub.artisanName.sabaiGrassBasketry', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.12.title', image: img('basketry', 'mizoram_thul_basketry_traditions_01.jpeg'), pricePaise: 88000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', craftNameKey: 'stub.craftName.sabaiGrassBasketry', giNo: 'GI-404', artisan: 'Shanti Soren', artisanNameKey: 'stub.artisanName.sabaiGrassBasketry', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.13.title', image: img('basketry', 'moonj_grass_basketry_01.jpeg'), pricePaise: 54000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', craftNameKey: 'stub.craftName.sikkiGrassBasketry', giNo: 'GI-403', artisan: 'Droupadi Devi', artisanNameKey: 'stub.artisanName.sikkiGrassBasketry', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.14.title', image: img('basketry', 'sarkanda_chick_basketry_01.jpeg'), pricePaise: 68000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', craftNameKey: 'stub.craftName.sabaiGrassBasketry', giNo: 'GI-404', artisan: 'Shanti Soren', artisanNameKey: 'stub.artisanName.sabaiGrassBasketry', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.15.title', image: img('basketry', 'thana_tribal_grass_basketry_01.jpeg'), pricePaise: 72000 },

  // ---- Block Printing ----
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', craftNameKey: 'stub.craftName.ajrakhBlockPrinting', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', artisanNameKey: 'stub.artisanName.ajrakhBlockPrinting', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.16.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_01.jpeg'), pricePaise: 465000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', craftNameKey: 'stub.craftName.ajrakhBlockPrinting', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', artisanNameKey: 'stub.artisanName.ajrakhBlockPrinting', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.17.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_02.jpeg'), pricePaise: 445000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', craftNameKey: 'stub.craftName.ajrakhBlockPrinting', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', artisanNameKey: 'stub.artisanName.ajrakhBlockPrinting', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.18.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_03.jpeg'), pricePaise: 398000 },

  // ---- Embroidery ----
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', craftNameKey: 'stub.craftName.kashmirPashminaSozni', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', artisanNameKey: 'stub.artisanName.kashmirPashminaSozni', district: 'Srinagar', state: 'JK', titleKey: 'stub.listing.19.title', image: img('embroidery', 'kashmir_pashmina_sozni_01.jpeg'), pricePaise: 950000, madeToOrder: true },
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', craftNameKey: 'stub.craftName.kashmirPashminaSozni', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', artisanNameKey: 'stub.artisanName.kashmirPashminaSozni', district: 'Srinagar', state: 'JK', titleKey: 'stub.listing.20.title', image: img('embroidery', 'kashmir_pashmina_sozni_02.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Furniture (extra images, folded into Woodwork category) ----
  { category: 'Woodwork', craftSlug: 'jodhpur-furniture', craftName: 'Jodhpur Handcrafted Furniture', craftNameKey: 'stub.craftName.jodhpurFurniture', giNo: 'GI-410', artisan: 'Gopal Suthar', artisanNameKey: 'stub.artisanName.jodhpurFurniture', district: 'Jodhpur', state: 'RJ', titleKey: 'stub.listing.21.title', image: img('furniture', 'jodhpur-furniture.jpg'), pricePaise: 1850000, madeToOrder: true },

  // ---- Home & Living (extra images, folded into Weaving / Metalwork) ----
  { category: 'Weaving', craftSlug: 'bhadohi-carpet', craftName: 'Bhadohi Hand-Knotted Carpet', craftNameKey: 'stub.craftName.bhadohiCarpet', giNo: 'GI-411', artisan: 'Mohammad Yasin Ansari', artisanNameKey: 'stub.artisanName.bhadohiCarpet', district: 'Bhadohi', state: 'UP', titleKey: 'stub.listing.22.title', image: img('home_and_living', 'bhadohi-carpet.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Metalwork', craftSlug: 'moradabad-metal', craftName: 'Moradabad Metal Craft', craftNameKey: 'stub.craftName.moradabadMetal', giNo: 'GI-412', artisan: 'Rameshwar Prasad Verma', artisanNameKey: 'stub.artisanName.moradabadMetal', district: 'Moradabad', state: 'UP', titleKey: 'stub.listing.23.title', image: img('home_and_living', 'moradabad-metal.jpg'), pricePaise: 185000 },

  // ---- Jewellery ----
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', craftNameKey: 'stub.craftName.jaipurMeenakari', giNo: 'GI-414', artisan: 'Mahesh Soni', artisanNameKey: 'stub.artisanName.jaipurMeenakari', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.24.title', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 620000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', craftNameKey: 'stub.craftName.jaipurMeenakari', giNo: 'GI-414', artisan: 'Mahesh Soni', artisanNameKey: 'stub.artisanName.jaipurMeenakari', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.25.title', image: img('jewellery', 'jadau_jewellery_01.jpeg'), pricePaise: 1250000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', craftNameKey: 'stub.craftName.jaipurMeenakari', giNo: 'GI-414', artisan: 'Mahesh Soni', artisanNameKey: 'stub.artisanName.jaipurMeenakari', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.26.title', image: img('jewellery', 'kundankari_jewellery_01.jpeg'), pricePaise: 980000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', craftNameKey: 'stub.craftName.jaipurMeenakari', giNo: 'GI-414', artisan: 'Mahesh Soni', artisanNameKey: 'stub.artisanName.jaipurMeenakari', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.27.title', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 545000 },
  { category: 'Jewellery', craftSlug: 'cuttack-silver-filigree', craftName: 'Cuttack Silver Filigree', craftNameKey: 'stub.craftName.cuttackSilverFiligree', giNo: 'GI-413', artisan: 'Pankaj Kumar Sahoo', artisanNameKey: 'stub.artisanName.cuttackSilverFiligree', district: 'Cuttack', state: 'OD', titleKey: 'stub.listing.28.title', image: img('jewellery', 'tarakashi_silver_filigree_02.jpeg'), pricePaise: 495000 },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', craftNameKey: 'stub.craftName.jaipurMeenakari', giNo: 'GI-414', artisan: 'Mahesh Soni', artisanNameKey: 'stub.artisanName.jaipurMeenakari', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.29.title', image: img('jewellery', 'temple_jewellery_01.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Leatherwork ----
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', craftNameKey: 'stub.craftName.shantiniketanLeather', giNo: 'GI-415', artisan: 'Bikash Karmakar', artisanNameKey: 'stub.artisanName.shantiniketanLeather', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.30.title', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 285000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', craftNameKey: 'stub.craftName.kolhapuriChappal', giNo: 'GI-416', artisan: 'Santosh Satpute', artisanNameKey: 'stub.artisanName.kolhapuriChappal', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.31.title', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 145000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', craftNameKey: 'stub.craftName.kutchLeathercraft', giNo: 'GI-417', artisan: 'Salim Marwada', artisanNameKey: 'stub.artisanName.kutchLeathercraft', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.32.title', image: img('leatherwork', 'jawaja_tanned_leatherwork_01.jpeg'), pricePaise: 185000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', craftNameKey: 'stub.craftName.kolhapuriChappal', giNo: 'GI-416', artisan: 'Santosh Satpute', artisanNameKey: 'stub.artisanName.kolhapuriChappal', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.33.title', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 135000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', craftNameKey: 'stub.craftName.kolhapuriChappal', giNo: 'GI-416', artisan: 'Santosh Satpute', artisanNameKey: 'stub.artisanName.kolhapuriChappal', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.34.title', image: img('leatherwork', 'kolhapuri_chappals_02.jpeg'), pricePaise: 142000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', craftNameKey: 'stub.craftName.kutchLeathercraft', giNo: 'GI-417', artisan: 'Salim Marwada', artisanNameKey: 'stub.artisanName.kutchLeathercraft', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.35.title', image: img('leatherwork', 'kutch_leathercraft_marwada_style_01.jpeg'), pricePaise: 225000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', craftNameKey: 'stub.craftName.kutchLeathercraft', giNo: 'GI-417', artisan: 'Salim Marwada', artisanNameKey: 'stub.artisanName.kutchLeathercraft', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.36.title', image: img('leatherwork', 'mojari_jutti_footwear_01.jpeg'), pricePaise: 165000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', craftNameKey: 'stub.craftName.shantiniketanLeather', giNo: 'GI-415', artisan: 'Bikash Karmakar', artisanNameKey: 'stub.artisanName.shantiniketanLeather', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.37.title', image: img('leatherwork', 'shantiniketan_embossed_leather_01.jpeg'), pricePaise: 245000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', craftNameKey: 'stub.craftName.shantiniketanLeather', giNo: 'GI-415', artisan: 'Bikash Karmakar', artisanNameKey: 'stub.artisanName.shantiniketanLeather', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.38.title', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 98000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', craftNameKey: 'stub.craftName.shantiniketanLeather', giNo: 'GI-415', artisan: 'Bikash Karmakar', artisanNameKey: 'stub.artisanName.shantiniketanLeather', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.39.title', image: img('leatherwork', 'shantiniketan_embossed_leather_03.jpeg'), pricePaise: 74000 },

  // ---- Metalwork ----
  { category: 'Metalwork', craftSlug: 'dhokra-casting', craftName: 'Dhokra Metal Casting', craftNameKey: 'stub.craftName.dhokraCasting', giNo: 'GI-418', artisan: 'Budheshwar Ghadwa', artisanNameKey: 'stub.artisanName.dhokraCasting', district: 'Bastar', state: 'CT', titleKey: 'stub.listing.40.title', image: img('metalwork', 'dhokra-casting.jpg'), pricePaise: 850000 },
  { category: 'Metalwork', craftSlug: 'bidriware', craftName: 'Bidriware Pure Silver Inlay', craftNameKey: 'stub.craftName.bidriware', giNo: 'GI-419', artisan: 'Abdul Rashid Guild', artisanNameKey: 'stub.artisanName.bidriware', district: 'Bidar', state: 'KA', titleKey: 'stub.listing.41.title', image: img('metalwork', 'bidriware.jpg'), pricePaise: 650000, madeToOrder: true },

  // ---- Paintings ----
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', craftNameKey: 'stub.craftName.madhubaniPainting', giNo: 'GI-420', artisan: 'Lakshmi Devi', artisanNameKey: 'stub.artisanName.madhubaniPainting', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.42.title', image: img('paintings', 'madhubani-painting.jpg'), pricePaise: 320000 },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', craftNameKey: 'stub.craftName.pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', artisanNameKey: 'stub.artisanName.pattachitra', district: 'Raghurajpur', state: 'OD', titleKey: 'stub.listing.43.title', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 285000 },
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', craftNameKey: 'stub.craftName.madhubaniPainting', giNo: 'GI-420', artisan: 'Lakshmi Devi', artisanNameKey: 'stub.artisanName.madhubaniPainting', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.44.title', image: img('paintings', 'madhubani_mithila_painting_01.jpeg'), pricePaise: 265000 },
  { category: 'Painting', craftSlug: 'pahari-miniature-painting', craftName: 'Pahari Miniature Painting', craftNameKey: 'stub.craftName.pahariMiniaturePainting', giNo: 'GI-422', artisan: 'Vijay Sharma', artisanNameKey: 'stub.artisanName.pahariMiniaturePainting', district: 'Kangra', state: 'HP', titleKey: 'stub.listing.45.title', image: img('paintings', 'pahari_miniature_painting_01.jpeg'), pricePaise: 385000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', craftNameKey: 'stub.craftName.pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', artisanNameKey: 'stub.artisanName.pattachitra', district: 'Raghurajpur', state: 'OD', titleKey: 'stub.listing.46.title', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 245000 },
  { category: 'Painting', craftSlug: 'tanjore-painting', craftName: 'Tanjore Painting', craftNameKey: 'stub.craftName.tanjorePainting', giNo: 'GI-423', artisan: 'Meenakshi Sundaram', artisanNameKey: 'stub.artisanName.tanjorePainting', district: 'Thanjavur', state: 'TN', titleKey: 'stub.listing.47.title', image: img('paintings', 'tanjore_painting_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', craftNameKey: 'stub.craftName.warliArt', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', artisanNameKey: 'stub.artisanName.warliArt', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.48.title', image: img('paintings', 'warli_art_01.jpeg'), pricePaise: 155000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', craftNameKey: 'stub.craftName.warliArt', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', artisanNameKey: 'stub.artisanName.warliArt', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.49.title', image: img('paintings', 'warli_art_02.jpeg'), pricePaise: 165000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', craftNameKey: 'stub.craftName.warliArt', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', artisanNameKey: 'stub.artisanName.warliArt', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.50.title', image: img('paintings', 'warli_art_03.jpeg'), pricePaise: 178000 },

  // ---- Pottery ----
  { category: 'Pottery', craftSlug: 'nizamabad-black-pottery', craftName: 'Nizamabad Black Clay Pottery', craftNameKey: 'stub.craftName.nizamabadBlackPottery', giNo: 'GI-425', artisan: 'Ram Prakash Prajapati', artisanNameKey: 'stub.artisanName.nizamabadBlackPottery', district: 'Azamgarh', state: 'UP', titleKey: 'stub.listing.51.title', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 320000 },
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', craftNameKey: 'stub.craftName.bluePottery', giNo: 'GI-426', artisan: 'Kripal Kumbhar', artisanNameKey: 'stub.artisanName.bluePottery', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.52.title', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 180000 },

  // ---- Stone Carving ----
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', craftNameKey: 'stub.craftName.agraMarbleInlay', giNo: 'GI-427', artisan: 'Mohammad Rizwan', artisanNameKey: 'stub.artisanName.agraMarbleInlay', district: 'Agra', state: 'UP', titleKey: 'stub.listing.53.title', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 385000 },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', craftNameKey: 'stub.craftName.soapstoneCraft', giNo: 'GI-428', artisan: 'Santosh Prajapati', artisanNameKey: 'stub.artisanName.soapstoneCraft', district: 'Mahoba', state: 'UP', titleKey: 'stub.listing.54.title', image: img('stone_carving', 'soapstone-craft.jpg'), pricePaise: 145000 },
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', craftNameKey: 'stub.craftName.agraMarbleInlay', giNo: 'GI-427', artisan: 'Mohammad Rizwan', artisanNameKey: 'stub.artisanName.agraMarbleInlay', district: 'Agra', state: 'UP', titleKey: 'stub.listing.55.title', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', craftNameKey: 'stub.craftName.jaisalmerSandstoneCarving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', artisanNameKey: 'stub.artisanName.jaisalmerSandstoneCarving', district: 'Jaisalmer', state: 'RJ', titleKey: 'stub.listing.56.title', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_01.jpeg'), pricePaise: 685000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', craftNameKey: 'stub.craftName.jaisalmerSandstoneCarving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', artisanNameKey: 'stub.artisanName.jaisalmerSandstoneCarving', district: 'Jaisalmer', state: 'RJ', titleKey: 'stub.listing.57.title', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_02.jpeg'), pricePaise: 720000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', craftNameKey: 'stub.craftName.soapstoneCraft', giNo: 'GI-428', artisan: 'Santosh Prajapati', artisanNameKey: 'stub.artisanName.soapstoneCraft', district: 'Mahoba', state: 'UP', titleKey: 'stub.listing.58.title', image: img('stone_carving', 'karnataka_schist_soapstone_carving_01.jpeg'), pricePaise: 195000 },
  { category: 'Stone carving', craftSlug: 'mahabalipuram-granite-carving', craftName: 'Mahabalipuram Granite Carving', craftNameKey: 'stub.craftName.mahabalipuramGraniteCarving', giNo: 'GI-430', artisan: 'Raju Sthapathi', artisanNameKey: 'stub.artisanName.mahabalipuramGraniteCarving', district: 'Mahabalipuram', state: 'TN', titleKey: 'stub.listing.59.title', image: img('stone_carving', 'mahabalipuram_granite_carving_01.jpeg'), pricePaise: 895000, madeToOrder: true },

  // ---- Weaving & Looms ----
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', craftNameKey: 'stub.craftName.banarasiBrocadeWeaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', artisanNameKey: 'stub.artisanName.banarasiBrocadeWeaving', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.60.title', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'patan-patola', craftName: 'Patan Patola', craftNameKey: 'stub.craftName.patanPatola', giNo: 'GI-232', artisan: 'Dinesh Salvi Guild', artisanNameKey: 'stub.artisanName.patanPatola', district: 'Salvi Wada, Patan', state: 'GJ', titleKey: 'stub.listing.61.title', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 12000000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', craftNameKey: 'stub.craftName.banarasiBrocadeWeaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', artisanNameKey: 'stub.artisanName.banarasiBrocadeWeaving', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.62.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_01.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', craftNameKey: 'stub.craftName.banarasiBrocadeWeaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', artisanNameKey: 'stub.artisanName.banarasiBrocadeWeaving', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.63.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_02.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', craftNameKey: 'stub.craftName.banarasiBrocadeWeaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', artisanNameKey: 'stub.artisanName.banarasiBrocadeWeaving', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.64.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_03.jpeg'), pricePaise: 1350000, madeToOrder: true },

  // ---- Woodwork ----
  { category: 'Woodwork', craftSlug: 'saharanpur-wood-carving', craftName: 'Saharanpur Wood Carving', craftNameKey: 'stub.craftName.saharanpurWoodCarving', giNo: 'GI-432', artisan: 'Irfan Ahmed Qureshi', artisanNameKey: 'stub.artisanName.saharanpurWoodCarving', district: 'Saharanpur', state: 'UP', titleKey: 'stub.listing.65.title', image: img('woodwork', 'saharanpur-wood-carving.jpg'), pricePaise: 165000 },

  // ---- Pottery (dataset batch) ----
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', craftNameKey: 'stub.craftName.bluePottery', giNo: 'GI-426', artisan: 'Suresh Kumar Prajapati', artisanNameKey: 'stub.artisanName.terracottaJarBlue', district: 'Khurja', state: 'UP', titleKey: 'stub.listing.66.title', image: img('pottery', 'terracotta-jar-floral-blue.jpeg'), pricePaise: 85000 },
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', craftNameKey: 'stub.craftName.bluePottery', giNo: 'GI-426', artisan: 'Suresh Kumar Prajapati', artisanNameKey: 'stub.artisanName.terracottaJarBlue', district: 'Khurja', state: 'UP', titleKey: 'stub.listing.67.title', image: img('pottery', 'terracotta-fish-plate-green.jpeg'), pricePaise: 72000 },
  { category: 'Pottery', craftSlug: 'nizamabad-black-pottery', craftName: 'Nizamabad Black Clay Pottery', craftNameKey: 'stub.craftName.nizamabadBlackPottery', giNo: 'GI-425', artisan: 'Ram Prakash Prajapati', artisanNameKey: 'stub.artisanName.nizamabadBlackPottery', district: 'Azamgarh', state: 'UP', titleKey: 'stub.listing.68.title', image: img('pottery', 'terracotta-pitcher-cups-blue.jpeg'), pricePaise: 145000 },
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', craftNameKey: 'stub.craftName.bluePottery', giNo: 'GI-426', artisan: 'Suresh Kumar Prajapati', artisanNameKey: 'stub.artisanName.terracottaJarBlue', district: 'Khurja', state: 'UP', titleKey: 'stub.listing.69.title', image: img('pottery', 'terracotta-tulip-bowl-orange.jpeg'), pricePaise: 98000 },
  { category: 'Pottery', craftSlug: 'nizamabad-black-pottery', craftName: 'Nizamabad Black Clay Pottery', craftNameKey: 'stub.craftName.nizamabadBlackPottery', giNo: 'GI-425', artisan: 'Ram Prakash Prajapati', artisanNameKey: 'stub.artisanName.nizamabadBlackPottery', district: 'Azamgarh', state: 'UP', titleKey: 'stub.listing.70.title', image: img('pottery', 'terracotta-bowls-green-glaze.jpeg'), pricePaise: 125000 },

  // ---- Woodwork (dataset batch — recategorized from pottery/hand carved wodden fish) ----
  { category: 'Woodwork', craftSlug: 'saharanpur-wood-carving', craftName: 'Saharanpur Wood Carving', craftNameKey: 'stub.craftName.saharanpurWoodCarving', giNo: 'GI-432', artisan: 'Mohan Lal Suthar', artisanNameKey: 'stub.artisanName.handCarvedWoodenFish', district: 'Saharanpur', state: 'UP', titleKey: 'stub.listing.71.title', image: img('woodwork', 'hand-carved-wooden-fish.jpeg'), pricePaise: 159200 },

  // ---- Weaving & Looms (Chanderi — dataset batch) ----
  { category: 'Weaving', craftSlug: 'chanderi-weaving', craftName: 'Chanderi Saree Weaving', craftNameKey: 'stub.craftName.chanderiWeaving', giNo: 'GI-253', artisan: 'Rajesh Kumar Koshthi', artisanNameKey: 'stub.artisanName.chanderiWeaving', district: 'Ashoknagar', state: 'MP', titleKey: 'stub.listing.72.title', image: img('weaving_and_looms', 'chanderi-saree-orange.jpeg'), pricePaise: 680000 },
  { category: 'Weaving', craftSlug: 'chanderi-weaving', craftName: 'Chanderi Saree Weaving', craftNameKey: 'stub.craftName.chanderiWeaving', giNo: 'GI-253', artisan: 'Rajesh Kumar Koshthi', artisanNameKey: 'stub.artisanName.chanderiWeaving', district: 'Ashoknagar', state: 'MP', titleKey: 'stub.listing.73.title', image: img('weaving_and_looms', 'chanderi-saree-white-red.jpeg'), pricePaise: 620000 },
  { category: 'Weaving', craftSlug: 'chanderi-weaving', craftName: 'Chanderi Saree Weaving', craftNameKey: 'stub.craftName.chanderiWeaving', giNo: 'GI-253', artisan: 'Rajesh Kumar Koshthi', artisanNameKey: 'stub.artisanName.chanderiWeaving', district: 'Ashoknagar', state: 'MP', titleKey: 'stub.listing.74.title', image: img('weaving_and_looms', 'chanderi-saree-peach.jpeg'), pricePaise: 720000 },
  { category: 'Weaving', craftSlug: 'chanderi-weaving', craftName: 'Chanderi Saree Weaving', craftNameKey: 'stub.craftName.chanderiWeaving', giNo: 'GI-253', artisan: 'Rajesh Kumar Koshthi', artisanNameKey: 'stub.artisanName.chanderiWeaving', district: 'Ashoknagar', state: 'MP', titleKey: 'stub.listing.75.title', image: img('weaving_and_looms', 'chanderi-saree-aqua.jpeg'), pricePaise: 650000 },

  // ---- Weaving & Looms (Kalamkari Kurtas — dataset batch) ----
  { category: 'Weaving', craftSlug: 'kalamkari', craftName: 'Kalamkari', craftNameKey: 'stub.craftName.kalamkari', giNo: 'GI-312', artisan: 'Venkataraman Munsad', artisanNameKey: 'stub.artisanName.kalamkari', district: 'Srikalahasti', state: 'AP', titleKey: 'stub.listing.76.title', image: img('weaving_and_looms', 'kalamkari-kurta-multicolor.jpeg'), pricePaise: 185000 },
  { category: 'Weaving', craftSlug: 'kalamkari', craftName: 'Kalamkari', craftNameKey: 'stub.craftName.kalamkari', giNo: 'GI-312', artisan: 'Venkataraman Munsad', artisanNameKey: 'stub.artisanName.kalamkari', district: 'Srikalahasti', state: 'AP', titleKey: 'stub.listing.77.title', image: img('weaving_and_looms', 'kalamkari-kurta-black.jpeg'), pricePaise: 210000 },

  // ---- Weaving & Looms (Paithani — dataset batch) ----
  { category: 'Weaving', craftSlug: 'paithani-weaving', craftName: 'Paithani Silk Weaving', craftNameKey: 'stub.craftName.paithaniWeaving', giNo: 'GI-350', artisan: 'Shankar Mahadeo Salve', artisanNameKey: 'stub.artisanName.paithaniWeaving', district: 'Yeola', state: 'MH', titleKey: 'stub.listing.78.title', image: img('weaving_and_looms', 'paithani-saree-blue-green.jpeg'), pricePaise: 1850000, madeToOrder: true },

  // ---- Weaving & Looms (Patan Patola — dataset batch) ----
  { category: 'Weaving', craftSlug: 'patan-patola', craftName: 'Patan Patola', craftNameKey: 'stub.craftName.patanPatola', giNo: 'GI-232', artisan: 'Dinesh Salvi Guild', artisanNameKey: 'stub.artisanName.patanPatola', district: 'Salvi Wada, Patan', state: 'GJ', titleKey: 'stub.listing.79.title', image: img('weaving_and_looms', 'patan-patola-saree-pink.jpeg'), pricePaise: 9500000, madeToOrder: true },

  // ---- Weaving & Looms (Tussar Silk — dataset batch) ----
  { category: 'Weaving', craftSlug: 'tussar-silk-weaving', craftName: 'Tussar Silk Weaving', craftNameKey: 'stub.craftName.tussarSilkWeaving', giNo: 'GI-351', artisan: 'Prabha Devi Jha', artisanNameKey: 'stub.artisanName.tussarSilkWeaving', district: 'Bhagalpur', state: 'BR', titleKey: 'stub.listing.80.title', image: img('weaving_and_looms', 'tussar-silk-saree-beige-embroidered.jpeg'), pricePaise: 780000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'tussar-silk-weaving', craftName: 'Tussar Silk Weaving', craftNameKey: 'stub.craftName.tussarSilkWeaving', giNo: 'GI-351', artisan: 'Prabha Devi Jha', artisanNameKey: 'stub.artisanName.tussarSilkWeaving', district: 'Bhagalpur', state: 'BR', titleKey: 'stub.listing.81.title', image: img('weaving_and_looms', 'tussar-silk-saree-pink-floral.jpeg'), pricePaise: 650000 },
  { category: 'Weaving', craftSlug: 'tussar-silk-weaving', craftName: 'Tussar Silk Weaving', craftNameKey: 'stub.craftName.tussarSilkWeaving', giNo: 'GI-351', artisan: 'Prabha Devi Jha', artisanNameKey: 'stub.artisanName.tussarSilkWeaving', district: 'Bhagalpur', state: 'BR', titleKey: 'stub.listing.82.title', image: img('weaving_and_looms', 'tussar-silk-saree-navy-floral.jpeg'), pricePaise: 720000 },

  // ---- Weaving & Looms (Zari Work Sari — dataset batch) ----
  { category: 'Weaving', craftSlug: 'zari-work', craftName: 'Zari Work', craftNameKey: 'stub.craftName.zariWork', giNo: 'GI-433', artisan: 'Ashok Kumar Gupta', artisanNameKey: 'stub.artisanName.zariWork', district: 'Surat', state: 'GJ', titleKey: 'stub.listing.83.title', image: img('weaving_and_looms', 'zari-work-sari-beige.jpeg'), pricePaise: 950000, madeToOrder: true },

  // ---- Block Printing (Sanganeri — dataset batch, recategorized from weaving and handlooms) ----
  { category: 'Block printing', craftSlug: 'sanganeri-block-printing', craftName: 'Sanganeri Block Printing', craftNameKey: 'stub.craftName.sanganeriBlockPrinting', giNo: 'GI-354', artisan: 'Ramawatar Chippa', artisanNameKey: 'stub.artisanName.sanganeriBlockPrinting', district: 'Sanganer, Jaipur', state: 'RJ', titleKey: 'stub.listing.84.title', image: img('block_printing', 'sanganeri-block-print-kurta-white.jpeg'), pricePaise: 245000 },
  { category: 'Block printing', craftSlug: 'sanganeri-block-printing', craftName: 'Sanganeri Block Printing', craftNameKey: 'stub.craftName.sanganeriBlockPrinting', giNo: 'GI-354', artisan: 'Ramawatar Chippa', artisanNameKey: 'stub.artisanName.sanganeriBlockPrinting', district: 'Sanganer, Jaipur', state: 'RJ', titleKey: 'stub.listing.85.title', image: img('block_printing', 'sanganeri-block-print-kurta-pink.jpeg'), pricePaise: 285000 },

  // ---- Weaving & Looms (Part 2 — special model-shot sarees) ----
  { category: 'Weaving', craftSlug: 'zari-work', craftName: 'Zari Work', craftNameKey: 'stub.craftName.zariWork', giNo: 'GI-433', artisan: 'Ashok Kumar Gupta', artisanNameKey: 'stub.artisanName.zariWork', district: 'Surat', state: 'GJ', titleKey: 'stub.listing.86.title', image: img('weaving_and_looms', 'zari-work-saree-beige.jpeg'), pricePaise: 950000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'tussar-silk-weaving', craftName: 'Tussar Silk Weaving', craftNameKey: 'stub.craftName.tussarSilkWeaving', giNo: 'GI-351', artisan: 'Prabha Devi Jha', artisanNameKey: 'stub.artisanName.tussarSilkWeaving', district: 'Bhagalpur', state: 'BR', titleKey: 'stub.listing.87.title', image: img('weaving_and_looms', 'purple-tussar-silk-saree.jpeg'), pricePaise: 1240800 },
  { category: 'Weaving', craftSlug: 'tussar-silk-weaving', craftName: 'Tussar Silk Weaving', craftNameKey: 'stub.craftName.tussarSilkWeaving', giNo: 'GI-351', artisan: 'Prabha Devi Jha', artisanNameKey: 'stub.artisanName.tussarSilkWeaving', district: 'Narsinghpur', state: 'MP', titleKey: 'stub.listing.88.title', image: img('weaving_and_looms', 'blue-mp-tussar-saree.jpeg'), pricePaise: 1060300 },
  { category: 'Weaving', craftSlug: 'zari-work', craftName: 'Zari Work', craftNameKey: 'stub.craftName.zariWork', giNo: 'GI-433', artisan: 'Ashok Kumar Gupta', artisanNameKey: 'stub.artisanName.zariWork', district: 'Surat', state: 'GJ', titleKey: 'stub.listing.89.title', image: img('weaving_and_looms', 'orange-zari-border-silk-saree.jpeg'), pricePaise: 900000 },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', craftNameKey: 'stub.craftName.banarasiBrocadeWeaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', artisanNameKey: 'stub.artisanName.banarasiBrocadeWeaving', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.90.title', image: img('weaving_and_looms', 'white-saree-maroon-border.jpeg'), pricePaise: 750000 },
];

// Id is the item's fixed position in ITEMS, so the same piece always gets the
// same /listing/{id} URL and stubListingById() can resolve it back.
function toListing(item: StubItem, t: TFn): ListingSummary {
  const seq = ITEMS.indexOf(item) + 1;
  const id = `stub-${item.craftSlug}-${seq}`;
  return {
    id,
    product_id: `stub-prod-${seq}`,
    artisan_id: `stub-artisan-${item.craftSlug}`,
    artisan_name: t(item.artisanNameKey),
    craft_name: t(item.craftNameKey),
    craft_slug: item.craftSlug,
    craft_gi_registration_no: item.giNo,
    gi_certified: true,
    artisan_verified: true,
    artisan_district: item.district,
    artisan_state_code: item.state,
    type: item.madeToOrder ? 'MADE_TO_ORDER' : 'READY_STOCK',
    price: { amount_paise: item.pricePaise, currency_code: 'INR' },
    image_url: item.image,
    translations: [
      {
        language: locale.code,
        title: t(item.titleKey),
        description: t('stub.listing.descriptionTemplate', {
          craft: t(item.craftNameKey),
          artisan: t(item.artisanNameKey),
          district: item.district,
        }),
      },
    ],
  } as ListingSummary;
}

/** category cover image for a craft-categories.ts id, e.g. 'weaving_and_looms' */
export function categoryCoverImage(zipCategoryId: string): string {
  return img(zipCategoryId, 'category_cover.jpg');
}

/**
 * Matches ArtisanCraftGrid's `/search?category=<name>` click-through, plus a
 * loose text match so `/search?q=ajrakh` style links (home page journal,
 * hero slides) also resolve to real pieces when the backend returns nothing.
 *
 * Takes the caller's reactive `t` (e.g. `const t = $derived(locale.t)`) so
 * titles/descriptions are built fresh in the active language on every call,
 * rather than baked in once at module load.
 */
export function stubListingsForQuery(q: string, t: TFn): ListingSummary[] {
  const needle = q.trim().toLowerCase();
  if (!needle) return [];
  return ITEMS.filter(
    (item) =>
      item.category.toLowerCase() === needle ||
      item.category.toLowerCase().includes(needle) ||
      item.craftName.toLowerCase().includes(needle) ||
      item.craftSlug.toLowerCase().includes(needle) ||
      t(item.titleKey).toLowerCase().includes(needle),
  ).map((item) => toListing(item, t));
}

/** Every stub piece, in catalogue order -- for "customers also viewed" style rows. */
export function allStubListings(t: TFn): ListingSummary[] {
  return ITEMS.map((item) => toListing(item, t));
}

/** Resolves a `stub-<craftSlug>-<n>` id produced by toListing() above. */
export function stubListingById(id: string, t: TFn): ListingSummary | undefined {
  const n = Number(id.match(/-(\d+)$/)?.[1]);
  const item = ITEMS[n - 1];
  return item && id === `stub-${item.craftSlug}-${n}` ? toListing(item, t) : undefined;
}
