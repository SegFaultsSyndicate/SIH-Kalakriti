import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import { BadgeGrid } from '@kalakriti/ui';

describe('BadgeGrid', () => {
  it('shows the empty state when the artisan has no badges', () => {
    render(BadgeGrid, { props: { catalog: [], granted: [], progress: [] } });
    expect(screen.getByText(/no badges yet/i)).toBeTruthy();
  });
});
