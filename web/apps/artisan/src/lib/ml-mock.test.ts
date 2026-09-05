// apps/artisan/src/lib/ml-mock.test.ts
import { describe, expect, it } from 'vitest';
import { assessPhotoQuality, buildMockResult, runMockPipeline, type PipelineStageState } from './ml-mock';

describe('assessPhotoQuality', () => {
  it('passes a normal-sized photo', async () => {
    const result = await assessPhotoQuality(new Blob([new Uint8Array(200_000)]));
    expect(result).toEqual({ passed: true, issues: [] });
  });

  it('flags a near-empty file as no subject', async () => {
    const result = await assessPhotoQuality(new Blob([new Uint8Array(1_000)]));
    expect(result.passed).toBe(false);
    expect(result.issues[0].code).toBe('no_subject');
  });

  it('flags a small-but-not-empty file as blurry', async () => {
    const result = await assessPhotoQuality(new Blob([new Uint8Array(40_000)]));
    expect(result.passed).toBe(false);
    expect(result.issues[0].code).toBe('blurry');
  });
});

describe('runMockPipeline', () => {
  it('gates the upload stage on the real signal, then runs the mock stages in order and returns a result', async () => {
    let uploadsLeft = 2;
    const uploadDone = async () => {
      uploadsLeft -= 1;
      return uploadsLeft <= 0;
    };

    const seen: PipelineStageState[][] = [];
    const result = await runMockPipeline('weaving', 'Silk stole', uploadDone, (s) => seen.push(s), 1);

    // Every stage reaches 'done', in declaration order.
    const finalSnapshot = seen[seen.length - 1];
    expect(finalSnapshot.map((s) => s.key)).toEqual(['upload', 'enhance', 'attributes', 'describe', 'translate']);
    expect(finalSnapshot.every((s) => s.status === 'done')).toBe(true);

    // The upload stage only completed after uploadDone actually returned true.
    expect(uploadsLeft).toBeLessThanOrEqual(0);

    expect(result.translations.map((t) => t.language).sort()).toEqual(['en', 'hi']);
    expect(result.translations.every((t) => t.machine_generated)).toBe(true);
    expect(result.attributes.length).toBeGreaterThan(0);
    expect(result.claims.length).toBeGreaterThan(0);
    // Every claim points at a sentence index and an attribute that exists.
    const attributeKeys = new Set(result.attributes.map((a) => a.key));
    for (const claim of result.claims) {
      expect(attributeKeys.has(claim.attributeKey)).toBe(true);
    }
  });

  describe('buildMockResult', () => {
    it('marks low-confidence attributes for artisan input', () => {
      const result = buildMockResult('weaving', 'Silk stole');
      expect(result.attributes.every((attribute) => typeof attribute.confidence === 'number')).toBe(true);
      expect(result.attributes.some((attribute) => attribute.needs_artisan_input)).toBe(true);
    });
  });
});
