<!--
  packages/ui/src/VoiceInput.svelte

    <VoiceInput onrecording={(blob) => queueForUpload(blob)} />
    <VoiceInput mode="hold" onrecording={handleClip} />

  The signature control: idle -> recording -> processing -> result | error.
  `mode="tap"` (default) toggles on click, for a user who cannot sustain a
  hold; `mode="hold"` starts on press and stops on release, for a noisy
  room where a stray second tap would otherwise restart it. Both are wired
  to the same button, so keyboard support (Enter/Space) works identically
  in either mode without a second code path.

  The waveform is @kalakriti/motion's VoiceWaveform driven by a real
  AnalyserNode -- this component never fakes amplitude. Every audio
  resource opened in start() (the stream's tracks, the AudioContext) is
  torn down in cleanup(), which also runs on unmount, so leaving the
  screen mid-recording cannot leave a live microphone behind.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { VoiceWaveform } from '@kalakriti/motion';
  import Button from './Button.svelte';

  type Phase = 'idle' | 'recording' | 'processing' | 'result' | 'error';

  interface Props {
    mode?: 'tap' | 'hold';
    disabled?: boolean;
    /** Fires once per completed recording, with the captured audio. */
    onrecording?: (blob: Blob) => void;
    class?: string;
  }

  let { mode = 'tap', disabled = false, onrecording, class: className }: Props = $props();

  const t = $derived(locale.t);

  let phase = $state<Phase>('idle');
  let analyser = $state<AnalyserNode | undefined>(undefined);
  let errorMessage = $state('');

  let stream: MediaStream | undefined;
  let audioCtx: AudioContext | undefined;
  let recorder: MediaRecorder | undefined;
  let chunks: Blob[] = [];

  const PHASE_LABEL: Record<Phase, () => string> = {
    idle: () => (mode === 'hold' ? t('ui.voice.idle.hold') : t('ui.voice.idle')),
    recording: () => (mode === 'hold' ? t('ui.voice.recording.hold') : t('ui.voice.recording')),
    processing: () => t('ui.voice.processing'),
    result: () => t('ui.voice.result'),
    error: () => errorMessage || t('ui.voice.error'),
  };
  const label = $derived(PHASE_LABEL[phase]());

  function cleanupAudio(): void {
    stream?.getTracks().forEach((track) => track.stop());
    void audioCtx?.close();
    stream = undefined;
    audioCtx = undefined;
    recorder = undefined;
    analyser = undefined;
  }

  async function start(): Promise<void> {
    if (phase === 'recording' || disabled) return;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      phase = 'error';
      errorMessage = t('ui.voice.permission');
      return;
    }
    audioCtx = new AudioContext();
    const node = audioCtx.createAnalyser();
    node.fftSize = 64;
    audioCtx.createMediaStreamSource(stream).connect(node);
    analyser = node;

    chunks = [];
    recorder = new MediaRecorder(stream);
    recorder.ondataavailable = (event) => {
      if (event.data.size > 0) chunks.push(event.data);
    };
    recorder.onstop = () => {
      const blob = new Blob(chunks, { type: recorder?.mimeType || 'audio/webm' });
      cleanupAudio();
      phase = 'processing';
      // A completed MediaRecorder blob needs no further work, but skipping
      // straight past "processing" would make the state invisible -- one
      // frame is enough for it to register before the result appears.
      requestAnimationFrame(() => {
        onrecording?.(blob);
        phase = 'result';
      });
    };
    recorder.start();
    phase = 'recording';
  }

  function stop(): void {
    if (phase !== 'recording') return;
    recorder?.stop();
  }

  function reset(): void {
    cleanupAudio();
    errorMessage = '';
    phase = 'idle';
  }

  function onclick(): void {
    if (mode !== 'tap') return;
    if (phase === 'recording') stop();
    else if (phase === 'idle' || phase === 'error') void start();
    else if (phase === 'result') reset();
  }

  function onpointerdown(): void {
    if (mode !== 'hold') return;
    if (phase === 'idle' || phase === 'error' || phase === 'result') void start();
  }

  function onpointerup(): void {
    if (mode === 'hold') stop();
  }

  // Only needed for hold mode: a real <button> already synthesizes a click
  // from Enter/Space, so tap mode's keyboard support comes free from the
  // native element and must NOT also run this or every key press would
  // toggle twice.
  function onkeydown(event: KeyboardEvent): void {
    if (event.repeat || (event.key !== 'Enter' && event.key !== ' ')) return;
    event.preventDefault();
    onpointerdown();
  }

  function onkeyup(event: KeyboardEvent): void {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    onpointerup();
  }

  $effect(() => {
    return () => cleanupAudio();
  });
</script>

<div class="k-voice {className || ''}" data-phase={phase}>
  <button
    type="button"
    class="k-voice__control"
    {disabled}
    aria-label={label}
    onclick={mode === 'tap' ? onclick : undefined}
    onpointerdown={mode === 'hold' ? onpointerdown : undefined}
    onpointerup={mode === 'hold' ? onpointerup : undefined}
    onpointerleave={mode === 'hold' ? onpointerup : undefined}
    onkeydown={mode === 'hold' ? onkeydown : undefined}
    onkeyup={mode === 'hold' ? onkeyup : undefined}
  >
    {#if phase === 'recording'}
      <Icon name="microphone" class="k-voice__icon k-voice__icon--live" />
    {:else if phase === 'processing'}
      <Icon name="sync" class="k-voice__icon k-voice__icon--spin" />
    {:else if phase === 'result'}
      <Icon name="success" class="k-voice__icon" />
    {:else if phase === 'error'}
      <Icon name="warning" class="k-voice__icon" />
    {:else}
      <Icon name="microphone" class="k-voice__icon" />
    {/if}
  </button>

  {#if phase === 'recording'}
    <VoiceWaveform {analyser} label={(p) => t('ui.voice.level', { p })} />
  {/if}

  <p class="k-voice__caption" role="status" aria-live="polite">{label}</p>

  {#if phase === 'result' || phase === 'error'}
    <Button variant="ghost" size="sm" onclick={reset}>
      {phase === 'error' ? t('state.error.retry') : t('ui.voice.discard')}
    </Button>
  {/if}
</div>
