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
function createSampleCraftBlob(): Promise<Blob> {
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

    // Rich Indigo Ground
    ctx.fillStyle = '#1e3a8a';
    ctx.fillRect(0, 0, 600, 600);

    // Terracotta Madder Red Border
    ctx.fillStyle = '#991b1b';
    ctx.fillRect(30, 30, 540, 540);

    // Deep Indigo Inner
    ctx.fillStyle = '#172554';
    ctx.fillRect(60, 60, 480, 480);

    // Geometric Star & Floral Block Motifs
    ctx.fillStyle = '#fef08a';
    ctx.strokeStyle = '#ffffff';
    ctx.lineWidth = 2;

    for (let x = 100; x <= 500; x += 80) {
      for (let y = 100; y <= 500; y += 80) {
        ctx.beginPath();
        ctx.arc(x, y, 16, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();

        // 8-pointed star petals
        for (let a = 0; a < Math.PI * 2; a += Math.PI / 4) {
          const px = x + Math.cos(a) * 26;
          const py = y + Math.sin(a) * 26;
          ctx.beginPath();
          ctx.arc(px, py, 6, 0, Math.PI * 2);
          ctx.fillStyle = '#fed7aa';
          ctx.fill();
        }
      }
    }

    // Central GI Seal Marker
    ctx.fillStyle = 'rgba(0, 0, 0, 0.4)';
    ctx.fillRect(150, 520, 300, 36);
    ctx.fillStyle = '#ffffff';
    ctx.font = 'bold 14px sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('GI-72 • DHAMADKA KUTCH AJRAKH', 300, 544);

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
    craftId: 'ajrakh-block-print',
    workingTitle: 'Kutch Natural Ajrakh Modal Silk Dupatta - 16-Stage Indigo Hand-Block',
    dimensions: {
      length_mm: 2400,
      width_mm: 900,
      height_mm: 2,
      weight_g: 220,
    },
    type: 'READY_STOCK',
    priceAmountPaise: 385000, // ₹3,850
    stockQuantity: 12,
    minOrderQuantity: 1,
    studioConfig: {
      backgroundMode: 'white',
      autoLightingApplied: true,
      brightnessOffset: 18,
      contrastOffset: 12,
    },
    attributes: [
      {
        key: 'craft',
        labelKey: 'listing.attribute.craft',
        value: 'Kutch Ajrakh Hand-Block Print (GI-72)',
        source: 'MODEL',
        confidence: 0.96,
        needs_artisan_input: false,
      },
      {
        key: 'material',
        labelKey: 'listing.attribute.material',
        value: 'Pure Modal Silk with Botanical Indigofera Tinctoria & Madder Root',
        source: 'MODEL',
        confidence: 0.91,
        needs_artisan_input: false,
      },
      {
        key: 'technique',
        labelKey: 'listing.attribute.technique',
        value: '16-Stage Mud-Resist Hand-Block Printing using Carved Teak Blocks',
        source: 'MODEL',
        confidence: 0.88,
        needs_artisan_input: false,
      },
    ],
    claims: [
      { sentenceIndex: 0, attributeKey: 'craft' },
      { sentenceIndex: 1, attributeKey: 'technique' },
      { sentenceIndex: 2, attributeKey: 'material' },
    ],
    translations: [
      {
        language: 'en',
        title: 'Kutch Natural Ajrakh Modal Silk Dupatta',
        description:
          'This authentic piece is crafted through traditional Kutch Ajrakh block printing. Hand-finished through 16 stages of botanical resist dyeing, it features pure modal silk with natural indigo and madder root.',
        machine_generated: true,
      },
      {
        language: 'hi',
        title: 'कच्छ प्राकृतिक अजरख मोडल सिल्क दुपट्टा',
        description:
          'यह प्रामाणिक कृति पारंपरिक कच्छ अजरख ब्लॉक प्रिंटिंग द्वारा हस्तनिर्मित है। 16 चरणों की प्राकृतिक डाई प्रक्रिया और शुद्ध मोडल सिल्क से तैयार।',
        machine_generated: true,
      },
    ],
  });

  return draft.id;
}
