// services/core-svc/cmd/seed-catalog/main.go
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	HindiName    string `json:"hindi_name"`
	Description  string `json:"description"`
}

type TitleDesc struct {
	EN string `json:"en"`
	HI string `json:"hi"`
}

type Pricing struct {
	Currency        string `json:"currency"`
	BasePricePaise  int64  `json:"base_price_paise"`
	AdvancePct      int    `json:"advance_percentage"`
}

type Inventory struct {
	ListingType      string `json:"listing_type"`
	StockQuantity    *int   `json:"stock_quantity,omitempty"`
	MinOrderQuantity int    `json:"min_order_quantity"`
	LeadTimeDays     *int   `json:"lead_time_days,omitempty"`
	MonthlyCapacity  *int   `json:"monthly_capacity,omitempty"`
}

type Dimensions struct {
	LengthMM int `json:"length_mm"`
	WidthMM  int `json:"width_mm"`
	HeightMM int `json:"height_mm"`
	WeightG  int `json:"weight_g"`
}

type Packaging struct {
	Fragile               bool `json:"fragile"`
	Oversized             bool `json:"oversized"`
	RequiresCustomCrating bool `json:"requires_custom_crating"`
	PackedLengthMM        int  `json:"packed_length_mm"`
	PackedWidthMM         int  `json:"packed_width_mm"`
	PackedHeightMM        int  `json:"packed_height_mm"`
	PackedWeightG         int  `json:"packed_weight_g"`
}

type Provenance struct {
	GICertified      bool   `json:"gi_certified"`
	GIRegistrationNo string `json:"gi_registration_no"`
	ClusterName      string `json:"cluster_name"`
	RegionState      string `json:"region_state"`
}

type MediaItem struct {
	URL       string `json:"url"`
	MediaType string `json:"media_type"`
	Angle     string `json:"angle"`
	IsPrimary bool   `json:"is_primary"`
}

type ListingDatasetItem struct {
	ListingID        string            `json:"listing_id"`
	ArtisanID        string            `json:"artisan_id"`
	ArtisanName      string            `json:"artisan_name"`
	ArtisanDistrict  string            `json:"artisan_district"`
	ArtisanStateCode string            `json:"artisan_state_code"`
	CraftCategory    string            `json:"craft_category"`
	CraftName        string            `json:"craft_name"`
	CraftSlug        string            `json:"craft_slug"`
	Title            TitleDesc         `json:"title"`
	Description      TitleDesc         `json:"description"`
	Pricing          Pricing           `json:"pricing"`
	Inventory        Inventory         `json:"inventory"`
	Dimensions       Dimensions        `json:"dimensions"`
	Packaging        Packaging         `json:"packaging"`
	Provenance       Provenance        `json:"provenance"`
	Media            []MediaItem       `json:"media"`
	Attributes       map[string]any    `json:"attributes"`
}

type Dataset struct {
	Categories []Category           `json:"categories"`
	Listings   []ListingDatasetItem `json:"listings"`
}

