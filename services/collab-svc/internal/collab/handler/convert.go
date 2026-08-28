// services/collab-svc/internal/collab/handler/convert.go
package handler

import (
	"fmt"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	commonv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/common/v1"
)

// parseUUID converts a request field to a UUID, reporting a field-named
// ErrInvalidInput rather than a bare parse error.
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s %q is not a valid uuid: %w", field, value, pkgdomain.ErrInvalidInput)
	}
	return id, nil
}

// parseOptionalUUID converts an optional request field to a UUID pointer.
func parseOptionalUUID(field string, value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	id, err := parseUUID(field, *value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// parseUUIDs converts a repeated request field to UUIDs.
func parseUUIDs(field string, values []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		id, err := parseUUID(field, v)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// moneyToProto renders paise as a common.v1.Money, INR always.
func moneyToProto(paise int64) *commonv1.Money {
	return &commonv1.Money{AmountPaise: paise, CurrencyCode: "INR"}
}
