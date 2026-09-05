// packages/voice/src/index.ts
export { speak, stopSpeaking, voiceAvailable, type SpeakOptions } from './speak';
export { listen, listenSupported, type ListenOptions, type ListenHandle } from './listen';
export { matchCommand, hasCommandGrammar, type VoiceCommand } from './commands';
export { buildReadingOrder, collectHeadings, collectLabelsAndValues } from './screen-reader';
export { default as ReadScreen } from './ReadScreen.svelte';
