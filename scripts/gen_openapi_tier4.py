#!/usr/bin/env python3
"""Adds the MoSJE tier-4 routes (services/bff/internal/bff/mosje/routes.go) to
services/bff/openapi.json, with schemas generated from the protos those routes
render. bff's mosje package serialises proto responses generically (snake_case
names, int64 as numbers, unset optional fields omitted), so the proto is the
source of truth and this script keeps the spec in step with it.

Re-run after changing a tier-4 proto or route, then `pnpm --filter
@kalakriti/api api:gen`. Idempotent: it replaces what it added last time.
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SPEC = ROOT / "services/bff/openapi.json"
PROTOS = ["finance/v1/finance.proto", "impact/v1/impact.proto", "assisted/v1/assisted.proto",
          "literacy/v1/literacy.proto", "insight/v1/insight.proto", "catalog/v1/media.proto"]

SCALARS = {"string": {"type": "string"}, "bool": {"type": "boolean"}, "double": {"type": "number"},
           "float": {"type": "number"}, "int32": {"type": "integer"}, "int64": {"type": "integer"},
           "uint32": {"type": "integer"}, "uint64": {"type": "integer"},
           "google.protobuf.Timestamp": {"type": "string", "format": "date-time"}}

BRACKETS = ["LT_3K", "B3K_6K", "B6K_10K", "B10K_15K", "GT_15K", "PREFER_NOT_TO_SAY"]
CORPS = ["NSFDC", "NBCFDC", "NSKFDC", "NDFDC", "PM_DAKSH", "PM_AJAY", "OTHER"]
ENUMS = {
    "FinanceLink.corporation": CORPS, "LinkFinanceRequest.corporation": CORPS,
    "FinanceLinkForReview.corporation": CORPS,
    "FinanceLink.status": ["SELF_REPORTED", "VERIFIED", "REJECTED"],
    "FinanceLinkForReview.status": ["SELF_REPORTED", "VERIFIED", "REJECTED"],
    "RepaymentCoverage.status": ["COVERED", "ALMOST", "NOT_YET", "NO_EMI"],
    "IncomeBaseline.monthly_bracket": BRACKETS, "SetIncomeBaselineRequest.monthly_bracket": BRACKETS,
    "IncomeBaseline.fair_income_bracket": BRACKETS, "SetIncomeBaselineRequest.fair_income_bracket": BRACKETS,
    "IncomeBaseline.source": ["SELF", "AGENT"],
    "OfflineSale.channel": ["FAIR", "LOCAL_MARKET", "DIRECT", "OTHER"],
    "LogOfflineSaleRequest.channel": ["FAIR", "LOCAL_MARKET", "DIRECT", "OTHER"],
    "IncomeSummary.insufficient_data": ["", "NO_BASELINE", "TOO_NEW", "NO_SALES"],
    "StaffAccount.role": ["FIELD_AGENT", "CLUSTER_OFFICER", "MINISTRY"],
    "CreateStaffRequest.role": ["FIELD_AGENT", "CLUSTER_OFFICER", "MINISTRY"],
    "LinkArtisanRequest.consent_method": ["ARTISAN_OTP", "VOICE_RECORDING"],
    "AgentArtisan.consent_method": ["ARTISAN_OTP", "VOICE_RECORDING"],
    "Helper.consent_method": ["ARTISAN_OTP", "VOICE_RECORDING"],
    "Lesson.code": ["PHOTO", "STORY", "PRICE", "ORDERS", "PAYMENTS", "SAFETY", "WHATSAPP", "LOAN"],
}
# Request fields a caller must send (proto3 cannot say so itself).
REQUIRED = {
    "LinkFinanceRequest": ["corporation", "reference", "consent_given", "consent_version"],
    "SetIncomeBaselineRequest": ["monthly_bracket"],
    "LogOfflineSaleRequest": ["id", "channel", "amount_paise", "sold_on"],
    "ReviewFinanceLinkRequest": ["verified"],
    "CreateStaffRequest": ["phone_e164", "display_name", "role"],
    "SetStaffActiveRequest": ["active"],
    "StartArtisanConsentRequest": ["phone_e164"],
    "LinkArtisanRequest": ["phone_e164", "consent_method"],
    "AttachVoiceConsentRequest": ["artisan_id", "media_id"],
}
# The assisted-registration payload: same shape as POST /artisans' body.
EXTERNAL = {"identity.v1.RegisterArtisanRequest": {
    "type": "object", "description": "Same fields as POST /artisans (languages as short names, e.g. \"HINDI\"); phone_e164 and idempotency_key are ignored.",
    "additionalProperties": True}}

FILTER_QUERY = ["state_code", "district", "social_category", "corporation", "from_month", "to_month"]

# (method, path, request message, response message, summary, path params, query params, public)
ROUTES = [
    ("get", "/finance/links", "ListMyFinanceLinksRequest", "ListMyFinanceLinksResponse", "List the caller's linked finance-corporation loans", [], [], False),
    ("post", "/finance/links", "LinkFinanceRequest", "LinkFinanceResponse", "Link a finance-corporation loan (DPDP consent required; only the last 4 characters of the reference are kept)", [], [], False),
    ("patch", "/finance/links/{id}", "UpdateFinanceLinkRequest", "UpdateFinanceLinkResponse", "Edit loan terms (resets verification)", ["id"], [], False),
    ("delete", "/finance/links/{id}", "DeleteFinanceLinkRequest", "DeleteFinanceLinkResponse", "Withdraw consent: permanently delete a finance link (artisan's own session only)", ["id"], [], False),
    ("get", "/finance/coverage", "GetRepaymentCoverageRequest", "GetRepaymentCoverageResponse", "How far this month's income covers the caller's EMIs", [], ["month"], False),
    ("get", "/admin/finance/links", "ListFinanceLinksForReviewRequest", "ListFinanceLinksForReviewResponse", "Finance links awaiting review (MINISTRY, scoped CLUSTER_OFFICER)", [], ["status", "state_code", "district"], False),
    ("post", "/admin/finance/links/{id}/review", "ReviewFinanceLinkRequest", "ReviewFinanceLinkResponse", "Verify or reject a finance link", ["id"], [], False),
    ("get", "/income/baseline", "GetIncomeBaselineRequest", "GetIncomeBaselineResponse", "The caller's registration-time income baseline", [], [], False),
    ("put", "/income/baseline", "SetIncomeBaselineRequest", "SetIncomeBaselineResponse", "Set the caller's income baseline", [], [], False),
    ("get", "/income/sales", "ListOfflineSalesRequest", "ListOfflineSalesResponse", "The caller's logged offline sales", [], ["limit"], False),
    ("post", "/income/sales", "LogOfflineSaleRequest", "LogOfflineSaleResponse", "Log an offline sale (client-generated id; safe to retry)", [], [], False),
    ("delete", "/income/sales/{id}", "DeleteOfflineSaleRequest", "DeleteOfflineSaleResponse", "Delete a mistaken offline sale", ["id"], [], False),
    ("get", "/income/summary", "GetMyIncomeSummaryRequest", "GetMyIncomeSummaryResponse", "The caller's income picture: last 3 months, 90-day totals, uplift over baseline", [], [], False),
    ("get", "/impact/summary", "GetImpactSummaryRequest", "GetImpactSummaryResponse", "Impact KPIs for a cohort (MINISTRY, scoped CLUSTER_OFFICER; cohorts under 5 suppressed)", [], FILTER_QUERY, False),
    ("get", "/impact/by-group", "GetImpactByGroupRequest", "GetImpactByGroupResponse", "Impact by district, social category or corporation (groups under 5 suppressed)", [], ["group_by"] + FILTER_QUERY, False),
    ("get", "/impact/sales-mix", "GetSalesMixRequest", "GetSalesMixResponse", "Monthly platform vs fair vs other offline sales", [], FILTER_QUERY, False),
    ("get", "/impact/finance-coverage", "GetFinanceCoverageRequest", "GetFinanceCoverageResponse", "Finance-corporation beneficiaries on the platform", [], FILTER_QUERY, False),
    ("get", "/impact/literacy-funnel", "GetLiteracyFunnelRequest", "GetLiteracyFunnelResponse", "Digital literacy progress by district", [], FILTER_QUERY, False),
    ("get", "/staff/me", "GetMyStaffAccountRequest", "GetMyStaffAccountResponse", "The caller's staff account", [], [], False),
    ("get", "/admin/staff", "ListStaffRequest", "ListStaffResponse", "Staff accounts (MINISTRY)", [], ["state_code"], False),
    ("post", "/admin/staff", "CreateStaffRequest", "CreateStaffResponse", "Create a staff account (MINISTRY)", [], [], False),
    ("post", "/admin/staff/{id}/active", "SetStaffActiveRequest", "SetStaffActiveResponse", "Activate or deactivate a staff account (MINISTRY)", ["id"], [], False),
    ("get", "/admin/staff/productivity", "ListAgentProductivityRequest", "ListAgentProductivityResponse", "Field agent productivity (MINISTRY, scoped CLUSTER_OFFICER)", [], ["state_code", "district"], False),
    ("post", "/assisted/consent/start", "StartArtisanConsentRequest", "StartArtisanConsentResponse", "Send a consent OTP to an artisan's phone (field staff)", [], [], False),
    ("post", "/assisted/link", "LinkArtisanRequest", "LinkArtisanResponse", "Link (and if new, register) an artisan with their consent (field staff)", [], [], False),
    ("post", "/assisted/voice-consent", "AttachVoiceConsentRequest", "AttachVoiceConsentResponse", "Attach a recorded voice consent to a link (field staff)", [], [], False),
    ("get", "/assisted/artisans", "ListMyArtisansRequest", "ListMyArtisansResponse", "Artisans the calling agent helps", [], [], False),
    ("get", "/assisted/review", "ListLinksForReviewRequest", "ListLinksForReviewResponse", "Voice-consent links awaiting officer review", [], ["state_code", "district"], False),
    ("post", "/assisted/review/{id}", "MarkLinkReviewedRequest", "MarkLinkReviewedResponse", "Mark a voice-consent link reviewed", ["id"], [], False),
    ("get", "/media/{id}/url", "GetMediaURLRequest", "GetMediaURLResponse", "A time-limited download URL for a media file (e.g. a voice consent under review)", ["id"], [], False),
    ("get", "/helpers", "ListMyHelpersRequest", "ListMyHelpersResponse", "People helping the caller (artisan)", [], [], False),
    ("delete", "/helpers/{id}", "RevokeHelperRequest", "RevokeHelperResponse", "Remove a helper's access immediately (artisan's own session only)", ["id"], [], False),
    ("get", "/listings/{id}/helper", "GetListingHelperRequest", "GetListingHelperResponse", "Which agent, if any, created this listing for the caller", ["id"], [], False),
    ("get", "/learn/lessons", "ListLessonsRequest", "ListLessonsResponse", "The digital literacy track and the caller's progress", [], [], False),
    ("post", "/learn/lessons/{code}/progress", "RecordLessonProgressRequest", "RecordLessonProgressResponse", "Record practice / quiz progress on a lesson", ["code"], [], False),
    ("get", "/learn/certificate", "GetLiteracyCertificateRequest", "GetLiteracyCertificateResponse", "The caller's digital literacy certificate, if issued", [], [], False),
    ("post", "/learn/certificate", "IssueLiteracyCertificateRequest", "IssueLiteracyCertificateResponse", "Issue the caller's certificate once all 8 lessons are complete", [], [], False),
    ("get", "/verify/certificate/{short_code}", "VerifyLiteracyCertificateRequest", "VerifyLiteracyCertificateResponse", "Verify a printed digital literacy certificate (public)", ["short_code"], [], True),
]
PATH_FIELD = {"id": ["id", "link_id", "listing_id", "media_id"], "code": ["lesson_code"], "short_code": ["short_code"]}


def parse_messages():
    msgs = {}
    for rel in PROTOS:
        text = (ROOT / "proto" / rel).read_text(encoding="utf-8")
        text = re.sub(r"//[^\n]*", "", text)
        for m in re.finditer(r"message\s+(\w+)\s*\{([^{}]*)\}", text):
            fields = []
            for f in re.finditer(r"(optional\s+|repeated\s+)?([\w.]+)\s+(\w+)\s*=\s*\d+\s*;", m.group(2)):
                label = (f.group(1) or "").strip()
                if f.group(2) == "reserved":
                    continue
                fields.append((label, f.group(2), f.group(3)))
            msgs[m.group(1)] = fields
    return msgs


def schema_for(msgs, name, used, request=False, skip=()):
    props, required = {}, []
    for label, typ, field in msgs[name]:
        if field in skip:
            continue
        if typ in SCALARS:
            s = dict(SCALARS[typ])
            if f"{name}.{field}" in ENUMS:
                s["enum"] = ENUMS[f"{name}.{field}"]
        elif typ in EXTERNAL:
            s = EXTERNAL[typ]
        else:
            used.add(typ)
            s = {"$ref": f"#/components/schemas/{typ}"}
        if label == "repeated":
            s = {"type": "array", "items": s}
        props[field] = s
        if request:
            continue
        if label == "repeated" or (label != "optional" and (typ in SCALARS)):
            required.append(field)
    out = {"type": "object", "properties": props}
    if request:
        required = REQUIRED.get(name, [])
    if required:
        out["required"] = required
    return out


def main():
    spec = json.loads(SPEC.read_text(encoding="utf-8"))
    msgs = parse_messages()
    marker = "x-kalakriti-generated"
    spec["paths"] = {p: v for p, v in spec["paths"].items() if not any(isinstance(o, dict) and o.get(marker) for o in v.values())}
    spec["components"]["schemas"] = {k: v for k, v in spec["components"]["schemas"].items() if not v.get(marker)}

    used = set()
    for method, path, req, resp, summary, path_params, query, public in ROUTES:
        op = {"summary": summary, marker: "scripts/gen_openapi_tier4.py"}
        if not public:
            op["security"] = [{"bearerAuth": []}]
        params = [{"name": p, "in": "path", "required": True, "schema": {"type": "string"}} for p in path_params]
        params += [{"name": q, "in": "query", "required": False,
                    "schema": {"type": "integer"} if q == "limit" else
                    {"type": "string", "enum": ["district", "social_category", "corporation"]} if q == "group_by" else
                    {"type": "string"}} for q in query]
        if params:
            op["parameters"] = params
        if method in ("post", "put", "patch"):
            skip = {f for p in path_params for f in PATH_FIELD[p]}
            if req == "IssueLiteracyCertificateRequest":
                skip.add("artisan_id")  # bff fills it from the caller's token
            body = schema_for(msgs, req, used, request=True, skip=skip)
            if body["properties"]:
                op["requestBody"] = {"required": bool(body.get("required")),
                                     "content": {"application/json": {"schema": body}}}
        used.add(resp)
        op["responses"] = {"200": {"description": "OK", "content": {"application/json": {"schema": {"$ref": f"#/components/schemas/{resp}"}}}}}
        if not public:
            op["responses"]["403"] = {"description": "Not allowed for this role, or outside the caller's scope"}
        spec["paths"].setdefault(path, {})[method] = op

    done = set()
    while used - done:
        name = sorted(used - done)[0]
        done.add(name)
        s = schema_for(msgs, name, used)
        s[marker] = True
        spec["components"]["schemas"][name] = s

    spec["paths"]["/impact/export.csv"] = {"get": {
        "summary": "Impact by group as CSV (groups under 5 show \"<5\" and no numbers)",
        marker: "scripts/gen_openapi_tier4.py", "security": [{"bearerAuth": []}],
        "parameters": [{"name": q, "in": "query", "required": False, "schema": {"type": "string"}} for q in ["group_by"] + FILTER_QUERY],
        "responses": {"200": {"description": "CSV", "content": {"text/csv": {"schema": {"type": "string"}}}},
                      "403": {"description": "Not allowed for this role"}}}}

    # Existing badge schema: the digital_ready badge's metric.
    metric = spec["components"]["schemas"]["Badge"]["properties"]["metric"]
    if "LESSONS_COMPLETED" not in metric["enum"]:
        metric["enum"].append("LESSONS_COMPLETED")

    # X-On-Behalf-Of is documented once, as a security-adjacent header.
    spec["components"].setdefault("parameters", {})["OnBehalfOf"] = {
        "name": "X-On-Behalf-Of", "in": "header", "required": False, "schema": {"type": "string"},
        "description": "Assisted mode: a FIELD_AGENT or CLUSTER_OFFICER acting for a linked artisan (artisan id). "
                       "Honoured only on allow-listed routes (services/bff/internal/bff/mosje/onbehalf.go); "
                       "every other route answers 403 when it is present."}
    SPEC.write_text(json.dumps(spec, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")
    print(f"{len(ROUTES)} routes, {len(done)} schemas")


if __name__ == "__main__":
    main()
