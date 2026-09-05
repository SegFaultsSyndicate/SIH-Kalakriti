// packages/ui/src/index.ts
//
// The shared primitive layer, built on @kalakriti/tokens. Import
// '@kalakriti/ui/ui.css' once, at the app root, alongside the asset-pack
// stylesheets it depends on (icons.css, illustrations.css, motion.css,
// patterns.css) -- see each component's own header for which ones it needs.
//
// Card and Skeleton are re-exported from @kalakriti/patterns rather than
// redefined here: Batch 6 already built the flat/hairline/printed surface
// and the weft-shimmer loading block this batch's brief asks for again, and
// this batch only added the `media` card variant and the `shape` skeleton
// presets those files were missing. One definition, not two that could
// drift apart.
export { Card, Skeleton } from '@kalakriti/patterns';

export { default as SkipLink } from './SkipLink.svelte';
export { default as VisuallyHidden } from './VisuallyHidden.svelte';
export { default as RouteAnnouncer } from './RouteAnnouncer.svelte';
export { focusMainHeading } from './focus';
export { a11y, TEXT_SCALE_STEPS, type TextScale, type Contrast } from './a11y.svelte';
export { default as AccessibilityControl } from './AccessibilityControl.svelte';
export { default as LanguageSelector } from './LanguageSelector.svelte';
export { default as Chip } from './Chip.svelte';
export { default as AccessibilityStatement } from './AccessibilityStatement.svelte';
export { default as CraftTerm } from './CraftTerm.svelte';

export { default as Button } from './Button.svelte';
export type { ButtonProps, ButtonVariant, ButtonSize } from './Button.svelte';

export { default as Label } from './Label.svelte';
export { default as Input } from './Input.svelte';
export { default as Textarea } from './Textarea.svelte';
export { default as Select } from './Select.svelte';
export { default as Checkbox } from './Checkbox.svelte';
export { default as Radio } from './Radio.svelte';
export { default as Switch } from './Switch.svelte';
export { default as NumberStepper } from './NumberStepper.svelte';
export { default as FieldGroup } from './FieldGroup.svelte';

export { default as VoiceInput } from './VoiceInput.svelte';
export { default as AudioPlayback } from './AudioPlayback.svelte';

export { default as Sheet } from './Sheet.svelte';
export { default as Dialog } from './Dialog.svelte';

export { default as ToastRegion } from './ToastRegion.svelte';
export { toastQueue, showToast } from './toast.svelte';
export type { ToastOptions, ToastVariant, ToastAction, ToastEntry } from './toast.svelte';

export { default as Tabs } from './Tabs.svelte';
export { default as Accordion } from './Accordion.svelte';
export { default as Tooltip } from './Tooltip.svelte';
export { default as Popover } from './Popover.svelte';

export { default as Image } from './Image.svelte';
export { default as EmptyState } from './EmptyState.svelte';
export { default as SectionHeader } from './SectionHeader.svelte';
export { default as Money } from './Money.svelte';
export { default as Stepper } from './Stepper.svelte';
export { default as SpeakButton } from './SpeakButton.svelte';
export { default as Keypad } from './Keypad.svelte';
export { default as OtpInput } from './OtpInput.svelte';
