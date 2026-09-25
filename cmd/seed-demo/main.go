// cmd/seed-demo/main.go
//
// Seeds real artisans, published listings and a bulk order through the
// actual BFF REST API -- not a direct-to-Postgres insert -- so the data
// exercises the same validation and state machine a real user would. `make
// seed` (cmd/seed-ontology) must already have loaded the craft ontology, and
// the compose stack must be up with AUTH_DEV_OTP_ENABLED=true (dev OTP code
// 000000).
//
// It also seeds the MoSJE tier-4 impact dashboard: SEED_MOSJE_PER_GROUP (env,
// default 6) artisans in each of mosjeGroups below, each with an income
// baseline, a recent offline sale and a finance link (about half verified),
// so /impact's district, social-category and corporation breakdowns clear
// the <5 suppression threshold and show real medians instead of every cell
// reading "<5". If POSTGRES_DSN is set, it also backdates those artisans'
// created_at past the 90-day TOO_NEW threshold (the one thing no API can do)
// and bootstraps one MINISTRY and one FIELD_AGENT staff account via
// services/core-svc/cmd/create-staff. Without POSTGRES_DSN, the MoSJE
// artisans/sales/finance links still get created, but uplift stays TOO_NEW
// and the admin staff pages stay empty until those two steps are run by hand.
//
// Two steps have no public, self-serve path in the product today and are
// documented gaps, not bugs this script works around invisibly:
//   - Submitting a listing for approval requires a CLUSTER_OFFICER or
//     MINISTRY principal (services/core-svc/internal/core/service/catalog.go
//     SubmitForApproval) -- there is no REST endpoint or OTP flow that grants
//     either role, so this script mints one directly with pkg/auth, using
//     the same JWT_SECRET the bff verifies against.
//   - Creating a bulk order requires a BUYER principal (order.go
//     CreateBulkOrder), but pkg/auth's Subject.Role is hardcoded to
//     RoleArtisan for every OTP login (services/core-svc/internal/core/
//     service/auth.go VerifyOtp) -- buyer identity is "external" per
//     CLAUDE.md, and nothing in this codebase issues a buyer token. This
//     script mints one the same way. A real deploy needs a real answer for
//     buyer auth before "create bulk order" works for an actual buyer.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
)

