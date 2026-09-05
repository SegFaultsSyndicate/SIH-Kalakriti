# Kalakriti — demo runbook

This is the real, current architecture: a pnpm workspace under `web/`
(artisan, buyer, admin — SvelteKit 2 / Svelte 5) talking to a Go BFF
(`services/bff`), which fronts core-svc over gRPC. There is no
docker-compose orchestration for this stack (an older, unrelated
implementation attempt left a `md build notes/DEMO.md` referencing one —
ignore it, it does not describe this codebase).

## Running it

```
pnpm install
pnpm -F @kalakriti/artisan dev   # :5173
pnpm -F @kalakriti/buyer dev     # :5174 (default SvelteKit port, or as configured)
pnpm -F @kalakriti/admin dev
```

Each app needs the BFF reachable at whatever `@kalakriti/api`'s transport is
configured to hit — see `packages/api/src/transport.ts`. Without it running,
every screen still boots (offline-first), but nothing beyond local/cached
state resolves.

## DEMO_MODE

`PUBLIC_DEMO_MODE=1` (read via `$lib/demo.ts` in each app) marks a demo
build. **It does not seed any data** — there is no seed-data backend
endpoint in this repository (only `services/core-svc/cmd/seed-ontology`,
which seeds the craft ontology, not artisans/listings/orders). Today the
flag is a hook for cosmetic/timing choices only (e.g. a fixed rather than
randomized empty-state illustration, so a rehearsed script always shows the
same screen). Building real deterministic demo seed data is backend work
outside this batch's frontend scope — flagged here rather than faked.

Until that exists, the script below walks the golden path against whatever
data is actually in the target environment. Rehearse it once against that
environment before presenting, and note the artisan/listing names it
actually uses.

## 8-minute script

Timings are targets for a rehearsed run, not hard limits.

| # | Time | Step | App | What to show |
|---|------|------|-----|---------------|
| 1 | 0:00 | Language + welcome | artisan | `/language` → pick Hindi, `/welcome` |
| 2 | 0:30 | Register | artisan | `/register/name` → `/register/craft` → `/register/district` → `/register/cluster` → `/register/pehchan` |
| 3 | 1:30 | Capture offline | artisan | Turn on airplane mode. `/listing/new/capture` → photograph a piece → `/listing/new/pricing` → `/listing/new/story` → `/listing/new/review` → `/listing/new/terms`. Point out the outbox indicator queuing the work. |
| 4 | 3:30 | Reconnect + publish | artisan | Turn airplane mode off. Watch the outbox drain (`$lib/sync.ts`'s SyncEngine). Listing reaches PUBLISHED. |
| 5 | 4:30 | Install prompt | artisan | Now that a listing has published, the install banner appears (`$lib/InstallPrompt.svelte`) — this is the point of the demo, not first load. |
| 6 | 5:00 | Buyer discovery | buyer | `/feed` (process-provenance clips) → `/search` cross-lingual: search a Hindi term, show English-titled results matching it. |
| 7 | 6:00 | Provenance | buyer/artisan | Open the published listing, scan/open its provenance QR (`/listings/[id]/provenance`), show the seal. |
| 8 | 7:00 | Ministry view | admin | `/moderation` queue, or `/insights` if the environment has order volume. |
| 9 | 7:45 | Wrap | — | Accessibility statement (`/accessibility`) and offline-first pitch. |

## Verifying the artisan flow offline (real device, not DevTools)

DevTools' network throttling does not exercise the same code path as an
actual radio dropout, and this session has no physical low-end Android
device available to borrow, so this line item is **not verified on real
hardware** — it is verified only via Playwright's `context.setOffline(true)`
(`e2e/tests/artisan/shell.spec.ts`) and manual airplane-mode testing on
whatever device runs this rehearsal. Before presenting: reproduce step 3-4
above on the actual demo device with real airplane mode, not just the
emulator/DevTools, and update this line once done.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Outbox never drains after reconnecting | BFF unreachable at the configured API base URL | Check `packages/api/src/transport.ts`'s base URL against what's actually running |
| Install prompt never appears | No listing has reached PUBLISHED on this device yet, or Chrome decided the page isn't installable this session | Confirm a listing shows `state: PUBLISHED` in `/listings`; `beforeinstallprompt` is Chromium-only, won't fire in Firefox/Safari |
| Update-available toast appears mid-capture | Should not happen — the SW never auto-claims (`clientsClaim: false`) and the prompt is dismissible, never forced | If seen, file it as a real bug, not a demo quirk |
| Hindi search doesn't match English titles | Craft ontology alias index stale | `POST /admin/crafts/refresh` (admin `/crafts` page or `RefreshCraftIndex`) |
| iOS: no install banner at all | Expected — iOS Safari never fires `beforeinstallprompt`. Install path is Share → Add to Home Screen, walk it manually if presenting on an iPhone. | — |

## Known gaps in this batch's e2e coverage

Only two of the four required Playwright journeys exist as of this batch:
offline-resilience at the shell level (`shell.spec.ts`) and the axe/visual
sweeps added this batch. Not yet written: buyer cross-lingual search,
bulk-order dropout/reallocation, and provenance QR verification, as
end-to-end specs — writing them against mocked BFF responses risked
committing tests that assert against a schema I had not verified this turn,
which is worse than an honest gap. See web/ACCESSIBILITY.md for the full
punch list.
