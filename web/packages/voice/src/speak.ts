// packages/voice/src/speak.ts
//
// Text-to-speech. The batch spec asks for a fast path playing pre-generated
// audio from a CDN for static strings, falling back to a live TTS endpoint
// for dynamic content -- but services/bff/openapi.json has no TTS endpoint
// and no CDN asset manifest exists to point a fast path at. Inventing either
// would mean guessing a response shape, which the project's absolute rule on
// generated types forbids.
//
// What's used instead: the browser's own SpeechSynthesis API, for every
// utterance, with no network round trip at all. This is arguably a better
// fit for the artisan's connectivity story than either spec'd path (both
// need a request), but it has a real failure mode a CDN path would not:
// SpeechSynthesis needs an installed voice for the target language, and
// Android TTS engines typically ship en-IN and hi-IN but nothing for most of
// the other 20 scheduled languages. voiceAvailable() below exists so a
// caller (ReadScreen.svelte) can detect that and say so, rather than the
// control silently doing nothing when pressed.

export interface SpeakOptions {
  /** BCP 47 tag. Defaults to the synthesizer's own default voice. */
  tag?: string;
  /** A new utterance cancels whatever is currently speaking. Default true. */
  interrupt?: boolean;
}

function supported(): boolean {
  return typeof speechSynthesis !== 'undefined';
}

/**
 * Whether a voice matching `tag` (or its primary subtag) is installed.
 * `getVoices()` can return an empty list before the browser has finished
 * loading them asynchronously -- that's "unknown", not "no voice", so an
 * empty list is treated as available rather than reported as a false
 * negative on the very first call.
 */
export function voiceAvailable(tag: string): boolean {
  if (!supported()) return false;
  const voices = speechSynthesis.getVoices();
  if (voices.length === 0) return true;
  const lower = tag.toLowerCase();
  const primary = lower.split('-')[0];
  return voices.some((v) => {
    const vLang = v.lang.toLowerCase();
    return vLang === lower || vLang === primary || vLang.startsWith(`${primary}-`);
  });
}

/** Resolves when the utterance finishes; rejects if it errors or is cancelled by a newer one. */
export function speak(text: string, options: SpeakOptions = {}): Promise<void> {
  return new Promise((resolve, reject) => {
    if (!supported()) {
      reject(new Error('speechSynthesis is not available'));
      return;
    }
    const { interrupt = true, tag } = options;
    if (interrupt) speechSynthesis.cancel();

    const utterance = new SpeechSynthesisUtterance(text);
    if (tag) utterance.lang = tag;
    utterance.onend = () => resolve();
    utterance.onerror = (event) => reject(new Error(event.error));
    speechSynthesis.speak(utterance);
  });
}

export function stopSpeaking(): void {
  if (supported()) speechSynthesis.cancel();
}
