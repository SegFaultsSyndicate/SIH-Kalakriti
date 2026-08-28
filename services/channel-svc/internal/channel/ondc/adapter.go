// services/channel-svc/internal/channel/ondc/adapter.go

// Package ondc exports published listings to ONDC retail schema with Ed25519 signing.
package ondc

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/segfaultsyndicate/kalakriti/pkg/crypto"
	"golang.org/x/crypto/blake2b"
)

// Adapter exports catalog to ONDC.
type Adapter struct {
	signer      *crypto.Signer
	subscriberID string
	subscriberURL string
	dryRun      bool
}

// Config holds ONDC subscriber identity.
type Config struct {
	PrivateKeyHex string
	KeyID         string
	SubscriberID  string
	SubscriberURL string
	DryRun        bool // If true, log payloads but don't send
}

// NewAdapter creates an ONDC adapter.
func NewAdapter(cfg Config) (*Adapter, error) {
	signer, err := crypto.NewSigner(cfg.PrivateKeyHex, cfg.KeyID)
	if err != nil {
		return nil, fmt.Errorf("ondc: invalid signing key: %w", err)
	}
	return &Adapter{
		signer:        signer,
		subscriberID:  cfg.SubscriberID,
		subscriberURL: cfg.SubscriberURL,
		dryRun:        cfg.DryRun,
	}, nil
}

// OnSearchItem is one catalog item in ONDC retail schema.
type OnSearchItem struct {
	ID          string            `json:"id"`
	Descriptor  Descriptor        `json:"descriptor"`
	Price       Price             `json:"price"`
	CategoryID  string            `json:"category_id"`
	FulfillmentID string          `json:"fulfillment_id,omitempty"`
	LocationID  string            `json:"location_id,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// Descriptor holds name, images, short_desc, long_desc.
type Descriptor struct {
	Name      string   `json:"name"`
	ShortDesc string   `json:"short_desc,omitempty"`
	LongDesc  string   `json:"long_desc,omitempty"`
	Images    []string `json:"images,omitempty"`
}

// Price holds currency and value.
type Price struct {
	Currency string `json:"currency"`
	Value    string `json:"value"`
}

// OnSearchPayload is the full on_search response.
type OnSearchPayload struct {
	Context Context `json:"context"`
	Message Message `json:"message"`
}

// Context holds transaction metadata.
type Context struct {
	Domain      string    `json:"domain"`
	Action      string    `json:"action"`
	BapID       string    `json:"bap_id"`
	BapURI      string    `json:"bap_uri"`
	BppID       string    `json:"bpp_id"`
	BppURI      string    `json:"bpp_uri"`
	TransactionID string  `json:"transaction_id"`
	MessageID   string    `json:"message_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// Message holds the catalog items.
type Message struct {
	Catalog Catalog `json:"catalog"`
}

// Catalog holds providers.
type Catalog struct {
	Providers []Provider `json:"providers"`
}

// Provider holds one seller's items.
type Provider struct {
	ID         string         `json:"id"`
	Descriptor Descriptor     `json:"descriptor"`
	Items      []OnSearchItem `json:"items"`
}

// BuildOnSearch builds an on_search payload from listings.
func (a *Adapter) BuildOnSearch(ctx context.Context, listings []Listing) (OnSearchPayload, error) {
	items := make([]OnSearchItem, len(listings))
	for i, l := range listings {
		items[i] = OnSearchItem{
			ID: l.ID,
			Descriptor: Descriptor{
				Name:      l.Title,
				ShortDesc: l.Description,
				Images:    l.ImageURLs,
			},
			Price: Price{
				Currency: l.Currency,
				Value:    fmt.Sprintf("%.2f", float64(l.PriceCents)/100.0),
			},
			CategoryID: l.CategoryID,
		}
	}

	payload := OnSearchPayload{
		Context: Context{
			Domain:        "retail",
			Action:        "on_search",
			BppID:         a.subscriberID,
			BppURI:        a.subscriberURL,
			TransactionID: "tx-" + time.Now().Format("20060102150405"),
			MessageID:     "msg-" + time.Now().Format("20060102150405"),
			Timestamp:     time.Now().UTC(),
		},
		Message: Message{
			Catalog: Catalog{
				Providers: []Provider{
					{
						ID: a.subscriberID,
						Descriptor: Descriptor{
							Name: "Kalakriti",
						},
						Items: items,
					},
				},
			},
		},
	}

	return payload, nil
}

// Sign signs the payload with blake2b + Ed25519.
func (a *Adapter) Sign(payload []byte) (signature string, err error) {
	hash := blake2b.Sum256(payload)
	sig := a.signer.Sign(hash[:])
	return hex.EncodeToString(sig), nil
}

// Verify checks a signature.
func Verify(publicKey ed25519.PublicKey, payload, signature []byte) bool {
	hash := blake2b.Sum256(payload)
	return crypto.Verify(publicKey, hash[:], signature)
}

// Listing is a minimal DTO for building ONDC items.
type Listing struct {
	ID          string
	Title       string
	Description string
	PriceCents  int64
	Currency    string
	CategoryID  string
	ImageURLs   []string
}