func main() {
	baseURL := envOr("BFF_BASE_URL", "http://localhost:8000")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required (must match the bff's own)")
	}
	numArtisans, _ := strconv.Atoi(envOr("SEED_NUM_ARTISANS", "3"))
	if numArtisans < 1 {
		numArtisans = 1
	}

	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     jwtSecret,
		Issuer:     "kalakriti",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("building issuer: %v", err)
	}

	c := &client{base: baseURL, http: &http.Client{Timeout: 30 * time.Second}}

	crafts, err := c.listCrafts()
	if err != nil {
		log.Fatalf("GET /crafts (did you run `make seed` first?): %v", err)
	}
	if len(crafts) == 0 {
		log.Fatal("no crafts in the ontology -- run `make seed` first")
	}

	// The one role gap this script papers over: no REST/OTP path issues a
	// CLUSTER_OFFICER or MINISTRY token, so it's minted directly here.
	officerToken, err := mint(issuer, auth.Subject{ID: "seed-officer", Role: auth.RoleMinistry})
	if err != nil {
		log.Fatalf("minting officer token: %v", err)
	}

	var listingIDs []string

	for i := 0; i < numArtisans; i++ {
		phone := fmt.Sprintf("+9198765%05d", 10000+i)
		craft := crafts[i%len(crafts)]

		log.Printf("[%d/%d] registering artisan %s (%s)", i+1, numArtisans, phone, craft["display_name"])

		preToken, err := c.otpLogin(phone)
		if err != nil {
			log.Fatalf("otp login (pre-registration) for %s: %v", phone, err)
		}

		artisanResp, err := c.postJSON("/api/v1/artisans", preToken, map[string]any{
			"display_name": fmt.Sprintf("Demo Artisan %d", i+1),
			"craft_ids":    []string{craft["id"].(string)},
			"languages":    []string{"ENGLISH", "HINDI"},
			"region":       map[string]any{"state_code": "IN-UP", "district": "Lucknow"},
		})
		if err != nil {
			log.Fatalf("registering artisan %s: %v", phone, err)
		}
		artisanID := artisanResp["artisan_id"].(string)

		// POST /artisans already mints the artisan-scoped token pair this
		// needs: the pre-registration token was issued before the profile
		// existed and carries no subject, so RegisterArtisan returns fresh
		// ones (see its handler in services/bff/internal/bff/handler/api.go,
		// and auth-flow.ts's completeOtpVerification for the client-side
		// counterpart). This used to throw that pair away and run a second
		// full OTP login per artisan instead -- which, at 2 OTP requests per
		// artisan against the /auth/otp/request limiter's 5-per-10-minutes-
		// per-IP budget (server.go), made `make demo-up` fail on the third
		// artisan every single time, from a completely clean stack.
		artisanToken, _ := artisanResp["access_token"].(string)
		if artisanToken == "" {
			log.Fatalf("registering artisan %s: response carried no access_token: %v", phone, artisanResp)
		}

		listingResp, err := c.postJSON("/api/v1/listings", artisanToken, map[string]any{
			"craft_id":       craft["id"],
			"working_title":  fmt.Sprintf("Handmade %s piece", craft["display_name"]),
			"type":           "READY_STOCK",
			"price":          map[string]any{"amount_paise": 150000 + i*10000},
			"stock_quantity": 10,
			"translations": []map[string]any{
				{
					"language":    "ENGLISH",
					"title":       fmt.Sprintf("Handmade %s piece", craft["display_name"]),
					"description": fmt.Sprintf("A demo-seeded %s listing, made by a real registered artisan.", craft["display_name"]),
				},
			},
		})
		if err != nil {
			log.Fatalf("creating listing for artisan %s: %v", artisanID, err)
		}
		// The bff returns "listing_id" here, not "id" -- see CreateListing in
		// services/bff/internal/bff/handler/api.go.
		listingID, _ := listingResp["listing_id"].(string)
		if listingID == "" {
			log.Fatalf("creating listing for artisan %s: response carried no listing_id: %v", artisanID, listingResp)
		}

		if _, err := c.postJSON(fmt.Sprintf("/api/v1/listings/%s/submit", listingID), officerToken, nil); err != nil {
			log.Fatalf("submitting listing %s: %v", listingID, err)
		}
		if _, err := c.postJSON(fmt.Sprintf("/api/v1/listings/%s/approve", listingID), artisanToken, nil); err != nil {
			log.Fatalf("approving listing %s: %v", listingID, err)
		}

		log.Printf("  published listing %s", listingID)
		listingIDs = append(listingIDs, listingID)
	}

	// The other role gap: no path issues a BUYER token either.
	buyerToken, err := mint(issuer, auth.Subject{ID: uuid.New().String(), Role: auth.RoleBuyer})
	if err != nil {
		log.Fatalf("minting buyer token: %v", err)
	}

	for _, listingID := range listingIDs {
		orderResp, err := c.postJSON("/api/v1/orders/bulk", buyerToken, map[string]any{
			"listing_id":  listingID,
			"quantity":    5,
			"required_by": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			"notes":       "seeded demo bulk order",
		})
		if err != nil {
			log.Fatalf("creating bulk order for listing %s: %v", listingID, err)
		}
		log.Printf("  bulk order %s placed against listing %s", orderResp["order_id"], listingID)
	}

	mosjeArtisanIDs, err := seedMosjeTier4(c, crafts, officerToken)
	if err != nil {
		log.Fatalf("seeding MoSJE tier 4 impact data: %v", err)
	}

	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		if err := backdateRegistration(dsn, mosjeArtisanIDs); err != nil {
			log.Fatalf("backdating MoSJE artisan registration: %v", err)
		}
		if err := createDemoStaff(dsn); err != nil {
			log.Fatalf("creating demo staff accounts: %v", err)
		}
	} else {
		log.Printf("POSTGRES_DSN not set: skipping registration backdate and staff account creation -- " +
			"the impact dashboard's uplift figures will read TOO_NEW (artisans need >90 days since " +
			"registration) and the admin staff pages will stay empty until those two steps are run by hand")
	}

	if _, err := c.postJSON("/api/v1/insights/refresh", officerToken, nil); err != nil {
		log.Fatalf("refreshing impact materialized views: %v", err)
	}

	log.Printf("done: %d artisans, %d published listings, %d bulk orders, %d MoSJE tier-4 artisans across %d impact groups",
		numArtisans, len(listingIDs), len(listingIDs), len(mosjeArtisanIDs), len(mosjeGroups))
}

