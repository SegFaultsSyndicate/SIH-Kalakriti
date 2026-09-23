package mosje

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/finance/v1"
)

func TestProtoMapShape(t *testing.T) {
	emi := int64(3_000_00)
	m := protoMap(&financev1.LinkFinanceResponse{Link: &financev1.FinanceLink{
		Id: "l1", ReferenceLast4: "1234", EmiPaise: &emi,
		ConsentAt: timestamppb.New(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)),
	}})
	link := m["link"].(map[string]any)
	if link["emi_paise"] != int64(300000) {
		t.Fatalf("int64 must stay a number: %#v", link["emi_paise"])
	}
	if _, ok := link["sanctioned_paise"]; ok {
		t.Fatal("unset optional fields must be omitted")
	}
	if link["status"] != "" {
		t.Fatalf("non-optional scalars are always present: %#v", link["status"])
	}
	if link["consent_at"] != "2026-09-01T10:00:00Z" {
		t.Fatalf("timestamp = %#v", link["consent_at"])
	}
	if _, ok := link["verified_at"]; ok {
		t.Fatal("unset message fields must be omitted")
	}
}
