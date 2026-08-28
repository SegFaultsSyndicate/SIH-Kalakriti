// pkg/qrcode/qrcode.go

// Package qrcode generates QR codes for provenance verification URLs.
package qrcode

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/skip2/go-qrcode"
)

// Generator creates QR codes as PNG and SVG.
type Generator struct {
	baseURL string
}

// NewGenerator builds a QR code generator with the given base verification URL.
func NewGenerator(baseURL string) *Generator {
	return &Generator{baseURL: baseURL}
}

// GeneratePNG creates a PNG QR code encoding the verification URL for the given short code.
func (g *Generator) GeneratePNG(shortCode string, size int) ([]byte, error) {
	url := fmt.Sprintf("%s/v/%s", g.baseURL, shortCode)
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("creating QR code: %w", err)
	}
	qr.DisableBorder = false
	img := qr.Image(size)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateSVG creates an SVG QR code encoding the verification URL for the given short code.
func (g *Generator) GenerateSVG(shortCode string, size int) ([]byte, error) {
	url := fmt.Sprintf("%s/v/%s", g.baseURL, shortCode)
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("creating QR code: %w", err)
	}
	qr.DisableBorder = false
	svg := qr.ToSmallString(false)
	return []byte(svg), nil
}

// VerificationURL returns the full verification URL for a short code.
func (g *Generator) VerificationURL(shortCode string) string {
	return fmt.Sprintf("%s/v/%s", g.baseURL, shortCode)
}
