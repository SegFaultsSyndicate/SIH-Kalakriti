# Kalakriti REST API Documentation

**Base URL:** `http://localhost:8000/api/v1`  
**Version:** v1  
**Last Updated:** 2026-08-28

All endpoints return JSON. Most require authentication via `Authorization: Bearer <token>` header.

---

## Authentication

### POST /auth/otp
Request one-time password for phone number.

**Auth Required:** No

**Request:**
```json
{
  "phone_e164": "+919876543210"
}
```

**Response:** 200 OK
```json
{
  "challenge_id": "01234567-89ab-cdef-0123-456789abcdef"
}
```

**Dev Mode:** When `AUTH_DEV_OTP_ENABLED=true`, OTP is always `000000`.

**Rate Limit:** 5 requests/min per IP (recommended)

---

### POST /auth/verify
Verify OTP and receive access token.

**Auth Required:** No

**Request:**
```json
{
  "challenge_id": "01234567-89ab-cdef-0123-456789abcdef",
  "phone_e164": "+919876543210",
  "code": "000000"
}
```

**Response:** 200 OK
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "01234567-89ab-cdef-0123-456789abcdef",
  "expires_in": 86400
}
```

**Errors:**
- `400` Invalid challenge ID or code
- `401` OTP expired or incorrect
- `429` Too many attempts

---

### POST /auth/refresh
Refresh an expired access token.

**Auth Required:** No (uses refresh token)

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:** 200 OK
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400
}
```

---

## Artisan Management

### POST /artisans
Register new artisan profile.

**Auth Required:** Yes

**Request:**
```json
{
  "full_name": "Sunita Devi",
  "craft": "madhubani",
  "village": "Jitwarpur",
  "district": "Madhubani",
  "state": "Bihar",
  "pincode": "847226",
  "language": "hi"
}
```

**Response:** 201 Created
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "full_name": "Sunita Devi",
  "craft": "madhubani",
  "village": "Jitwarpur",
  "district": "Madhubani",
  "state": "Bihar",
  "pincode": "847226",
  "language": "hi",
  "created_at": "2026-08-28T12:00:00Z"
}
```

---

### GET /artisans/{id}
Get artisan profile details.

**Auth Required:** Yes

**Response:** 200 OK
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "full_name": "Sunita Devi",
  "craft": "madhubani",
  "village": "Jitwarpur",
  "district": "Madhubani",
  "state": "Bihar",
  "bio": "Traditional Madhubani painter...",
  "listings_count": 12,
  "followers_count": 245,
  "created_at": "2026-08-28T12:00:00Z"
}
```

---

### PATCH /artisans/{id}
Update artisan profile.

**Auth Required:** Yes (must be profile owner)

**Request:**
```json
{
  "bio": "Award-winning Madhubani artist with 20 years experience",
  "profile_image_url": "https://..."
}
```

**Response:** 200 OK
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "bio": "Award-winning Madhubani artist with 20 years experience",
  "updated_at": "2026-08-28T12:30:00Z"
}
```

---

## Media Upload

Media upload is a 3-step process: request URL → upload file → confirm.

### POST /media/upload-url
Request presigned upload URL.

**Auth Required:** Yes

**Request:**
```json
{
  "artisan_id": "01234567-89ab-cdef-0123-456789abcdef",
  "content_type": "image/jpeg",
  "size_bytes": 524288
}
```

**Response:** 200 OK
```json
{
  "media_id": "fedcba98-7654-3210-fedc-ba9876543210",
  "upload_url": "http://localhost:9000/kalakriti-media/media/...",
  "expires_at": "2026-08-28T13:00:00Z"
}
```

**Step 2:** PUT file to `upload_url` (not through BFF):
```bash
curl -X PUT --upload-file photo.jpg \
  -H "Content-Type: image/jpeg" \
  "http://localhost:9000/kalakriti-media/..."