func mint(issuer *auth.Issuer, sub auth.Subject) (string, error) {
	pair, err := issuer.Issue(sub)
	if err != nil {
		return "", err
	}
	return pair.AccessToken, nil
}

type client struct {
	base string
	http *http.Client
}

func (c *client) listCrafts() ([]map[string]any, error) {
	resp, err := c.http.Get(c.base + "/api/v1/crafts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Crafts []map[string]any `json:"crafts"`
	}
	if err := decodeOrErr(resp, &body); err != nil {
		return nil, err
	}
	return body.Crafts, nil
}

// otpLogin requests and verifies an OTP with the dev code, returning an
// access token. Requires AUTH_DEV_OTP_ENABLED=true on core-svc.
func (c *client) otpLogin(phone string) (string, error) {
	if _, err := c.postJSON("/api/v1/auth/otp/request", "", map[string]any{"phone": phone}); err != nil {
		return "", fmt.Errorf("otp request: %w", err)
	}
	resp, err := c.postJSON("/api/v1/auth/otp/verify", "", map[string]any{"phone": phone, "otp": "000000"})
	if err != nil {
		return "", fmt.Errorf("otp verify: %w", err)
	}
	token, _ := resp["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("otp verify response carried no access_token: %v", resp)
	}
	return token, nil
}

func (c *client) postJSON(path, token string, body any) (map[string]any, error) {
	return c.doJSON(http.MethodPost, path, token, body)
}

func (c *client) putJSON(path, token string, body any) (map[string]any, error) {
	return c.doJSON(http.MethodPut, path, token, body)
}

func (c *client) findExistingFinanceLink(token, corporation, referenceLast4 string, originalErr error) (map[string]any, error) {
	links, err := c.getJSON("/api/v1/finance/links", token)
	if err != nil {
		return nil, fmt.Errorf("existing finance-link lookup failed: %v; original create failed: %w", err, originalErr)
	}
	items, ok := links["links"].([]any)
	if !ok {
		return nil, fmt.Errorf("existing finance-link response had no links array; original create failed: %w", originalErr)
	}
	for _, item := range items {
		link, ok := item.(map[string]any)
		if ok && link["corporation"] == corporation && link["reference_last4"] == referenceLast4 {
			return map[string]any{"link": link}, nil
		}
	}
	return nil, fmt.Errorf("no matching existing finance link; original create failed: %w", originalErr)
}

func (c *client) getJSON(path, token string) (map[string]any, error) {
	return c.doJSON(http.MethodGet, path, token, nil)
}

