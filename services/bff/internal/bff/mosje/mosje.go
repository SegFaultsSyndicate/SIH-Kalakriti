// Package mosje is bff's REST surface for the MoSJE tier-4 features: finance
// corporation linkage (F12), income and impact (F13), assisted mode (F14)
// and digital literacy (F15).
//
// Unlike the older handler/client split (a map[string]any adapter per RPC),
// these routes call the generated gRPC clients directly and render the proto
// response generically (see protoMap): the proto is the contract, and
// openapi.json documents the same snake_case shape.
package mosje

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	assistedv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/assisted/v1"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	financev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/finance/v1"
	impactv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/impact/v1"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
	literacyv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/literacy/v1"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/client"
)

// Handler holds the backend clients.
type Handler struct {
	Finance  financev1.FinanceServiceClient
	Income   impactv1.IncomeServiceClient
	Staff    assistedv1.StaffServiceClient
	Assisted assistedv1.AssistedServiceClient
	Literacy literacyv1.LiteracyServiceClient
	Insight  insightv1.InsightServiceClient
	Media    catalogv1.MediaServiceClient
}

// New builds the handler over core-svc and insight-svc connections.
func New(core, insight grpc.ClientConnInterface) *Handler {
	return &Handler{
		Finance:  financev1.NewFinanceServiceClient(core),
		Income:   impactv1.NewIncomeServiceClient(core),
		Staff:    assistedv1.NewStaffServiceClient(core),
		Assisted: assistedv1.NewAssistedServiceClient(core),
		Literacy: literacyv1.NewLiteracyServiceClient(core),
		Insight:  insightv1.NewInsightServiceClient(insight),
		Media:    catalogv1.NewMediaServiceClient(core),
	}
}

// ResolveOnBehalf implements Resolver over core-svc, with the agent's token.
func (h *Handler) ResolveOnBehalf(ctx context.Context, artisanID string) (bool, error) {
	ctx, cancel := client.Outgoing(ctx)
	defer cancel()
	resp, err := h.Assisted.ResolveOnBehalf(ctx, &assistedv1.ResolveOnBehalfRequest{ArtisanId: artisanID})
	if err != nil {
		return false, client.Err(err)
	}
	return resp.GetAllowed(), nil
}

// unary adapts one gRPC call to an HTTP handler: the JSON body (for
// POST/PUT/PATCH) is decoded into the request with protojson, fill copies
// path/query/principal values on top, and the response is rendered with
// protoMap.
func unary[Req any, PReq interface {
	*Req
	proto.Message
}, Resp proto.Message](
	rpc func(context.Context, PReq, ...grpc.CallOption) (Resp, error),
	fill func(r *http.Request, req PReq) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := PReq(new(Req))
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if err := decode(r, req); err != nil {
				httpx.Error(w, err)
				return
			}
		}
		if fill != nil {
			if err := fill(r, req); err != nil {
				httpx.Error(w, err)
				return
			}
		}
		ctx, cancel := client.Outgoing(r.Context())
		defer cancel()
		resp, err := rpc(ctx, req)
		if err != nil {
			httpx.Error(w, client.Err(err))
			return
		}
		httpx.JSON(w, http.StatusOK, protoMap(resp))
	}
}

func decode(r *http.Request, m proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return domain.InvalidInput("cannot read request body")
	}
	if len(body) == 0 {
		return nil
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, m); err != nil {
		return domain.InvalidInput(fmt.Sprintf("invalid request body: %v", err))
	}
	return nil
}

// registrationLanguages lets /assisted/link take the same short language
// names POST /artisans does ("HINDI"), mapping them to the proto enum names
// protojson expects ("LANGUAGE_HINDI").
func registrationLanguages(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.Error(w, domain.InvalidInput("cannot read request body"))
			return
		}
		var m map[string]any
		if json.Unmarshal(body, &m) == nil {
			if reg, ok := m["registration"].(map[string]any); ok {
				if langs, ok := reg["languages"].([]any); ok {
					for i, l := range langs {
						if s, ok := l.(string); ok && !strings.HasPrefix(s, "LANGUAGE_") {
							langs[i] = "LANGUAGE_" + s
						}
					}
					body, _ = json.Marshal(m)
				}
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next(w, r)
	}
}

// self is the caller's own subject (the artisan being acted for, in
// assisted mode).
func self(r *http.Request) (string, error) {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		return "", domain.Unauthenticated("sign in first")
	}
	return p.Subject, nil
}

