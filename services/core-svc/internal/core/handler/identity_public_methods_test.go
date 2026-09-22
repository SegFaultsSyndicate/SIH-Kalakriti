package handler

import "testing"

// Regression coverage for F-9 (WIRING_AUDIT_PLAN.md): the bff mounts
// POST /companies and GET /companies/:id as unauthenticated routes, and
// neither service.B2B.RegisterCompany nor .GetCompany requires a
// principal -- both RPCs must be on this allow-list or the gRPC auth
// interceptor rejects the call before the intentionally-public handler
// ever runs.
func TestPublicMethodsAllowsCompanyRegistrationAndLookup(t *testing.T) {
	methods := PublicMethods()

	for _, m := range []string{
		"/b2b.v1.B2BService/RegisterCompany",
		"/b2b.v1.B2BService/GetCompany",
	} {
		if _, ok := methods[m]; !ok {
			t.Errorf("PublicMethods() missing %s -- POST/GET /companies would 401 for an anonymous caller", m)
		}
	}
}
