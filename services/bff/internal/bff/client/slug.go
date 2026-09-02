// services/bff/internal/bff/client/slug.go
package client

import (
	"regexp"
	"strings"
)

// slugSeparator joins a slugified title to the id it resolves to. slugify
// never itself produces a double dash, so splitting on the last occurrence
// of this separator recovers the id unambiguously — no stored slug column,
// no migration, no new RPC: the slug is entirely derived and reversible.
const slugSeparator = "--"

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases s and collapses every run of non-alphanumeric
// characters to a single dash, trimming leading/trailing dashes.
func slugify(s string) string {
	s = slugNonAlnum.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(s, "-")
}

// buildSlug derives a public slug for name (a listing title or artisan
// display name) and id. Empty name still yields a parseable slug — just an
// id with no human-readable prefix.
func buildSlug(name, id string) string {
	if s := slugify(name); s != "" {
		return s + slugSeparator + id
	}
	return id
}

// parseSlugID recovers the id buildSlug encoded into slug. ok is false for
// anything not shaped like a slug this package built.
func parseSlugID(slug string) (id string, ok bool) {
	if i := strings.LastIndex(slug, slugSeparator); i >= 0 {
		return slug[i+len(slugSeparator):], true
	}
	if slug != "" {
		return slug, true
	}
	return "", false
}
