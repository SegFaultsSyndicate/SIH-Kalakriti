// services/channel-svc/internal/channel/export/indiahandmade_test.go
package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCSV(t *testing.T) {
	records := []IndiaHandmadeRecord{
		{
			ArtisanID:    "artisan-1",
			ArtisanName:  "Lakshmi Devi",
			ProductID:    "prod-1",
			ProductTitle: "Kanchipuram Saree",
			Description:  "Traditional silk saree",
			Craft:        "Weaving",
			Price:        "5000.00",
			Currency:     "INR",
			ImageURL:     "https://example.com/img1.jpg",
			GINumber:     "GI-123",
		},
	}

	var buf bytes.Buffer
	err := WriteCSV(&buf, records)
	require.NoError(t, err)

	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + 1 record.
	require.Len(t, rows, 2)
	assert.Equal(t, "artisan_id", rows[0][0])
	assert.Equal(t, "artisan-1", rows[1][0])
	assert.Equal(t, "Lakshmi Devi", rows[1][1])
}

func TestWriteJSON(t *testing.T) {
	records := []IndiaHandmadeRecord{
		{
			ArtisanID:    "artisan-1",
			ArtisanName:  "Lakshmi Devi",
			ProductID:    "prod-1",
			ProductTitle: "Kanchipuram Saree",
			Description:  "Traditional silk saree",
			Craft:        "Weaving",
			Price:        "5000.00",
			Currency:     "INR",
			ImageURL:     "https://example.com/img1.jpg",
			GINumber:     "GI-123",
		},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, records)
	require.NoError(t, err)

	var decoded []IndiaHandmadeRecord
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	require.Len(t, decoded, 1)
	assert.Equal(t, "artisan-1", decoded[0].ArtisanID)
	assert.Equal(t, "Lakshmi Devi", decoded[0].ArtisanName)
}

func TestEmptyRecords(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCSV(&buf, []IndiaHandmadeRecord{})
	require.NoError(t, err)

	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	require.NoError(t, err)

	// Just header.
	require.Len(t, rows, 1)
}
