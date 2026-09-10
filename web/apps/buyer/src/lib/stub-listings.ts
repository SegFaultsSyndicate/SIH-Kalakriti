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
 */

import type { components } from '@kalakriti/api';

type ListingSummary = components['schemas']['ListingSummary'];

interface StubItem {
  category: string;
  craftSlug: string;
  craftName: string;
  giNo: string;
  artisan: string;
  district: string;
  state: string;
  title: string;
  image: string;
  pricePaise: number;
  madeToOrder?: boolean;
}

function img(dir: string, file: string): string {
  return `/craft-images/${dir}/${file}`;
}

const ITEMS: StubItem[] = [
  // ---- Bamboo Craft ----
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', title: 'Artisanal Hand-Split Bamboo Hexagonal Weave Pendant Lamp', image: img('bamboo_craft', 'assam-bamboo-craft.jpg'), pricePaise: 285000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', title: 'Ergonomic Hand-Bent Solid Bamboo Accent Stool (Mora)', image: img('bamboo_craft', 'tripura-bamboo-craft.jpg'), pricePaise: 320000, madeToOrder: true },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', title: 'Assam Jaapi & Structural Weaving', image: img('bamboo_craft', 'assam_jaapi_structural_weaving_01.jpeg'), pricePaise: 195000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', title: 'Hand-Turned Bamboo Craft Mug', image: img('bamboo_craft', 'bamboo_craft_mug_01.jpeg'), pricePaise: 65000 },
  { category: 'Bamboo craft', craftSlug: 'assam-bamboo-craft', craftName: 'Assam Bamboo & Cane Craft', giNo: 'GI-401', artisan: 'Bhaben Kalita', district: 'Barpeta', state: 'AS', title: 'Kerala Bamboo Reed & Ply Mat Craft', image: img('bamboo_craft', 'kerala_bamboo_reed_ply_mat_craft_01.jpeg'), pricePaise: 220000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', title: 'Meghalaya Knup Rain Shield Craft', image: img('bamboo_craft', 'meghalaya_knup_rain_shield_craft_01.jpeg'), pricePaise: 145000 },
  { category: 'Bamboo craft', craftSlug: 'tripura-bamboo-craft', craftName: 'Tripura Bamboo & Cane Craft', giNo: 'GI-402', artisan: 'Sudip Debbarma', district: 'West Tripura', state: 'TR', title: 'Tripura Cane & Split Bamboo Furniture', image: img('bamboo_craft', 'tripura_cane_split_bamboo_furniture_01.jpeg'), pricePaise: 480000, madeToOrder: true },

  // ---- Basketry ----
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', title: 'Golden Sikki Grass Handwoven Royal Pauti Storage Box with Lid', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 145000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', title: 'Eco-Friendly Braided Sabai Grass Laundry & Planter Hamper', image: img('basketry', 'sikki-grass-basketry.jpg'), pricePaise: 95000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', title: 'Assamese Cane & Bamboo Basketry', image: img('basketry', 'assamese_cane_bamboo_basketry_01.jpeg'), pricePaise: 78000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', title: 'Kottan Palm Leaf Basketry', image: img('basketry', 'kottan_palm_leaf_basketry_01.jpeg'), pricePaise: 62000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', title: 'Mizoram Thul & Basketry Traditions', image: img('basketry', 'mizoram_thul_basketry_traditions_01.jpeg'), pricePaise: 88000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', title: 'Moonj Grass Basketry', image: img('basketry', 'moonj_grass_basketry_01.jpeg'), pricePaise: 54000 },
  { category: 'Basketry', craftSlug: 'sikki-grass-basketry', craftName: 'Sikki Grass Basketry', giNo: 'GI-403', artisan: 'Droupadi Devi', district: 'Madhubani', state: 'BR', title: 'Sarkanda & Chick Basketry', image: img('basketry', 'sarkanda_chick_basketry_01.jpeg'), pricePaise: 68000 },
  { category: 'Basketry', craftSlug: 'sabai-grass-basketry', craftName: 'Mayurbhanj Sabai Grass Craft', giNo: 'GI-404', artisan: 'Shanti Soren', district: 'Mayurbhanj', state: 'OD', title: 'Thana Tribal Grass Basketry', image: img('basketry', 'thana_tribal_grass_basketry_01.jpeg'), pricePaise: 72000 },

  // ---- Block Printing ----
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', title: 'The Monsoon Indigo Edition: Ajrakh & Dabu Resists', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_01.jpeg'), pricePaise: 465000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', title: 'Monsoon Indigo Edition — Ajrakh & Dabu Resists', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_02.jpeg'), pricePaise: 445000, madeToOrder: true },
  { category: 'Block printing', craftSlug: 'ajrakh-block-printing', craftName: 'Ajrakh Block Printing', giNo: 'GI-405', artisan: 'Dr. Ismail Mohammed Khatri', district: 'Kutch', state: 'GJ', title: 'Monsoon Indigo Edition — Dabu Mud-Resist Yardage', image: img('block_printing', 'ajrakh_dabu_monsoon_indigo_03.jpeg'), pricePaise: 398000 },

  // ---- Embroidery ----
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', district: 'Srinagar', state: 'JK', title: 'Kashmir Pashmina: Royal Needlework & Sozni', image: img('embroidery', 'kashmir_pashmina_sozni_01.jpeg'), pricePaise: 950000, madeToOrder: true },
  { category: 'Embroidery', craftSlug: 'kashmir-pashmina-sozni', craftName: 'Kashmir Pashmina Sozni Embroidery', giNo: 'GI-409', artisan: 'Ghulam Nabi Mir', district: 'Srinagar', state: 'JK', title: 'Kashmir Pashmina Sozni Needlework Stole', image: img('embroidery', 'kashmir_pashmina_sozni_02.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Furniture (extra images, folded into Woodwork category) ----
  { category: 'Woodwork', craftSlug: 'jodhpur-furniture', craftName: 'Jodhpur Handcrafted Furniture', giNo: 'GI-410', artisan: 'Gopal Suthar', district: 'Jodhpur', state: 'RJ', title: 'Jodhpur Hand-Turned Camel Bone Inlay Moroccan Pattern Bedside Cabinet', image: img('furniture', 'jodhpur-furniture.jpg'), pricePaise: 1850000, madeToOrder: true },

  // ---- Home & Living (extra images, folded into Weaving / Metalwork) ----
  { category: 'Weaving', craftSlug: 'bhadohi-carpet', craftName: 'Bhadohi Hand-Knotted Carpet', giNo: 'GI-411', artisan: 'Mohammad Yasin Ansari', district: 'Bhadohi', state: 'UP', title: 'Bhadohi Master-Knotted Pure Bikaner Wool Area Rug (4x6 ft)', image: img('home_and_living', 'bhadohi-carpet.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Metalwork', craftSlug: 'moradabad-metal', craftName: 'Moradabad Metal Craft', giNo: 'GI-412', artisan: 'Rameshwar Prasad Verma', district: 'Moradabad', state: 'UP', title: 'Ayurvedic Hand-Hammered Pure Copper Water Carafe & Glass Set', image: img('home_and_living', 'moradabad-metal.jpg'), pricePaise: 185000 },

  // ---- Jewellery ----
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', title: 'Jaipur Vitreous Champlevé Enamel & Kundan Chandbali Earrings', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 620000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', title: 'Jadau', image: img('jewellery', 'jadau_jewellery_01.jpeg'), pricePaise: 1250000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', title: 'Kundankari Jewellery', image: img('jewellery', 'kundankari_jewellery_01.jpeg'), pricePaise: 980000, madeToOrder: true },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', title: 'Meenakari / Enamelwork', image: img('jewellery', 'meenakari_enamelwork_01.jpeg'), pricePaise: 545000 },
  { category: 'Jewellery', craftSlug: 'cuttack-silver-filigree', craftName: 'Cuttack Silver Filigree', giNo: 'GI-413', artisan: 'Pankaj Kumar Sahoo', district: 'Cuttack', state: 'OD', title: 'Tarakashi Silver Filigree Necklace Set', image: img('jewellery', 'tarakashi_silver_filigree_02.jpeg'), pricePaise: 495000 },
  { category: 'Jewellery', craftSlug: 'jaipur-meenakari', craftName: 'Jaipur Meenakari Jewellery', giNo: 'GI-414', artisan: 'Mahesh Soni', district: 'Jaipur', state: 'RJ', title: 'Temple Jewellery', image: img('jewellery', 'temple_jewellery_01.jpeg'), pricePaise: 875000, madeToOrder: true },

  // ---- Leatherwork ----
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', title: 'Shantiniketan Embossed Batik Vegetable-Tanned Leather Tote Bag', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 285000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', title: 'Heritage Hand-Braided Vegetable-Tanned Kolhapuri Chappals', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 145000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', title: 'Jawaja Tanned Leatherwork', image: img('leatherwork', 'jawaja_tanned_leatherwork_01.jpeg'), pricePaise: 185000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', title: 'Kolhapuri Chappals', image: img('leatherwork', 'kolhapuri_chappals_01.jpeg'), pricePaise: 135000 },
  { category: 'Leatherwork', craftSlug: 'kolhapuri-chappal', craftName: 'Kolhapuri Chappal', giNo: 'GI-416', artisan: 'Santosh Satpute', district: 'Kolhapur', state: 'MH', title: 'Kolhapuri Chappals — Braided Tan', image: img('leatherwork', 'kolhapuri_chappals_02.jpeg'), pricePaise: 142000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', title: 'Kutch Leathercraft / Marwada Style', image: img('leatherwork', 'kutch_leathercraft_marwada_style_01.jpeg'), pricePaise: 225000 },
  { category: 'Leatherwork', craftSlug: 'kutch-leathercraft', craftName: 'Kutch Leathercraft (Marwada Style)', giNo: 'GI-417', artisan: 'Salim Marwada', district: 'Kutch', state: 'GJ', title: 'Mojari & Jutti Footwear', image: img('leatherwork', 'mojari_jutti_footwear_01.jpeg'), pricePaise: 165000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', title: 'Shantiniketan Embossed Leather Sling Bag', image: img('leatherwork', 'shantiniketan_embossed_leather_01.jpeg'), pricePaise: 245000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', title: 'Shantiniketan Embossed Leather Journal Cover', image: img('leatherwork', 'shantiniketan_embossed_leather_02.jpeg'), pricePaise: 98000 },
  { category: 'Leatherwork', craftSlug: 'shantiniketan-leather', craftName: 'Shantiniketan Leather Goods', giNo: 'GI-415', artisan: 'Bikash Karmakar', district: 'Bolpur Shantiniketan', state: 'WB', title: 'Shantiniketan Embossed Leather Wallet', image: img('leatherwork', 'shantiniketan_embossed_leather_03.jpeg'), pricePaise: 74000 },

  // ---- Metalwork ----
  { category: 'Metalwork', craftSlug: 'dhokra-casting', craftName: 'Dhokra Metal Casting', giNo: 'GI-418', artisan: 'Budheshwar Ghadwa', district: 'Bastar', state: 'CT', title: 'Bastar Cire-Perdue Lost-Wax Sacred Bull (Nandi) Bronze Sculpture', image: img('metalwork', 'dhokra-casting.jpg'), pricePaise: 850000 },
  { category: 'Metalwork', craftSlug: 'bidriware', craftName: 'Bidriware Pure Silver Inlay', giNo: 'GI-419', artisan: 'Abdul Rashid Guild', district: 'Bidar', state: 'KA', title: 'Bidriware Pure Silver Wire Tarkashi Inlay Floral Aftaba Vessel', image: img('metalwork', 'bidriware.jpg'), pricePaise: 650000, madeToOrder: true },

  // ---- Paintings ----
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', giNo: 'GI-420', artisan: 'Lakshmi Devi', district: 'Madhubani', state: 'BR', title: 'Traditional Madhubani Kohbar Nuptial Painting on Handmade Paper', image: img('paintings', 'madhubani-painting.jpg'), pricePaise: 320000 },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', district: 'Raghurajpur', state: 'OD', title: 'Tussar Silk Pattachitra Tree of Life & Lord Jagannath Scroll', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 285000 },
  { category: 'Painting', craftSlug: 'madhubani-painting', craftName: 'Madhubani Painting', giNo: 'GI-420', artisan: 'Lakshmi Devi', district: 'Madhubani', state: 'BR', title: 'Madhubani / Mithila Painting', image: img('paintings', 'madhubani_mithila_painting_01.jpeg'), pricePaise: 265000 },
  { category: 'Painting', craftSlug: 'pahari-miniature-painting', craftName: 'Pahari Miniature Painting', giNo: 'GI-422', artisan: 'Vijay Sharma', district: 'Kangra', state: 'HP', title: 'Traditional Pahari Miniature Painting', image: img('paintings', 'pahari_miniature_painting_01.jpeg'), pricePaise: 385000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'pattachitra', craftName: 'Raghurajpur Pattachitra', giNo: 'GI-421', artisan: 'Rabi Narayan Maharana', district: 'Raghurajpur', state: 'OD', title: 'Pattachitra', image: img('paintings', 'pattachitra_01.jpeg'), pricePaise: 245000 },
  { category: 'Painting', craftSlug: 'tanjore-painting', craftName: 'Tanjore Painting', giNo: 'GI-423', artisan: 'Meenakshi Sundaram', district: 'Thanjavur', state: 'TN', title: 'Tanjore Painting', image: img('paintings', 'tanjore_painting_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', title: 'Warli Art', image: img('paintings', 'warli_art_01.jpeg'), pricePaise: 155000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', title: 'Warli Art — Harvest Motif', image: img('paintings', 'warli_art_02.jpeg'), pricePaise: 165000 },
  { category: 'Painting', craftSlug: 'warli-art', craftName: 'Warli Art', giNo: 'GI-424', artisan: 'Jivya Soma Mashe Guild', district: 'Dahanu', state: 'MH', title: 'Warli Art — Tarpa Dance Circle', image: img('paintings', 'warli_art_03.jpeg'), pricePaise: 178000 },

  // ---- Pottery ----
  { category: 'Pottery', craftSlug: 'nizamabad-black-pottery', craftName: 'Nizamabad Black Clay Pottery', giNo: 'GI-425', artisan: 'Ram Prakash Prajapati', district: 'Azamgarh', state: 'UP', title: 'Nizamabad Mirror-Burnished Black Clay Vedic Handi with Silver Inlay', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 320000 },
  { category: 'Pottery', craftSlug: 'blue-pottery', craftName: 'Jaipur Blue Pottery', giNo: 'GI-426', artisan: 'Kripal Kumbhar', district: 'Jaipur', state: 'RJ', title: 'Jaipur Turquoise Cobalt Glazed Quartz Blue Pottery Amphora Vase', image: img('pottery', 'nizamabad-black-pottery.jpg'), pricePaise: 180000 },

  // ---- Stone Carving ----
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', giNo: 'GI-427', artisan: 'Mohammad Rizwan', district: 'Agra', state: 'UP', title: 'Makrana White Marble Parchin Kari Pietra Dura Floral Inlay Plate', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 385000 },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', giNo: 'GI-428', artisan: 'Santosh Prajapati', district: 'Mahoba', state: 'UP', title: 'Hand-Chiseled Soapstone Jali Perforated Aromatherapy Diffuser', image: img('stone_carving', 'soapstone-craft.jpg'), pricePaise: 145000 },
  { category: 'Stone carving', craftSlug: 'agra-marble-inlay', craftName: 'Agra Marble Inlay Craft', giNo: 'GI-427', artisan: 'Mohammad Rizwan', district: 'Agra', state: 'UP', title: 'Agra Marble Inlay Work / Parchin Kari', image: img('stone_carving', 'agra_marble_inlay_parchin_kari_01.jpeg'), pricePaise: 425000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', district: 'Jaisalmer', state: 'RJ', title: 'Jaisalmer & Jaipur Sandstone Carving', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_01.jpeg'), pricePaise: 685000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'jaisalmer-sandstone-carving', craftName: 'Jaisalmer & Jaipur Sandstone Carving', giNo: 'GI-429', artisan: 'Bhanwar Lal Mistri', district: 'Jaisalmer', state: 'RJ', title: 'Jaisalmer Sandstone Jali Screen Panel', image: img('stone_carving', 'jaisalmer_jaipur_sandstone_carving_02.jpeg'), pricePaise: 720000, madeToOrder: true },
  { category: 'Stone carving', craftSlug: 'soapstone-craft', craftName: 'Gorara Stone Craft', giNo: 'GI-428', artisan: 'Santosh Prajapati', district: 'Mahoba', state: 'UP', title: 'Karnataka Schist & Soapstone Carving', image: img('stone_carving', 'karnataka_schist_soapstone_carving_01.jpeg'), pricePaise: 195000 },
  { category: 'Stone carving', craftSlug: 'mahabalipuram-granite-carving', craftName: 'Mahabalipuram Granite Carving', giNo: 'GI-430', artisan: 'Raju Sthapathi', district: 'Mahabalipuram', state: 'TN', title: 'Mahabalipuram Granite Carving', image: img('stone_carving', 'mahabalipuram_granite_carving_01.jpeg'), pricePaise: 895000, madeToOrder: true },

  // ---- Weaving & Looms ----
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', title: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Mulberry Silk Saree', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 2450000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'patan-patola', craftName: 'Patan Patola', giNo: 'GI-232', artisan: 'Dinesh Salvi Guild', district: 'Salvi Wada, Patan', state: 'GJ', title: 'Patan Double-Ikat Shikargah Royal Heritage Silk Saree', image: img('weaving_and_looms', 'banarasi-brocade-weaving.jpg'), pricePaise: 12000000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', title: 'Zari Brocades and Kadwa Weaves of Varanasi', image: img('weaving_and_looms', 'banarasi_brocade_weaving_01.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', title: 'Zari Brocades and Kadwa Weaves of Varanasi', image: img('weaving_and_looms', 'banarasi_brocade_weaving_02.jpeg'), pricePaise: 1200000, madeToOrder: true },
  { category: 'Weaving', craftSlug: 'banarasi-brocade-weaving', craftName: 'Banarasi Brocade Weaving', giNo: 'GI-99', artisan: 'Mohammad Kabir Ansari', district: 'Varanasi Weavers Colony', state: 'UP', title: 'Banarasi Brocade Weaving — Kadwa Detail', image: img('weaving_and_looms', 'banarasi_brocade_weaving_03.jpeg'), pricePaise: 1350000, madeToOrder: true },

  // ---- Woodwork ----
  { category: 'Woodwork', craftSlug: 'saharanpur-wood-carving', craftName: 'Saharanpur Wood Carving', giNo: 'GI-432', artisan: 'Irfan Ahmed Qureshi', district: 'Saharanpur', state: 'UP', title: 'Hand-Chiseled Sheesham Wood Floral Jali Openwork Incense Box', image: img('woodwork', 'saharanpur-wood-carving.jpg'), pricePaise: 165000 },
];

