<!--
  Kalakriti <UploadZone /> — Batch 4

    <UploadZone state="idle">Drag a photo here, or tap to choose</UploadZone>
    <UploadZone state={dropState} />

  One asset (upload-zone.svg), five states, switched by CSS in icons.css —
  see the header comment there for why it isn't five separate SVGs.
-->
<script>
  import UploadIcon from "./src/upload-zone.svg";

  /**
   * @typedef {object} Props
   * @property {"idle"|"drag"|"uploading"|"success"|"error"} [state]
   * @property {import("svelte").Snippet} [children] The zone label.
   */

  /** @type {Props & Record<string, unknown>} */
  let { state = "idle", children, ...rest } = $props();
</script>

<div class="k-upload-zone" data-state={state} {...rest}>
  <UploadIcon aria-hidden="true" focusable="false" />
  <div class="k-upload-zone__label">{@render children?.()}</div>
</div>
