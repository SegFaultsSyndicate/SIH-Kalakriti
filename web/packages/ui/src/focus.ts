// packages/ui/src/focus.ts
//
// Move focus to a page's main heading after navigation. SvelteKit does not
// reset focus on client-side routing, so without this a screen reader user
// who activates a link keeps hearing the OLD page while the new one has
// already rendered underneath them.

/**
 * Focuses the first heading inside `containerSelector` (default: the h1 in
 * <main id="main-content">), giving it tabindex="-1" first if it isn't
 * already focusable so the browser doesn't silently refuse the call.
 */
export function focusMainHeading(containerSelector = '#main-content'): void {
  if (typeof document === 'undefined') return;
  const container = document.querySelector(containerSelector);
  // The page already put the cursor in a field on purpose (e.g. the OTP
  // boxes' autofocus) -- don't steal it. The route title is still announced
  // by RouteAnnouncer's live region.
  const active = document.activeElement;
  if (active && container?.contains(active) && active.matches('input, textarea, select')) return;
  const heading = container?.querySelector<HTMLElement>('h1, [role="heading"][aria-level="1"]');
  const target = heading ?? (container as HTMLElement | null);
  if (!target) return;
  if (!target.hasAttribute('tabindex')) target.setAttribute('tabindex', '-1');
  // Lets reset.css drop the focus ring on this target: it's a non-interactive
  // heading focused by script only so screen readers announce the new page,
  // and a ring around it reads as a stray "selected" box to sighted users.
  target.setAttribute('data-route-focus', '');
  target.focus();
}
