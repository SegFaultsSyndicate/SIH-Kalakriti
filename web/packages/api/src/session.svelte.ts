// packages/api/src/session.svelte.ts
//
// Session shape the spec can actually back today: POST /auth/otp/verify
// returns only {access_token, refresh_token} (services/bff/openapi.json) --
// no principal, role, or language field anywhere in the response. Those, if
// they exist, live in the JWT's claims; decodeJwtClaims in jwt.ts reads them
// generically since the spec never names them. `language` here is the app's
// own UI locale (set via setAcceptLanguage in config.ts), not a claim.

import { decodeJwtClaims } from './jwt';
import { getAcceptLanguage } from './config';

export type SessionStatus = 'anonymous' | 'authenticated';

class SessionState {
  #status = $state<SessionStatus>('anonymous');
  #claims = $state<Record<string, unknown> | null>(null);
  #language = $state(getAcceptLanguage());

  get status(): SessionStatus {
    return this.#status;
  }

  get claims(): Record<string, unknown> | null {
    return this.#claims;
  }

  get language(): string {
    return this.#language;
  }

  establish(accessToken: string): void {
    this.#claims = decodeJwtClaims(accessToken);
    this.#status = 'authenticated';
  }

  clear(): void {
    this.#claims = null;
    this.#status = 'anonymous';
  }
}

/** One instance per app, same reasoning as packages/i18n's locale store: no SSR, so no request to leak across. */
export const session = new SessionState();