```

---

### POST /media/{id}/confirm
Confirm upload completed, trigger ML pipeline.

**Auth Required:** Yes

**Response:** 200 OK
```json
{
  "media_id": "fedcba98-7654-3210-fedc-ba9876543210",
  "status": "uploaded",
  "pipeline_triggered": true
}
```

**Side Effect:** ML pipeline runs asynchronously (~5s in mock mode), creates draft listing.

---

## Listings

### POST /listings
Create listing manually (alternative to ML pipeline).

**Auth Required:** Yes

**Request:**
```json
{
  "artisan_id": "01234567-89ab-cdef-0123-456789abcdef",
  "title": "Madhubani Fish Painting",
  "description": "Traditional fish motif in natural dyes",
  "craft": "madhubani",
  "price_paise": 250000,
  "dimensions": {"width_cm": 30, "height_cm": 40},
  "media_ids": ["fedcba98-7654-3210-fedc-ba9876543210"]
}
```

**Response:** 201 Created
```json
{
  "id": "listing-uuid",
  "status": "draft",
  "title": "Madhubani Fish Painting",
  "created_at": "2026-08-28T12:35:00Z"
}
```

---

### GET /listings/{id}
Get listing details.

**Auth Required:** No (public listings), Yes (draft listings)

**Response:** 200 OK
```json
{
  "id": "listing-uuid",
  "artisan_id": "artisan-uuid",
  "artisan_name": "Sunita Devi",
  "title": "Madhubani Fish Painting",
  "description": "Traditional fish motif...",
  "craft": "madhubani",
  "price_paise": 250000,
  "price_display": "₹2,500",
  "status": "published",
  "media": [
    {
      "id": "media-uuid",
      "url": "https://...",
      "thumbnail_url": "https://..."
    }
  ],
  "created_at": "2026-08-28T12:00:00Z",
  "published_at": "2026-08-28T12:10:00Z"
}
```

---

### GET /listings
List/filter listings.

**Auth Required:** Optional (see more if authenticated)

**Query Params:**
- `status` — draft | pending_review | published | sold
- `craft` — madhubani | warli | kalamkari | ...
- `artisan_id` — filter by artisan UUID
- `min_price_paise` — minimum price
- `max_price_paise` — maximum price
- `limit` — results per page (default 20, max 100)
- `offset` — pagination offset

**Example:** `GET /listings?status=published&craft=madhubani&limit=10`

**Response:** 200 OK
```json
{
  "listings": [
    {
      "id": "listing-uuid",
      "title": "Madhubani Fish Painting",
      "price_paise": 250000,
      "thumbnail_url": "https://...",
      "artisan_name": "Sunita Devi"
    }
  ],
  "total": 47,
  "limit": 10,
  "offset": 0
}
```

---

### PATCH /listings/{id}
Update draft listing.

**Auth Required:** Yes (must be listing owner)

**Request:**
```json
{
  "title": "Updated Title",
  "price_paise": 300000
}
```

**Response:** 200 OK

---

### POST /listings/{id}/submit
Submit listing for review.

**Auth Required:** Yes (must be listing owner)

**Response:** 200 OK
```json
{
  "id": "listing-uuid",
  "status": "pending_review",
  "submitted_at": "2026-08-28T12:40:00Z"
}
```

---

### POST /listings/{id}/approve
Approve listing (admin only).

**Auth Required:** Yes (admin role)

**Response:** 200 OK
```json
{
  "id": "listing-uuid",
  "status": "published",
  "published_at": "2026-08-28T12:45:00Z"
}
```

---

## Search

### GET /search
Hybrid search (BM25 + vector similarity).

**Auth Required:** Optional

**Query Params:**
- `q` — search query (English/Hindi/Hinglish)
- `craft` — filter by craft type
- `min_price_paise` — minimum price
- `max_price_paise` — maximum price
- `limit` — results per page (default 20, max 50)

**Example:** `GET /search?q=fish+painting&craft=madhubani`

**Response:** 200 OK
```json
{
  "query": "fish painting",
  "results": [
    {
      "id": "listing-uuid",
      "title": "Madhubani Fish Painting",
      "score": 0.89,
      "price_paise": 250000,
      "thumbnail_url": "https://...",
      "artisan_name": "Sunita Devi",
      "craft": "madhubani"
    }
  ],
  "total": 12,
  "took_ms": 45
}
```

**Search Supports:**
- English: "fish painting"
- Hindi: "मछली चित्र"
- Hinglish: "fish ka painting"
- Transliterated: "machli painting"

---

### GET /suggest
Autocomplete suggestions.

**Auth Required:** No

**Query Params:**
- `q` — prefix (min 2 chars)

**Example:** `GET /suggest?q=madh`

**Response:** 200 OK
```json
{
  "suggestions": ["madhubani", "madhubani fish", "madhubani peacock"]
}
```

---

### POST /search/voice
Voice search (audio → text → results).

**Auth Required:** Yes

**Request:** Multipart form with `audio` field (WAV/MP3/OGG)

**Response:** 200 OK
```json
{
  "transcription": "madhubani fish painting",
  "language": "hi-IN",
  "results": [...]
}
```

---

## Pricing

### POST /pricing/advise
Get ML-powered pricing advice.

**Auth Required:** Yes

**Request:**
```json
{
  "craft": "madhubani",
  "dimensions": {"width_cm": 30, "height_cm": 40},
  "complexity": "medium",
  "region": "Bihar"
}
```

**Response:** 200 OK
```json
{
  "suggested_price_paise": 250000,
  "price_range": {
    "min_paise": 200000,
    "max_paise": 300000
  },
  "comparable_listings": 15,
  "confidence": 0.85
}
```

---

## Bulk Orders

### POST /orders
Create bulk order (100+ units).

**Auth Required:** Yes (buyer account)

**Request:**
```json
{
  "buyer_id": "buyer-uuid",
  "lots": [
    {
      "craft": "madhubani",
      "quantity": 150,
      "unit_price_paise": 200000,
      "deadline_days": 45
    },
    {
      "craft": "warli",
      "quantity": 200,
      "unit_price_paise": 180000,
      "deadline_days": 45
    }
  ]
}
```

**Response:** 201 Created
```json
{
  "order_id": "order-uuid",
  "status": "matching",
  "total_quantity": 350,
  "total_paise": 66000000,
  "created_at": "2026-08-28T13:00:00Z"
}
```

**Side Effect:** Matching algorithm runs, artisans receive lot invitations.

---

### GET /orders/{id}
Get order status and allocation details.

**Auth Required:** Yes (buyer or assigned artisan)

**Response:** 200 OK
```json
{
  "id": "order-uuid",
  "status": "in_production",
  "lots": [
    {
      "id": "lot-uuid",
      "craft": "madhubani",
      "quantity": 150,
      "allocated_to": [
        {
          "artisan_id": "artisan-1-uuid",
          "artisan_name": "Sunita Devi",
          "quantity": 50,
          "status": "accepted",
          "deadline": "2026-10-12T00:00:00Z"
        },
        {
          "artisan_id": "artisan-2-uuid",
          "artisan_name": "Ravi Kumar",
          "quantity": 50,
          "status": "in_production",
          "deadline": "2026-10-12T00:00:00Z"
        },
        {
          "artisan_id": "artisan-3-uuid",
          "artisan_name": "Meera Singh",
          "quantity": 50,
          "status": "accepted",
          "deadline": "2026-10-12T00:00:00Z"
        }
      ]
    }
  ],
  "created_at": "2026-08-28T13:00:00Z"
}
```

---

### POST /orders/{order_id}/lots/{lot_id}/respond
Artisan responds to lot invitation.

**Auth Required:** Yes (invited artisan)

**Request:**
```json
{
  "response": "accept",
  "message": "I can deliver 50 units by the deadline"
}
```

**Values:** `accept` | `decline` | `counter`

**Response:** 200 OK

---

### GET /orders/{id}/watch
Server-sent events stream for order updates.

**Auth Required:** Yes

**Response:** `text/event-stream`
```
event: allocation_accepted
data: {"lot_id": "...", "artisan_id": "...", "timestamp": "..."}

