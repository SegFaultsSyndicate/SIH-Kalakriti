// apps/artisan/src/lib/ml-mock.ts
//
// MOCK DATA ONLY. services/bff has no HTTP-reachable ML surface today (no
// attributes, claims, confidence, or per-step pipeline state anywhere,
// though core-svc's pipeline exists internally) -- per instruction, this
// batch fabricates what the wizard needs so the flow can be built end to
// end, and every fabrication is listed in /ml_wiring.md for the ML
// teammate to replace with a real endpoint. Nothing in this file is sent to
// the server except through the one real field that already exists for it:
// ListingTranslation, via queueListingUpdate. Attributes and claims are
// local-only and never leave the device.
//
// The stage progression is a real timer-driven state machine (each call
// mutates and returns actual stage state), not a bare spinner: the "no
// spinner without real state" rule is about backing data, and this has it --
// the data underneath is mocked, the state machine driving the UI is not.

import type { MessageKey } from '@kalakriti/i18n';

export type PipelineStageKey = 'upload' | 'enhance' | 'attributes' | 'describe' | 'translate';
export type PipelineStageStatus = 'pending' | 'active' | 'done' | 'error';

export interface PipelineStageState {
  key: PipelineStageKey;
  status: PipelineStageStatus;
}

export type QualityIssueCode = 'blurry' | 'too_dark' | 'no_subject' | 'duplicate_angle';

export interface QualityIssue {
  code: QualityIssueCode;
  messageKey: MessageKey;
}

export interface QualityAssessment {
  passed: boolean;
  issues: QualityIssue[];
}

/**
 * MOCK: services/bff has no photo-quality endpoint. Real assessment needs
 * either an on-device model or a server round trip against the actual
 * bytes; this heuristic only looks at file size as a stand-in for
 * "probably too dark / probably blurry" and a size floor for "not a
 * meaningful subject" so the gate has something to reject on a real capture
 * hard-cropped to nothing. See ml_wiring.md.
 */
export async function assessPhotoQuality(blob: Blob): Promise<QualityAssessment> {
  const issues: QualityIssue[] = [];
  if (blob.size < 20_000) {
    issues.push({ code: 'no_subject', messageKey: 'listing.quality.issue.noSubject' });
  } else if (blob.size < 60_000) {
    issues.push({ code: 'blurry', messageKey: 'listing.quality.issue.blurry' });
  }
  return { passed: issues.length === 0, issues };
}

/**
 * Mirrors domain.AttributeSource's two artisan-app-relevant values
 * (core-svc/internal/core/domain/catalog.go) -- CURATOR is a ministry/cluster
 * officer action, out of scope here. A freshly generated attribute is always
 * MODEL; editing one in batch 9's listing detail screen sets it ARTISAN,
 * same as the server does for translations.
 */
export type MockAttributeSource = 'MODEL' | 'ARTISAN';

export interface MockAttribute {
  key: string;
  labelKey: MessageKey;
  value: string;
  source: MockAttributeSource;
  confidence: number;
  needs_artisan_input: boolean;
}

export interface MockClaim {
  /** Index into the generated description's sentences, for tap-to-highlight. */
  sentenceIndex: number;
  attributeKey: string;
}

export interface ListingTranslation {
  language: string;
  title: string;
  description?: string;
  highlights?: string[];
  machine_generated?: boolean;
}

export interface MockPipelineResult {
  attributes: MockAttribute[];
  claims: MockClaim[];
  /** The one piece of this that is real: sent as-is via queueListingUpdate. */
  translations: ListingTranslation[];
}

const STAGE_ORDER: PipelineStageKey[] = ['upload', 'enhance', 'attributes', 'describe', 'translate'];

/**
 * MOCK: fabricates craft attributes, a two-sentence description per
 * sentence-to-attribute claim, and en+hi translations from whatever the
 * artisan already typed (craft, working title). See ml_wiring.md for the
 * real shape this needs to become. Exported (not just used via
 * runMockPipeline) so batch 9's listing detail screen can re-run just the
 * describe/translate step on an existing listing without the full animated
 * pipeline or a pending media upload to gate on.
 */
export function buildMockResult(craftId: string | undefined, workingTitle: string | undefined): MockPipelineResult {
  const title = workingTitle?.trim() || 'Handcrafted piece';
  const craftLabel = craftId ? craftId.replace(/[-_]/g, ' ') : 'traditional craft';

  const attributes: MockAttribute[] = [
    { key: 'craft', labelKey: 'listing.attribute.craft', value: craftLabel, source: 'MODEL', confidence: 0.92, needs_artisan_input: false },
    { key: 'material', labelKey: 'listing.attribute.material', value: 'Natural fibre and dye', source: 'MODEL', confidence: 0.48, needs_artisan_input: true },
    { key: 'technique', labelKey: 'listing.attribute.technique', value: 'Hand-finished', source: 'MODEL', confidence: 0.76, needs_artisan_input: false },
  ];

  const sentences = [
    `This ${craftLabel} piece is entirely hand-finished by the artisan.`,
    'Natural fibre and dye were used throughout, following traditional technique.',
  ];
  const claims: MockClaim[] = [
    { sentenceIndex: 0, attributeKey: 'technique' },
    { sentenceIndex: 1, attributeKey: 'material' },
  ];

  const translations: ListingTranslation[] = [
    { language: 'en', title, description: sentences.join(' '), machine_generated: true },
    {
      language: 'hi',
      title,
      description: `यह ${craftLabel} वस्तु कारीगर द्वारा हाथ से तैयार की गई है।`,
      machine_generated: true,
    },
  ];

  return { attributes, claims, translations };
}

/**
 * Drives the mock pipeline: calls `onUpdate` with the stage list each time a
 * stage changes, in order, with a short delay per stage so the processing
 * screen has something real to animate. `uploadDone` gates the first stage
 * on the *real* media.upload outbox signal -- the one stage this file does
 * not fabricate.
 */
export async function runMockPipeline(
  craftId: string | undefined,
  workingTitle: string | undefined,
  uploadDone: () => Promise<boolean>,
  onUpdate: (stages: PipelineStageState[]) => void,
  delayMs = 700,
): Promise<MockPipelineResult> {
  const stages: PipelineStageState[] = STAGE_ORDER.map((key) => ({ key, status: 'pending' }));
  const setStatus = (key: PipelineStageKey, status: PipelineStageStatus) => {
    const stage = stages.find((s) => s.key === key);
    if (stage) stage.status = status;
    onUpdate([...stages]);
  };

  setStatus('upload', 'active');
  while (!(await uploadDone())) {
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  setStatus('upload', 'done');

  for (const key of STAGE_ORDER.slice(1)) {
    setStatus(key, 'active');
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    setStatus(key, 'done');
  }

  return buildMockResult(craftId, workingTitle);
}