// protoMap renders a message as snake_case JSON with int64 as numbers (not
// protojson's strings), timestamps as RFC 3339, unset optional fields
// omitted and every other scalar present even when zero.
func protoMap(m proto.Message) map[string]any {
	return messageMap(m.ProtoReflect())
}

func messageMap(m protoreflect.Message) map[string]any {
	out := map[string]any{}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.HasPresence() && !m.Has(fd) {
			continue
		}
		v := m.Get(fd)
		name := string(fd.Name())
		switch {
		case fd.IsList():
			list := v.List()
			items := make([]any, list.Len())
			for j := range items {
				items[j] = scalar(fd, list.Get(j))
			}
			out[name] = items
		default:
			out[name] = scalar(fd, v)
		}
	}
	return out
}

func scalar(fd protoreflect.FieldDescriptor, v protoreflect.Value) any {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		msg := v.Message()
		if ts, ok := msg.Interface().(*timestamppb.Timestamp); ok {
			return ts.AsTime().UTC().Format(time.RFC3339)
		}
		return messageMap(msg)
	case protoreflect.EnumKind:
		if ev := fd.Enum().Values().ByNumber(v.Enum()); ev != nil {
			return string(ev.Name())
		}
		return int32(v.Enum())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return int32(v.Int())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return v.Int()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return v.Uint()
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return v.Float()
	case protoreflect.BoolKind:
		return v.Bool()
	case protoreflect.BytesKind:
		return v.Bytes()
	default:
		return v.String()
	}
}

// impactFilter reads the shared dashboard filter from the query string.
// insight-svc clamps it to a cluster officer's own scope.
func impactFilter(r *http.Request) *insightv1.ImpactFilter {
	q := r.URL.Query()
	return &insightv1.ImpactFilter{
		StateCode: q.Get("state_code"), District: q.Get("district"),
		SocialCategory: q.Get("social_category"), Corporation: q.Get("corporation"),
		FromMonth: q.Get("from_month"), ToMonth: q.Get("to_month"),
	}
}

func optQuery(r *http.Request, key string) *string {
	if v := r.URL.Query().Get(key); v != "" {
		return &v
	}
	return nil
}

// exportImpactCSV streams GetImpactByGroup as CSV. A suppressed group keeps
// its label and says "<5" -- its numbers never leave insight-svc.
func (h *Handler) exportImpactCSV(w http.ResponseWriter, r *http.Request) {
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "district"
	}
	ctx, cancel := client.Outgoing(r.Context())
	defer cancel()
	resp, err := h.Insight.GetImpactByGroup(ctx, &insightv1.GetImpactByGroupRequest{GroupBy: groupBy, Filter: impactFilter(r)})
	if err != nil {
		httpx.Error(w, client.Err(err))
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="kalakriti-impact-%s-%s.csv"`, groupBy, time.Now().UTC().Format("2006-01-02")))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"group", "state_code", "artisans", "active_sellers_90d", "with_baseline",
		"median_baseline_monthly_inr", "median_current_monthly_inr", "median_uplift_pct",
		"platform_income_90d_inr", "offline_income_90d_inr", "fair_income_90d_inr"})
	inr := func(p int64) string { return strconv.FormatFloat(float64(p)/100, 'f', 2, 64) }
	for _, row := range resp.GetRows() {
		if row.GetSuppressed() {
			_ = cw.Write([]string{row.GetGroup(), row.GetStateCode(), "<5", "", "", "", "", "", "", "", ""})
			continue
		}
		uplift := ""
		if row.MedianUpliftPct != nil {
			uplift = strconv.FormatFloat(row.GetMedianUpliftPct(), 'f', 1, 64)
		}
		baseline, current := "", ""
		if row.GetWithBaselineCount() >= 5 {
			baseline, current = inr(row.GetMedianBaselineMonthlyPaise()), inr(row.GetMedianCurrentMonthlyPaise())
		}
		_ = cw.Write([]string{
			row.GetGroup(), row.GetStateCode(), strconv.FormatInt(row.GetArtisanCount(), 10),
			strconv.FormatInt(row.GetActiveSellers_90D(), 10), strconv.FormatInt(row.GetWithBaselineCount(), 10),
			baseline, current, uplift,
			inr(row.GetPlatformIncomePaise_90D()), inr(row.GetOfflineIncomePaise_90D()), inr(row.GetFairIncomePaise_90D()),
		})
	}
	cw.Flush()
}
