// packages/voice/src/screen-reader.ts
//
// Building the "read this screen" reading order: headings first (the
// outline), then labels and their current values (the content) -- per the
// batch spec, in that order, not interleaved by DOM position. Pure DOM
// traversal, no speech here, so it's testable with jsdom.

export function collectHeadings(root: Element): string[] {
  return Array.from(root.querySelectorAll('h1, h2, h3, h4, h5, h6'))
    .map((el) => el.textContent?.trim())
    .filter((text): text is string => Boolean(text));
}

function labelledControlValue(control: Element): string {
  if (
    control instanceof HTMLInputElement ||
    control instanceof HTMLTextAreaElement ||
    control instanceof HTMLSelectElement
  ) {
    return control.value.trim();
  }
  return '';
}

/**
 * Labels paired with their control's value (or just the label if there's no
 * value yet), followed by paragraphs and list items not already covered by
 * a label -- content the artisan wrote, like a listing's description.
 */
export function collectLabelsAndValues(root: Element): string[] {
  const out: string[] = [];
  const covered = new Set<Element>();

  for (const label of Array.from(root.querySelectorAll('label'))) {
    const labelText = label.textContent?.trim();
    if (!labelText) continue;

    const forId = label.getAttribute('for');
    const control = forId
      ? label.ownerDocument.getElementById(forId)
      : label.querySelector('input, select, textarea');

    if (control) covered.add(control);
    const value = control ? labelledControlValue(control) : '';
    out.push(value ? `${labelText}: ${value}` : labelText);
  }

  for (const el of Array.from(root.querySelectorAll('p, li'))) {
    if (el.closest('label') || covered.has(el)) continue;
    const text = el.textContent?.trim();
    if (text) out.push(text);
  }

  return out;
}

/** The full reading order for one screen: headings, then labels and values. */
export function buildReadingOrder(root: Element): string[] {
  return [...collectHeadings(root), ...collectLabelsAndValues(root)];
}
