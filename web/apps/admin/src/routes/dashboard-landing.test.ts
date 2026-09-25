import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it, vi } from 'vitest';

const { goto } = vi.hoisted(() => ({ goto: vi.fn() }));

vi.mock('$app/navigation', () => ({ goto }));
vi.mock('@kalakriti/api', () => ({
  session: { status: 'authenticated', claims: { role: 'MINISTRY' } },
}));
vi.mock('@kalakriti/i18n', () => ({ locale: { t: (key: string) => key } }));
vi.mock('@kalakriti/icons', () => ({ Icon: () => null }));

import Dashboard from './+page.svelte';

describe('admin dashboard landing', () => {
  it('keeps ministry users on the dashboard', async () => {
    render(Dashboard);
    await tick();

    expect(screen.getByRole('heading', { level: 1, name: 'nav.dashboard' })).toBeTruthy();
    expect(goto).not.toHaveBeenCalled();
  });
});
