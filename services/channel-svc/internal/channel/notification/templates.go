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
	CompanyInterestReceived NotificationKind = "COMPANY_INTEREST_RECEIVED"
	CompanyInterestAccepted NotificationKind = "COMPANY_INTEREST_ACCEPTED"
	BoutiqueMatchFound      NotificationKind = "BOUTIQUE_MATCH_FOUND"
	CompanyRegistered       NotificationKind = "COMPANY_REGISTERED"
	CompanyVerified         NotificationKind = "COMPANY_VERIFIED"
	CompanyRejected         NotificationKind = "COMPANY_REJECTED"
	CompanySaleSettled      NotificationKind = "COMPANY_SALE_SETTLED"
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
		CompanyInterestReceived: {
			English: {Title: "New Business Inquiry", Body: "{company_name} ({company_type}) wants to work with you. Check your leads."},
			Hindi:   {Title: "नई व्यापारिक पूछताछ", Body: "{company_name} ({company_type}) आपके साथ काम करना चाहता है। अपने लीड्स देखें।"},
		},
		CompanyInterestAccepted: {
			English: {Title: "Interest Accepted", Body: "{artisan_name} has accepted your interest. You can now start a partnership."},
			Hindi:   {Title: "रुचि स्वीकार", Body: "{artisan_name} ने आपकी रुचि स्वीकार कर ली है। अब आप साझेदारी शुरू कर सकते हैं।"},
		},
		BoutiqueMatchFound: {
			English: {Title: "Boutique Match", Body: "We found a boutique match: {boutique_name} in {city}. Score: {score}."},
			Hindi:   {Title: "बुटीक मैच", Body: "हमने एक बुटीक मैच पाया: {boutique_name}, {city} में। स्कोर: {score}।"},
		},
		CompanyRegistered: {
			English: {Title: "Registration Received", Body: "Namaste {contact_name}, your application for {company_name} ({company_type}) has been submitted for Ministry verification."},
			Hindi:   {Title: "पंजीकरण प्राप्त हुआ", Body: "नमस्ते {contact_name}, {company_name} ({company_type}) के लिए आपका आवेदन मंत्रालय सत्यापन हेतु जमा कर दिया गया है।"},
		},
		CompanyVerified: {
			English: {Title: "Company Verified", Body: "Congratulations {contact_name}! {company_name} has been verified on Kalakriti. Commission rate: {rate}%. You may now connect with master artisans."},
			Hindi:   {Title: "कंपनी सत्यापित", Body: "बधाई हो {contact_name}! {company_name} कलाकृति पर सत्यापित हो गया है। कमीशन दर: {rate}%। अब आप शिल्पकारों से जुड़ सकते हैं।"},
		},
		CompanyRejected: {
			English: {Title: "Application Update", Body: "Namaste {contact_name}, your application for {company_name} was not approved. Reason: {reason}."},
			Hindi:   {Title: "आवेदन स्थिति", Body: "नमस्ते {contact_name}, {company_name} के लिए आपका आवेदन स्वीकृत नहीं हुआ। कारण: {reason}।"},
		},
		CompanySaleSettled: {
			English: {Title: "Sale Settled", Body: "Order #{order_id} fulfilled for {product_name}! Gross: Rs {gross_paise}. Platform fee ({rate}%): Rs {fee_paise}. Net: Rs {net_paise}."},
			Hindi:   {Title: "बिक्री निपटान", Body: "{product_name} के लिए ऑर्डर #{order_id} पूरा हुआ! कुल: रु {gross_paise}। प्लेटफॉर्म शुल्क ({rate}%): रु {fee_paise}। शुद्ध: रु {net_paise}।"},
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
