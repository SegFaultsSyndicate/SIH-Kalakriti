package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"

	"github.com/ZoroNewbie00/kalakriti/pkg/crypto"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/impact"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
)

// ImpactStore is the read side of the ministry impact dashboard (F13) and
// the literacy certificate store (F15).
type ImpactStore interface {
	ListImpactArtisans(ctx context.Context, f domain.ImpactFilter) ([]domain.ImpactArtisan, error)
	GetSalesMix(ctx context.Context, f domain.ImpactFilter) ([]domain.SalesMixMonth, error)
	ListFinanceCoverageFacts(ctx context.Context, f domain.ImpactFilter) ([]domain.FinanceCoverageFact, error)
	GetLiteracyFunnel(ctx context.Context, f domain.ImpactFilter) ([]domain.LiteracyFunnelRow, error)
	GetImpactViewsRefreshedAt(ctx context.Context) (*time.Time, error)

	CountCompletedLessons(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CreateLiteracyCertificate(ctx context.Context, c domain.LiteracyCertificate) (*domain.LiteracyCertificate, error)
	GetLiteracyCertificateByArtisan(ctx context.Context, artisanID uuid.UUID) (*domain.LiteracyCertificate, error)
	GetLiteracyCertificateByCode(ctx context.Context, code string) (*domain.LiteracyCertificate, error)
	GetArtisanName(ctx context.Context, artisanID uuid.UUID) (string, error)
}

// Group-by dimensions for GetImpactByGroup.
const (
	GroupDistrict       = "district"
	GroupSocialCategory = "social_category"
	GroupCorporation    = "corporation"
)

// LessonCount is the number of lessons in the literacy track; a certificate
// is issued only once all of them are complete.
const LessonCount = 8

// --- dashboard -----------------------------------------------------------------

func (s *Service) GetImpactSummary(ctx context.Context, f domain.ImpactFilter) (domain.ImpactSummary, error) {
	artisans, err := s.impact.ListImpactArtisans(ctx, f)
	if err != nil {
		return domain.ImpactSummary{}, err
	}
	funnel, err := s.impact.GetLiteracyFunnel(ctx, f)
	if err != nil {
		return domain.ImpactSummary{}, err
	}
	sum := summarize(artisans, time.Now())
	for _, r := range funnel {
		sum.CertificatesIssued += r.Certified
	}
	if sum.Suppressed {
		sum.CertificatesIssued = 0
	}
	sum.RefreshedAt, err = s.impact.GetImpactViewsRefreshedAt(ctx)
	return sum, err
}

func (s *Service) GetImpactByGroup(ctx context.Context, groupBy string, f domain.ImpactFilter) ([]domain.ImpactGroupRow, error) {
	if groupBy != GroupDistrict && groupBy != GroupSocialCategory && groupBy != GroupCorporation {
		return nil, pkgdomain.InvalidInput("group_by must be district, social_category or corporation")
	}
	artisans, err := s.impact.ListImpactArtisans(ctx, f)
	if err != nil {
		return nil, err
	}
	return groupImpact(artisans, groupBy, time.Now()), nil
}

func (s *Service) GetSalesMix(ctx context.Context, f domain.ImpactFilter) ([]domain.SalesMixMonth, error) {
	months, err := s.impact.GetSalesMix(ctx, f)
	if err != nil {
		return nil, err
	}
	for i := range months {
		if impact.Suppressed(int(months[i].ArtisanCount)) {
			months[i] = domain.SalesMixMonth{Month: months[i].Month, Suppressed: true}
		}
	}
	return months, nil
}

func (s *Service) GetFinanceCoverage(ctx context.Context, f domain.ImpactFilter) ([]domain.FinanceCoverageRow, error) {
	facts, err := s.impact.ListFinanceCoverageFacts(ctx, f)
	if err != nil {
		return nil, err
	}
	return financeCoverage(facts), nil
}

func (s *Service) GetLiteracyFunnel(ctx context.Context, f domain.ImpactFilter) ([]domain.LiteracyFunnelRow, error) {
	rows, err := s.impact.GetLiteracyFunnel(ctx, f)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if impact.Suppressed(int(rows[i].Artisans)) {
			rows[i] = domain.LiteracyFunnelRow{StateCode: rows[i].StateCode, District: rows[i].District, Suppressed: true}
		}
	}
	return rows, nil
}

