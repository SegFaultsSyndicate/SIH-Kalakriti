// apps/artisan/src/app.d.ts
/// <reference types="@sveltejs/kit" />
/// <reference types="vite/client" />
// vite-plugin-pwa/client re-exports the others with explicit .d.ts import
// specifiers, which TypeScript does not resolve, so the vanillajs entry --
// which is where `virtual:pwa-register` is actually declared -- is referenced
// directly.
/// <reference types="vite-plugin-pwa/vanillajs" />
/// <reference types="vite-plugin-pwa/svelte" />
/// <reference types="vite-plugin-pwa/info" />

declare module 'virtual:pwa-register' {
  export function registerSW(options?: {
    immediate?: boolean;
    onNeedRefresh?: () => void;
    onOfflineReady?: () => void;
  }): (reloadPage?: boolean) => Promise<void>;
}

declare global {
  namespace App {
    interface Error {
      /** Message key, so the error boundary can translate rather than echo. */
      messageKey?: string;
    }
    // Locals, PageData, PageState and Platform stay empty: this app has no
    // server, so there is nothing for a hook to put in them.
  }
}

export {};
