<!--
  Kalakriti <QRFrame /> — Batch 6

    <QRFrame>
      <img src={qrDataUrl} alt="Scan to verify this item's provenance" />
    </QRFrame>

  Draws the jaali surround only. The slot's size is fixed by the frame's
  own quiet-zone math (see src/qr-frame.svg's comment) — do not shrink or
  pad whatever you put in the slot, that reopens the quiet-zone risk the
  frame exists to prevent.

  NOT verified against a real phone scan. Confirm with a printed tag before
  relying on this in production — see the SVG's own comment.
-->
<script>
  import Frame from './src/qr-frame.svg';
</script>

<div class="k-qr-frame" {...$$restProps}>
  <Frame aria-hidden="true" focusable="false" />
  <div class="k-qr-frame__slot">
    <slot />
  </div>
</div>