// summarize computes the headline KPIs. A cohort under MinCohort is
// suppressed whole; the median uplift needs MinCohort qualifying artisans
// of its own, since it is a statistic about a (smaller) sub-cohort.
func summarize(artisans []domain.ImpactArtisan, now time.Time) domain.ImpactSummary {
	if impact.Suppressed(len(artisans)) {
		return domain.ImpactSummary{Suppressed: true}
	}
	sum := domain.ImpactSummary{Beneficiaries: int64(len(artisans))}
	var uplifts []float64
	var platform, offline int64
	for _, a := range artisans {
		if a.Platform90d+a.Offline90d > 0 {
			sum.ActiveSellers90d++
		}
		platform += a.Platform90d
		offline += a.Offline90d
		if len(a.Corporations) > 0 {
			sum.FinanceLinked++
		}
		if a.FinanceVerified {
			sum.FinanceVerified++
		}
		if pct, reason := impact.Uplift(facts(a), now); reason == "" {
			uplifts = append(uplifts, pct)
		}
	}
	sum.UpliftSample = int64(len(uplifts))
	if !impact.Suppressed(len(uplifts)) {
		m, _ := impact.Median(uplifts)
		sum.MedianUpliftPct = &m
	}
	if !impact.Suppressed(int(sum.ActiveSellers90d)) && platform+offline > 0 {
		share := float64(platform) / float64(platform+offline) * 100
		sum.DigitalSharePct = &share
	}
	return sum
}

func facts(a domain.ImpactArtisan) impact.Facts {
	return impact.Facts{
		RegisteredAt: a.RegisteredAt, BaselineBracket: a.BaselineBracket, BaselineExact: a.BaselineMonthlyPaise,
		Platform90d: a.Platform90d, Offline90d: a.Offline90d,
	}
}

// groupImpact buckets artisans by one dimension. An artisan linked to two
// corporations counts once in each corporation's row (never twice in one).
func groupImpact(artisans []domain.ImpactArtisan, groupBy string, now time.Time) []domain.ImpactGroupRow {
	type key struct{ state, group string }
	buckets := map[key][]domain.ImpactArtisan{}
	for _, a := range artisans {
		switch groupBy {
		case GroupDistrict:
			k := key{a.StateCode, a.District}
			buckets[k] = append(buckets[k], a)
		case GroupSocialCategory:
			k := key{"", a.SocialCategory}
			buckets[k] = append(buckets[k], a)
		case GroupCorporation:
			for _, c := range a.Corporations {
				k := key{"", c}
				buckets[k] = append(buckets[k], a)
			}
		}
	}
	rows := make([]domain.ImpactGroupRow, 0, len(buckets))
	for k, members := range buckets {
		row := domain.ImpactGroupRow{Group: k.group, StateCode: k.state}
		if impact.Suppressed(len(members)) {
			row.Suppressed = true
			rows = append(rows, row)
			continue
		}
		row.ArtisanCount = int64(len(members))
		var baselines, currents []int64
		var uplifts []float64
		for _, a := range members {
			if a.Platform90d+a.Offline90d > 0 {
				row.ActiveSellers90d++
			}
			row.PlatformIncomePaise += a.Platform90d
			row.OfflineIncomePaise += a.Offline90d
			row.FairIncomePaise += a.Fair90d
			// Baseline and current medians are over the same artisans (those
			// with a usable baseline), so "before vs after" compares like with like.
			if b, ok := impact.BaselineMonthlyPaise(a.BaselineBracket, a.BaselineMonthlyPaise); a.BaselineBracket != "" && ok {
				baselines = append(baselines, b)
				currents = append(currents, impact.CurrentMonthlyPaise(a.Platform90d, a.Offline90d))
			}
			if pct, reason := impact.Uplift(facts(a), now); reason == "" {
				uplifts = append(uplifts, pct)
			}
		}
		row.WithBaselineCount = int64(len(baselines))
		if !impact.Suppressed(len(baselines)) {
			row.MedianBaselinePaise, _ = impact.Median(baselines)
			row.MedianCurrentPaise, _ = impact.Median(currents)
		}
		if !impact.Suppressed(len(uplifts)) {
			m, _ := impact.Median(uplifts)
			row.MedianUpliftPct = &m
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StateCode != rows[j].StateCode {
			return rows[i].StateCode < rows[j].StateCode
		}
		return rows[i].Group < rows[j].Group
	})
	return rows
}