func (c *client) doJSON(method, path, token string, body any) (map[string]any, error) {
	time.Sleep(150 * time.Millisecond)
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[string]any
	if err := decodeOrErr(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeOrErr(resp *http.Response, v any) error {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s -> %d: %s", resp.Request.Method, resp.Request.URL, resp.StatusCode, string(raw))
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// financeConsentVersion must match domain.FinanceConsentVersion
// (services/core-svc/internal/core/domain/finance.go): core-svc rejects a
// finance link whose consent_version isn't the current one.
const financeConsentVersion = "finance-consent-2026-09"

// mosjeGroup is one impact-dashboard cohort: same state+district, same
// social category, same finance corporation. impact.MinCohort (pkg/impact)
// is 5, so mosjePerGroup below must be at least that for every group-by
// dimension (district, social_category, corporation) to show real numbers
// instead of "<5" suppression once seeded.
type mosjeGroup struct {
	stateCode, district, socialCategory, corporation string
}

var mosjeGroups = []mosjeGroup{
	{stateCode: "IN-UP", district: "Varanasi", socialCategory: "SC", corporation: "NSFDC"},
	{stateCode: "IN-MP", district: "Bhopal", socialCategory: "OBC", corporation: "NBCFDC"},
}

const mosjeMinTenureDays = 120 // > impact.MinTenure (90 days), with headroom

// formatMosjePhone gives each (group, index) pair its own phone number, so
// no two MoSJE artisans ever collide on the identity the BFF's OTP login
// keys on. gi is single-digit by construction (len(mosjeGroups) stays small).
func formatMosjePhone(groupIndex, indexInGroup int) string {
	return fmt.Sprintf("+9198764%01d%04d", groupIndex, 10000+indexInGroup)
}

// parseMosjePerGroupEnv reads SEED_MOSJE_PER_GROUP (val overrides the env
// lookup in tests), clamped to impact.MinCohort (5) so a misconfigured
// override can't silently make every group unsuppressible-threshold-blind.
func parseMosjePerGroupEnv(val string) (int, error) {
	if val == "" {
		val = envOr("SEED_MOSJE_PER_GROUP", "6")
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	if n < 5 {
		n = 5 // impact.MinCohort
	}
	return n, nil
}

// seedMosjeTier4 registers mosjePerGroup artisans per mosjeGroups entry,
// each with an income baseline, a recent offline sale (so they clear
// NO_SALES) and a finance link (about half verified by the ministry
// officer, so financeAdmin's review queue and the "checked" KPI both have
// something real to show). It returns every artisan id created, for the
// registered_at backdate that only direct SQL can do (TOO_NEW is gated on
// artisan.created_at -- migrations/queries/impact.sql's
// GetArtisanIncomeFacts -- and nothing in the product lets an artisan or
// officer edit that after the fact).
func seedMosjeTier4(c *client, crafts []map[string]any, officerToken string) ([]string, error) {
	numPerGroup, err := parseMosjePerGroupEnv("")
	if err != nil {
		return nil, fmt.Errorf("parsing SEED_MOSJE_PER_GROUP: %w", err)
	}
	brackets := []string{"LT_3K", "B3K_6K", "B6K_10K", "B10K_15K", "GT_15K"}
	channels := []string{"FAIR", "LOCAL_MARKET", "DIRECT", "OTHER"}

	var artisanIDs []string
	for gi, g := range mosjeGroups {
		for i := 0; i < numPerGroup; i++ {
			phone := formatMosjePhone(gi, i)
			craft := crafts[(gi*numPerGroup+i)%len(crafts)]

			log.Printf("[mosje %d/%d group %d] registering artisan %s (%s, %s)",
				i+1, numPerGroup, gi+1, phone, g.district, g.socialCategory)

			preToken, err := c.otpLogin(phone)
			if err != nil {
				return nil, fmt.Errorf("otp login (pre-registration) for %s: %w", phone, err)
			}
			artisanResp, err := c.postJSON("/api/v1/artisans", preToken, map[string]any{
				"display_name":    fmt.Sprintf("Demo MoSJE Artisan %d-%d", gi+1, i+1),
				"craft_ids":       []string{craft["id"].(string)},
				"languages":       []string{"ENGLISH", "HINDI"},
				"region":          map[string]any{"state_code": g.stateCode, "district": g.district},
				"social_category": g.socialCategory,
			})
			if err != nil {
				return nil, fmt.Errorf("registering MoSJE artisan %s: %w", phone, err)
			}
			artisanID := artisanResp["artisan_id"].(string)
			artisanIDs = append(artisanIDs, artisanID)

			artisanToken, err := c.otpLogin(phone)
			if err != nil {
				return nil, fmt.Errorf("otp login (post-registration) for %s: %w", phone, err)
			}

			bracket := brackets[i%len(brackets)]
			if _, err := c.putJSON("/api/v1/income/baseline", artisanToken, map[string]any{
				"monthly_bracket": bracket,
			}); err != nil {
				return nil, fmt.Errorf("setting income baseline for %s: %w", artisanID, err)
			}

			soldOn := time.Now().AddDate(0, 0, -(i % 60)).Format("2006-01-02")
			if _, err := c.postJSON("/api/v1/income/sales", artisanToken, map[string]any{
				"id":           uuid.New().String(),
				"channel":      channels[i%len(channels)],
				"amount_paise": int64(180000 + i*15000),
				"sold_on":      soldOn,
			}); err != nil {
				return nil, fmt.Errorf("logging offline sale for %s: %w", artisanID, err)
			}

			financeResp, err := c.postJSON("/api/v1/finance/links", artisanToken, map[string]any{
				"corporation":      g.corporation,
				"reference":        fmt.Sprintf("DEMO-%d-%d-%08d", gi, i, 10000000+i),
				"sanctioned_paise": int64(5000000 + i*100000),
				"emi_paise":        int64(150000 + i*5000),
				"emi_day_of_month": 1 + i%28,
				"consent_given":    true,
				"consent_version":  financeConsentVersion,
			})
			if err != nil {
				financeResp, err = c.findExistingFinanceLink(artisanToken, g.corporation, fmt.Sprintf("%04d", (10000000+i)%10000), err)
				if err != nil {
					return nil, fmt.Errorf("linking finance for %s: %w", artisanID, err)
				}
			}

			// Verify roughly half of each group's links, so financeAdmin's
			// review queue and the "checked" KPI both have real data.
			if i%2 == 0 {
				linkID := financeResp["link"].(map[string]any)["id"].(string)
				if _, err := c.postJSON("/api/v1/admin/finance/links/"+linkID+"/review", officerToken, map[string]any{
					"verified": true,
				}); err != nil {
					return nil, fmt.Errorf("reviewing finance link %s: %w", linkID, err)
				}
			}
		}
	}
	return artisanIDs, nil
}

// backdateRegistration sets each artisan's created_at far enough in the
// past to clear impact.MinTenure (90 days) -- registration date can't be
// backdated through any API, so this is the one direct-SQL step in an
// otherwise API-only seeder. Every artisan gets a slightly different
// offset so they don't all share one timestamp.
func backdateRegistration(dsn string, artisanIDs []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{DSN: dsn, MaxConns: 2, MinConns: 1})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	for i, id := range artisanIDs {
		backdate := time.Now().AddDate(0, 0, -(mosjeMinTenureDays + i%30))
		if _, err := pool.Exec(ctx, `UPDATE artisan SET created_at = $1 WHERE id = $2`, backdate, id); err != nil {
			return fmt.Errorf("backdating artisan %s: %w", id, err)
		}
	}
	log.Printf("backdated created_at for %d MoSJE artisans to >%d days ago", len(artisanIDs), mosjeMinTenureDays)
	return nil
}

// createDemoStaff bootstraps one MINISTRY and one FIELD_AGENT staff account
// through the real create-staff CLI (services/core-svc/cmd/create-staff) --
// this seeder never writes the staff_account table directly, since
// create-staff already owns that schema knowledge and is the documented
// bootstrap path (see its own doc comment). Requires being run with the
// repo root as the working directory (as `make seed-demo` does).
func createDemoStaff(dsn string) error {
	accounts := []struct{ phone, name, role string }{
		{"+919800000001", "Demo Ministry Officer", "MINISTRY"},
		{"+919800000002", "Demo Field Agent", "FIELD_AGENT"},
	}
	for _, a := range accounts {
		cmd := exec.Command("go", "run", "./services/core-svc/cmd/create-staff",
			"-dsn", dsn, "-phone", a.phone, "-name", a.name, "-role", a.role)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Idempotent re-run tolerance: a unique-constraint conflict on a
			if bytes.Contains(out, []byte("duplicate")) || bytes.Contains(out, []byte("unique")) || bytes.Contains(out, []byte("conflict")) {
				log.Printf("staff account for %s already exists, skipping", a.phone)
				continue
			}
			return fmt.Errorf("create-staff %s: %w\n%s", a.role, err, out)
		}
		log.Printf("%s", bytes.TrimSpace(out))
	}
	return nil
}
