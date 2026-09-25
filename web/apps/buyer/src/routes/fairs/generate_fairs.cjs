const fs = require('fs');

const FAIRS = [
  {
    id: 'surajkund-2026',
    title: 'Surajkund International Crafts Mela',
    edition: '39th Annual Edition',
    venue: 'Surajkund Mela Grounds, Faridabad',
    city: 'Faridabad',
    state: 'Haryana',
    dates: 'Feb 2 – Feb 18, 2026',
    description: "The world's largest open-air handicraft mela showcasing authentic rural craft lineages, living pit looms, and live woodcarving demonstrations by national awardees.",
    partnerMinistry: 'Ministry of Tourism & Haryana Tourism Council',
  },
  {
    id: 'dilli-haat-ina',
    title: 'Dilli Haat Master Crafts Fortnight',
    edition: 'National Artisan Rotation',
    venue: 'INA Market Complex, New Delhi',
    city: 'New Delhi',
    state: 'Delhi NCR',
    dates: 'Year-Round Rotating Fortnights',
    description: 'A permanent cultural marketplace enabling empanelled weavers and rural craft cooperatives to bypass wholesale brokers and sell directly to metropolitan buyers.',
    partnerMinistry: 'Delhi Tourism & DC (Handicrafts)',
  },
  {
    id: 'shilp-samagam-2026',
    title: 'Shilp Samagam Apex Handicraft Expo',
    edition: '2026 National Pavilion',
    venue: 'Major Dhyan Chand National Stadium, India Gate',
    city: 'New Delhi',
    state: 'Delhi NCR',
    dates: 'Nov 1 – Nov 15, 2026',
    description: 'The apex central government exhibition uniting GI-certified craft clusters, tribal SHG micro-entrepreneurs, and PM Vishwakarma certified master lineages.',
    partnerMinistry: 'Ministry of Social Justice & Ministry of Textiles',
  },
  {
    id: 'saras-mela-2026',
    title: 'Saras Mela Rural Livelihoods Festival',
    edition: 'Eastern India Pavilion',
    venue: 'Gandhi Maidan, Patna',
    city: 'Patna',
    state: 'Bihar',
    dates: 'Dec 10 – Dec 22, 2026',
    description: 'Dedicated rural livelihoods mela fostering institutional B2B procurement and direct market linkages for women artisan self-help groups (SHGs).',
    partnerMinistry: 'Ministry of Rural Development (MoRD)',
  },
  {
    id: 'hunar-haat-mumbai',
    title: 'Hunar Haat Heritage Pavilion',
    edition: 'Western Metropolitan Expo',
    venue: 'MMRDA Grounds, Bandra-Kurla Complex',
    city: 'Mumbai',
    state: 'Maharashtra',
    dates: 'Jan 14 – Jan 25, 2026',
    description: 'High-volume urban expo linking master leather crafters, bell metal founders, and handloom cooperatives directly with interior design houses and exporters.',
    partnerMinistry: 'Ministry of Minority Affairs',
  },
];

let out = "";
for (const fair of FAIRS) {
  out += `  'fairs.data.${fair.id}.title': ${JSON.stringify(fair.title)},\n`;
  out += `  'fairs.data.${fair.id}.edition': ${JSON.stringify(fair.edition)},\n`;
  out += `  'fairs.data.${fair.id}.venue': ${JSON.stringify(fair.venue)},\n`;
  out += `  'fairs.data.${fair.id}.city': ${JSON.stringify(fair.city)},\n`;
  out += `  'fairs.data.${fair.id}.state': ${JSON.stringify(fair.state)},\n`;
  out += `  'fairs.data.${fair.id}.dates': ${JSON.stringify(fair.dates)},\n`;
  out += `  'fairs.data.${fair.id}.description': ${JSON.stringify(fair.description)},\n`;
  out += `  'fairs.data.${fair.id}.partnerMinistry': ${JSON.stringify(fair.partnerMinistry)},\n`;
}

const enTsFile = "c:/projects/kalakriti/web/packages/i18n/src/messages/en.ts";
let enTsContent = fs.readFileSync(enTsFile, 'utf8');

if (!enTsContent.includes("'fairs.data.surajkund-2026.title'")) {
  enTsContent = enTsContent.replace('} as const;', out + '} as const;');
  fs.writeFileSync(enTsFile, enTsContent);
  console.log("Updated en.ts with fairs");
} else {
  console.log("en.ts already updated with fairs");
}
