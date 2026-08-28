package domain

import "time"

type ArtisanCategoryRow struct {
	StateCode      string
	District       string
	SocialCategory string
	ArtisanCount   int32
	VerifiedCount  int32
}

type ListingCraftMonthRow struct {
	CraftID       string
	CraftName     string
	Month         time.Time
	ListingCount  int32
	ArtisanCount  int32
}

type EarningsDistrictRow struct {
	StateCode     string
	District      string
	TotalGMVPaise int64
	TotalNetPaise int64
	ArtisanCount  int32
	AvgEarnings   int64
}

type IncomeComparisonRow struct {
	StateCode        string
	District         string
	MedianBeforePaise int64
	MedianAfterPaise  int64
	ArtisanCount     int32
}

type DyingCraftRow struct {
	CraftID         string
	CraftName       string
	DeclineRate     float32
	PeakArtisans    int32
	CurrentArtisans int32
}
