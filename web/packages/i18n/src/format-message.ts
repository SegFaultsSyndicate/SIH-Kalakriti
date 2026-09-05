// packages/i18n/src/format-message.ts
//
// Pure message resolution: placeholder substitution and ICU-lite plural key
// selection. Split out of locale.svelte.ts (which the Svelte compiler must
// process for its runes) so this logic is plain, testable TypeScript.

export type MessageValues = Record<string, string | number>;

/**
 * Substitute {placeholders}. Deliberately dumb -- no expression syntax, no
 * nesting. Anything beyond value substitution is a decision for the call
 * site, which has the context; a mini-language in the catalogue would hide
 * that decision.
 */
export function interpolate(template: string, values?: MessageValues): string {
  if (!values) return template;
  return template.replace(/\{(\w+)\}/g, (whole, name: string) =>
    name in values ? String(values[name]) : whole,
  );
}

/**
 * Which catalogue key to look up for a plural message. `base.<category>`
 * (per Intl.PluralRules for the given tag) if `hasKey` confirms it exists
 * anywhere in the fallback chain, else `base.other` -- every language has an
 * "other" category, so that key is required to exist for any pluralized base.
 */
export function selectPluralKey(
  base: string,
  count: number,
  tag: string,
  hasKey: (key: string) => boolean,
): string {
  const category = new Intl.PluralRules(tag).select(count);
  const categoryKey = `${base}.${category}`;
  return hasKey(categoryKey) ? categoryKey : `${base}.other`;
}
