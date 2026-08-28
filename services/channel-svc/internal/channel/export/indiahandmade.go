// services/channel-svc/internal/channel/export/indiahandmade.go

// Package export generates IndiaHandmade-compatible CSV/JSON catalog exports.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// IndiaHandmadeRecord is one catalog row in IndiaHandmade format.
type IndiaHandmadeRecord struct {
	ArtisanID    string `json:"artisan_id" csv:"artisan_id"`
	ArtisanName  string `json:"artisan_name" csv:"artisan_name"`
	ProductID    string `json:"product_id" csv:"product_id"`
	ProductTitle string `json:"product_title" csv:"product_title"`
	Description  string `json:"description" csv:"description"`
	Craft        string `json:"craft" csv:"craft"`
	Price        string `json:"price" csv:"price"`
	Currency     string `json:"currency" csv:"currency"`
	ImageURL     string `json:"image_url" csv:"image_url"`
	GINumber     string `json:"gi_number" csv:"gi_number"`
}

// WriteCSV writes records as CSV.
func WriteCSV(w io.Writer, records []IndiaHandmadeRecord) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header.
	if err := cw.Write([]string{
		"artisan_id", "artisan_name", "product_id", "product_title",
		"description", "craft", "price", "currency", "image_url", "gi_number",
	}); err != nil {
		return fmt.Errorf("csv header: %w", err)
	}

	// Rows.
	for _, r := range records {
		if err := cw.Write([]string{
			r.ArtisanID, r.ArtisanName, r.ProductID, r.ProductTitle,
			r.Description, r.Craft, r.Price, r.Currency, r.ImageURL, r.GINumber,
		}); err != nil {
			return fmt.Errorf("csv row: %w", err)
		}
	}

	return nil
}

// WriteJSON writes records as JSON array.
func WriteJSON(w io.Writer, records []IndiaHandmadeRecord) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}
