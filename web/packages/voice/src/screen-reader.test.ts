// packages/voice/src/screen-reader.test.ts
import { afterEach, describe, expect, it } from 'vitest';
import { buildReadingOrder, collectHeadings, collectLabelsAndValues } from './screen-reader';

// jsdom's `document` persists across tests in this file; a stale #t from a
// previous test would otherwise leak into getElementById lookups below.
afterEach(() => {
  document.body.innerHTML = '';
});

function root(html: string): HTMLElement {
  const el = document.createElement('main');
  el.innerHTML = html;
  document.body.appendChild(el);
  return el;
}

describe('collectHeadings', () => {
  it('collects headings in document order across levels', () => {
    const el = root('<h1>Listing</h1><p>x</p><h2>Photos</h2>');
    expect(collectHeadings(el)).toEqual(['Listing', 'Photos']);
  });

  it('skips an empty heading', () => {
    const el = root('<h1></h1><h2>Real</h2>');
    expect(collectHeadings(el)).toEqual(['Real']);
  });
});

describe('collectLabelsAndValues', () => {
  it('pairs a label with its control value via for=', () => {
    const el = root('<label for="t">Title</label><input id="t" value="Silk scarf" />');
    expect(collectLabelsAndValues(el)).toEqual(['Title: Silk scarf']);
  });

  it('pairs a label wrapping its control', () => {
    const el = root('<label>Title<input value="Silk scarf" /></label>');
    expect(collectLabelsAndValues(el)).toEqual(['Title: Silk scarf']);
  });

  it('reads a label alone when the control has no value yet', () => {
    const el = root('<label for="t">Title</label><input id="t" />');
    expect(collectLabelsAndValues(el)).toEqual(['Title']);
  });

  it('includes paragraphs and list items not covered by a label', () => {
    const el = root('<p>Hand-thrown clay pot.</p><ul><li>Fired at 1200C</li></ul>');
    expect(collectLabelsAndValues(el)).toEqual(['Hand-thrown clay pot.', 'Fired at 1200C']);
  });

  it('does not double-count a control wrapped inside its own label as a paragraph', () => {
    const el = root('<label>Title<input value="x" /></label><p>Description</p>');
    expect(collectLabelsAndValues(el)).toEqual(['Title: x', 'Description']);
  });
});

describe('buildReadingOrder', () => {
  it('puts every heading before any label or value', () => {
    const el = root(
      '<h1>Listing</h1><label for="t">Title</label><input id="t" value="Pot" /><h2>Photos</h2><p>3 uploaded</p>',
    );
    expect(buildReadingOrder(el)).toEqual(['Listing', 'Photos', 'Title: Pot', '3 uploaded']);
  });
});
