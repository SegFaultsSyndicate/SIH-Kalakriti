// apps/buyer/src/lib/gallery-images.ts
//
// Photos under static/craft-images/<folder>/, listed for /gallery. Static
// files cannot be globbed without bundling copies of them, so this is a
// snapshot of the directory.
// ponytail: snapshot -- re-list the folders here when photos are added.
export const GALLERY_IMAGES: Readonly<Record<string, readonly string[]>> = {
  'bamboo_craft': [
    'assam-bamboo-craft.jpg',
    'assam_jaapi_structural_weaving_01.jpeg',
    'bamboo_craft_mug_01.jpeg',
    'kerala_bamboo_reed_ply_mat_craft_01.jpeg',
    'meghalaya_knup_rain_shield_craft_01.jpeg',
    'tripura-bamboo-craft.jpg',
    'tripura_cane_split_bamboo_furniture_01.jpeg'
  ],
  'basketry': [
    'assamese_cane_bamboo_basketry_01.jpeg',
    'kottan_palm_leaf_basketry_01.jpeg',
    'mizoram_thul_basketry_traditions_01.jpeg',
    'moonj_grass_basketry_01.jpeg',
    'sabai-grass-basketry.jpg',
    'sarkanda_chick_basketry_01.jpeg',
    'sikki-grass-basketry.jpg',
    'thana_tribal_grass_basketry_01.jpeg'
  ],
  'block_printing': [
    'ajrakh-block-printing.jpg',
    'ajrakh_dabu_monsoon_indigo_01.jpeg',
    'ajrakh_dabu_monsoon_indigo_02.jpeg',
    'ajrakh_dabu_monsoon_indigo_03.jpeg',
    'sanganeri-block-printing.jpg'
  ],
  'embroidery': [
    'kashmir_pashmina_sozni_01.jpeg',
    'kashmir_pashmina_sozni_02.jpeg',
    'kutch-embroidery.jpg',
    'lucknow-chikankari.jpg'
  ],
  'furniture': [
    'jodhpur-furniture.jpg',
    'saharanpur-wood-carving.jpg'
  ],
  'home_and_living': [
    'bhadohi-carpet.jpg',
    'moradabad-metal.jpg'
  ],
  'jewellery': [
    'cuttack-silver-filigree.jpg',
    'jadau_jewellery_01.jpeg',
    'jaipur-meenakari.jpg',
    'kundankari_jewellery_01.jpeg',
    'meenakari_enamelwork_01.jpeg',
    'tarakashi_silver_filigree_01.jpeg',
    'tarakashi_silver_filigree_02.jpeg',
    'temple_jewellery_01.jpeg'
  ],
  'leatherwork': [
    'jawaja_tanned_leatherwork_01.jpeg',
    'kolhapuri-chappal.jpg',
    'kolhapuri_chappals_01.jpeg',
    'kolhapuri_chappals_02.jpeg',
    'kutch_leathercraft_marwada_style_01.jpeg',
    'mojari_jutti_footwear_01.jpeg',
    'shantiniketan-leather.jpg',
    'shantiniketan_embossed_leather_01.jpeg',
    'shantiniketan_embossed_leather_02.jpeg',
    'shantiniketan_embossed_leather_03.jpeg'
  ],
  'metalwork': [
    'bidriware.jpg',
    'dhokra-casting.jpg'
  ],
  'paintings': [
    'madhubani-painting.jpg',
    'madhubani_mithila_painting_01.jpeg',
    'pahari_miniature_painting_01.jpeg',
    'pattachitra.jpg',
    'pattachitra_01.jpeg',
    'tanjore_painting_01.jpeg',
    'warli_art_01.jpeg',
    'warli_art_02.jpeg',
    'warli_art_03.jpeg'
  ],
  'pottery': [
    'blue-pottery.jpg',
    'nizamabad-black-pottery.jpg'
  ],
  'stone_carving': [
    'agra-marble-inlay.jpg',
    'agra_marble_inlay_parchin_kari_01.jpeg',
    'jaisalmer_jaipur_sandstone_carving_01.jpeg',
    'jaisalmer_jaipur_sandstone_carving_02.jpeg',
    'karnataka_schist_soapstone_carving_01.jpeg',
    'mahabalipuram_granite_carving_01.jpeg',
    'soapstone-craft.jpg'
  ],
  'weaving_and_looms': [
    'banarasi-brocade-weaving.jpg',
    'banarasi_brocade_weaving_01.jpeg',
    'banarasi_brocade_weaving_02.jpeg',
    'banarasi_brocade_weaving_03.jpeg',
    'patan-patola.jpg'
  ],
  'woodwork': [
    'channapatna-toys.jpg',
    'saharanpur-wood-carving.jpg'
  ]
};
