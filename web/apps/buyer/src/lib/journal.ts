// apps/buyer/src/lib/journal.ts
//
// The Virasat journal's essays: one source for /journal (index),
// /journal/[slug] (essay page) and /rss.xml. Text lives in the i18n
// catalogues under journal.essay.<slug>.*; this only holds what isn't prose.
import type { MessageKey } from '@kalakriti/i18n';

export interface Essay {
  slug: 'ajrakh' | 'patola' | 'dhokra';
  heroImage: string;
  /** ISO date the essay was published -- RSS <pubDate> and the index's sort. */
  published: string;
  paragraphs: number;
}

export const ESSAYS: readonly Essay[] = [
  { slug: 'ajrakh', heroImage: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_03.jpeg', published: '2026-03-12', paragraphs: 4 },
  { slug: 'patola', heroImage: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg', published: '2026-05-04', paragraphs: 4 },
  { slug: 'dhokra', heroImage: '/craft-images/metalwork/dhokra-casting.jpg', published: '2026-07-21', paragraphs: 4 },
];

type Field = 'title' | 'author' | 'readTime' | 'cluster' | 'region' | 'lead' | 'quote' | 'quoteAuthor';

export const essayKey = (e: Essay, field: Field): MessageKey => `journal.essay.${e.slug}.${field}` as MessageKey;

export const paragraphKeys = (e: Essay): MessageKey[] =>
  Array.from({ length: e.paragraphs }, (_, i) => `journal.essay.${e.slug}.body.${i + 1}` as MessageKey);