let seq = 0;
function toListing(item: StubItem): ListingSummary {
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
    translations: [{ language: 'en', title: item.title, description: `Handcrafted ${item.craftName} by ${item.artisan}, ${item.district}.` }],
  } as ListingSummary;
}

const PAIRS: { category: string; listing: ListingSummary }[] = ITEMS.map((item) => ({
  category: item.category.toLowerCase(),
  listing: toListing(item),
}));

export const STUB_LISTINGS: ListingSummary[] = PAIRS.map((p) => p.listing);

/** category cover image for a craft-categories.ts id, e.g. 'weaving_and_looms' */
export function categoryCoverImage(zipCategoryId: string): string {
  return img(zipCategoryId, 'category_cover.jpg');
}

/**
 * Matches ArtisanCraftGrid's `/search?category=<name>` click-through, plus a
 * loose text match so `/search?q=ajrakh` style links (home page journal,
 * hero slides) also resolve to real pieces when the backend returns nothing.
 */
export function stubListingsForQuery(q: string): ListingSummary[] {
  const needle = q.trim().toLowerCase();
  if (!needle) return [];
  return PAIRS.filter(
    ({ category, listing }) =>
      category === needle ||
      category.includes(needle) ||
      (listing.craft_name ?? '').toLowerCase().includes(needle) ||
      (listing.craft_slug ?? '').toLowerCase().includes(needle) ||
      (listing.translations?.[0]?.title ?? '').toLowerCase().includes(needle),
  ).map((p) => p.listing);
}
