const fs = require('fs');

const CASE_STUDIES = [
  {
    id: 'dhamadka-ajrakh',
    artisanName: 'Ismail Mohammad Khatri',
    title: 'The Alchemists of Dhamadka',
    subtitle: 'Revitalizing 16-stage natural indigo mud-resist and securing living wages for 42 master families.',
    cluster: 'Kutch Artisans Collective',
    region: 'Dhamadka, Gujarat',
    giTag: 'GI-72 (Kutch Ajrakh)',
    metrics: [
      { label: 'Artisan Income Surge', value: '+337%' },
      { label: 'Active Pit Vats', value: '18 Vats' },
      { label: 'Middleman Deduction', value: '0%' },
      { label: 'Sealed Provenance Proofs', value: '1,240' },
    ],
    beforeAfterRows: [
      {
        dimension: 'Monthly Household Income',
        before: '₹7,200 (Extreme seasonal swings, poverty during monsoon)',
        after: '₹28,400 (+294% steady recurring monthly income)',
      },
      {
        dimension: 'Intermediary Deductions',
        before: '38% retail margin retained by urban retail brokers',
        after: '0% broker loss (100% direct DBT transfer to artisan bank)',
      },
      {
        dimension: 'Market Reach',
        before: 'Only 2 physical melas (Surajkund & Dilli Haat) per year',
        after: 'Continuous 365-day digital storefront + QR stall placards',
      },
      {
        dimension: 'Authenticity Proof',
        before: 'No proof; undercut by ₹300 synthetic chemical screenprints',
        after: 'Spectroscopic botanical indigo proof & cryptographic NFC seal',
      },
      {
        dimension: 'Payment Settlement',
        before: '6 to 9 months delayed payment on credit consignment',
        after: 'Automated 48-hour escrow payout upon carrier scan',
      },
    ],
    problem: 'For decades, synthetic chemical block-prints imported by commercial fast-fashion houses severely undercut authentic hand block-printers in Dhamadka. Chemical substitutes flooded markets at 20% of the price, forcing traditional masters to dilute ancient 16-stage botanical resist techniques.',
    intervention: 'Through Kalakriti, the Dhamadka Guild implemented spectroscopic verification of real indigofera tinctoria vat fermentation and acacia arabica gum resists. Every stole and yardage length receives an immutable Ed25519 cryptographic seal and NFC QR tag before leaving the cluster.',
    impact: 'Authenticity verification allowed the guild to sell directly to luxury design houses and foreign cultural missions. Household incomes rose by 3.37x, enabling the creation of a collective natural dye seed bank and apprenticeship fund for 14 young printers.',
    quote: {
      text: 'When buyers scan our cloth and see our hands in the indigo vat, they understand why 16 stages cannot be hurried. Kalakriti gave our honor back.',
      author: 'Ismail Mohammad Khatri',
      role: 'Master Craftsman & National Awardee, Dhamadka',
    },
  },
  {
    id: 'varanasi-kadwa',
    artisanName: 'Eshaan',
    title: 'Varanasi Kadwa Pit-Loom Revival',
    subtitle: 'Eliminating predatory 38% broker fees and automating direct DBT payouts for heritage silk weavers.',
    cluster: 'Varanasi Silk Weaver Facility Centre',
    region: 'Varanasi, Uttar Pradesh',
    giTag: 'GI-99 (Banarasi Kadwa)',
    metrics: [
      { label: 'Direct Disbursements', value: '₹1.48 Cr' },
      { label: 'Broker Fee Cut', value: '0%' },
      { label: 'Lead Time Reduction', value: '68%' },
      { label: 'Active Pit Looms', value: '86 Looms' },
    ],
    beforeAfterRows: [
      {
        dimension: 'Monthly Weaver Earnings',
        before: '₹8,200 (Forced to take moneylender loans at 36% APR)',
        after: '₹32,600 (+298% increase, guaranteed living wage)',
      },
      {
        dimension: 'Order Predictability',
        before: 'Unpredictable bazaar spot sales, loom idle for 4 months/yr',
        after: '86 pit-looms booked 10 months in advance via GeM lots',
      },
      {
        dimension: 'Middleman Cut',
        before: 'Bazaar traders pocketed 35-40% markups',
        after: '0% middleman margin; 100% DBT bank settlement',
      },
      {
        dimension: 'Institutional Access',
        before: 'Zero access to government summits or corporate gifting',
        after: 'Direct procurement by state delegations & hotel chains',
      },
    ],
    problem: 'Master zari weavers in Varanasi were dependent on multi-tier bazaar intermediaries who retained 35-40% of retail margins and delayed payments for up to 9 months, forcing weaving families into cyclical monsoon indebtedness.',
    intervention: 'Kalakriti’s institutional procurement hub allowed state summit hospitality groups and corporate gifting consortia to place collective bulk orders directly with the Pariwar Bunkar SHG. Payment escrow is automated with direct DBT transfers upon loom milestone confirmation.',
    impact: 'Disbursements totaling ₹1.48 Cr reached weavers within 48 hours of dispatch. 86 pit-looms are now permanently booked across 12-month advance cycles, completely eliminating seasonal poverty.',
    quote: {
      text: 'For three generations, we never saw the face of the final buyer. Now our name and loom coordinates travel with every saree we interlock.',
      author: 'Eshaan',
      role: 'Lead Pit-Loom Weaver, Varanasi Silk Guild',
    },
  },
  {
    id: 'bastar-lost-wax',
    artisanName: 'Budheshwar Ghadwa',
    title: 'Preserving Bastar Lost-Wax Metallurgy',
    subtitle: 'Sustaining 4,000-year-old Harappan cire-perdue casting through collective metal financing.',
    cluster: 'Bastar Bell Metal Guild',
    region: 'Kondagaon, Chhattisgarh',
    giTag: 'GI-117 (Bastar Dhokra)',
    metrics: [
      { label: 'Raw Metal Financed', value: '14.2 Tons' },
      { label: 'Apprentices Trained', value: '28' },
      { label: 'Institutional Kits', value: '640 Lots' },
      { label: 'Pehchan Verified', value: '100%' },
    ],
    beforeAfterRows: [
      {
        dimension: 'Monthly Artisan Wage',
        before: '₹7,100 (Unviable due to 60% scrap metal price inflation)',
        after: '₹24,500 (+245% uplift with raw material financing)',
      },
      {
        dimension: 'Raw Material Financing',
        before: 'Artisans had to borrow at exploitative rates to buy beeswax',
        after: 'Ministry inventory credit line: 14.2 Tons financed upfront',
      },
      {
        dimension: 'Youth Apprenticeship',
        before: '0 young apprentices; youth migrating to construction labor',
        after: '28 tribal youth trained in 4,000-year lost-wax casting',
      },
      {
        dimension: 'Pehchan Recognition',
        before: 'Anonymous tribal makers without government MSME cards',
        after: '100% Pehchan & PM Vishwakarma verified ID cards',
      },
    ],
    problem: 'Volatile copper and zinc scrap pricing made it nearly impossible for tribal Dhokra sculptors to afford raw beeswax, clay, and casting metals upfront, threatening to wipe out one of humanity’s oldest non-ferrous casting lineages.',
    intervention: 'The platform introduced raw material inventory credits backed by institutional summit procurement kits. Sculptors record the molten pour on their artisan app, proving unadulterated bell metal alloys.',
    impact: '28 new tribal apprentices have joined the Kondagaon furnaces. Bastar lost-wax sculptures now grace prominent institutional lobbies and diplomatic gifts worldwide.',
    quote: {
      text: 'Our ancestors used beeswax from the sal forests. Kalakriti ensures our furnaces burn bright for the next generation without fear of debt.',
      author: 'Budheshwar Ghadwa',
      role: 'Senior Cire-Perdue Metalworker, Kondagaon',
    },
  },
];

