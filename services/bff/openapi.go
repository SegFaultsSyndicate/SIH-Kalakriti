// Package assets embeds files the bff binary must serve regardless of the
// working directory it was launched from.
package assets

import _ "embed"

//go:embed openapi.json
var OpenAPIJSON []byte
