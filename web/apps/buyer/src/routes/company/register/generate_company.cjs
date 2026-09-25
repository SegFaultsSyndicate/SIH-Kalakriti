const fs = require('fs');

const COMPANY_TYPE_OPTIONS = [
  { value: 'RETAILER', label: 'Commercial Retailer / Heritage Brand (0.5% Platform Fee)' },
  { value: 'BOUTIQUE', label: 'Curated Boutique Studio (0.5% Platform Fee)' },
  { value: 'EXPORTER', label: 'International Craft Exporter (1.0% Platform Fee)' },
  { value: 'INSTITUTION', label: 'Government / State Institution (0.5% Platform Fee)' },
];

const CRAFT_OPTIONS = [
  { id: 'craft-bagru', name: 'Bagru Hand Block Print', state: 'Rajasthan' },
  { id: 'craft-sanganeri', name: 'Sanganeri Print', state: 'Rajasthan' },
  { id: 'craft-chanderi', name: 'Chanderi Weaving', state: 'Madhya Pradesh' },
  { id: 'craft-banarasi-brocade', name: 'Banarasi Brocade & Zari', state: 'Uttar Pradesh' },
  { id: 'craft-paithani', name: 'Paithani Silk', state: 'Maharashtra' },
  { id: 'craft-madhubani', name: 'Mithila / Madhubani Painting', state: 'Bihar' },
  { id: 'craft-kutch-embroidery', name: 'Kutch Rogan & Embroidery', state: 'Gujarat' },
  { id: 'craft-tanjore', name: 'Thanjavur Gold Leaf Painting', state: 'Tamil Nadu' },
  { id: 'craft-pochampally', name: 'Pochampally Ikat', state: 'Telangana' },
  { id: 'craft-dhokra', name: 'Dhokra Lost-Wax Bell Metal', state: 'Odisha / Chhattisgarh' },
];

const STEP_TITLES = [
  'Legal Entity & Contact',
  'Legitimacy Audit & Documents',
  'Sourcing & Studio Profile',
];

let out = "";

for (const opt of COMPANY_TYPE_OPTIONS) {
  out += `  'company.type.${opt.value}': ${JSON.stringify(opt.label)},\n`;
}

for (const craft of CRAFT_OPTIONS) {
  out += `  'company.craft.${craft.id}.name': ${JSON.stringify(craft.name)},\n`;
  out += `  'company.craft.${craft.id}.state': ${JSON.stringify(craft.state)},\n`;
}

for (let i = 0; i < STEP_TITLES.length; i++) {
  out += `  'company.step.${i}': ${JSON.stringify(STEP_TITLES[i])},\n`;
}

const enTsFile = "c:/projects/kalakriti/web/packages/i18n/src/messages/en.ts";
let enTsContent = fs.readFileSync(enTsFile, 'utf8');

if (!enTsContent.includes("'company.type.RETAILER'")) {
  enTsContent = enTsContent.replace('} as const;', out + '} as const;');
  fs.writeFileSync(enTsFile, enTsContent);
  console.log("Updated en.ts with company registration");
} else {
  console.log("en.ts already updated with company registration");
}
