// services/core-svc/internal/core/handler/audit.go

package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// readPrefixes mark RPCs that change nothing; they are not audited.
var readPrefixes = []string{"Get", "List", "Search", "Resolve"}

func isRead(fullMethod string) bool {
	name := fullMethod[strings.LastIndex(fullMethod, "/")+1:]
	for _, p := range readPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// AuditInterceptor writes one audit_log row for every successful write an
// agent makes on an artisan's behalf (a principal with Actor set). It sits
// after the auth interceptor, so the principal is already verified.
//
// The row's resource is the id the response names (e.g. the new listing), so
// "who created this listing for me" and agent productivity can be answered
// from audit_log alone. An audit failure is logged, not returned: the write
// has already committed and failing the RPC would only invite a duplicate.
func AuditInterceptor(w service.AuditWriter, log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		resp, err := next(ctx, req)
		if err != nil || isRead(info.FullMethod) {
			return resp, err
		}
		p, ok := auth.PrincipalFrom(ctx)
		if !ok || p.Actor == "" {
			return resp, err
		}
		if aerr := service.AuditOnBehalf(ctx, w, p, info.FullMethod, resourceIDFrom(resp)); aerr != nil {
			log.ErrorContext(ctx, "assisted audit write failed", "method", info.FullMethod, "error", aerr)
		}
		return resp, nil
	}
}

// resourceIDFrom finds the id a response is about: a top-level "id" field, or
// the "id" of its first populated message field (CreateListingResponse.listing.id).
func resourceIDFrom(resp any) *uuid.UUID {
	m, ok := resp.(proto.Message)
	if !ok {
		return nil
	}
	if id := idField(m.ProtoReflect()); id != nil {
		return id
	}
	var found *uuid.UUID
	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() {
			found = idField(v.Message())
		}
		return found == nil
	})
	return found
}

func idField(m protoreflect.Message) *uuid.UUID {
	fd := m.Descriptor().Fields().ByName("id")
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return nil
	}
	id, err := uuid.Parse(m.Get(fd).String())
	if err != nil {
		return nil
	}
	return &id
}
