// scripts/seed/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	crafts = []string{"madhubani", "warli", "pottery", "weaving", "metalwork", "embroidery", "woodcarving", "terracotta"}
	states = []string{"bihar", "maharashtra", "rajasthan", "gujarat", "karnataka", "west bengal"}

	artisans = []struct {
		name  string
		craft string
		state string
		story string
	}{
		{"Lakshmi Devi", "madhubani", "bihar", "Fourth generation Madhubani artist from Mithila"},
		{"Ram Kumar", "pottery", "rajasthan", "Blue pottery specialist from Jaipur"},
		{"Sita Sharma", "weaving", "gujarat", "Traditional Patola weaver"},
		{"Mohan Lal", "metalwork", "rajasthan", "Brass and copper artisan"},
		{"Geeta Patel", "embroidery", "gujarat", "Kutch embroidery expert"},
		{"Rajesh Singh", "woodcarving", "karnataka", "Sandalwood carver from Mysore"},
		{"Priya Devi", "warli", "maharashtra", "Warli tribal art from Thane"},
		{"Amit Kumar", "terracotta", "west bengal", "Bankura horse sculptor"},
		{"Sunita Rani", "madhubani", "bihar", "Contemporary Madhubani innovator"},
		{"Vijay Sharma", "pottery", "rajasthan", "Traditional wheel-thrown pottery"},
		{"Anita Devi", "weaving", "gujarat", "Silk saree weaver"},
		{"Rakesh Patel", "metalwork", "karnataka", "Bronze casting specialist"},
		{"Meena Singh", "embroidery", "rajasthan", "Mirror work artisan"},
		{"Suresh Kumar", "woodcarving", "karnataka", "Temple carving craftsman"},
		{"Kavita Sharma", "warli", "maharashtra", "Warli painting educator"},
		{"Dinesh Lal", "terracotta", "west bengal", "Clay doll maker"},
		{"Poonam Devi", "madhubani", "bihar", "Natural dye Madhubani artist"},
		{"Ashok Kumar", "pottery", "rajasthan", "Terracotta tile maker"},
		{"Rekha Patel", "weaving", "gujarat", "Cotton weaver"},
		{"Mahesh Singh", "metalwork", "rajasthan", "Silver jewelry craftsman"},
	}
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://kalakriti:kalakriti@localhost:5432/kalakriti?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Check if already seeded
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = 'artisan'").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		fmt.Println("Database already seeded, skipping")
		return
	}

	fmt.Println("Seeding database...")

	// Seed artisans
	artisanIDs := make([]uuid.UUID, len(artisans))
	for i, a := range artisans {
		id := uuid.New()
		artisanIDs[i] = id

		_, err := pool.Exec(ctx, `
			INSERT INTO users (id, phone, role, name, craft_type, state, bio, onboarding_complete, created_at, updated_at)
			VALUES ($1, $2, 'artisan', $3, $4, $5, $6, true, NOW(), NOW())
		`, id, fmt.Sprintf("+919%09d", 100000000+i), a.name, a.craft, a.state, a.story)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created artisan: %s (%s, %s)\n", a.name, a.craft, a.state)
	}

	// Seed listings
	listingTitles := []struct {
		title     string
		craft     string
		price     int64
		mto       bool
		leadDays  int
		minQty    int
	}{
		{"Traditional Madhubani Fish Painting", "madhubani", 250000, false, 0, 1},
		{"Blue Pottery Vase", "pottery", 180000, false, 0, 1},
		{"Handwoven Patola Silk Saree", "weaving", 5000000, true, 30, 1},
		{"Brass Lord Ganesha Statue", "metalwork", 350000, false, 0, 1},
		{"Kutch Mirror Work Wall Hanging", "embroidery", 150000, true, 15, 1},
		{"Sandalwood Carved Elephant", "woodcarving", 800000, false, 0, 1},
		{"Warli Tribal Art Canvas", "warli", 120000, false, 0, 1},
		{"Bankura Terracotta Horse", "terracotta", 90000, false, 0, 1},
		{"Madhubani Peacock Painting", "madhubani", 280000, false, 0, 1},
		{"Terracotta Garden Planter", "pottery", 45000, false, 0, 5},
		{"Cotton Dhurrie Rug 4x6", "weaving", 220000, true, 20, 1},
		{"Copper Water Jug", "metalwork", 65000, false, 0, 10},
		{"Rajasthani Embroidered Cushion Cover", "embroidery", 35000, true, 10, 2},
		{"Wooden Incense Box", "woodcarving", 55000, false, 0, 5},
		{"Warli Painting Greeting Cards", "warli", 5000, false, 0, 50},
		{"Clay Diwali Diyas Set", "terracotta", 12000, false, 0, 20},
		{"Madhubani Coasters Set of 4", "madhubani", 18000, false, 0, 10},
		{"Blue Pottery Tile", "pottery", 25000, false, 0, 50},
		{"Handloom Cotton Stole", "weaving", 95000, true, 12, 1},
		{"Brass Puja Thali", "metalwork", 145000, false, 0, 2},
	}

	listingIDs := make([]uuid.UUID, 0)
	for i := 0; i < 60; i++ {
		id := uuid.New()
		listingIDs = append(listingIDs, id)

		tpl := listingTitles[i%len(listingTitles)]
		artisanIdx := i % len(artisanIDs)

		status := "published"
		if i%10 == 0 {
			status = "draft"
		}

		imageURL := fmt.Sprintf("https://storage.kalakriti.in/images/%s.jpg", id)

		_, err := pool.Exec(ctx, `
			INSERT INTO listings (
				id, artisan_id, title_en, title_hi, description_en, status,
				category, tags, price_paise, is_made_to_order, lead_time_days,
				min_order_qty, stock_qty, image_urls, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		`, id, artisanIDs[artisanIdx], tpl.title, tpl.title, "Authentic handcrafted "+tpl.craft,
			status, tpl.craft, []string{tpl.craft, "handmade", "traditional"}, tpl.price,
			tpl.mto, tpl.leadDays, tpl.minQty, 100, []string{imageURL})
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Created %d listings\n", len(listingIDs))

	// 5 listings with provenance
	for i := 0; i < 5; i++ {
		listingID := listingIDs[i]
		artisanID := artisanIDs[i]
		provenanceID := uuid.New()

		videoURL := fmt.Sprintf("https://storage.kalakriti.in/videos/%s.mp4", provenanceID)
		qrCode := fmt.Sprintf("PROV_%s", provenanceID.String()[:8])

		_, err := pool.Exec(ctx, `
			INSERT INTO provenance (
				id, listing_id, artisan_id, process_videos, materials, techniques,
				location_lat, location_lng, sealed_at, qr_code, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, NOW())
		`, provenanceID, listingID, artisanID, []string{videoURL},
			[]string{"natural dyes", "cotton canvas"}, []string{"traditional brush painting"},
			25.5941, 85.1376, qrCode)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Created 5 provenance records")

	// 3 bulk orders
	buyerID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, phone, role, name, created_at, updated_at)
		VALUES ($1, '+919999999999', 'buyer', 'Bulk Buyer Corp', NOW(), NOW())
	`, buyerID)
	if err != nil {
		log.Fatal(err)
	}

	orderStates := []string{"allocating", "in_production", "completed"}
	for i, state := range orderStates {
		orderID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO orders (
				id, buyer_id, order_type, total_paise, status, created_at, updated_at
			) VALUES ($1, $2, 'bulk', $3, $4, NOW(), NOW())
		`, orderID, buyerID, 500000+(int64(i)*100000), state)
		if err != nil {
			log.Fatal(err)
		}

		// Allocations
		for j := 0; j < 3; j++ {
			_, err := pool.Exec(ctx, `
				INSERT INTO order_allocations (
					id, order_id, artisan_id, quantity, status, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			`, uuid.New(), orderID, artisanIDs[j], 150+j*25, state)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	fmt.Println("Created 3 bulk orders with allocations")

	// 200 follows
	for i := 0; i < 200; i++ {
		followerIdx := i % len(artisanIDs)
		followeeIdx := (i + 1) % len(artisanIDs)
		if followerIdx == followeeIdx {
			continue
		}

		_, err := pool.Exec(ctx, `
			INSERT INTO follows (follower_id, followee_id, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT DO NOTHING
		`, artisanIDs[followerIdx], artisanIDs[followeeIdx])
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Created 200 follow relationships")

	// Some notifications
	for i := 0; i < 50; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO notifications (
				id, user_id, type, title, body, read, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`, uuid.New(), artisanIDs[i%len(artisanIDs)], "order_update",
			"New Order", "You have a new order!", i%3 == 0)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Created notifications")

	fmt.Println("✓ Seed complete")
}