func main() {
	ctx := context.Background()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	// Locate data directory
	cwd, _ := os.Getwd()
	datasetPath := filepath.Join(cwd, "data", "craft-listings-dataset.json")
	if _, err := os.Stat(datasetPath); os.IsNotExist(err) {
		// try parent dir
		datasetPath = filepath.Join(cwd, "..", "..", "data", "craft-listings-dataset.json")
	}

	bytes, err := os.ReadFile(datasetPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", datasetPath, err)
	}

	var ds Dataset
	if err := json.Unmarshal(bytes, &ds); err != nil {
		log.Fatalf("failed to parse dataset: %v", err)
	}

	fmt.Printf("Loaded dataset: %d categories, %d listings\n", len(ds.Categories), len(ds.Listings))

	categorySlugMap := map[string]string{
		"weaving_and_looms": "weaving-and-looms",
		"block_printing":    "block-printing",
		"pottery":           "pottery",
		"metalwork":         "metalwork",
		"woodwork":          "woodwork",
		"embroidery":        "embroidery",
		"home_and_living":   "home-and-living",
		"furniture":         "furniture",
		"paintings":         "paintings",
		"basketry":          "basketry",
		"jewellery":         "jewellery",
		"leatherwork":       "leatherwork",
		"stone_carving":     "stone-carving",
		"bamboo_craft":      "bamboo-craft",
	}

	// 1. Seed root categories in craft table
	categoryUUIDMap := make(map[string]uuid.UUID)
	for _, cat := range ds.Categories {
		slug, ok := categorySlugMap[cat.CategoryID]
		if !ok {
			slug = strings.ReplaceAll(cat.CategoryID, "_", "-")
		}

		catUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("category:"+slug))
		var existingID uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO craft (id, code, display_name, parent_craft_id, techniques, materials)
			VALUES ($1, $2, $3, NULL, '{}', '{}')
			ON CONFLICT (code) DO UPDATE SET display_name = EXCLUDED.display_name
			RETURNING id
		`, catUUID, slug, cat.CategoryName).Scan(&existingID)
		if err != nil {
			log.Fatalf("failed to insert category craft %s: %v", slug, err)
		}
		categoryUUIDMap[cat.CategoryID] = existingID
		categoryUUIDMap[slug] = existingID

		// Add aliases for category
		aliases := []struct {
			name   string
			script string
			lang   string
		}{
			{cat.CategoryName, "Latn", "ENGLISH"},
			{cat.HindiName, "Deva", "HINDI"},
		}
		for _, a := range aliases {
			if a.name == "" {
				continue
			}
			aliasUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("alias:"+slug+":"+a.name+":"+a.script))
			_, _ = pool.Exec(ctx, `
				INSERT INTO craft_alias (id, craft_id, alias, script, language, source)
				VALUES ($1, $2, $3, $4, $5, 'curator')
				ON CONFLICT (lower(alias), script) DO UPDATE SET craft_id = EXCLUDED.craft_id
			`, aliasUUID, existingID, a.name, a.script, a.lang)
		}
	}
	fmt.Printf("✓ Seeded %d root categories\n", len(ds.Categories))

	// 2. Process each listing
	artisanPhoneCounter := 100
	for i, l := range ds.Listings {
		// Parent category UUID
		parentID, ok := categoryUUIDMap[l.CraftCategory]
		if !ok {
			slug := strings.ReplaceAll(l.CraftCategory, "_", "-")
			parentID = categoryUUIDMap[slug]
		}

		// Collect materials and techniques from attributes
		var materials []string
		var techniques []string
		for k, v := range l.Attributes {
			valStr := stringify(v)
			kLower := strings.ToLower(k)
			if strings.Contains(kLower, "material") || strings.Contains(kLower, "clay") || strings.Contains(kLower, "fabric") || strings.Contains(kLower, "wood") || strings.Contains(kLower, "silk") {
				materials = append(materials, valStr)
			}
			if strings.Contains(kLower, "technique") || strings.Contains(kLower, "weave") || strings.Contains(kLower, "firing") || strings.Contains(kLower, "casting") {
				techniques = append(techniques, valStr)
			}
		}
		if len(materials) == 0 {
			materials = []string{"Natural Organic Material"}
		}
		if len(techniques) == 0 {
			techniques = []string{"Traditional Handcrafted"}
		}

		// A. Child craft record
		childCraftUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("craft:"+l.CraftSlug))
		var giReg *string
		if l.Provenance.GICertified && l.Provenance.GIRegistrationNo != "" {
			reg := l.Provenance.GIRegistrationNo
			giReg = &reg
		}

		var childCraftID uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO craft (id, code, display_name, parent_craft_id, gi_registration_no, techniques, materials)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (code) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				parent_craft_id = COALESCE(EXCLUDED.parent_craft_id, craft.parent_craft_id),
				techniques = EXCLUDED.techniques,
				materials = EXCLUDED.materials
			RETURNING id
		`, childCraftUUID, l.CraftSlug, l.CraftName, parentID, giReg, techniques, materials).Scan(&childCraftID)
		if err != nil {
			log.Fatalf("failed to insert child craft %s: %v", l.CraftSlug, err)
		}

		// Aliases for child craft
		childAliases := []struct {
			name   string
			script string
			lang   string
		}{
			{l.CraftName, "Latn", "ENGLISH"},
		}
		for _, ca := range childAliases {
			aliasUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("alias:"+l.CraftSlug+":"+ca.name+":"+ca.script))
			_, _ = pool.Exec(ctx, `
				INSERT INTO craft_alias (id, craft_id, alias, script, language, source)
				VALUES ($1, $2, $3, $4, $5, 'curator')
				ON CONFLICT (lower(alias), script) DO UPDATE SET craft_id = EXCLUDED.craft_id
			`, aliasUUID, childCraftID, ca.name, ca.script, ca.lang)
		}

		// B. Cluster
		clusterName := l.Provenance.ClusterName
		if clusterName == "" {
			clusterName = fmt.Sprintf("%s Craft Cluster", l.ArtisanDistrict)
		}
		clusterUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("cluster:"+l.ArtisanStateCode+":"+l.ArtisanDistrict+":"+clusterName))
		_, err = pool.Exec(ctx, `
			INSERT INTO cluster (id, name, state_code, district)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, state_code = EXCLUDED.state_code, district = EXCLUDED.district
		`, clusterUUID, clusterName, l.ArtisanStateCode, l.ArtisanDistrict)
		if err != nil {
			log.Fatalf("failed to insert cluster: %v", err)
		}

		// C. Artisan
		artisanUUID, err := uuid.Parse(l.ArtisanID)
		if err != nil {
			artisanUUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("artisan:"+l.ArtisanName))
		}
		phone := fmt.Sprintf("+919876543%03d", artisanPhoneCounter)
		artisanPhoneCounter++
		bio := fmt.Sprintf("Master artisan practising %s in %s, %s. Dedicated to authentic handmade heritage.", l.CraftName, l.ArtisanDistrict, l.ArtisanStateCode)

		_, err = pool.Exec(ctx, `
			INSERT INTO artisan (
				id, display_name, phone_e164, primary_cluster_id, state_code,
				district, languages, verified, bio
			)
			VALUES ($1, $2, $3, $4, $5, $6, ARRAY['ENGLISH', 'HINDI']::language_code[], true, $7)
			ON CONFLICT (id) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				primary_cluster_id = EXCLUDED.primary_cluster_id,
				state_code = EXCLUDED.state_code,
				district = EXCLUDED.district,
				bio = EXCLUDED.bio,
				verified = true
		`, artisanUUID, l.ArtisanName, phone, clusterUUID, l.ArtisanStateCode, l.ArtisanDistrict, bio)
		if err != nil {
			log.Fatalf("failed to insert artisan %s: %v", l.ArtisanName, err)
		}

		// Link artisan craft
		_, _ = pool.Exec(ctx, `
			INSERT INTO artisan_craft (artisan_id, craft_id, is_primary)
			VALUES ($1, $2, true)
			ON CONFLICT (artisan_id, craft_id) DO UPDATE SET is_primary = true
		`, artisanUUID, childCraftID)

		// D. Product
		productUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("product:"+l.ListingID))
		_, err = pool.Exec(ctx, `
			INSERT INTO product (
				id, artisan_id, craft_id, working_title, length_mm, width_mm, height_mm, weight_g,
				materials, techniques
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET
				working_title = EXCLUDED.working_title,
				length_mm = EXCLUDED.length_mm,
				width_mm = EXCLUDED.width_mm,
				height_mm = EXCLUDED.height_mm,
				weight_g = EXCLUDED.weight_g,
				materials = EXCLUDED.materials,
				techniques = EXCLUDED.techniques
		`, productUUID, artisanUUID, childCraftID, l.Title.EN, l.Dimensions.LengthMM, l.Dimensions.WidthMM, l.Dimensions.HeightMM, l.Dimensions.WeightG, materials, techniques)
		if err != nil {
			log.Fatalf("failed to insert product: %v", err)
		}

		// E. Media
		var primaryMediaID uuid.UUID
		for mIdx, media := range l.Media {
			hasher := sha256.New()
			hasher.Write([]byte(media.URL))
			shaHex := hex.EncodeToString(hasher.Sum(nil))

			mediaUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("media:"+l.ListingID+":"+media.URL))
			var savedMediaID uuid.UUID
			err := pool.QueryRow(ctx, `
				INSERT INTO media (
					id, artisan_id, product_id, bucket, object_key, kind, mime_type, size_bytes,
					sha256_hex, state, confirmed_at
				) VALUES ($1, $2, $3, 'kalakriti-media', $4, 'IMAGE', 'image/jpeg', 150000, $5, 'READY', now())
				ON CONFLICT (bucket, object_key) DO UPDATE SET
					product_id = EXCLUDED.product_id,
					state = 'READY',
					confirmed_at = now()
				RETURNING id
			`, mediaUUID, artisanUUID, productUUID, media.URL, shaHex).Scan(&savedMediaID)
			if err != nil {
				log.Fatalf("failed to insert media for %s: %v", media.URL, err)
			}

			if mIdx == 0 || media.IsPrimary {
				primaryMediaID = savedMediaID
			}
		}

		// F. Listing
		listingUUID, err := uuid.Parse(l.ListingID)
		if err != nil {
			listingUUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("listing:"+l.Title.EN))
		}

		listingType := l.Inventory.ListingType
		if listingType != "MADE_TO_ORDER" && listingType != "READY_STOCK" {
			listingType = "READY_STOCK"
		}

		var stockQty *int
		var leadTime *int
		if listingType == "READY_STOCK" {
			qty := 5
			if l.Inventory.StockQuantity != nil && *l.Inventory.StockQuantity > 0 {
				qty = *l.Inventory.StockQuantity
			}
			stockQty = &qty
		} else {
			days := 21
			if l.Inventory.LeadTimeDays != nil && *l.Inventory.LeadTimeDays > 0 {
				days = *l.Inventory.LeadTimeDays
			}
			leadTime = &days
		}

		minOrderQty := l.Inventory.MinOrderQuantity
		if minOrderQty <= 0 {
			minOrderQty = 1
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO listing (
				id, product_id, artisan_id, type, state, price_paise, currency_code,
				stock_quantity, min_order_quantity, lead_time_days, capacity_per_month,
				packaging_fragile, packaging_oversized, packaging_requires_custom_crating,
				packed_length_mm, packed_width_mm, packed_height_mm, packed_weight_g,
				gi_certified, published_at
			) VALUES (
				$1, $2, $3, $4, 'PUBLISHED', $5, 'INR',
				$6, $7, $8, $9,
				$10, $11, $12,
				$13, $14, $15, $16,
				$17, now()
			)
			ON CONFLICT (id) DO UPDATE SET
				price_paise = EXCLUDED.price_paise,
				stock_quantity = EXCLUDED.stock_quantity,
				lead_time_days = EXCLUDED.lead_time_days,
				state = 'PUBLISHED',
				gi_certified = EXCLUDED.gi_certified,
				published_at = COALESCE(listing.published_at, now())
		`, listingUUID, productUUID, artisanUUID, listingType, l.Pricing.BasePricePaise,
			stockQty, minOrderQty, leadTime, l.Inventory.MonthlyCapacity,
			l.Packaging.Fragile, l.Packaging.Oversized, l.Packaging.RequiresCustomCrating,
			l.Packaging.PackedLengthMM, l.Packaging.PackedWidthMM, l.Packaging.PackedHeightMM, l.Packaging.PackedWeightG,
			l.Provenance.GICertified)
		if err != nil {
			log.Fatalf("failed to insert listing %s: %v", l.Title.EN, err)
		}

		// G. Listing Media Link
		if primaryMediaID != uuid.Nil {
			_, _ = pool.Exec(ctx, `DELETE FROM listing_media WHERE listing_id = $1`, listingUUID)
			_, err = pool.Exec(ctx, `
				INSERT INTO listing_media (listing_id, media_id, ordinal, role)
				VALUES ($1, $2, 0, 'PRIMARY_IMAGE')
				ON CONFLICT (listing_id, media_id) DO UPDATE SET role = 'PRIMARY_IMAGE', ordinal = 0
			`, listingUUID, primaryMediaID)
			if err != nil {
				log.Fatalf("failed to link listing media: %v", err)
			}
		}

		// H. Translations (EN & HI)
		_, err = pool.Exec(ctx, `
			INSERT INTO listing_translation (listing_id, language, title, description, machine_generated)
			VALUES ($1, 'ENGLISH', $2, $3, false)
			ON CONFLICT (listing_id, language) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				machine_generated = false
		`, listingUUID, l.Title.EN, l.Description.EN)
		if err != nil {
			log.Fatalf("failed to insert EN translation: %v", err)
		}

		if l.Title.HI != "" {
			hiDesc := l.Description.HI
			if hiDesc == "" {
				hiDesc = l.Description.EN
			}
			_, err = pool.Exec(ctx, `
				INSERT INTO listing_translation (listing_id, language, title, description, machine_generated)
				VALUES ($1, 'HINDI', $2, $3, false)
				ON CONFLICT (listing_id, language) DO UPDATE SET
					title = EXCLUDED.title,
					description = EXCLUDED.description,
					machine_generated = false
			`, listingUUID, l.Title.HI, hiDesc)
			if err != nil {
				log.Fatalf("failed to insert HI translation: %v", err)
			}
		}

		// I. Attributes
		for attrName, attrVal := range l.Attributes {
			valStr := stringify(attrVal)
			attrUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("attr:"+l.ListingID+":"+attrName+":"+valStr))
			_, _ = pool.Exec(ctx, `
				INSERT INTO listing_attribute (id, listing_id, name, value, confidence, source)
				VALUES ($1, $2, $3, $4, 1.0, 'ARTISAN')
				ON CONFLICT (listing_id, name, value) DO NOTHING
			`, attrUUID, listingUUID, attrName, valStr)
		}

		// J. Search Index (listing_search)
		// Document EN contains all searchable keywords: title, description, craft name, category, artisan name, district, state, materials, techniques
		categoryDisplay := l.CraftCategory
		for _, cat := range ds.Categories {
			if cat.CategoryID == l.CraftCategory {
				categoryDisplay = cat.CategoryName
				break
			}
		}

		docEN := fmt.Sprintf("%s %s %s %s %s %s %s %s %s",
			l.Title.EN,
			l.Description.EN,
			l.CraftName,
			categoryDisplay,
			l.ArtisanName,
			l.ArtisanDistrict,
			l.ArtisanStateCode,
			strings.Join(materials, " "),
			strings.Join(techniques, " "),
		)

		_, err = pool.Exec(ctx, `
			INSERT INTO listing_search (
				listing_id, language, artisan_id, craft_id, cluster_id, listing_type, price_paise,
				lead_time_days, gi_certified, state_code, district, materials, document, document_tsv, indexed_at
			) VALUES (
				$1, 'ENGLISH', $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12, to_tsvector('simple', unaccent($12)), now()
			)
			ON CONFLICT (listing_id, language) DO UPDATE SET
				craft_id = EXCLUDED.craft_id,
				artisan_id = EXCLUDED.artisan_id,
				cluster_id = EXCLUDED.cluster_id,
				listing_type = EXCLUDED.listing_type,
				price_paise = EXCLUDED.price_paise,
				gi_certified = EXCLUDED.gi_certified,
				state_code = EXCLUDED.state_code,
				district = EXCLUDED.district,
				materials = EXCLUDED.materials,
				document = EXCLUDED.document,
				document_tsv = EXCLUDED.document_tsv,
				indexed_at = now()
		`, listingUUID, artisanUUID, childCraftID, clusterUUID, listingType, l.Pricing.BasePricePaise,
			leadTime, l.Provenance.GICertified, l.ArtisanStateCode, l.ArtisanDistrict, materials, docEN)
		if err != nil {
			log.Fatalf("failed to index EN listing_search: %v", err)
		}

		if l.Title.HI != "" {
			docHI := fmt.Sprintf("%s %s %s %s %s %s %s",
				l.Title.HI,
				l.Description.HI,
				l.CraftName,
				categoryDisplay,
				l.ArtisanName,
				l.ArtisanDistrict,
				l.ArtisanStateCode,
			)
			_, _ = pool.Exec(ctx, `
				INSERT INTO listing_search (
					listing_id, language, artisan_id, craft_id, cluster_id, listing_type, price_paise,
					lead_time_days, gi_certified, state_code, district, materials, document, document_tsv, indexed_at
				) VALUES (
					$1, 'HINDI', $2, $3, $4, $5, $6,
					$7, $8, $9, $10, $11, $12, to_tsvector('simple', unaccent($12)), now()
				)
				ON CONFLICT (listing_id, language) DO UPDATE SET
					craft_id = EXCLUDED.craft_id,
					artisan_id = EXCLUDED.artisan_id,
					cluster_id = EXCLUDED.cluster_id,
					listing_type = EXCLUDED.listing_type,
					price_paise = EXCLUDED.price_paise,
					gi_certified = EXCLUDED.gi_certified,
					state_code = EXCLUDED.state_code,
					district = EXCLUDED.district,
					materials = EXCLUDED.materials,
					document = EXCLUDED.document,
					document_tsv = EXCLUDED.document_tsv,
					indexed_at = now()
			`, listingUUID, artisanUUID, childCraftID, clusterUUID, listingType, l.Pricing.BasePricePaise,
				leadTime, l.Provenance.GICertified, l.ArtisanStateCode, l.ArtisanDistrict, materials, docHI)
		}

		fmt.Printf("  [%d/%d] Seeded listing: %s (Craft: %s, Artisan: %s)\n",
			i+1, len(ds.Listings), l.Title.EN, l.CraftName, l.ArtisanName)
	}

	fmt.Println("\n🎉 Successfully seeded complete catalog data into PostgreSQL!")
}

func stringify(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		var parts []string
		for _, p := range val {
			parts = append(parts, fmt.Sprintf("%v", p))
		}
		return strings.Join(parts, ", ")
	case []string:
		return strings.Join(val, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}
