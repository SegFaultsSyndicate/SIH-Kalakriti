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
import type { MessageKey, MessageValues } from '@kalakriti/i18n';

type ListingSummary = components['schemas']['ListingSummary'];
type TFn = (key: MessageKey, values?: MessageValues) => string;

interface StubItem {
  category: string;
  craftSlug: string;
  craftName: string;
  giNo: string;
  artisan: string;
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
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.01.title', image: img('bamboo_craft', 'assam-bamboo-craft.jpg'), pricePaise: 285000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.02.title', image: img('bamboo_craft', 'tripura-bamboo-craft.jpg'), pricePaise: 320000, madeToOrder: true },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.03.title', image: img('bamboo_craft', 'assam_jaapi_structural_weaving_01.jpeg'), pricePaise: 195000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.04.title', image: img('bamboo_craft', 'bamboo_craft_mug_01.jpeg'), pricePaise: 65000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', titleKey: 'stub.listing.05.title', image: img('bamboo_craft', 'kerala_bamboo_reed_ply_mat_craft_01.jpeg'), pricePaise: 220000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.06.title', image: img('bamboo_craft', 'meghalaya_knup_rain_shield_craft_01.jpeg'), pricePaise: 145000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', titleKey: 'stub.listing.07.title', image: img('bamboo_craft', 'tripura_cane_split_bamboo_furniture_01.jpeg'), pricePaise: 480000, madeToOrder: true },

  // ---- Basketry ----
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.08.title', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 145000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.09.title', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 95000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.10.title', image: img('basketry', 'assamese_cane_bamboo_basketry_01.jpeg'), pricePaise: 78000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.11.title', image: img('basketry', 'kottan_palm_leaf_basketry_01.jpeg'), pricePaise: 62000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.12.title', image: img('basketry', 'mizoram_thul_basketry_traditions_01.jpeg'), pricePaise: 88000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.13.title', image: img('basketry', 'moonj_grass_basketry_01.jpeg'), pricePaise: 54000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.14.title', image: img('basketry', 'sarkanda_chick_basketry_01.jpeg'), pricePaise: 68000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', titleKey: 'stub.listing.15.title', image: img('basketry', 'thana_tribal_grass_basketry_01.jpeg'), pricePaise: 72000 },

  // ---- Block Printing ----
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.16.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_01.jpeg'), pricePaise: 465000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.17.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_02.jpeg'), pricePaise: 445000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.18.title', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_03.jpeg'), pricePaise: 398000 },

  // ---- Embroidery ----
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', district: 'Srinagar', state: 'JK', titleKey: 'stub.listing.19.title', image: img('embroidery', 'kashmir_pashmina_sozni_01.jpeg'), pricePaise: 950000, madeToOrder: true },
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', district: 'Srinagar', state: 'JK', titleKey: 'stub.listing.20.title', image: img('embroidery', 'kashmir_pashmina_sozni_02.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Furniture (extra images, folded into Woodwork category) ----
  { category: 'Woodwork', craftSlug: 'jodhpur-furniture', craftName: 'Jodhpur Handcrafted Furniture', giNo: 'GI-410', artisan: 'Gopal Suthar', district: 'Jodhpur', state: 'RJ', titleKey: 'stub.listing.21.title', image: img('furniture', 'jodhpur-furniture.jpg'), pricePaise: 1850000, madeToOrder: true },

  // ---- Home & Living (extra images, folded into Weaving / Metalwork) ----
  { category: 'Weaving', craftSlug: 'bhadohi-carpet', craftName: 'Bhadohi Hand-Knotted Carpet', giNo: 'GI-411', artisan: 'Mohammad Yasin Ansari', district: 'Bhadohi', state: 'UP', titleKey: 'stub.listing.22.title', image: img('home_and_living', 'bhadohi-carpet.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Metalwork', craftSlug: 'moradabad-metal', craftName: 'Moradabad Metal Craft', giNo: 'GI-412', artisan: 'Rameshwar Prasad Verma', district: 'Moradabad', state: 'UP', titleKey: 'stub.listing.23.title', image: img('home_and_living', 'moradabad-metal.jpg'), pricePaise: 185000 },

  // ---- Jewellery ----
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.24.title', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 620000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.25.title', image: img('jewellery', 'jadau_jewellery_01.jpeg'), pricePaise: 1250000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.26.title', image: img('jewellery', 'kundankari_jewellery_01.jpeg'), pricePaise: 980000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.27.title', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 545000 },
  { category: 'Jewellery', craftSlug: 'cuttack-silver-filigree', craftName: 'Cuttack Silver Filigree', giNo: 'GI-413', artisan: 'Pankaj Kumar Sahoo', district: 'Cuttack', state: 'OD', titleKey: 'stub.listing.28.title', image: img('jewellery', 'tarakashi_silver_filigree_02.jpeg'), pricePaise: 495000 },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.29.title', image: img('jewellery', 'temple_jewellery_01.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Leatherwork ----
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.30.title', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 285000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.31.title', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 145000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.32.title', image: img('leatherwork', 'jawaja_tanned_leatherwork_01.jpeg'), pricePaise: 185000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.33.title', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 135000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', titleKey: 'stub.listing.34.title', image: img('leatherwork', 'kolhapuri_chappals_02.jpeg'), pricePaise: 142000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.35.title', image: img('leatherwork', 'kutch_leathercraft_marwada_style_01.jpeg'), pricePaise: 225000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', titleKey: 'stub.listing.36.title', image: img('leatherwork', 'mojari_jutti_footwear_01.jpeg'), pricePaise: 165000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.37.title', image: img('leatherwork', 'shantiniketan_embossed_leather_01.jpeg'), pricePaise: 245000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.38.title', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 98000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', titleKey: 'stub.listing.39.title', image: img('leatherwork', 'shantiniketan_embossed_leather_03.jpeg'), pricePaise: 74000 },

  // ---- Metalwork ----
  { category: 'Metalwork', craftSlug: 'dhokra-casting', craftName: 'Dhokra Metal Casting', giNo: 'GI-418', artisan: 'Budheshwar Ghadwa', district: 'Bastar', state: 'CT', titleKey: 'stub.listing.40.title', image: img('metalwork', 'dhokra-casting.jpg'), pricePaise: 850000 },
  { category: 'Metalwork', craftSlug: 'bidriware', craftName: 'Bidriware Pure Silver Inlay', giNo: 'GI-419', artisan: 'Abdul Rashid Guild', district: 'Bidar', state: 'KA', titleKey: 'stub.listing.41.title', image: img('metalwork', 'bidriware.jpg'), pricePaise: 650000, madeToOrder: true },

  // ---- Paintings ----
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', giNo: 'GI-420', artisan: 'Lakshmi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.42.title', image: img('paintings', 'madhubani-painting.jpg'), pricePaise: 320000 },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', district: 'Raghurajpur', state: 'OD', titleKey: 'stub.listing.43.title', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 285000 },
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', giNo: 'GI-420', artisan: 'Lakshmi Devi', district: 'Madhubani', state: 'BR', titleKey: 'stub.listing.44.title', image: img('paintings', 'madhubani_mithila_painting_01.jpeg'), pricePaise: 265000 },
  { category: 'Painting', craftSlug: 'pahari-miniature-painting', craftName: 'Pahari Miniature Painting', giNo: 'GI-422', artisan: 'Vijay Sharma', district: 'Kangra', state: 'HP', titleKey: 'stub.listing.45.title', image: img('paintings', 'pahari_miniature_painting_01.jpeg'), pricePaise: 385000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', district: 'Raghurajpur', state: 'OD', titleKey: 'stub.listing.46.title', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 245000 },
  { category: 'Painting', craftSlug: 'tanjore-painting', craftName: 'Tanjore Painting', giNo: 'GI-423', artisan: 'Meenakshi Sundaram', district: 'Thanjavur', state: 'TN', titleKey: 'stub.listing.47.title', image: img('paintings', 'tanjore_painting_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.48.title', image: img('paintings', 'warli_art_01.jpeg'), pricePaise: 155000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.49.title', image: img('paintings', 'warli_art_02.jpeg'), pricePaise: 165000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', titleKey: 'stub.listing.50.title', image: img('paintings', 'warli_art_03.jpeg'), pricePaise: 178000 },

  // ---- Pottery ----
  { category: 'Pottery', craftSlug: 'nizamabad-black-pottery', craftName: 'Nizamabad Black Clay Pottery', giNo: 'GI-425', artisan: 'Ram Prakash Prajapati', district: 'Azamgarh', state: 'UP', titleKey: 'stub.listing.51.title', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 320000 },
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', giNo: 'GI-426', artisan: 'Kripal Kumbhar', district: 'Jaipur', state: 'RJ', titleKey: 'stub.listing.52.title', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 180000 },

  // ---- Stone Carving ----
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', giNo: 'GI-427', artisan: 'Mohammad Rizwan', district: 'Agra', state: 'UP', titleKey: 'stub.listing.53.title', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 385000 },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', giNo: 'GI-428', artisan: 'Santosh Prajapati', district: 'Mahoba', state: 'UP', titleKey: 'stub.listing.54.title', image: img('stone_carving', 'soapstone-craft.jpg'), pricePaise: 145000 },
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', giNo: 'GI-427', artisan: 'Mohammad Rizwan', district: 'Agra', state: 'UP', titleKey: 'stub.listing.55.title', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', district: 'Jaisalmer', state: 'RJ', titleKey: 'stub.listing.56.title', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_01.jpeg'), pricePaise: 685000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', district: 'Jaisalmer', state: 'RJ', titleKey: 'stub.listing.57.title', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_02.jpeg'), pricePaise: 720000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', giNo: 'GI-428', artisan: 'Santosh Prajapati', district: 'Mahoba', state: 'UP', titleKey: 'stub.listing.58.title', image: img('stone_carving', 'karnataka_schist_soapstone_carving_01.jpeg'), pricePaise: 195000 },
  { category: 'Stone carving', craftSlug: 'mahabalipuram-granite-carving', craftName: 'Mahabalipuram Granite Carving', giNo: 'GI-430', artisan: 'Raju Sthapathi', district: 'Mahabalipuram', state: 'TN', titleKey: 'stub.listing.59.title', image: img('stone_carving', 'mahabalipuram_granite_carving_01.jpeg'), pricePaise: 895000, madeToOrder: true },

  // ---- Weaving & Looms ----
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.60.title', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'patan-patola', craftName: 'Patan Patola', giNo: 'GI-232', artisan: 'Dinesh Salvi Guild', district: 'Salvi Wada, Patan', state: 'GJ', titleKey: 'stub.listing.61.title', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 12000000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.62.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_01.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.63.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_02.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', titleKey: 'stub.listing.64.title', image: img('weaving_and_looms', 'banarasi_brocade_weaving_03.jpeg'), pricePaise: 1350000, madeToOrder: true },

  // ---- Woodwork ----
  { category: 'Woodwork', craftSlug: 'saharanpur-wood-carving', craftName: 'Saharanpur Wood Carving', giNo: 'GI-432', artisan: 'Irfan Ahmed Qureshi', district: 'Saharanpur', state: 'UP', titleKey: 'stub.listing.65.title', image: img('woodwork', 'saharanpur-wood-carving.jpg'), pricePaise: 165000 },
];

let seq = 0;
function toListing(item: StubItem, t: TFn): ListingSummary {
  seq += 1;
  const id = `stub-${item.craftSlug}-${seq}`;
  return {
    id,
    product_id: `stub-prod-${seq}`,
    artisan_id: `stub-artisan-${item.craftSlug}`,
    artisan_name: item.artisan,
    craft_name: item.craftName,
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
        language: 'en',
        title: t(item.titleKey),
        description: t('stub.listing.descriptionTemplate', {
          craft: item.craftName,
          artisan: item.artisan,
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
