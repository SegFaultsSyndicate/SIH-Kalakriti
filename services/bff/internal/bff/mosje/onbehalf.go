package mosje

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// HeaderOnBehalfOf names the artisan a field agent is acting for.
const HeaderOnBehalfOf = "X-On-Behalf-Of"

// Resolver answers "may this agent act for this artisan right now".
type Resolver interface {
	ResolveOnBehalf(ctx context.Context, artisanID string) (bool, error)
}

// onBehalfAllowed is the complete list of routes an agent may call for an
// artisan (method + gin route pattern). Everything else is refused: money
// movement, orders, follows, webhooks, phone change, consent withdrawal
// (finance link delete, helper revoke) and anything else not listed stays
// with the artisan's own phone.
var onBehalfAllowed = map[string]bool{
	"GET /api/v1/artisans/me":                   true,
	"PATCH /api/v1/artisans/me":                 true,
	"GET /api/v1/listings":                      true,
	"GET /api/v1/listings/:id":                  true,
	"GET /api/v1/listings/:id/attributes":       true,
	"POST /api/v1/listings":                     true,
	"PATCH /api/v1/listings/:id":                true,
	"POST /api/v1/listings/:id/media":           true,
	"POST /api/v1/listings/:id/submit":          true,
	"POST /api/v1/media/upload-url":             true,
	"POST /api/v1/media/:id/confirm":            true,
	"POST /api/v1/pricing/advise":               true,
	"GET /api/v1/badges/me/progress":            true,
	"GET /api/v1/schemes/match":                 true,
	"POST /api/v1/statements":                   true,
	"GET /api/v1/statements":                    true,
	"GET /api/v1/statements/:id":                true,
	"GET /api/v1/income/baseline":               true,
	"PUT /api/v1/income/baseline":               true,
	"GET /api/v1/income/sales":                  true,
	"POST /api/v1/income/sales":                 true,
	"DELETE /api/v1/income/sales/:id":           true,
	"GET /api/v1/income/summary":                true,
	"GET /api/v1/finance/links":                 true,
	"POST /api/v1/finance/links":                true,
	"PATCH /api/v1/finance/links/:id":           true,
	"GET /api/v1/finance/coverage":              true,
	"GET /api/v1/learn/lessons":                 true,
	"POST /api/v1/learn/lessons/:code/progress": true,
	"GET /api/v1/learn/certificate":             true,
	"POST /api/v1/learn/certificate":            true,
}

// OnBehalf turns an agent's request carrying X-On-Behalf-Of into a request
// by that artisan: after checking the route is allow-listed and core-svc
// confirms an active consent link, it swaps in a short-lived ARTISAN token
// whose `act` claim names the agent. Downstream services see an ordinary
// artisan call (so every ownership check still applies) and core-svc's
// audit interceptor records who really made each write.
//
// It must run after Auth (or OptionalAuth); without the header it does
// nothing.
func OnBehalf(issuer *auth.Issuer, resolver Resolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		target := c.GetHeader(HeaderOnBehalfOf)
		if target == "" {
			c.Next()
			return
		}
		fail := func(err error) {
			httpx.Error(c.Writer, err)
			c.Abort()
		}
		p, ok := auth.PrincipalFrom(c.Request.Context())
		if !ok {
			fail(domain.Unauthenticated("sign in before acting for an artisan"))
			return
		}
		if !p.HasRole(auth.RoleFieldAgent, auth.RoleClusterOfficer) {
			fail(domain.Forbidden("only field staff can act for an artisan"))
			return
		}
		if !onBehalfAllowed[c.Request.Method+" "+c.FullPath()] {
			fail(domain.Forbidden("this action is not available in assisted mode; the artisan must do it on their own phone"))
			return
		}
		if _, err := uuid.Parse(target); err != nil {
			fail(domain.InvalidInput(HeaderOnBehalfOf + " must be an artisan id"))
			return
		}
		allowed, err := resolver.ResolveOnBehalf(c.Request.Context(), target)
		if err != nil {
			fail(err)
			return
		}
		if !allowed {
			fail(domain.Forbidden("you are not linked to this artisan, or they removed your access"))
			return
		}
		pair, err := issuer.Issue(auth.Subject{ID: target, Role: auth.RoleArtisan, Language: p.Language, Actor: p.Subject})
		if err != nil {
			fail(err)
			return
		}
		claims, err := issuer.Verify(pair.AccessToken, auth.KindAccess)
		if err != nil {
			fail(err)
			return
		}
		acting, err := claims.Principal()
		if err != nil {
			fail(err)
			return
		}
		ctx := auth.ContextWithPrincipal(c.Request.Context(), acting)
		ctx = auth.ContextWithToken(ctx, pair.AccessToken)
		c.Request = c.Request.WithContext(ctx)
		c.Header("Vary", HeaderOnBehalfOf)
		c.Next()
	}
}
