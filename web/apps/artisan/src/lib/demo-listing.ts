// apps/artisan/src/lib/demo-listing.ts
//
// Generates a pre-configured sample listing draft for the "Digital Literacy Demo Mode"
// walkthrough. Enables judges and low-digital-literacy artisans to experience the
// end-to-end listing wizard (AI Studio, Voice Story, Ontology, Fair Pricing) in 60 seconds
// without requiring physical camera capture or typing.

import { db, type MediaRecord } from '@kalakriti/offline';
import { createDraft, patchFields } from '$lib/listing-draft';

/**
 * Creates a sample SVG canvas blob representing an authentic Ajrakh block-printed textile.
 */
/**
 * Creates a sample craft image blob representing the Paithani Silk Saree with golden zari border.
 */
async function createSampleCraftBlob(): Promise<Blob> {
  // First attempt to fetch the actual white-saree-maroon-border image asset if reachable
  try {
    const urls = [
      '/craft-images/weaving_and_looms/paithani-saree-blue-green.jpeg',
      'http://localhost:5173/craft-images/weaving_and_looms/paithani-saree-blue-green.jpeg',
    ];
    for (const url of urls) {
      const res = await fetch(url).catch(() => null);
      if (res && res.ok) {
        const b = await res.blob();
        if (b.size > 100) return b;
      }
    }
  } catch { /* ignore and use canvas */ }

  return new Promise((resolve) => {
    if (typeof document === 'undefined') {
      resolve(new Blob(['mock craft'], { type: 'image/jpeg' }));
      return;
    }

    const canvas = document.createElement('canvas');
    canvas.width = 600;
    canvas.height = 600;
    const ctx = canvas.getContext('2d');
    if (!ctx) {
      resolve(new Blob(['mock craft'], { type: 'image/jpeg' }));
      return;
    }

    // Elegant Cream/White Silk Ground
    ctx.fillStyle = '#fbfbf9';
    ctx.fillRect(0, 0, 600, 600);

    // Deep Maroon Border
    ctx.fillStyle = '#831843';
    ctx.fillRect(0, 480, 600, 120);
    ctx.fillRect(0, 0, 600, 40);

    // Golden Zari Weft Bands
    ctx.fillStyle = '#d97706';
    ctx.fillRect(0, 470, 600, 10);
    ctx.fillRect(0, 40, 600, 6);

    // Intricate Peacock & Oblique Border motifs in gold
    ctx.fillStyle = '#f59e0b';
    for (let x = 30; x < 580; x += 55) {
      ctx.beginPath();
      ctx.arc(x, 535, 14, 0, Math.PI * 2);
      ctx.fill();

      ctx.beginPath();
      ctx.moveTo(x, 505);
      ctx.lineTo(x + 10, 520);
      ctx.lineTo(x - 10, 520);
      ctx.closePath();
      ctx.fillStyle = '#fbbf24';
      ctx.fill();
    }

    // Saree Body Subtle Golden Buttis
    ctx.fillStyle = '#b45309';
    for (let x = 60; x <= 540; x += 80) {
      for (let y = 90; y <= 430; y += 70) {
        ctx.beginPath();
        ctx.arc(x, y, 5, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    // GI Seal Marker
    ctx.fillStyle = 'rgba(0, 0, 0, 0.45)';
    ctx.fillRect(130, 420, 340, 32);
    ctx.fillStyle = '#ffffff';
    ctx.font = 'bold 13px sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('GI-350 \u2022 PAITHANI WEAVING \u2022 HANDLOOM', 300, 441);

    canvas.toBlob((blob) => {
      resolve(blob || new Blob(['mock craft'], { type: 'image/jpeg' }));
    }, 'image/jpeg', 0.95);
  });
}

export async function launchDemoListing(): Promise<string> {
  const draft = await createDraft();
  const sampleBlob = await createSampleCraftBlob();

  const photoRecord: MediaRecord = {
    id: crypto.randomUUID(),
    kind: 'photo',
    blob: sampleBlob,
    mimeType: 'image/jpeg',
    byteSize: sampleBlob.size,
    uploaded: false,
    capturedAt: Date.now(),
    order: 0,
  };

  await db.media.add(photoRecord);
  await db.drafts.update(draft.id, {
    mediaIds: [photoRecord.id],
  });

  await patchFields(draft.id, {
    craftId: 'paithani-weaving',
    workingTitle: 'Paithani Silk Saree (Weaving)',
    dimensions: {
      length_mm: 5500,
      width_mm: 1140,
      height_mm: 2,
      weight_g: 780,
    },
    type: 'READY_STOCK',
    priceAmountPaise: 1850000, // ₹18,500
    stockQuantity: 3,
    minOrderQuantity: 1,
    studioConfig: {
      backgroundMode: 'white',
      autoLightingApplied: true,
      brightnessOffset: 15,
      contrastOffset: 10,
    },
    attributes: [
      {
        name: 'craft',
        value: 'Paithani Silk Handloom Weaving (GI-350)',
        source: 'MODEL',
        confidence: 0.98,
      },
      {
        name: 'material',
        value: 'Pure Mulberry Silk & Fine Gold Zari Weft',
        source: 'MODEL',
        confidence: 0.95,
      },
      {
        name: 'technique',
        value: 'Tapestry Weaving & Hand-Interlocked Kadwa Border',
        source: 'MODEL',
        confidence: 0.92,
      },
    ],
    translations: [
      {
        language: 'en',
        title: 'Paithani Silk Saree (Weaving)',
        description:
          'Exquisite handwoven Paithani Silk Saree handcrafted by Eshaan in Varanasi. Woven from pure mulberry silk with traditional oblique square borders and delicate zari buttis.',
        machine_generated: true,
      },
      {
        language: 'hi',
        title: '\u092a\u0948\u0920\u0923\u0940 \u0938\u093f\u0932\u094d\u0915 \u0938\u093e\u0921\u093c\u0940 (\u0939\u0924\u0915\u0930\u0918\u093e)',
        description:
          '\u0936\u0941\u0926\u094d\u0927 \u092e\u0932\u092c\u0930\u0940 \u0938\u093f\u0932\u094d\u0915 \u0914\u0930 \u0938\u0941\u0928\u0939\u0930\u0940 \u091c\u0930\u0940 \u092c\u0949\u0930\u094d\u0921\u0930 \u0915\u0947 \u0938\u093e\u0925 \u0939\u0938\u094d\u0924\u0928\u093f\u0930\u094d\u092e\u093f\u0924 \u092a\u0948\u0920\u0923\u0940 \u0938\u093e\u0921\u093c\u0940\u0964',
        machine_generated: true,
      },
    ],
  });

  return draft.id;
}

/**
 * Removes obsolete Ajrakh drafts and ensures a clean Paithani Weaving draft exists.
 */
export async function cleanupLegacyDraftsAndSeedPaithani(): Promise<void> {
  try {
    const rows = await db.drafts.toArray();
    let hasPaithaniDraft = false;
    for (const row of rows) {
      const title = String((row.fields as { workingTitle?: string })?.workingTitle || '');
      const craftId = String((row.fields as { craftId?: string })?.craftId || '');
      const isAjrakh = craftId === 'ajrakh-block-print' || title.toLowerCase().includes('ajrakh');
      if (isAjrakh) {
        await db.drafts.delete(row.id);
        for (const mid of row.mediaIds) {
          await db.media.delete(mid).catch(() => {});
        }
      } else if (craftId === 'paithani-weaving' || title.toLowerCase().includes('paithani')) {
        hasPaithaniDraft = true;
      }
    }

    if (!hasPaithaniDraft) {
      await launchDemoListing();
    }
  } catch { /* ignore */ }
}
