<!--
  Kalakriti <UploadZone /> — Batch 4

    <UploadZone state="idle">Drag a photo here, or tap to choose</UploadZone>
    <UploadZone state={dropState} />

  One asset (upload-zone.svg), five states, switched by CSS in icons.css —
  see the header comment there for why it isn't five separate SVGs.
-->
<script>
  import UploadIcon from './src/upload-zone.svg';

  /** @type {'idle'|'drag'|'uploading'|'success'|'error'} */
  export let state = 'idle';
</script>

<div class="k-upload-zone" data-state={state} {...$$restProps}>
  <UploadIcon aria-hidden="true" focusable="false" />
  <div class="k-upload-zone__label"><slot /></div>
</div>
