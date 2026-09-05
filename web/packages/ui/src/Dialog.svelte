<!--
  packages/ui/src/Dialog.svelte

    <Dialog bind:open={confirmOpen} title="Delete this listing?">
      <p>This cannot be undone.</p>
      <Button variant="danger" onclick={confirmDelete}>Delete</Button>
    </Dialog>

  Built on the native <dialog> element via showModal(), not a hand-rolled
  focus trap: a native modal dialog already traps focus, blocks
  interaction with the rest of the page, closes on Escape, and exposes
  aria-modal, all without a line of that logic to get subtly wrong here.
  What native <dialog> does NOT do is restore focus to the element that
  opened it -- that part is handled explicitly below.
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
  let previousActive: HTMLElement | null = null;
  let previousOverflow = '';

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
  class="k-dialog {className || ''}"
  aria-labelledby={titleId}
  onclose={onNativeClose}
  onclick={onBackdropClick}
>
  <div class="k-dialog__header">
    <h2 class="k-dialog__title" id={titleId}>{title}</h2>
    <button type="button" class="k-dialog__close" onclick={close} aria-label={t('ui.dialog.close')}>
      <Icon name="close" />
    </button>
  </div>
  <div class="k-dialog__body">
    {@render children()}
  </div>
</dialog>