event: production_started
data: {"lot_id": "...", "artisan_id": "..."}

event: qc_passed
data: {"lot_id": "...", "units": 50}
```

**Events:** allocation_accepted | production_started | qc_passed | qc_failed | shipped | delivered

---

## Social

### POST /follow/{artisan_id}
Follow an artisan.

**Auth Required:** Yes

**Response:** 200 OK

---

### DELETE /follow/{artisan_id}
Unfollow an artisan.

**Auth Required:** Yes

**Response:** 204 No Content

---

### GET /feed
Get personalized feed (followed artisans' new listings).

**Auth Required:** Yes

**Query Params:**
- `limit` — results per page (default 20)
- `offset` — pagination offset

**Response:** 200 OK
```json
{
  "items": [
    {
      "type": "new_listing",
      "artisan_id": "artisan-uuid",
      "artisan_name": "Sunita Devi",
      "listing_id": "listing-uuid",
      "listing_title": "New Madhubani Fish Painting",
      "thumbnail_url": "https://...",
      "created_at": "2026-08-28T12:00:00Z"
    }
  ],
  "total": 15,
  "limit": 20,
  "offset": 0
}
```

---

## Income Statements

### POST /statements
Generate income statement PDF for artisan.

**Auth Required:** Yes (artisan or admin)

**Request:**
```json
{
  "artisan_id": "artisan-uuid",
  "year": 2026,
  "month": 7
}
```

**Response:** 200 OK
```json
{
  "pdf_url": "https://s3.../statements/abc123.pdf",
  "qr_code": "XYZ123ABC456",
  "gross_paise": 12500000,
  "fees_paise": 312500,
  "net_paise": 12187500,
  "order_count": 15,
  "signature": "base64..."
}
```

**PDF Contents:**
- Artisan details
- Month/year
- Order breakdown
- Gross earnings
- Platform fees (2.5%)
- Net payout
- Ed25519 signature
- QR code for verification

---

### GET /statements/{code}/verify
Verify statement authenticity via QR code.

**Auth Required:** No (public verification)

**Response:** 200 OK
```json
{
  "valid": true,
  "artisan_name": "Sunita Devi",
  "year": 2026,
  "month": 7,
  "net_paise": 12187500,
  "verified_at": "2026-08-28T13:30:00Z"
}
```

**Response (invalid):** 404 Not Found
```json
{
  "valid": false
}
```

---

## Analytics & Insights

### GET /insights/earnings-by-district
Get earnings aggregated by district.

**Auth Required:** Yes (admin or researcher)

**Query Params:**
- `state` — filter by state
- `year` — filter by year
- `month` — filter by month

**Response:** 200 OK
```json
{
  "districts": [
    {
      "district": "Madhubani",
      "state": "Bihar",
      "artisan_count": 45,
      "total_earnings_paise": 125000000,
      "avg_earnings_paise": 2777777
    }
  ]
}
```

---

### GET /insights/income-comparison
Compare income across crafts/regions.

**Auth Required:** Yes (admin or researcher)

**Query Params:**
- `craft` — compare by craft type
- `region` — compare by region
- `period` — month | quarter | year

**Response:** 200 OK
```json
{
  "comparison": [
    {
      "craft": "madhubani",
      "avg_income_paise": 3500000,
      "artisan_count": 120
    },
    {
      "craft": "warli",
      "avg_income_paise": 2800000,
      "artisan_count": 85
    }
  ]
}
```

---

### GET /insights/dying-crafts
Identify crafts with declining artisan participation.

**Auth Required:** Yes (admin or researcher)

**Query Params:**
- `limit` — number of crafts to return (default 10)

**Response:** 200 OK
```json
{
  "crafts": [
    {
      "craft": "chikankari",
      "artisan_count": 12,
      "trend": "declining",
      "yoy_change_pct": -15.5,
      "avg_age": 58
    }
  ]
}
```

---

## Provenance Verification

### GET /v/{code}
Public provenance verification page (HTML or JSON).

**Auth Required:** No

**HTML Response:** Server-rendered page with:
- Product photo
- Artisan details
- Craft lineage
- QR verification status
- Purchase history (anonymized)

**JSON Response:** (if `Accept: application/json`)
```json
{
  "code": "ABC123XYZ",
  "product_id": "listing-uuid",
  "artisan_id": "artisan-uuid",
  "artisan_name": "Sunita Devi",
  "craft": "madhubani",
  "signature_valid": true,
  "created_at": "2026-08-28T12:00:00Z"
}
```

---

## SEO & Static Pages

### GET /listing/{slug}
SEO-optimized listing page (server-rendered HTML).

**Auth Required:** No

**Example:** `/listing/madhubani-fish-painting-sunita-devi`

**Response:** HTML with OpenGraph tags for social sharing.

---

### GET /artisan/{slug}
SEO-optimized artisan profile page.

**Auth Required:** No

**Example:** `/artisan/sunita-devi-madhubani`

---

### GET /sitemap.xml
XML sitemap for search engines.

**Auth Required:** No

**Response:** XML with all published listings + artisan profiles.

---

### GET /robots.txt
Robots exclusion file.

**Auth Required:** No

---

## Error Responses

All errors follow this format:

```json
{
  "error": "human-readable error message",
  "code": "ERROR_CODE",
  "request_id": "req-uuid",
  "timestamp": "2026-08-28T13:45:00Z"
}
```

**Common Status Codes:**
- `400 Bad Request` — Invalid input
- `401 Unauthorized` — Missing or invalid auth token
- `403 Forbidden` — Insufficient permissions
- `404 Not Found` — Resource doesn't exist
- `409 Conflict` — Duplicate or conflicting operation
- `422 Unprocessable Entity` — Validation failed
- `429 Too Many Requests` — Rate limit exceeded
- `500 Internal Server Error` — Server error
- `503 Service Unavailable` — Downstream service down

---

## Idempotency

All `POST`, `PATCH`, `DELETE` endpoints support idempotency.

**Header:** `Idempotency-Key: <uuid>`

Same key within 24 hours returns cached response (status + body).

**Example:**
```bash
curl -X POST http://localhost:8000/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: 01234567-89ab-cdef-0123-456789abcdef" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

