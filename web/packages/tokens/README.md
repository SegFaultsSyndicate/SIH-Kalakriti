# @kalakriti/tokens

The design system's ground floor: colour, type, space, line weight, motion, and
the element defaults every surface inherits. Nothing in this package renders a
component. Nothing outside it may hardcode a colour, a size or a duration.

## What to import

```css
@import '@kalakriti/tokens'; /* palette + scale + fonts, and the layer order */
@import '@kalakriti/tokens/reset'; /* the reset */
@import '@kalakriti/tokens/base'; /* element defaults */
```

Import them in that order, at the app root, once. `@kalakriti/tokens` is the
file that declares the cascade layer order:

```css
@layer reset, tokens, base, pattern, component, utility, app;
```

Every other stylesheet in the system opts into one of those layers, so an app
can override a component without a specificity fight and without `!important`.
A Svelte component's own scoped `<style>` block is unlayered and therefore
beats all of them, which is the correct default.

Optionally, and **not in the artisan app**:

```css
@import '@kalakriti/tokens/fonts-indic'; /* self-hosted Anek Indic faces */
```

## Files

| File                 | What lives there                                                      |
| -------------------- | --------------------------------------------------------------------- |
| `palette.css`        | Raw OKLCH-derived scales, and the light/dark/high-contrast aliases     |
| `scale.css`          | Type, space, measure, radius, line weight, elevation, motion, z-index  |
| `fonts.css`          | `@font-face` via fontsource, the family stacks, per-script leading     |
| `fonts-indic.css`    | Optional self-hosted Indic faces. Costs bytes. Read the header first.  |
| `reset.css`          | Modern reset, including the focus ring that must never be removed      |
| `base.css`           | Element defaults, paper grain, skip link, visually-hidden              |
| `brand.js`           | The three hex values the PWA manifest needs, checked against palette   |
| `motifs.md`          | The motif grammar: which tradition, where it is licensed, what is misuse |
| `stroke-geometry-spec.md` | Stroke weights, optical correction, the kolam dot grid            |
| `type-specimens.html` | Design review artefact. Open it in a browser.                         |
| `palette.html`       | Asset-track colour contact sheet                                       |

## Fonts, and why the Indic scripts are not self-hosted by default

Display and body are both **Anek Latin** (Ek Type, OFL, variable 100–800), one
nine-script superfamily drawn on one skeleton. Hierarchy comes from weight,
size and tracking, not from a second download.

The Indic scripts fall through to the platform face. Measured `woff2` payloads,
weight axis only, against a 120 KB per-locale font budget:

| Face                                   | Script subset |
| -------------------------------------- | ------------- |
| Anek Latin                             | **43.7 KB**   |
| Anek Devanagari                        | 251.6 KB      |
| Noto Sans Devanagari (variable)        | 118.4 KB      |
| Mukta, two static weights              | 197.2 KB      |
| Anek Bangla                            | 152.0 KB      |
| Anek Tamil                             | 49.7 KB       |

Hindi is the default locale. Every complete Devanagari face breaks the budget on
its own, before Latin, and every Android in the target range already ships Noto
Sans Devanagari. Spending a quarter of a megabyte over 2G to replace glyphs the
phone already has is the wrong trade, so the default stack names the platform
faces and downloads nothing beyond Latin. Buyer and admin opt into
`fonts-indic.css`; artisan must not.

There is deliberately **no `<link rel="preload">`** for the font. `adapter-static`
emits a content-hashed filename that cannot be known at authoring time, so a
hardcoded preload would 404 silently and cost a round trip for nothing.
`font-display: swap` plus a system fallback in the stack is the correct 2G
behaviour anyway: text paints immediately in a face the device already has.

---

# Design law

A checklist a reviewer runs against any new screen. Every line is a reject, not
a suggestion. If a screen needs an exception, the exception goes in the PR
description, not in the CSS.

## Banned, without exception

- [ ] No purple, violet or indigo-to-pink gradient. No gradient as a primary
      surface at all. (`--k-indigo-*` is Ajrakh dye, a flat colour, and is not
      an exception to this.)
- [ ] No glassmorphism, no `backdrop-filter: blur`, no neumorphism.
- [ ] No floating rounded-2xl white card with a soft shadow on a grey ground.
      Radii stop at `--k-radius-lg` (8px) and there is no card shadow token.
- [ ] No emoji as an icon or a bullet. Anywhere. Including code comments.
- [ ] No centred hero with a big gradient headline and two side-by-side CTAs.
- [ ] No "trusted by" logo wall, no invented testimonial, no pill badge with a
      coloured dot.
- [ ] No Inter, Poppins or Montserrat as the display face.
- [ ] No undraw / blob-people illustration. Illustrations come from
      `@kalakriti/illustrations` and are Warli-derived geometry.
- [ ] No drop shadow as the primary means of separation.

## Required instead

- [ ] **Separation is by line and space.** 1px hairlines
      (`--k-hairline` + `--k-border-hairline`), generous whitespace, honest
      borders. Elevation exists only for things that genuinely float: menu,
      sheet, modal. There are exactly three elevation tokens and no fourth.
- [ ] **Editorial section pattern.** Lowercase kicker, large heading,
      right-aligned "view all".
- [ ] **Product-forward.** The craft photograph is the hero. Ornament frames
      it; ornament never replaces it.
- [ ] **Government-portal trust cues.** Emblem and wordmark lockup, a visible
      language selector, a skip-to-content link, an accessibility control,
      high-contrast body text, plain-language labels, and no dark patterns.
- [ ] **Texture over gloss.** Khadi and paper grain, block-print and jaali
      motifs used structurally. Never a sticker in a corner.
- [ ] **Asymmetry.** A centred column with everything stacked in it is the
      generic layout this system rejects.
- [ ] **Craft names, GI tags, district names and awardee status are first-class
      content**, not metadata in small grey text.

## Tokens

- [ ] No component references a raw palette step. `var(--k-terracotta-700)` in
      a component is a bug; `var(--k-accent-primary-bg)` is the fix.
- [ ] No hardcoded hex, rgb, px font-size, or millisecond duration anywhere
      outside this package.
- [ ] Theme switching changes token values only. If a layout shifts when the
      theme changes, a component is reading something it should not.
- [ ] A new semantic alias is added to all three themes in the same commit, or
      it is not added.

## Accessibility, non-negotiable

- [ ] Semantic HTML first. A `div` with a click handler is a reject.
- [ ] Every interactive element is keyboard-reachable and has a visible focus
      ring. `outline: none` without a replacement is a reject.
- [ ] Every text/background pair used meets 4.5:1; non-text UI boundaries meet
      3:1. Verified in `type-specimens.html`, not estimated.
- [ ] Meaning is never carried by colour alone. Each state has a shape, a
      position or a word as well.
- [ ] Text scales to 200% with no loss of function. No `px` font sizes; every
      `clamp()` term is rem-based, including the ceiling.
- [ ] `prefers-reduced-motion` and `prefers-contrast` are respected. Any new
      animation has a documented reduced-motion behaviour.
- [ ] Touch targets in the artisan app are at least `--k-touch-min` (44px).
- [ ] Every screen is operable by voice, which means every control has an
      accessible name that matches its visible label.
- [ ] No user-visible string is hardcoded in a component. Everything goes
      through `@kalakriti/i18n`.

## Money

- [ ] Money is `int64` paise on the wire and never rendered raw. It goes
      through `formatMoney` from `@kalakriti/i18n`.
- [ ] No float arithmetic on a money value, anywhere, ever.
- [ ] Prices and quantities render with tabular numerals (`.k-tabular`, or
      inside a `table`, which gets it by default).
