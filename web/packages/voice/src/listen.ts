// packages/voice/src/listen.ts
//
// Speech-to-text. Same spec gap as speak.ts: no ASR endpoint exists in
// services/bff/openapi.json to post a recording to. Used instead: the
// browser's SpeechRecognition, which streams interim results natively --
// something a record-then-POST-a-blob endpoint could not give without a
// custom streaming protocol of its own. @kalakriti/ui's VoiceInput.svelte
// still owns raw audio *capture* (a MediaRecorder blob, for upload); this is
// the separate, live-transcript path for voice input and voice navigation.

export function listenSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    (typeof window.SpeechRecognition !== 'undefined' ||
      typeof window.webkitSpeechRecognition !== 'undefined')
  );
}

export interface ListenOptions {
  /** BCP 47 tag for recognition. */
  tag?: string;
  /** Called with the best-effort transcript so far, including interim (non-final) words. */
  onPartial?: (transcript: string) => void;
}

export interface ListenHandle {
  /** Resolves with the final transcript when recognition ends. */
  result: Promise<string>;
  /** Ends recognition early, resolving `result` with whatever was captured. */
  stop: () => void;
}

/** Starts listening immediately. Returns a handle rather than a promise so the caller can stop it. */
export function listen(options: ListenOptions = {}): ListenHandle {
  const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
  if (!Ctor) {
    return { result: Promise.reject(new Error('SpeechRecognition is not available')), stop: () => {} };
  }

  const recognition = new Ctor();
  recognition.lang = options.tag ?? 'en-IN';
  recognition.interimResults = true;
  recognition.continuous = false;
  recognition.maxAlternatives = 1;

  let finalTranscript = '';

  const result = new Promise<string>((resolve, reject) => {
    recognition.onresult = (event) => {
      let interim = '';
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const r = event.results[i];
        if (r.isFinal) finalTranscript += r[0].transcript;
        else interim += r[0].transcript;
      }
      options.onPartial?.((finalTranscript + interim).trim());
    };
    recognition.onerror = (event) => reject(new Error(event.error));
    recognition.onend = () => resolve(finalTranscript.trim());
    recognition.start();
  });

  return { result, stop: () => recognition.stop() };
}
