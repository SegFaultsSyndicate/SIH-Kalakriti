<!--
  packages/ui/src/Button.svelte

    <Button onclick={save}>Save listing</Button>
    <Button variant="danger" size="lg">Delete</Button>
    <Button icon="trash" label="Delete photograph" variant="ghost" />
    <Button loading>Submitting</Button>

  Two shapes, enforced by the exported type rather than a runtime check: a
  button either carries visible content (`children`), or is icon-only and
  MUST carry `label` -- `<Button icon="trash" />` alone is a compile error,
  not a lint warning caught later.

  Loading preserves width: the content span stays in the layout at
  visibility:hidden rather than being removed, so the button does not
  reflow the instant a request starts. The state change is announced once,
  politely, rather than by a spinner alone.
-->
<script lang="ts" module>
  import type { Snippet } from 'svelte';
  import type { IconName } from '@kalakriti/icons';

  export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
  export type ButtonSize = 'sm' | 'md' | 'lg' | 'xl';

  interface Common {
    variant?: ButtonVariant;
    /** xl is the artisan default: the whole system's minimum comfortable tap target. */
    size?: ButtonSize;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    /** Must come from a real pending request -- never a decorative delay. */
    loading?: boolean;
    class?: string;
    onclick?: (event: MouseEvent) => void;
  }

  interface IconOnly extends Common {
    icon: IconName;
    /** Required: this is the button's only accessible name. */
    label: string;
    children?: undefined;
  }

  interface WithContent extends Common {
    icon?: IconName;
    label?: string;
    children: Snippet;
  }

  export type ButtonProps = IconOnly | WithContent;
</script>

<script lang="ts">
  import { Icon, Spinner } from '@kalakriti/icons';
  import { locale } from '@kalakriti/i18n';

  let {
    variant = 'primary',
    size = 'xl',
    type = 'button',
    disabled = false,
    loading = false,
    icon,
    label,
    class: className,
    children,
    ...rest
  }: ButtonProps = $props();

  const t = $derived(locale.t);
  const iconOnly = $derived(!children);
</script>

<button
  class="k-button k-button--{variant} k-button--{size} {className || ''}"
  class:k-button--icon-only={iconOnly}
  {type}
  disabled={disabled || loading}
  aria-busy={loading || undefined}
  aria-label={iconOnly ? label : undefined}
  {...rest}
>
  <span class="k-button__content" class:k-button__content--hidden={loading}>
    {#if icon}
      <Icon name={icon} class="k-button__icon" />
    {/if}
    {#if children}
      {@render children()}
    {/if}
  </span>
  {#if loading}
    <span class="k-button__spinner" role="status" aria-live="polite">
      <Spinner title={t('ui.button.loading')} />
    </span>
  {/if}
</button>
