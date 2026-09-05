// packages/offline/src/connection.test.ts
import { afterEach, describe, expect, it } from 'vitest';
import { isSlowConnection, saveDataEnabled, shouldConserveData } from './connection';

function setConnection(value: { saveData?: boolean; effectiveType?: string } | undefined) {
  Object.defineProperty(navigator, 'connection', { value, configurable: true });
}

afterEach(() => setConnection(undefined));

describe('connection', () => {
  it('defaults to false with no Network Information API', () => {
    expect(saveDataEnabled()).toBe(false);
    expect(isSlowConnection()).toBe(false);
    expect(shouldConserveData()).toBe(false);
  });

  it('reads saveData', () => {
    setConnection({ saveData: true });
    expect(saveDataEnabled()).toBe(true);
    expect(shouldConserveData()).toBe(true);
  });

  it('treats 2g and slow-2g as slow, 4g as not', () => {
    setConnection({ effectiveType: '2g' });
    expect(isSlowConnection()).toBe(true);
    setConnection({ effectiveType: 'slow-2g' });
    expect(isSlowConnection()).toBe(true);
    setConnection({ effectiveType: '4g' });
    expect(isSlowConnection()).toBe(false);
  });
});
