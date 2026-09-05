<!--
  packages/ui/src/AccessibilityStatement.svelte

  Used from each app's routes/accessibility/+page.svelte as
  `import { AccessibilityStatement } from '@kalakriti/ui'`, then rendered as
  `<AccessibilityStatement />`.

  Shared body text for the statement page every app links to from
  AccessibilityControl. One definition so the three apps can't drift apart
  on what they claim to support; each app supplies its own route/layout
  chrome around it.

  (The usage example above is written as prose rather than a fenced code
  block: a literal `<script>...</script>` pair inside this leading HTML
  comment made the Svelte compiler see two top-level script blocks in this
  file and fail with "A component can have a single top-level <script>
  element" -- it does not parse comments as fully opaque before scanning for
  script tags. Keep any future example here free of literal <script> tags.)
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';

  const t = $derived(locale.t);
</script>

<h1>{t('a11y.statement.title')}</h1>
<p>{t('a11y.statement.body')}</p>

<h2>{t('a11y.statement.conformance.title')}</h2>
<p>{t('a11y.statement.conformance.body')}</p>

<h2>{t('a11y.statement.gaps.title')}</h2>
<ul>
  <li>{t('a11y.statement.gaps.item1')}</li>
  <li>{t('a11y.statement.gaps.item2')}</li>
  <li>{t('a11y.statement.gaps.item3')}</li>
</ul>

<h2>{t('a11y.statement.contact.title')}</h2>
<p>{t('a11y.statement.contact.body')}</p>
