// pkg/topics/topics.go

// Package topics holds the canonical Kafka topic names for the platform.
// Producers and consumers must reference these constants, never string literals.
package topics

// Topic names. Each maps to exactly one envelope message in events/v1/events.proto.
const (
	// MediaUploaded carries events.v1.MediaUploaded.
	MediaUploaded = "media.uploaded"
	// MediaEnhanced carries events.v1.MediaEnhanced.
	MediaEnhanced = "media.enhanced"

	// CatalogAttributesExtracted carries events.v1.CatalogAttributesExtracted.
	CatalogAttributesExtracted = "catalog.attributes.extracted"
	// CatalogListingDrafted carries events.v1.CatalogListingDrafted.
	CatalogListingDrafted = "catalog.listing.drafted"
	// CatalogListingPublished carries events.v1.CatalogListingPublished.
	CatalogListingPublished = "catalog.listing.published"
	// CatalogProvenanceSealed carries events.v1.CatalogProvenanceSealed.
	CatalogProvenanceSealed = "catalog.provenance.sealed"

	// SearchIndexRequested carries events.v1.SearchIndexRequested.
	SearchIndexRequested = "search.index.requested"

	// OrderBulkRequested carries events.v1.OrderBulkRequested.
	OrderBulkRequested = "order.bulk.requested"
	// OrderLotOffered carries events.v1.OrderLotOffered.
	OrderLotOffered = "order.lot.offered"
	// OrderLotAccepted carries events.v1.OrderLotAccepted.
	OrderLotAccepted = "order.lot.accepted"
	// OrderLotDeclined carries events.v1.OrderLotDeclined.
	OrderLotDeclined = "order.lot.declined"
	// OrderLotExpired carries events.v1.OrderLotExpired.
	OrderLotExpired = "order.lot.expired"
	// OrderLotProgressed carries events.v1.OrderLotProgressed.
	OrderLotProgressed = "order.lot.progressed"
	// OrderLotCompleted carries events.v1.OrderLotCompleted.
	OrderLotCompleted = "order.lot.completed"
	// OrderFulfilmentCompleted carries events.v1.OrderFulfilmentCompleted.
	OrderFulfilmentCompleted = "order.fulfilment.completed"
	// OrderBulkCancelled carries events.v1.OrderBulkCancelled.
	OrderBulkCancelled = "order.bulk.cancelled"
	// OrderAmendmentProposed carries a buyer amendment proposal (batch 13:
	// reduced quantity or an extended deadline) raised when accepted lots
	// cannot cover an order's quantity.
	OrderAmendmentProposed = "order.amendment.proposed"
	// OrderAmendmentDecided carries the buyer's accept/decline of an
	// amendment proposal.
	OrderAmendmentDecided = "order.amendment.decided"

	// PaymentSplitRequested carries events.v1.PaymentSplitRequested.
	PaymentSplitRequested = "payment.split.requested"
	// PaymentSettled carries events.v1.PaymentSettled.
	PaymentSettled = "payment.settled"
	// EscrowMilestoneReleased carries events.v1.EscrowMilestoneReleased.
	EscrowMilestoneReleased = "escrow.milestone.released"

	// ArtisanRegistered carries events.v1.ArtisanRegistered.
	ArtisanRegistered = "artisan.registered"
	// ArtisanFollowed carries events.v1.ArtisanFollowed.
	ArtisanFollowed = "artisan.followed"

	// DisputeRaised carries events.v1.DisputeRaised.
	DisputeRaised = "dispute.raised"
	// DisputeResolved carries events.v1.DisputeResolved.
	DisputeResolved = "dispute.resolved"

	// ShipmentDispatched carries events.v1.ShipmentDispatched.
	ShipmentDispatched = "shipment.dispatched"
	// ShipmentDelivered carries events.v1.ShipmentDelivered.
	ShipmentDelivered = "shipment.delivered"

	// CompanyInterestExpressed is emitted when a company expresses interest in an artisan.
	// CompanyRegistered is emitted when a new company submits registration.
	CompanyRegistered = "company.registered"
	// CompanyVerified is emitted when an admin approves a company's registration.
	CompanyVerified = "company.verified"
	// CompanyRejected is emitted when an admin rejects a company's registration.
	CompanyRejected = "company.rejected"
	// CompanySaleSettled is emitted when a purchase of a company product has platform fee deducted.
	CompanySaleSettled = "company.sale.settled"
	// CompanyInterestExpressed is emitted when a company reaches out to an artisan.
	CompanyInterestExpressed = "company.interest.expressed"
	// CompanyInterestAccepted is emitted when an artisan accepts a company's interest.
	CompanyInterestAccepted = "company.interest.accepted"
	// SupplyPartnershipCreated is emitted when a supply partnership is established.
	SupplyPartnershipCreated = "supply.partnership.created"
	// BoutiqueMatchFound is emitted when a new boutique-artisan match is computed.
	BoutiqueMatchFound = "boutique.match.found"
)

// DLQSuffix is appended to a topic name to form its dead-letter topic.
const DLQSuffix = ".dlq"

// All lists every topic the platform produces, in dependency order.
var All = []string{
	MediaUploaded,
	MediaEnhanced,
	CatalogAttributesExtracted,
	CatalogListingDrafted,
	CatalogListingPublished,
	CatalogProvenanceSealed,
	SearchIndexRequested,
	OrderBulkRequested,
	OrderLotOffered,
	OrderLotAccepted,
	OrderLotDeclined,
	OrderLotExpired,
	OrderLotProgressed,
	OrderLotCompleted,
	OrderFulfilmentCompleted,
	OrderBulkCancelled,
	OrderAmendmentProposed,
	OrderAmendmentDecided,
	PaymentSplitRequested,
	PaymentSettled,
	EscrowMilestoneReleased,
	ArtisanRegistered,
	ArtisanFollowed,
	DisputeRaised,
	DisputeResolved,
	ShipmentDispatched,
	ShipmentDelivered,
	CompanyRegistered,
	CompanyVerified,
	CompanyRejected,
	CompanySaleSettled,
	CompanyInterestExpressed,
	CompanyInterestAccepted,
	SupplyPartnershipCreated,
	BoutiqueMatchFound,
}

// DLQ returns the dead-letter topic for the given topic.
func DLQ(topic string) string { return topic + DLQSuffix }

// IsKnown reports whether the name is a topic this platform produces.
func IsKnown(topic string) bool {
	for _, t := range All {
		if t == topic {
			return true
		}
	}
	return false
}
