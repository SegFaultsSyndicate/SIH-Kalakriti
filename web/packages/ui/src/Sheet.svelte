<!--
  packages/ui/src/Sheet.svelte

    <Sheet bind:open={filtersOpen} title="Filter results">
      ...
    </Sheet>

  Same native <dialog> foundation as Dialog.svelte (focus trap, Escape,
  aria-modal, scroll lock, focus restoration all come from there -- see
  its header comment). This adds the mobile bottom-sheet presentation and
  drag-to-dismiss on the handle. Dragging is a bonus gesture, not the only
  way out: the same visible close button Dialog uses is always present,
  because a drag gesture has no keyboard equivalent and one is required.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    open?: boolean;
    title: string;
    class?: string;
    children: Snippet;
  }

  let { open = $bindable(false), title, class: className, children }: Props = $props();

  const t = $derived(locale.t);
  const uid = $props.id();
  const titleId = uid + '-title';

  let dialogEl: HTMLDialogElement;
  let sheetEl: HTMLDivElement;
  let previousActive: HTMLElement | null = null;
  let previousOverflow = '';

  let dragStartY = 0;
  let dragOffset = $state(0);
  let dragging = $state(false);

  const DISMISS_THRESHOLD_PX = 96;

  function close(): void {
    dialogEl.close();
  }

  function onNativeClose(): void {
    open = false;
    document.documentElement.style.overflow = previousOverflow;
    previousActive?.focus();
  }

  function onBackdropClick(event: MouseEvent): void {
    if (event.target === dialogEl) close();
  }

  function onHandlePointerDown(event: PointerEvent): void {
    dragging = true;
    dragStartY = event.clientY;
    sheetEl.setPointerCapture(event.pointerId);
  }

  function onHandlePointerMove(event: PointerEvent): void {
    if (!dragging) return;
    dragOffset = Math.max(0, event.clientY - dragStartY);
  }

  function onHandlePointerUp(): void {
    if (!dragging) return;
    dragging = false;
    if (dragOffset > DISMISS_THRESHOLD_PX) close();
    dragOffset = 0;
  }

  $effect(() => {
    if (!dialogEl) return;
    if (open && !dialogEl.open) {
      previousActive = document.activeElement as HTMLElement | null;
      previousOverflow = document.documentElement.style.overflow;
      document.documentElement.style.overflow = 'hidden';
      dialogEl.showModal();
    } else if (!open && dialogEl.open) {
      close();
    }
  });
</script>

<dialog
  bind:this={dialogEl}
  class="k-dialog k-sheet {className || ''}"
  aria-labelledby={titleId}
  onclose={onNativeClose}
  onclick={onBackdropClick}
>
  <div
    class="k-sheet__panel"
    bind:this={sheetEl}
    style={dragOffset ? `transform:translateY(${dragOffset}px)` : undefined}
    class:k-sheet__panel--dragging={dragging}
  >
    <!--
      Decorative and non-focusable on purpose: this is a bonus pointer
      gesture, not a control. The keyboard-accessible way to close is the
      k-dialog__close button below, always present.
    -->
    <div
      class="k-sheet__handle"
      aria-hidden="true"
      onpointerdown={onHandlePointerDown}
      onpointermove={onHandlePointerMove}
      onpointerup={onHandlePointerUp}
      onpointercancel={onHandlePointerUp}
    >
      <span class="k-sheet__grip"></span>
    </div>

    <div class="k-dialog__header">
      <h2 class="k-dialog__title" id={titleId}>{title}</h2>
      <button type="button" class="k-dialog__close" onclick={close} aria-label={t('ui.sheet.close')}>
        <Icon name="close" />
      </button>
    </div>
    <div class="k-dialog__body">
      {@render children()}
    </div>
  </div>
</dialog>
