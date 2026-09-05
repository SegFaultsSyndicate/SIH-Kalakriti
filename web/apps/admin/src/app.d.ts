// apps/admin/src/app.d.ts
/// <reference types="@sveltejs/kit" />
/// <reference types="vite/client" />

declare global {
  namespace App {
    interface Error {
      messageKey?: string;
    }
  }
}

export {};
