// apps/buyer/src/lib/order-store.ts
//
// fulfilment.proto has no ListOrders-for-a-buyer RPC (GetOrder only fetches
// one order by id) -- there is nowhere on the backend to page a buyer's own
// order history from. This device remembers the order ids it has created,
// so /orders can rehydrate each one via the real GetOrder. Documented as a
// gap in ml_wiring.md: history is per-device, not per-account.

const KEY = 'kalakriti.buyer.orderIds';

export function rememberOrder(orderId: string): void {
  try {
    const ids = listRememberedOrders();
    if (!ids.includes(orderId)) {
      ids.unshift(orderId);
      localStorage.setItem(KEY, JSON.stringify(ids.slice(0, 50)));
    }
  } catch {
    // Private window / storage disabled -- the order still succeeded server-side.
  }
}

export function listRememberedOrders(): string[] {
  try {
    const raw = localStorage.getItem(KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}