// financeCoverage summarises links per corporation. Coverage ratio is an
// artisan's trailing average monthly income over their total EMI to that
// corporation, only for links with an EMI on record.
func financeCoverage(facts []domain.FinanceCoverageFact) []domain.FinanceCoverageRow {
	type agg struct {
		artisans map[uuid.UUID]bool
		verified map[uuid.UUID]bool
		self     map[uuid.UUID]bool
		active   map[uuid.UUID]bool
		emi      map[uuid.UUID]int64
		income   map[uuid.UUID]int64
	}
	byCorp := map[string]*agg{}
	for _, f := range facts {
		a := byCorp[f.Corporation]
		if a == nil {
			a = &agg{map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{}, map[uuid.UUID]bool{},
				map[uuid.UUID]int64{}, map[uuid.UUID]int64{}}
			byCorp[f.Corporation] = a
		}
		a.artisans[f.ArtisanID] = true
		if f.Status == "VERIFIED" {
			a.verified[f.ArtisanID] = true
		} else {
			a.self[f.ArtisanID] = true
		}
		if f.Platform90d+f.Offline90d > 0 {
			a.active[f.ArtisanID] = true
		}
		a.emi[f.ArtisanID] += f.EmiPaise
		a.income[f.ArtisanID] = impact.CurrentMonthlyPaise(f.Platform90d, f.Offline90d)
	}
	rows := make([]domain.FinanceCoverageRow, 0, len(byCorp))
	for corp, a := range byCorp {
		row := domain.FinanceCoverageRow{Corporation: corp}
		if impact.Suppressed(len(a.artisans)) {
			row.Suppressed = true
			rows = append(rows, row)
			continue
		}
		row.Beneficiaries = int64(len(a.artisans))
		row.Verified = int64(len(a.verified))
		row.SelfReported = int64(len(a.self))
		row.ActiveSellers90d = int64(len(a.active))
		var ratios []float64
		for id, emi := range a.emi {
			if emi > 0 {
				ratios = append(ratios, float64(a.income[id])/float64(emi))
			}
		}
		if !impact.Suppressed(len(ratios)) {
			m, _ := impact.Median(ratios)
			row.MedianCoverageRatio = &m
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Corporation < rows[j].Corporation })
	return rows
}

// --- literacy certificate ----------------------------------------------------------

