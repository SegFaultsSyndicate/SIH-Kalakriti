// apps/buyer/src/lib/wishlist.svelte.ts
//
// Reactive Client-Side Wishlist Manager:
// - Persists user wishlisted craft items to localStorage
// - Synchronizes reactive heart states across ListingCards, GI Directory, and Account
// - Emits friendly toast confirmations upon adding / removing craft treasures

import { showToast } from '@kalakriti/ui';

class WishlistStore {
  // Set of wishlisted item IDs
  items = $state<Record<string, boolean>>({});

  constructor() {
    if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
      try {
        const raw = localStorage.getItem('kalakriti.buyer.wishlist');
        if (raw) {
          this.items = JSON.parse(raw);
        }
      } catch {
        this.items = {};
      }
    }
  }

  has(id: string): boolean {
    return !!this.items[id];
  }

  toggle(id: string, title?: string): boolean {
    const nextState = !this.items[id];
    this.items[id] = nextState;

    if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem('kalakriti.buyer.wishlist', JSON.stringify(this.items));
      } catch {}
    }

    showToast({
      message: title
        ? `"${title}" has been ${nextState ? 'saved to your craft wishlist' : 'removed from your wishlist'}.`
        : nextState
          ? 'Item saved to your craft wishlist.'
          : 'Item removed from your craft wishlist.',
      variant: nextState ? 'success' : 'info',
    });

    return nextState;
  }

  getCount(): number {
    return Object.values(this.items).filter(Boolean).length;
  }
}

export const wishlist = new WishlistStore();
