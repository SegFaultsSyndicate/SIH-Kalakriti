// packages/illustrations/illustrations.d.ts
//
// Hand-written for the same reason as packages/icons/icons.d.ts: the barrel is
// a .js file re-exporting .svelte components, which TypeScript cannot see
// through on its own.
import type { Component, Snippet } from 'svelte';

export interface IllustrationProps {
  name: string;
  /**
   * Accessible name. Omit for a decorative scene, which most empty-state art
   * is -- the surrounding copy carries the meaning.
   */
  title?: string;
  /** Any CSS length. Sets width only. */
  size?: string;
  class?: string;
  [key: string]: unknown;
}

export interface ProcessSequenceProps {
  craft: 'blockprint' | 'weaving' | 'pottery';
  /** Four labels in step order, from the i18n layer. */
  labels?: string[];
  [key: string]: unknown;
}

export interface HeroBackdropProps {
  element?: string;
  children?: Snippet;
  [key: string]: unknown;
}

export const Illustration: Component<IllustrationProps>;
export const ProcessSequence: Component<ProcessSequenceProps>;
export const HeroBackdrop: Component<HeroBackdropProps>;

export interface IllustrationManifestEntry {
  name: string;
  group: string;
  file: string;
}
export const ILLUSTRATIONS: readonly IllustrationManifestEntry[];
