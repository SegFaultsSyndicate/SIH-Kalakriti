// packages/ui/src/toast.svelte.ts
//
// The toast queue. A module-scope class rather than a component prop chain,
// for the same reason packages/i18n/src/locale.svelte.ts is one: `toast.show()`
// needs to be callable from anywhere -- a submit handler, an offline sync
// callback -- without threading a prop down to wherever <ToastRegion> happens
// to be mounted.
//
// Auto-dismiss timers pause on hover/focus by tracking a deadline rather than
// a running countdown: pausing just clears the timeout and records how much
// was left, resuming restarts a fresh timeout for that remainder. Nothing
// polls, and unmounting a paused toast leaves no timer running because
// dismiss() always clears it.

export interface ToastAction {
  label: string;
  onclick: () => void;
}

export type ToastVariant = 'info' | 'success' | 'error';

export interface ToastOptions {
  message: string;
  variant?: ToastVariant;
  /** ms before auto-dismiss, or 0 to require manual dismissal. */
  duration?: number;
  action?: ToastAction;
}

export interface ToastEntry {
  id: number;
  message: string;
  variant: ToastVariant;
  action?: ToastAction;
}

interface Internal extends ToastEntry {
  /** ms left before auto-dismiss, updated whenever the timer is paused. */
  remaining: number;
  timer?: ReturnType<typeof setTimeout>;
  /** When the current timer was armed, so pause() can compute what is left. */
  armedAt?: number;
}

let nextId = 0;

class ToastQueue {
  #items = $state<Internal[]>([]);

  get items(): ToastEntry[] {
    return this.#items;
  }

  show(options: ToastOptions): number {
    const id = nextId++;
    const variant = options.variant ?? 'info';
    const entry: Internal = {
      id,
      message: options.message,
      variant,
      action: options.action,
      remaining: options.duration ?? (variant === 'error' ? 8000 : 5000),
    };
    this.#items = [...this.#items, entry];
    this.#arm(entry);
    return id;
  }

  dismiss(id: number): void {
    const entry = this.#items.find((item) => item.id === id);
    if (entry?.timer) clearTimeout(entry.timer);
    this.#items = this.#items.filter((item) => item.id !== id);
  }

  pause(id: number): void {
    const entry = this.#items.find((item) => item.id === id);
    if (!entry?.timer || entry.armedAt === undefined) return;
    clearTimeout(entry.timer);
    entry.remaining = Math.max(0, entry.remaining - (Date.now() - entry.armedAt));
    entry.timer = undefined;
  }

  resume(id: number): void {
    const entry = this.#items.find((item) => item.id === id);
    if (entry && !entry.timer) this.#arm(entry);
  }

  #arm(entry: Internal): void {
    if (entry.remaining <= 0) return; // 0 means "stays until dismissed"
    entry.armedAt = Date.now();
    entry.timer = setTimeout(() => this.dismiss(entry.id), entry.remaining);
  }
}

export const toastQueue = new ToastQueue();

export function showToast(options: ToastOptions): number {
  return toastQueue.show(options);
}
