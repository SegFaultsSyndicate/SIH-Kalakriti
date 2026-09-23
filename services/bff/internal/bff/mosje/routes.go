package mosje

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	assistedv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/assisted/v1"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	financev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/finance/v1"
	impactv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/impact/v1"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
	literacyv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/literacy/v1"
)

// Mount wires every tier-4 route. idem wraps a mutating handler with the
// idempotency middleware; every POST/PUT/PATCH/DELETE here goes through it.
// Authorisation (which role may call what) is enforced by core-svc and
// insight-svc, which see the forwarded token.
func (h *Handler) Mount(public, authed *gin.RouterGroup, idem func(http.HandlerFunc) http.HandlerFunc) {
	w := httpx.WrapHandler
	mut := func(f http.HandlerFunc) gin.HandlerFunc { return w(idem(f)) }
	param := httpx.URLParam

	// --- F12 finance corporation linkage ---
	authed.GET("/finance/links", w(unary(h.Finance.ListMyFinanceLinks, nil)))
	authed.POST("/finance/links", mut(unary(h.Finance.LinkFinance, nil)))
	authed.PATCH("/finance/links/:id", mut(unary(h.Finance.UpdateFinanceLink, func(r *http.Request, req *financev1.UpdateFinanceLinkRequest) error {
		req.Id = param(r, "id")
		return nil
	})))
	authed.DELETE("/finance/links/:id", mut(unary(h.Finance.DeleteFinanceLink, func(r *http.Request, req *financev1.DeleteFinanceLinkRequest) error {
		req.Id = param(r, "id")
		return nil
	})))
	authed.GET("/finance/coverage", w(unary(h.Finance.GetRepaymentCoverage, func(r *http.Request, req *financev1.GetRepaymentCoverageRequest) error {
		req.Month = r.URL.Query().Get("month")
		return nil
	})))
	authed.GET("/admin/finance/links", w(unary(h.Finance.ListFinanceLinksForReview, func(r *http.Request, req *financev1.ListFinanceLinksForReviewRequest) error {
		req.Status, req.StateCode, req.District = optQuery(r, "status"), optQuery(r, "state_code"), optQuery(r, "district")
		return nil
	})))
	authed.POST("/admin/finance/links/:id/review", mut(unary(h.Finance.ReviewFinanceLink, func(r *http.Request, req *financev1.ReviewFinanceLinkRequest) error {
		req.Id = param(r, "id")
		return nil
	})))

	// --- F13 income (artisan) ---
	authed.GET("/income/baseline", w(unary(h.Income.GetIncomeBaseline, nil)))
	authed.PUT("/income/baseline", mut(unary(h.Income.SetIncomeBaseline, nil)))
	authed.GET("/income/sales", w(unary(h.Income.ListOfflineSales, func(r *http.Request, req *impactv1.ListOfflineSalesRequest) error {
		if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
			req.Limit = int32(n)
		}
		return nil
	})))
	authed.POST("/income/sales", mut(unary(h.Income.LogOfflineSale, nil)))
	authed.DELETE("/income/sales/:id", mut(unary(h.Income.DeleteOfflineSale, func(r *http.Request, req *impactv1.DeleteOfflineSaleRequest) error {
		req.Id = param(r, "id")
		return nil
	})))
	authed.GET("/income/summary", w(unary(h.Income.GetMyIncomeSummary, nil)))

	// --- F13 impact dashboard (MINISTRY / scoped CLUSTER_OFFICER) ---
	authed.GET("/impact/summary", w(unary(h.Insight.GetImpactSummary, func(r *http.Request, req *insightv1.GetImpactSummaryRequest) error {
		req.Filter = impactFilter(r)
		return nil
	})))
	authed.GET("/impact/by-group", w(unary(h.Insight.GetImpactByGroup, func(r *http.Request, req *insightv1.GetImpactByGroupRequest) error {
		req.GroupBy, req.Filter = r.URL.Query().Get("group_by"), impactFilter(r)
		return nil
	})))
	authed.GET("/impact/sales-mix", w(unary(h.Insight.GetSalesMix, func(r *http.Request, req *insightv1.GetSalesMixRequest) error {
		req.Filter = impactFilter(r)
		return nil
	})))
	authed.GET("/impact/finance-coverage", w(unary(h.Insight.GetFinanceCoverage, func(r *http.Request, req *insightv1.GetFinanceCoverageRequest) error {
		req.Filter = impactFilter(r)
		return nil
	})))
	authed.GET("/impact/literacy-funnel", w(unary(h.Insight.GetLiteracyFunnel, func(r *http.Request, req *insightv1.GetLiteracyFunnelRequest) error {
		req.Filter = impactFilter(r)
		return nil
	})))
	authed.GET("/impact/export.csv", w(h.exportImpactCSV))

	// --- F14 staff accounts and assisted mode ---
	authed.GET("/staff/me", w(unary(h.Staff.GetMyStaffAccount, nil)))
	authed.GET("/admin/staff", w(unary(h.Staff.ListStaff, func(r *http.Request, req *assistedv1.ListStaffRequest) error {
		req.StateCode = optQuery(r, "state_code")
		return nil
	})))
	authed.POST("/admin/staff", mut(unary(h.Staff.CreateStaff, nil)))
	authed.POST("/admin/staff/:id/active", mut(unary(h.Staff.SetStaffActive, func(r *http.Request, req *assistedv1.SetStaffActiveRequest) error {
		req.Id = param(r, "id")
		return nil
	})))
	authed.GET("/admin/staff/productivity", w(unary(h.Staff.ListAgentProductivity, func(r *http.Request, req *assistedv1.ListAgentProductivityRequest) error {
		req.StateCode, req.District = optQuery(r, "state_code"), optQuery(r, "district")
		return nil
	})))
	authed.POST("/assisted/consent/start", mut(unary(h.Assisted.StartArtisanConsent, nil)))
	authed.POST("/assisted/link", mut(registrationLanguages(unary(h.Assisted.LinkArtisan, nil))))
	authed.POST("/assisted/voice-consent", mut(unary(h.Assisted.AttachVoiceConsent, nil)))
	authed.GET("/assisted/artisans", w(unary(h.Assisted.ListMyArtisans, nil)))
	authed.GET("/assisted/review", w(unary(h.Assisted.ListLinksForReview, func(r *http.Request, req *assistedv1.ListLinksForReviewRequest) error {
		req.StateCode, req.District = optQuery(r, "state_code"), optQuery(r, "district")
		return nil
	})))
	authed.POST("/assisted/review/:id", mut(unary(h.Assisted.MarkLinkReviewed, func(r *http.Request, req *assistedv1.MarkLinkReviewedRequest) error {
		req.LinkId = param(r, "id")
		return nil
	})))
	// A time-limited download link, so an officer can listen to a recorded
	// voice consent before marking it reviewed. core-svc's GetMediaURL only
	// serves a not-yet-servable file to its owner or to officers/ministry.
	authed.GET("/media/:id/url", w(unary(h.Media.GetMediaURL, func(r *http.Request, req *catalogv1.GetMediaURLRequest) error {
		req.MediaId = param(r, "id")
		return nil
	})))
	authed.GET("/helpers", w(unary(h.Assisted.ListMyHelpers, nil)))
	authed.DELETE("/helpers/:id", mut(unary(h.Assisted.RevokeHelper, func(r *http.Request, req *assistedv1.RevokeHelperRequest) error {
		req.LinkId = param(r, "id")
		return nil
	})))
	authed.GET("/listings/:id/helper", w(unary(h.Assisted.GetListingHelper, func(r *http.Request, req *assistedv1.GetListingHelperRequest) error {
		req.ListingId = param(r, "id")
		return nil
	})))

	// --- F15 digital literacy ---
	authed.GET("/learn/lessons", w(unary(h.Literacy.ListLessons, nil)))
	authed.POST("/learn/lessons/:code/progress", mut(unary(h.Literacy.RecordLessonProgress, func(r *http.Request, req *literacyv1.RecordLessonProgressRequest) error {
		req.LessonCode = param(r, "code")
		return nil
	})))
	authed.GET("/learn/certificate", w(unary(h.Insight.GetLiteracyCertificate, func(r *http.Request, req *insightv1.GetLiteracyCertificateRequest) error {
		var err error
		req.ArtisanId, err = self(r)
		return err
	})))
	authed.POST("/learn/certificate", mut(unary(h.Insight.IssueLiteracyCertificate, func(r *http.Request, req *insightv1.IssueLiteracyCertificateRequest) error {
		var err error
		req.ArtisanId, err = self(r)
		return err
	})))
	// The only new public route: anyone holding a printed certificate can check it.
	public.GET("/verify/certificate/:short_code", w(unary(h.Insight.VerifyLiteracyCertificate, func(r *http.Request, req *insightv1.VerifyLiteracyCertificateRequest) error {
		req.ShortCode = param(r, "short_code")
		return nil
	})))
}
