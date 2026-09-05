# Accessibility — manual verification checklist

Automated coverage (axe-core, contrast ratios, keyboard-never-loses-focus,
360px/200%-text-scale layout) runs in CI — see `.github/workflows/ci.yml`
and `e2e/tests/*/a11y-sweep.spec.ts`. This document is for what automation
cannot check, or hasn't been wired to check yet. An unchecked box here is an
honest statement that the check has not been run — not a claim it failed.

## Manual keyboard traversal (per flow)

Tab/Shift+Tab/Enter/Space/Escape through the whole flow with a mouse
unplugged. Confirm: every interactive element reachable, focus order
matches visual order, no trap, Escape closes any open Sheet/Dialog.

- [ ] Artisan: language → welcome → registration (5 steps)
- [ ] Artisan: capture → pricing → story → review → terms → publish
- [ ] Artisan: listings list → listing detail → provenance
- [ ] Artisan: orders list → order detail → lot offer
- [ ] Artisan: profile, earnings, notifications
- [ ] Buyer: home → search → listing detail → bulk order
- [ ] Buyer: feed (process clips), artisan profile, craft page
- [ ] Buyer: orders list → order detail
- [ ] Admin: login → verify → dashboard
- [ ] Admin: clusters, crafts, insights, moderation
- [ ] Admin: command palette (Ctrl/Cmd+K) open, search, select, close

## Screen reader pass (NVDA or VoiceOver)

Not run this batch — no NVDA/VoiceOver session available in this sandbox.
Needs a real Windows+NVDA or Mac+VoiceOver machine.

- [ ] Artisan capture flow: photograph a piece, price it, add a story,
      review, publish — narrated start to finish, noting anything
      confusing (not just anything invalid)
- [ ] Buyer listing page: navigate from search result to listing detail,
      read price/craft/artisan/provenance, add to order

## Contrast

Automated — `scripts/check-contrast.mjs`, runs in CI (`pnpm check`). Covers
13 token pairs (text/surface, accent, focus-ring, border-interactive,
inverse-surface) across light, dark and high-contrast themes against real
WCAG ratios computed from `packages/tokens/src/palette.css`. No manual
follow-up needed unless a new token pair is introduced that the script's
`PAIRS` array doesn't yet cover — check there first before adding a new row
here.

## 200% zoom

This design system implements "200%" via its own text-scale control
(`packages/tokens/src/scale.css`, `AccessibilityControl`), checked at 360px
+ max scale in `e2e/tests/artisan/stories.spec.ts`'s no-overflow assertion.
Browser-native page zoom (Ctrl/Cmd + repeatedly) has not been separately
swept across all three apps' routes.

- [ ] Browser-native 200% zoom, artisan: registration, capture, listings
- [ ] Browser-native 200% zoom, buyer: search, listing detail, bulk order
- [ ] Browser-native 200% zoom, admin: dashboard, clusters, moderation

## Low-end Android device, network disabled

Not run on real hardware this batch — no device available in this sandbox.
Verified instead via Playwright's `context.setOffline(true)`
(`e2e/tests/artisan/shell.spec.ts`) and DevTools throttling, which is a
different code path than an actual radio dropout.

- [ ] Full artisan flow (capture → publish) on a real low-end Android
      device, airplane mode, not DevTools

## Lighthouse CI, throttled 4G

`.github/workflows/ci.yml`'s `lighthouse` job runs `@lhci/cli` against
artisan and buyer with simulated 4G throttling and a 0.9 minScore
assertion for performance and accessibility (`web/lighthouserc.json`). Set
`continue-on-error: true` until a few real CI runs confirm the assertion
isn't flaky on GitHub's shared runners — this session has no outbound
network to run it and verify end-to-end.

- [ ] Confirm first few CI runs pass consistently, then drop
      `continue-on-error`

## iOS PWA quirks

Documented, not code-workaroundable (these are real iOS Safari
limitations, not bugs in this app):

- **No Background Sync API.** The outbox drains on the next foreground
  visit / reconnect event instead, which is what `$lib/sync.ts`'s
  SyncEngine already does — iOS just never gets the "drain while
  backgrounded" behavior a BackgroundSync-capable browser would.
- **No `beforeinstallprompt`.** `InstallPrompt.svelte` never appears on
  iOS Safari by design — there is no API to intercept. Install path is
  Share → Add to Home Screen, manual, documented in `web/DEMO.md`'s
  troubleshooting table.
- **Aggressive storage eviction.** Safari can evict IndexedDB/Cache
  Storage for a site not opened in 7 days, with no event fired to warn
  the app. `packages/offline/src/storage.ts`'s `requestPersistentStorage()`
  asks for the (Chromium-only) persistence guarantee; on iOS this call is
  a no-op and eviction risk is real for an artisan who doesn't reopen the
  app for a week with unsynced drafts. No code fix exists for this on the
  web platform today — worth flagging to the artisan as "sync before you
  put the phone away for a long trip," not solvable purely client-side.

## E2E journeys still needed

Two of four required Playwright journeys are not yet written (see
`web/DEMO.md`'s gaps section for why — writing them against unverified
mocked schemas risked shipping tests that assert something false):

- [x] Offline capture through resilience (shell-level; `shell.spec.ts`)
- [ ] Offline capture through **published listing**, end to end
- [ ] Buyer cross-lingual search
- [ ] Bulk order with dropout and reallocation
- [ ] Provenance QR verification

## Visual regression baselines

`e2e/tests/artisan/visual.spec.ts` exists (stories page × light/dark/
high-contrast) but no baseline PNGs are committed — this sandbox has no
Playwright browser binaries and screenshots are platform-specific. Run in
CI once and commit the resulting `*-snapshots/` directory:

```
pnpm -F @kalakriti/e2e exec playwright install --with-deps chromium
pnpm e2e -- --update-snapshots
```

- [ ] Generate and commit baselines from a CI (Linux) run
- [ ] Extend to buyer home and admin dashboard, all three themes
