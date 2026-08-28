// pkg/domain/domain_test.go
package domain

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestHTTPStatusMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "not found", err: NotFound("listing"), want: http.StatusNotFound},
		{name: "conflict", err: Conflict("state"), want: http.StatusConflict},
		{name: "invalid input", err: InvalidInput("price"), want: http.StatusBadRequest},
		{name: "forbidden", err: Forbidden("owner"), want: http.StatusForbidden},
		{name: "unavailable", err: Unavailable("kafka"), want: http.StatusServiceUnavailable},
		{name: "unmapped", err: errors.New("boom"), want: http.StatusInternalServerError},
		{name: "wrapped not found", err: fmt.Errorf("get: %w", ErrNotFound), want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := HTTPStatus(tt.err); got != tt.want {
				t.Errorf("HTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestWriteHTTPErrorHidesInternalDetail(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteHTTPError(rec, errors.New("leaked db password in this message"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if want := "internal error"; !strings.Contains(body, want) {
		t.Errorf("body %q does not contain %q", body, want)
	}
	if strings.Contains(body, "leaked db password") {
		t.Errorf("body %q leaks internal error detail", body)
	}
}

func TestWriteHTTPErrorShowsMappedDetail(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteHTTPError(rec, NotFound("listing 123"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "listing 123") {
		t.Errorf("body %q should contain the mapped error detail", rec.Body.String())
	}
}

func TestGRPCStatusMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "nil", err: nil, want: codes.OK},
		{name: "not found", err: NotFound("artisan"), want: codes.NotFound},
		{name: "conflict", err: Conflict("state"), want: codes.AlreadyExists},
		{name: "invalid input", err: InvalidInput("phone"), want: codes.InvalidArgument},
		{name: "forbidden", err: Forbidden("owner"), want: codes.PermissionDenied},
		{name: "unavailable", err: Unavailable("kafka"), want: codes.Unavailable},
		{name: "unmapped", err: errors.New("boom"), want: codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GRPCStatus(tt.err).Code(); got != tt.want {
				t.Errorf("GRPCStatus(%v).Code() = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestGRPCErrorUnmappedHidesDetail(t *testing.T) {
	t.Parallel()
	err := GRPCError(errors.New("leaked db password in this message"))
	if strings.Contains(err.Error(), "leaked db password") {
		t.Errorf("GRPCError leaked internal detail: %v", err)
	}
}

func TestGRPCErrorNilIsNil(t *testing.T) {
	t.Parallel()
	if err := GRPCError(nil); err != nil {
		t.Errorf("GRPCError(nil) = %v, want nil", err)
	}
}

func TestConstructorsWrapSentinels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err      error
		sentinel error
	}{
		{NotFound("x"), ErrNotFound},
		{Conflict("x"), ErrConflict},
		{InvalidInput("x"), ErrInvalidInput},
		{Forbidden("x"), ErrForbidden},
		{Unavailable("x"), ErrUnavailable},
		{NotFoundf("listing %s", "abc"), ErrNotFound},
		{Conflictf("state %s", "abc"), ErrConflict},
	}
	for _, c := range cases {
		if !errors.Is(c.err, c.sentinel) {
			t.Errorf("errors.Is(%v, %v) = false, want true", c.err, c.sentinel)
		}
	}
}