let out = "";
for (const study of CASE_STUDIES) {
  out += `  'caseStudies.${study.id}.title': ${JSON.stringify(study.title)},\n`;
  out += `  'caseStudies.${study.id}.subtitle': ${JSON.stringify(study.subtitle)},\n`;
  out += `  'caseStudies.${study.id}.cluster': ${JSON.stringify(study.cluster)},\n`;
  out += `  'caseStudies.${study.id}.region': ${JSON.stringify(study.region)},\n`;
  out += `  'caseStudies.${study.id}.giTag': ${JSON.stringify(study.giTag)},\n`;
  
  study.metrics.forEach((m, i) => {
    out += `  'caseStudies.${study.id}.metrics.${i}.label': ${JSON.stringify(m.label)},\n`;
    out += `  'caseStudies.${study.id}.metrics.${i}.value': ${JSON.stringify(m.value)},\n`;
  });
  
  study.beforeAfterRows.forEach((r, i) => {
    out += `  'caseStudies.${study.id}.rows.${i}.dimension': ${JSON.stringify(r.dimension)},\n`;
    out += `  'caseStudies.${study.id}.rows.${i}.before': ${JSON.stringify(r.before)},\n`;
    out += `  'caseStudies.${study.id}.rows.${i}.after': ${JSON.stringify(r.after)},\n`;
  });
  
  out += `  'caseStudies.${study.id}.problem': ${JSON.stringify(study.problem)},\n`;
  out += `  'caseStudies.${study.id}.intervention': ${JSON.stringify(study.intervention)},\n`;
  out += `  'caseStudies.${study.id}.impact': ${JSON.stringify(study.impact)},\n`;
  out += `  'caseStudies.${study.id}.quote.text': ${JSON.stringify(study.quote.text)},\n`;
  out += `  'caseStudies.${study.id}.quote.author': ${JSON.stringify(study.quote.author)},\n`;
  out += `  'caseStudies.${study.id}.quote.role': ${JSON.stringify(study.quote.role)},\n`;
}

const enTsFile = "c:/projects/kalakriti/web/packages/i18n/src/messages/en.ts";
let enTsContent = fs.readFileSync(enTsFile, 'utf8');

if (!enTsContent.includes("'caseStudies.dhamadka-ajrakh.title'")) {
  enTsContent = enTsContent.replace('} as const;', out + '} as const;');
  fs.writeFileSync(enTsFile, enTsContent);
  console.log("Updated en.ts");
} else {
  console.log("en.ts already updated");
}
