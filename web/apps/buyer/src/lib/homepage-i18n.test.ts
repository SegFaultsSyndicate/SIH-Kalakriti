import { describe, it, expect, beforeEach } from 'vitest';
import { locale } from '@kalakriti/i18n';

describe('Homepage products i18n localization', () => {
  beforeEach(async () => {
    await locale.init();
  });

  it('translates fallback listing titles across multiple languages', async () => {
    // English
    await locale.set('en');
    expect(locale.t('home.fallbackListing.gi1.title')).toBe(
      'Kashi Kadwa Pure Silver-Gilt Pit-Loom Saree'
    );
    expect(locale.t('home.fallbackListing.gi1.craftName')).toBe('Banarasi Brocade Weaving');
    expect(locale.t('home.fallbackListing.gi1.artisanName')).toBe('Mohammad Kabir Ansari');

    // Hindi
    await locale.set('hi');
    expect(locale.t('home.fallbackListing.gi1.title')).toBe(
      'काशी कड़वा शुद्ध रजत-जरी पिट-लूम साड़ी'
    );
    expect(locale.t('home.fallbackListing.gi1.craftName')).toBe('बनारसी ब्रोकेड बुनाई');
    expect(locale.t('home.fallbackListing.gi1.artisanName')).toBe('मोहम्मद कबीर अंसारी');

    // Bengali
    await locale.set('bn');
    expect(locale.t('home.fallbackListing.gi1.title')).toBe(
      'কাশী কড়য়া খাঁটি রুপার জরি পিট-লুম শাড়ি'
    );
    expect(locale.t('home.fallbackListing.gi1.craftName')).toBe('বেনারসি ব্রোকেড বুনন');

    // Tamil
    await locale.set('ta');
    expect(locale.t('home.fallbackListing.gi1.title')).toBe(
      'காசி கட்வா தூய வெள்ளி ஜரி பிட்-லூம் புடவை'
    );
    expect(locale.t('home.fallbackListing.gi1.craftName')).toBe('பனாரசி ப்ரோகேட் நெசவு');

    // Marathi
    await locale.set('mr');
    expect(locale.t('home.fallbackListing.gi1.title')).toBe(
      'काशी कडवा शुद्ध रौप्य जरी खड्डा-माग साडी'
    );
    expect(locale.t('home.fallbackListing.gi1.craftName')).toBe('बनारसी ब्रोकेड विणकाम');

    // Reset back to English
    await locale.set('en');
  });

  it('translates all fallback products in GI and New Arrivals rails', async () => {
    await locale.set('hi');

    const listingKeys = [
      'home.fallbackListing.gi1.title',
      'home.fallbackListing.gi2.title',
      'home.fallbackListing.gi3.title',
      'home.fallbackListing.gi4.title',
      'home.fallbackListing.arr1.title',
      'home.fallbackListing.arr2.title',
      'home.fallbackListing.arr3.title',
    ] as const;

    for (const key of listingKeys) {
      const translated = locale.t(key);
      expect(translated).toBeDefined();
      expect(translated.length).toBeGreaterThan(0);
      // Ensure it is not English
      expect(translated).toMatch(/[\u0900-\u097F]/);
    }

    await locale.set('en');
  });

  it('translates discovery tags on homepage', async () => {
    await locale.set('hi');
    expect(locale.t('home.discoveryTag.ajrakh.name')).toBe('कच्छ अजरक');
    expect(locale.t('home.discoveryTag.banarasi.name')).toBe('बनारसी कड़वा');
    expect(locale.t('home.discoveryTag.pashmina.name')).toBe('कश्मीर पश्मीना');

    await locale.set('en');
    expect(locale.t('home.discoveryTag.ajrakh.name')).toBe('Kutch Ajrakh');
  });
});