---

## Rate Limiting

**Global Default:** 100 requests/min per IP, 200 requests/min per authenticated user

**Per-Endpoint (recommended):**
- `/auth/otp` — 5/min
- `/auth/verify` — 10/min
- `/media/upload-url` — 20/min
- `/search` — 50/min
- `/orders` — 10/min

**Headers in Response:**
- `X-RateLimit-Limit` — requests allowed in window
- `X-RateLimit-Remaining` — requests remaining
- `X-RateLimit-Reset` — Unix timestamp when limit resets

**On Limit Exceeded:** 429 with `Retry-After` header (seconds).

---

## OpenAPI Spec

**Coming Soon:** `/api/v1/openapi.json`

---

## Testing

**Health Check:**
```bash
curl http://localhost:8000/healthz
# {"status": "ok"}
```

**Test Auth Flow:**
```bash
# 1. Request OTP
curl -X POST http://localhost:8000/api/v1/auth/otp \
  -H "Content-Type: application/json" \
  -d '{"phone_e164": "+919876543210"}'

# 2. Verify (dev mode OTP: 000000)
curl -X POST http://localhost:8000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "challenge_id": "<from-step-1>",
    "phone_e164": "+919876543210",
    "code": "000000"
  }'

# 3. Use token
export TOKEN="<access_token-from-step-2>"
curl http://localhost:8000/api/v1/listings \
  -H "Authorization: Bearer $TOKEN"
```

---

## Support

**Base URL (local):** http://localhost:8000/api/v1  
**Base URL (production):** TBD

**Issues:** Report to backend team  
**Rate Limit Increase:** Contact admin

**Implementation:** `services/bff/internal/bff/handler/*.go`