// IssueLiteracyCertificate issues the artisan's certificate once all lessons
// are complete. Idempotent: a second call returns the existing certificate.
func (s *Service) IssueLiteracyCertificate(ctx context.Context, artisanID uuid.UUID) (*domain.LiteracyCertificate, error) {
	if existing, err := s.impact.GetLiteracyCertificateByArtisan(ctx, artisanID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	done, err := s.impact.CountCompletedLessons(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	if done < LessonCount {
		return nil, pkgdomain.InvalidInput(fmt.Sprintf("complete all %d lessons first (%d done)", LessonCount, done))
	}
	name, err := s.impact.GetArtisanName(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	code, err := s.certCodes.Generate(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating short code: %w", err)
	}
	qrPNG, err := qrcode.Encode(s.CertificateVerifyURL(code), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generating QR code: %w", err)
	}
	issued := time.Now().UTC()
	pdfBytes, err := certificatePDF(name, code, issued, qrPNG)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(pdfBytes)
	cert := domain.LiteracyCertificate{
		ID: ids.New(), ArtisanID: artisanID, ShortCode: code,
		Signature: s.signer.Sign(hash[:]), PublicKeyID: s.signer.KeyID(),
	}
	cert.S3Key = fmt.Sprintf("literacy-certificates/%s/%s.pdf", artisanID, cert.ID)
	putURL, err := s.s3.PresignedPutURL(ctx, cert.S3Key, "application/pdf", 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("generating S3 put URL: %w", err)
	}
	if err := uploadToPresignedURL(ctx, putURL, pdfBytes, "application/pdf"); err != nil {
		return nil, fmt.Errorf("uploading certificate: %w", err)
	}
	created, err := s.impact.CreateLiteracyCertificate(ctx, cert)
	if err != nil {
		return nil, err
	}
	if created == nil { // lost a race with a concurrent issue
		return s.impact.GetLiteracyCertificateByArtisan(ctx, artisanID)
	}
	return created, nil
}

func (s *Service) GetLiteracyCertificate(ctx context.Context, artisanID uuid.UUID) (*domain.LiteracyCertificate, error) {
	return s.impact.GetLiteracyCertificateByArtisan(ctx, artisanID)
}

// VerifyLiteracyCertificate checks a certificate by short code: it must
// exist and its stored PDF must still match the signature.
func (s *Service) VerifyLiteracyCertificate(ctx context.Context, code string) (*domain.LiteracyCertificate, bool, error) {
	cert, err := s.impact.GetLiteracyCertificateByCode(ctx, code)
	if err != nil || cert == nil {
		return nil, false, err
	}
	getURL, err := s.s3.PresignedGetURL(ctx, cert.S3Key, 5*time.Minute)
	if err != nil {
		return cert, false, fmt.Errorf("generating S3 get URL: %w", err)
	}
	pdfBytes, err := downloadFromPresignedURL(ctx, getURL)
	if err != nil {
		return cert, false, fmt.Errorf("downloading certificate: %w", err)
	}
	hash := sha256.Sum256(pdfBytes)
	pub, err := hex.DecodeString(s.signer.PublicKeyHex())
	if err != nil {
		return cert, false, err
	}
	if cert.PublicKeyID != s.signer.KeyID() {
		// Signed with a key this service no longer holds (e.g. a rotated or
		// ephemeral dev key): it cannot be vouched for, so it is not valid.
		return cert, false, nil
	}
	return cert, crypto.Verify(pub, hash[:], cert.Signature), nil
}

// CertificateVerifyURL is the public page a certificate's QR code opens.
func (s *Service) CertificateVerifyURL(code string) string {
	return s.verifyBaseURL + "/verify/certificate/" + code
}

// CertificateDownloadURL presigns the certificate PDF.
func (s *Service) CertificateDownloadURL(ctx context.Context, c *domain.LiteracyCertificate) (string, error) {
	return s.s3.PresignedGetURL(ctx, c.S3Key, s.presignExpiry)
}

func certificatePDF(name, code string, issued time.Time, qrPNG []byte) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("NotoSans", "", notoSansDevanagariTTF)
	pdf.AddPage()
	pdf.SetFont("NotoSans", "", 24)
	pdf.CellFormat(0, 16, "Certificate of Digital Readiness / डिजिटल तत्परता प्रमाणपत्र", "", 1, "C", false, 0, "")
	pdf.Ln(8)
	pdf.SetFont("NotoSans", "", 14)
	pdf.CellFormat(0, 10, "This certifies that", "", 1, "C", false, 0, "")
	pdf.SetFont("NotoSans", "", 22)
	pdf.CellFormat(0, 14, name, "", 1, "C", false, 0, "")
	pdf.SetFont("NotoSans", "", 14)
	pdf.MultiCell(0, 8, "has completed all eight Kalakriti digital literacy lessons: product photos, voice stories, "+
		"fair pricing, orders, digital payments, staying safe from fraud, sharing on WhatsApp, and loans and schemes.", "", "C", false)
	pdf.Ln(6)
	pdf.CellFormat(0, 8, "Issued: "+issued.Format("2 January 2006"), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Verification code: "+code, "", 1, "C", false, 0, "")
	pdf.RegisterImageReader("qr_"+code, "PNG", bytes.NewReader(qrPNG))
	pdf.Image("qr_"+code, 128, pdf.GetY()+4, 40, 40, false, "", 0, "")
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("rendering certificate: %w", err)
	}
	return buf.Bytes(), nil
}
