// scripts/generate-qr-sheet/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

func main() {
	// Generate sample QR codes for demo
	qrCodes := []struct {
		label string
		data  string
	}{
		{"Provenance #1", "https://kalakriti.in/verify/prov/abc123"},
		{"Provenance #2", "https://kalakriti.in/verify/prov/def456"},
		{"Provenance #3", "https://kalakriti.in/verify/prov/ghi789"},
		{"Income Statement - Aug 2026", "https://kalakriti.in/verify/income/xyz890"},
		{"Income Statement - Jul 2026", "https://kalakriti.in/verify/income/mno345"},
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Kalakriti QR Code Sheet")
	pdf.Ln(15)

	x, y := 20.0, 40.0
	size := 40.0
	cols := 3

	for i, qr := range qrCodes {
		// Generate QR code
		filename := fmt.Sprintf("/tmp/qr_%d.png", i)
		err := qrcode.WriteFile(qr.data, qrcode.Medium, 256, filename)
		if err != nil {
			log.Fatal(err)
		}

		// Calculate position
		col := i % cols
		row := i / cols
		posX := x + float64(col)*60
		posY := y + float64(row)*60

		// Add QR image
		pdf.Image(filename, posX, posY, size, size, false, "", 0, "")

		// Add label
		pdf.SetFont("Arial", "", 8)
		pdf.SetXY(posX, posY+size+2)
		pdf.Cell(size, 5, qr.label)

		// Cleanup
		os.Remove(filename)
	}

	err := pdf.OutputFileAndClose("qr-sheet.pdf")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ QR sheet generated: qr-sheet.pdf")
}
