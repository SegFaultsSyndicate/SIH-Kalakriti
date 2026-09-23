# MoSJE tier 4 — pitch / demo notes

Notes for whoever demos this batch to a non-engineering audience (MoSJE
stakeholders, a program reviewer). Written against the actual code, not the
original build spec — where a claim below can't be backed by a route, a
migration or a running check, it's called out as such rather than implied.

## What this batch adds, in plain terms

- **"Is Kalakriti actually raising artisans' income?"** — an artisan can
  record what they earned before joining (`income.baseline`), and the app
  then compares that against their trailing 90-day income (platform sales +
  offline sales they log by hand) to show real growth, honestly declining to
  guess when it can't: no baseline given, fewer than 90 days on the
  platform, or no sales yet all show their own explanation instead of a
  fabricated number.
- **A ministry-facing impact dashboard** (`/impact`) rolls the same figures
  up by district, social category and finance corporation — medians only,
  and any group under 5 artisans is hidden ("<5", never a bare 0), so no
  individual's income is ever identifiable from an aggregate.
- **Government loan linkage** — an artisan can link an existing loan from
  NSFDC/NBCFDC/NSKFDC/NDFDC/PM-DAKSH/PM-AJAY and see whether this month's
  sales cover the EMI. The full loan number is never stored: only its last 4
  characters and a salted hash (for de-duplication) survive past the request
  that submitted it.
- **Field-agent "assisted mode"** — an artisan who can't operate the app
  themselves (no smartphone literacy, no signal) can have a CSC field agent
  act for them, with consent captured either by OTP or a recorded voice
  statement an officer later reviews. The agent's access is scoped to a
  fixed allow-list of routes (profile, listings, media, pricing — never
  payments, orders, follows or consent withdrawal) and is revocable by the
  artisan at any time.
- **An 8-lesson digital-skills track** ending in a verifiable "Digital
  Ready" certificate — anyone holding a printed certificate can check it's
  genuine at a public URL with no login.

## What's genuinely demoable right now

Every route above exists, is wired into the real bff/core-svc/insight-svc
services, and is covered by the checked-in unit/integration test suites
(`go test`, the frontend `vitest` suites). `services/bff/openapi.json` and
the generated frontend client (`web/packages/api`) match the real route
table — see `docs/API.md` for the full map.

## What is NOT verified — say this out loud before anyone asks

- **No end-to-end run against a live stack.** This environment has no
  Docker, so `make migrate-up`, `make demo-up` and the Go/TS integration
  test suites that need a real Postgres have not been executed here. Go
  unit tests, `go build`/`go vet`, and the frontend `vitest`/`svelte-check`
  suites that don't need a database have run and pass.
- **Demo data is seeded but unrun for the same reason.** `cmd/seed-demo`
  seeds artisans, income history and finance links across two impact-
  dashboard cohorts (`CLAUDE.md`'s MoSJE tier 4 section has the details) —
  written and `go build`/`go vet`/`go test`-clean, not exercised against a
  live database.
- **No real buyer or ministry sign-up path exists yet.** Every OTP login
  mints an artisan token (`VerifyOtp` hardcodes `RoleArtisan`); MINISTRY and
  CLUSTER_OFFICER accounts exist only via the `create-staff` CLI bootstrap
  or an existing ministry user creating more through `/admin/staff`. Don't
  demo "a buyer signs up" as a real flow — there isn't one.
- **brx (Bodo), ks (Kashmiri) and sd (Sindhi) translations for this batch
  are machine-assisted, not reviewed by a native speaker.** Every other
  locale's tier-4 translation is a human/reviewed batch from earlier in this
  project; these three are new and should get a review pass before being
  called production-quality in front of a Ministry audience that includes
  speakers of those languages.
- **The nskfdc and pm_daksh scheme URLs are unverified** — their own domains
  didn't resolve when `037_finance_link.sql` was written, so both point at
  the general `socialjustice.gov.in` portal instead. Don't present those two
  as "the corporation's own site."

## Suggested demo path

1. Register an artisan (or use a seeded one), set an income baseline, log an
   offline sale — show `/income/summary` reflect it. Registering fresh
   won't show growth: it needs the 90-day backdate `cmd/seed-demo` does when
   `POSTGRES_DSN` is set, or a real 90 days to pass.
2. Link a demo loan, have a ministry account review and verify it, show
   `/finance/coverage`.
3. Open `/impact` as a ministry account with seeded data loaded — filter by
   district/corporation, show a populated (not suppressed) group.
4. Walk through one literacy lesson, issue the certificate, scan/open its
   public verify link on a second device or incognito tab to show it needs
   no login.
5. If assisted mode is in scope for the audience: add an artisan as a field
   agent, capture consent, show the "Helping {name}" banner and the scoped
   action set.
