// services/channel-svc/internal/channel/notification/templates.go

// Package notification handles notification rendering and delivery.
package notification

import (
	"fmt"
	"strings"
)

// NotificationKind matches the DB enum.
type NotificationKind string

const (
	LotOffered           NotificationKind = "LOT_OFFERED"
	LotExpiring          NotificationKind = "LOT_EXPIRING"
	ListingApprovalDue   NotificationKind = "LISTING_APPROVAL_DUE"
	PaymentSettled       NotificationKind = "PAYMENT_SETTLED"
	QCFailed             NotificationKind = "QC_FAILED"
	DisputeRaised        NotificationKind = "DISPUTE_RAISED"
	ShipmentDelivered    NotificationKind = "SHIPMENT_DELIVERED"
	ArtisanFollowed      NotificationKind = "ARTISAN_FOLLOWED"
)

// Language matches the DB enum.
type Language string

const (
	English Language = "ENGLISH"
	Hindi   Language = "HINDI"
	Bengali Language = "BENGALI"
	Tamil   Language = "TAMIL"
)

// Template holds title and body for a notification kind.
type Template struct {
	Title string
	Body  string
}

// Render fills template variables.
func (t Template) Render(vars map[string]string) Template {
	title := t.Title
	body := t.Body
	for k, v := range vars {
		placeholder := "{" + k + "}"
		title = strings.ReplaceAll(title, placeholder, v)
		body = strings.ReplaceAll(body, placeholder, v)
	}
	return Template{Title: title, Body: body}
}

// GetTemplate returns the template for a kind and language.
func GetTemplate(kind NotificationKind, lang Language) (Template, error) {
	templates := map[NotificationKind]map[Language]Template{
		LotOffered: {
			English: {Title: "New Lot Offer", Body: "You have a new lot offer for {product_name}. Respond by {deadline}."},
			Hindi:   {Title: "नया लॉट ऑफ़र", Body: "{product_name} के लिए आपको एक नया लॉट ऑफ़र मिला है। {deadline} तक जवाब दें।"},
		},
		LotExpiring: {
			English: {Title: "Lot Expiring Soon", Body: "Your lot offer for {product_name} expires in {hours} hours."},
			Hindi:   {Title: "लॉट जल्द समाप्त हो रहा है", Body: "{product_name} के लिए आपका लॉट ऑफ़र {hours} घंटे में समाप्त हो जाएगा।"},
		},
		ListingApprovalDue: {
			English: {Title: "Listing Pending Review", Body: "Your listing {title} is waiting for approval."},
			Hindi:   {Title: "लिस्टिंग समीक्षाधीन है", Body: "आपकी लिस्टिंग {title} अनुमोदन की प्रतीक्षा में है।"},
		},
		PaymentSettled: {
			English: {Title: "Payment Received", Body: "₹{amount} has been credited for order #{order_id}."},
			Hindi:   {Title: "भुगतान प्राप्त हुआ", Body: "ऑर्डर #{order_id} के लिए ₹{amount} जमा हो गया है।"},
		},
		QCFailed: {
			English: {Title: "Quality Check Failed", Body: "Order #{order_id} did not pass quality inspection. Reason: {reason}"},
			Hindi:   {Title: "गुणवत्ता जांच विफल", Body: "ऑर्डर #{order_id} गुणवत्ता निरीक्षण में विफल रहा। कारण: {reason}"},
		},
		DisputeRaised: {
			English: {Title: "Dispute Raised", Body: "A dispute has been raised for order #{order_id}. Review it now."},
			Hindi:   {Title: "विवाद उठाया गया", Body: "ऑर्डर #{order_id} के लिए विवाद उठाया गया है। अभी समीक्षा करें।"},
		},
		ShipmentDelivered: {
			English: {Title: "Order Delivered", Body: "Your order #{order_id} has been delivered."},
			Hindi:   {Title: "ऑर्डर डिलीवर हो गया", Body: "आपका ऑर्डर #{order_id} डिलीवर हो गया है।"},
		},
		ArtisanFollowed: {
			English: {Title: "New Follower", Body: "{follower_name} is now following your work."},
			Hindi:   {Title: "नया फॉलोअर", Body: "{follower_name} अब आपके काम को फॉलो कर रहा है।"},
		},
	}

	langMap, ok := templates[kind]
	if !ok {
		return Template{}, fmt.Errorf("no template for kind %s", kind)
	}
	tpl, ok := langMap[lang]
	if !ok {
		// Fallback to English.
		tpl = langMap[English]
	}
	return tpl, nil
}
