// packages/i18n/src/dnt-terms.ts
//
// A craft term flagged "do not translate": rendered verbatim, never passed
// through interpolation or any client-side transformation. CraftTerm.svelte
// (in @kalakriti/ui) is the component that renders one.
//
// This is a plain, hand-written type, not a generated one, and that is
// deliberate rather than an oversight: the batch spec assumes a
// `DescriptionResult` envelope with a `dnt_terms` field, but no such schema
// exists in services/bff/openapi.json (the ML description-generation
// endpoint isn't in the spec yet). Inventing that envelope's shape would
// violate the rule that request/response types are generated, not guessed.
// What's real is the concept -- a term with a canonical name, region and GI
// status that must render unchanged -- so that's what's typed here. When the
// endpoint lands, the caller maps its actual response shape onto this type;
// this type does not change.
export interface DntTerm {
  /** The term as it must appear on screen -- exact script, exact spelling. */
  term: string;
  canonicalName: string;
  region?: string;
  giStatus?: string;
}
