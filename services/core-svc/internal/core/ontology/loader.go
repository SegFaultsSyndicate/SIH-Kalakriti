// services/core-svc/internal/core/ontology/loader.go
package ontology

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// Writer is what the loader writes through. Both methods are upserts keyed on a
// natural key — the craft's code and the (alias, script) pair — which is what
// makes a re-run of the seed a no-op rather than a pile of duplicates.
type Writer interface {
	// UpsertCraft inserts or updates one craft by code and returns its id,
	// which is the existing id when the row was already there.
	UpsertCraft(ctx context.Context, c domain.Craft) (uuid.UUID, error)
	// UpsertCraftAlias inserts or updates one alias by (lower(alias), script).
	UpsertCraftAlias(ctx context.Context, a domain.CraftAlias) error
}

// LoadResult reports what a load did, for the CLI to print and the tests to assert.
type LoadResult struct {
	Crafts  int
	Aliases int
}

// listSeparator separates repeated values inside one CSV field, e.g.
// "resist-dyeing|block-printing". A comma cannot be used, and a semicolon reads
// badly in a spreadsheet.
const listSeparator = "|"

// Load reads a craft CSV and an alias CSV and writes both into the store. It is
// idempotent: running it twice leaves the same rows, with the same ids.
//
// The craft file is written in two passes because a craft may name a parent that
// appears later in the file: the first pass creates every craft without its
// parent, the second sets the parents once every code has an id.
//
// crafts.csv: code,display_name,parent_code,gi_registration_no,techniques,materials
// aliases.csv: craft_code,alias,script,language,source
func Load(ctx context.Context, w Writer, craftsCSV, aliasesCSV io.Reader) (LoadResult, error) {
	craftRows, err := readCSV(craftsCSV, "crafts", 6)
	if err != nil {
		return LoadResult{}, err
	}
	aliasRows, err := readCSV(aliasesCSV, "aliases", 5)
	if err != nil {
		return LoadResult{}, err
	}

	idsByCode := make(map[string]uuid.UUID, len(craftRows))
	parents := make(map[string]string, len(craftRows))

	for i, row := range craftRows {
		craft, parentCode, err := craftFromRow(row)
		if err != nil {
			return LoadResult{}, fmt.Errorf("crafts.csv line %d: %w", i+2, err)
		}
		id, err := w.UpsertCraft(ctx, craft)
		if err != nil {
			return LoadResult{}, fmt.Errorf("upserting craft %q: %w", craft.Code, err)
		}
		idsByCode[craft.Code] = id
		if parentCode != "" {
			parents[craft.Code] = parentCode
		}
	}

	for i, row := range craftRows {
		craft, _, err := craftFromRow(row)
		if err != nil {
			return LoadResult{}, fmt.Errorf("crafts.csv line %d: %w", i+2, err)
		}
		parentCode, ok := parents[craft.Code]
		if !ok {
			continue
		}
		parentID, ok := idsByCode[parentCode]
		if !ok {
			return LoadResult{}, fmt.Errorf("crafts.csv line %d: parent craft %q is not in the file", i+2, parentCode)
		}
		craft.ID = idsByCode[craft.Code]
		craft.ParentCraftID = &parentID
		if _, err := w.UpsertCraft(ctx, craft); err != nil {
			return LoadResult{}, fmt.Errorf("setting the parent of craft %q: %w", craft.Code, err)
		}
	}

	for i, row := range aliasRows {
		craftCode := strings.TrimSpace(row[0])
		craftID, ok := idsByCode[craftCode]
		if !ok {
			return LoadResult{}, fmt.Errorf("aliases.csv line %d: craft %q is not in crafts.csv", i+2, craftCode)
		}
		alias := domain.CraftAlias{
			CraftID:  craftID,
			Alias:    strings.TrimSpace(row[1]),
			Script:   strings.TrimSpace(row[2]),
			Language: strings.ToUpper(strings.TrimSpace(row[3])),
			Source:   strings.TrimSpace(row[4]),
		}
		if alias.Source == "" {
			alias.Source = "curator"
		}
		if err := alias.Validate(); err != nil {
			return LoadResult{}, fmt.Errorf("aliases.csv line %d: %w", i+2, err)
		}
		if err := w.UpsertCraftAlias(ctx, alias); err != nil {
			return LoadResult{}, fmt.Errorf("upserting alias %q: %w", alias.Alias, err)
		}
	}

	return LoadResult{Crafts: len(craftRows), Aliases: len(aliasRows)}, nil
}

// craftFromRow converts one CSV record, returning the parent's code separately
// because it cannot be resolved to an id until every craft has one.
func craftFromRow(row []string) (domain.Craft, string, error) {
	code := strings.TrimSpace(row[0])
	if code == "" {
		return domain.Craft{}, "", errors.New("code is required")
	}
	displayName := strings.TrimSpace(row[1])
	if displayName == "" {
		return domain.Craft{}, "", fmt.Errorf("craft %q has no display_name", code)
	}

	craft := domain.Craft{
		Code:        code,
		DisplayName: displayName,
		Techniques:  splitList(row[4]),
		Materials:   splitList(row[5]),
	}
	if gi := strings.TrimSpace(row[3]); gi != "" {
		craft.GIRegistrationNo = &gi
	}
	return craft, strings.TrimSpace(row[2]), nil
}

// readCSV reads a whole file, skipping the header row and checking the width so
// a column added upstream fails loudly rather than shifting every field.
func readCSV(r io.Reader, what string, columns int) ([][]string, error) {
	if r == nil {
		return nil, nil
	}
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = columns
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading the %s csv: %w", what, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("the %s csv is empty", what)
	}
	return rows[1:], nil
}

// splitList splits a pipe-separated CSV field, dropping blanks.
func splitList(field string) []string {
	out := []string{}
	for _, part := range strings.Split(field, listSeparator) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
