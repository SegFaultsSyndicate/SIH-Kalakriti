// cmd/seed-demo/main.go
//
// Seeds real artisans, published listings and a bulk order through the
// actual BFF REST API -- not a direct-to-Postgres insert -- so the data
// exercises the same validation and state machine a real user would. `make
// seed` (cmd/seed-ontology) must already have loaded the craft ontology, and
// the compose stack must be up with AUTH_DEV_OTP_ENABLED=true (dev OTP code
// 000000).
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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
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

		// Re-verify: the profile now exists, so this token carries artisanID
		// as its subject (see VerifyOtp in services/core-svc/internal/core/
		// service/auth.go) -- the pre-registration token above never will.
		artisanToken, err := c.otpLogin(phone)
		if err != nil {
			log.Fatalf("otp login (post-registration) for %s: %v", phone, err)
		}

		listingResp, err := c.postJSON("/api/v1/listings", artisanToken, map[string]any{
			"craft_id":      craft["id"],
			"working_title": fmt.Sprintf("Handmade %s piece", craft["display_name"]),
			"type":          "READY_STOCK",
			"price":         map[string]any{"amount_paise": 150000 + i*10000},
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
		listingID := listingResp["id"].(string)

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
		log.Printf("  bulk order %s placed against listing %s", orderResp["id"], listingID)
	}

	log.Printf("done: %d artisans, %d published listings, %d bulk orders", numArtisans, len(listingIDs), len(listingIDs))
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
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, reader)
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
