// pkg/auth/interceptor.go
package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authorizationHeader is the gRPC metadata key carrying the bearer token.
// gRPC lowercases metadata keys, so this must stay lowercase.
const authorizationHeader = "authorization"

const bearerPrefix = "bearer "

// Verifier is the slice of Issuer the interceptor needs. Declaring it as an
// interface keeps the interceptor testable without minting real tokens.
type Verifier interface {
	Verify(token string, want TokenKind) (*Claims, error)
}

// PublicMethods is the set of fully-qualified gRPC method names that may be
// called without a token, e.g. "/identity.v1.IdentityService/RequestOtp".
type PublicMethods map[string]struct{}

// NewPublicMethods builds a PublicMethods set from full method names.
func NewPublicMethods(methods ...string) PublicMethods {
	set := make(PublicMethods, len(methods))
	for _, m := range methods {
		set[m] = struct{}{}
	}
	return set
}

// UnaryServerInterceptor validates the bearer token on every unary call and puts
// the resulting Principal in the context. Calls to a method in public are passed
// through unauthenticated; every other method requires a valid access token.
//
// A missing, malformed, expired or forged token all produce codes.Unauthenticated
// with the same generic message, so a caller learns nothing about why.
func UnaryServerInterceptor(v Verifier, public PublicMethods) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := public[info.FullMethod]; ok {
			// Still attach a principal when a token happens to be present, so a
			// public RPC can tell an anonymous caller from a signed-in one.
			if p, err := principalFromMetadata(ctx, v); err == nil {
				ctx = ContextWithPrincipal(ctx, p)
			}
			return handler(ctx, req)
		}

		p, err := principalFromMetadata(ctx, v)
		if err != nil {
			return nil, err
		}
		return handler(ContextWithPrincipal(ctx, p), req)
	}
}

// StreamServerInterceptor is the streaming counterpart of UnaryServerInterceptor.
func StreamServerInterceptor(v Verifier, public PublicMethods) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if _, ok := public[info.FullMethod]; ok {
			return handler(srv, ss)
		}
		p, err := principalFromMetadata(ss.Context(), v)
		if err != nil {
			return err
		}
		return handler(srv, &principalStream{ServerStream: ss, ctx: ContextWithPrincipal(ss.Context(), p)})
	}
}

// principalStream overrides Context so downstream handlers see the principal.
type principalStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the principal-carrying context.
func (s *principalStream) Context() context.Context { return s.ctx }

// unauthenticated is the single error every auth failure renders as, so the
// wire response never distinguishes expired from forged from absent.
func unauthenticated() error {
	return status.Error(codes.Unauthenticated, "invalid or missing access token")
}

func principalFromMetadata(ctx context.Context, v Verifier) (Principal, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, unauthenticated()
	}
	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return Principal{}, unauthenticated()
	}
	raw := values[0]
	if len(raw) < len(bearerPrefix) || !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
		return Principal{}, unauthenticated()
	}
	token := strings.TrimSpace(raw[len(bearerPrefix):])
	if token == "" {
		return Principal{}, unauthenticated()
	}

	claims, err := v.Verify(token, KindAccess)
	if err != nil {
		return Principal{}, unauthenticated()
	}
	p, err := claims.Principal()
	if err != nil {
		return Principal{}, unauthenticated()
	}
	return p, nil
}

// BearerMetadata builds outgoing gRPC metadata carrying an access token, for
// service-to-service calls that forward the caller's identity.
func BearerMetadata(accessToken string) metadata.MD {
	return metadata.Pairs(authorizationHeader, "Bearer "+accessToken)
}
