import { request } from './transport';
import { getRefreshToken as getStoredRefreshToken, setRefreshToken as setStoredRefreshToken, setAccessToken } from './auth';
import { getSessionRefreshHandler } from './config';

export function setRefreshToken(token: string | undefined): void {
  setStoredRefreshToken(token);
}

export function getRefreshToken(): string | undefined {
  return getStoredRefreshToken();
}

export async function refreshSession(token = getStoredRefreshToken()): Promise<boolean> {
  if (!token) {
    setAccessToken(undefined);
    return false;
  }

  try {
    const response = (await request('/auth/refresh', {
      method: 'POST',
      body: { refresh_token: token },
      timeoutMs: 10_000,
    })) as { access_token?: string };
    if (response.access_token === undefined) {
      throw new Error('Refresh response did not include an access token');
    }
    setAccessToken(response.access_token);
    getSessionRefreshHandler()?.(response.access_token);
    return true;
  } catch {
    setStoredRefreshToken(undefined);
    setAccessToken(undefined);
    return false;
  }
}
