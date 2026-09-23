// services/core-svc/internal/core/domain/finance_test.go
package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

func TestHashReferenceNeverKeepsThePlaintext(t *testing.T) {
	salt := []byte("test-salt-at-least-16-bytes")
	ref := "NSFDC/UP/2024/0098765"

	last4, hash, err := HashReference(salt, ref)
	if err != nil {
		t.Fatalf("HashReference: %v", err)
	}
	if last4 != "8765" {
		t.Errorf("last4 = %q, want 8765", last4)
	}
	norm := NormalizeReference(ref)
	if bytes.Contains(hash, []byte(norm)) || bytes.Contains(hash, []byte(ref)) {
		t.Fatal("the stored hash must not contain the reference")
	}
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32 (HMAC-SHA256)", len(hash))
	}

	// Same number typed differently -> same hash (de-duplication works).
	_, again, _ := HashReference(salt, " nsfdc-up-2024-0098765 ")
	if !bytes.Equal(hash, again) {
		t.Error("formatting differences must not change the hash")
	}
	// A different salt yields a different hash: the salt is a real secret.
	_, other, _ := HashReference([]byte("another-salt-entirely-xx"), ref)
	if bytes.Equal(hash, other) {
		t.Error("hash must depend on the salt")
	}
}

func TestHashReferenceRejectsTooShortOrLong(t *testing.T) {
	for _, bad := range []string{"", "12", "---", strings.Repeat("9", 41)} {
		if _, _, err := HashReference([]byte("s"), bad); !errors.Is(err, pkgdomain.ErrInvalidInput) {
			t.Errorf("HashReference(%q) err = %v, want ErrInvalidInput", bad, err)
		}
	}
}

func TestComputeCoverage(t *testing.T) {
	today := time.Date(2026, 9, 20, 15, 0, 0, 0, time.UTC)
	month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	emi, day := int64(200_000), int32(25)
	links := []FinanceLink{{Status: FinanceVerified, EmiPaise: &emi, EmiDayOfMonth: &day}}

	c := ComputeCoverage(links, month, 150_000, 60_000, 90_000, today)
	if c.Status != CoverageCovered || c.CoverageRatio < 1.04 || c.CoverageRatio > 1.06 {
		t.Errorf("coverage = %+v, want COVERED at 1.05", c)
	}
	if c.PendingPaise != 90_000 {
		t.Error("pending must be reported")
	}
	if c.DaysToEmi != 5 || !c.AnyVerified {
		t.Errorf("days_to_emi = %d verified = %v, want 5 true", c.DaysToEmi, c.AnyVerified)
	}

	// Pending (unsettled) payouts never count towards coverage.
	if c := ComputeCoverage(links, month, 100_000, 0, 1_000_000, today); c.Status != CoverageNotYet {
		t.Errorf("status = %s, want NOT_YET: pending must not count", c.Status)
	}
	if c := ComputeCoverage(links, month, 160_000, 0, 0, today); c.Status != CoverageAlmost {
		t.Errorf("status = %s, want ALMOST at 0.8", c.Status)
	}

	// Rejected links contribute nothing; no EMI at all is its own state.
	rejected := []FinanceLink{{Status: FinanceRejected, EmiPaise: &emi}}
	if c := ComputeCoverage(rejected, month, 1, 0, 0, today); c.Status != CoverageNoEMI || c.DaysToEmi != -1 {
		t.Errorf("rejected-only coverage = %+v, want NO_EMI / -1", c)
	}
}

func TestDaysUntilDayWrapsToNextMonth(t *testing.T) {
	today := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	if d := DaysUntilDay(today, 27); d != 0 {
		t.Errorf("same day = %d, want 0", d)
	}
	if d := DaysUntilDay(today, 3); d != 6 {
		t.Errorf("wrapped = %d, want 6 (Oct 3)", d)
	}
}
