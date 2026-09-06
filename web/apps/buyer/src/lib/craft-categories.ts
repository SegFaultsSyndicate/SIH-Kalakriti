/**
 * apps/buyer/src/lib/craft-categories.ts
 *
 * The 12 authentic National Artisan Craft Categories of Kalakriti.
 * Aligned 1:1 with the artisan onboarding ontology and @kalakriti/icons.
 */

import type { IconName } from '@kalakriti/icons';

export interface CraftCategory {
  id: string;
  name: string;
  hindiName: string;
  icon: IconName;
  subtitle: string;
  tagline: string;
  query: string;
  sampleImage: string;
  regions: string[];
  giCount: number;
}

export const ARTISAN_CRAFT_CATEGORIES: readonly CraftCategory[] = [
  {
    id: 'weaving',
    name: 'Weaving',
    hindiName: 'बुनाई',
    icon: 'weaving',
    subtitle: 'Handlooms, Brocades & Sarees',
    tagline: 'Pit-loom and jacquard traditions woven with pure silk, khadi, and fine cotton threads.',
    query: 'weaving',
    sampleImage: 'https://images.unsplash.com/photo-1610030469983-98e550d6193c?auto=format&fit=crop&w=600&q=80',
    regions: ['Varanasi (UP)', 'Chanderi (MP)', 'Kanchipuram (TN)', 'Patan (Gujarat)'],
    giCount: 38,
  },
  {
    id: 'block-printing',
    name: 'Block printing',
    hindiName: 'ठप्पा छपाई',
    icon: 'block-printing',
    subtitle: 'Natural Dyes & Hand Woodblocks',
    tagline: 'Multi-stage river-washed mud resist (Dabu) and ancient geometric Ajrakh block prints.',
    query: 'block printing',
    sampleImage: 'https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=600&q=80',
    regions: ['Kutch (Gujarat)', 'Bagru (Rajasthan)', 'Machilipatnam (AP)'],
    giCount: 14,
  },
  {
    id: 'pottery',
    name: 'Pottery',
    hindiName: 'कुम्हारी व मृत्तिका',
    icon: 'pottery',
    subtitle: 'Terracotta & Studio Ceramics',
    tagline: 'Clay pottery fired with smoke, lead-free glazes, and Khurja/Jaipur blue pottery techniques.',
    query: 'pottery',
    sampleImage: 'https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=600&q=80',
    regions: ['Khurja (UP)', 'Nizamabad (UP)', 'Jaipur (Rajasthan)', 'Bankura (WB)'],
    giCount: 11,
  },
  {
    id: 'metalwork',
    name: 'Metalwork',
    hindiName: 'धातुकर्म व ढोकरा',
    icon: 'metalwork',
    subtitle: 'Bell Metal, Brass & Lost-Wax',
    tagline: 'Ancient cire-perdue lost-wax Dhokra bronze metallurgy and Moradabad hammered brassware.',
    query: 'metalwork',
    sampleImage: 'https://images.unsplash.com/photo-1579783900882-c0d3dad7b119?auto=format&fit=crop&w=600&q=80',
    regions: ['Bastar (Chhattisgarh)', 'Moradabad (UP)', 'Bidar (Karnataka)', 'Thanjavur (TN)'],
    giCount: 19,
  },
  {
    id: 'woodwork',
    name: 'Woodwork',
    hindiName: 'काष्ठकला व नक्काशी',
    icon: 'woodwork',
    subtitle: 'Carved Sheesham & Rosewood',
    tagline: 'Hand-chiselled brass inlay woodwork, Kashmiri walnut carving, and Channapatna lac-turnery.',
    query: 'woodwork',
    sampleImage: 'https://images.unsplash.com/photo-1513519245088-0e12902e5a38?auto=format&fit=crop&w=600&q=80',
    regions: ['Saharanpur (UP)', 'Srinagar (J&K)', 'Channapatna (Karnataka)', 'Jodhpur (RJ)'],
    giCount: 16,
  },
  {
    id: 'embroidery',
    name: 'Embroidery',
    hindiName: 'कशीदाकारी व जरदोजी',
    icon: 'embroidery',
    subtitle: 'Zardozi, Kantha & Chikankari',
    tagline: 'Intricate needle-and-thread needlework using metallic coils, silk floss, and running stitch patterns.',
    query: 'embroidery',
    sampleImage: 'https://images.unsplash.com/photo-1590736969955-71cc94801759?auto=format&fit=crop&w=600&q=80',
    regions: ['Lucknow (UP)', 'Shantiniketan (WB)', 'Kashmir', 'Kutch (Gujarat)'],
    giCount: 21,
  },
  {
    id: 'painting',
    name: 'Painting',
    hindiName: 'चित्रकला व लोक कला',
    icon: 'painting',
    subtitle: 'Folk Art & Natural Pigments',
    tagline: 'Centuries-old canvas and scroll folk art using stone, vermillion, indigo, and organic mineral colors.',
    query: 'painting',
    sampleImage: 'https://images.unsplash.com/photo-1579783902614-a3fb3927b675?auto=format&fit=crop&w=600&q=80',
    regions: ['Madhubani (Bihar)', 'Raghurajpur (Odisha)', 'Nathdwara (RJ)', 'Warli (MH)'],
    giCount: 27,
  },
  {
    id: 'basketry',
    name: 'Basketry',
    hindiName: 'टोकरी व घास शिल्प',
    icon: 'basketry',
    subtitle: 'Sabai, Moonj & Sikki Grass',
    tagline: 'Eco-friendly natural plant fibers platted by women-led self-help groups into durable lifestyle wares.',
    query: 'basketry',
    sampleImage: 'https://images.unsplash.com/photo-1590736969955-71cc94801759?auto=format&fit=crop&w=600&q=80',
    regions: ['Mayurbhanj (Odisha)', 'Madhubani (Bihar)', 'Prayagraj (UP)'],
    giCount: 8,
  },
  {
    id: 'jewellery',
    name: 'Jewellery',
    hindiName: 'आभूषण व तारकशी',
    icon: 'jewellery',
    subtitle: 'Silver Filigree & Meenakari',
    tagline: 'Hair-thin silver wires woven into ethereal ornaments and Jaipur vitreous enamel work.',
    query: 'jewellery',
    sampleImage: 'https://images.unsplash.com/photo-1535632066927-ab7c9ab60908?auto=format&fit=crop&w=600&q=80',
    regions: ['Cuttack (Odisha)', 'Karimnagar (Telangana)', 'Jaipur (Rajasthan)'],
    giCount: 12,
  },
  {
    id: 'leather',
    name: 'Leatherwork',
    hindiName: 'चर्मशिल्प',
    icon: 'leather',
    subtitle: 'Embossed Leather & Footwear',
    tagline: 'Vegetable-tanned leather handcrafted with batik dye embossing and traditional footwear artistry.',
    query: 'leatherwork',
    sampleImage: 'https://images.unsplash.com/photo-1548036328-c9fa89d128fa?auto=format&fit=crop&w=600&q=80',
    regions: ['Shantiniketan (WB)', 'Kolhapur (MH)', 'Indore (MP)'],
    giCount: 7,
  },
  {
    id: 'stone',
    name: 'Stone carving',
    hindiName: 'प्रस्तर शिल्प व पच्चीकारी',
    icon: 'stone',
    subtitle: 'Marble Inlay & Soapstone',
    tagline: 'Imperial Pietra Dura marble inlay with semi-precious gems and soft chlorite stone sculptures.',
    query: 'stone carving',
    sampleImage: 'https://images.unsplash.com/photo-1568605117036-5fe5e7bab0b7?auto=format&fit=crop&w=600&q=80',
    regions: ['Agra (UP)', 'Puri (Odisha)', 'Varanasi (UP)', 'Mamallapuram (TN)'],
    giCount: 9,
  },
  {
    id: 'bamboo',
    name: 'Bamboo craft',
    hindiName: 'बांस व बेंत शिल्प',
    icon: 'bamboo',
    subtitle: 'North-East Cane & Bamboo',
    tagline: 'Sustainable split bamboo furniture, wicker screens, and tribal lattice craftsmanship.',
    query: 'bamboo craft',
    sampleImage: 'https://images.unsplash.com/photo-1596178065887-1198b6148b2b?auto=format&fit=crop&w=600&q=80',
    regions: ['Assam', 'Tripura', 'Nagaland', 'Kerala'],
    giCount: 15,
  },
];
