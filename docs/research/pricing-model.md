# AEGIS Pricing & Monetization Model

Last updated: 2026-02-26

## Positioning

AEGIS occupies a unique position: **the only platform that combines browser-based de-identification, automated defacing, protocol compliance, QC, and BIDS conversion** in a single cloud-hosted service. Competitors force customers to choose between research tools (XNAT, Flywheel) and enterprise de-identification (ENCOG, MIRC CTP). AEGIS serves both from a shared core.

Pricing should reflect this dual-market advantage while remaining accessible to grant-funded academic labs.

---

## Tier Structure

### Tier 0: Open Source (Self-Hosted)

| | |
|---|---|
| **Price** | Free (MIT license) |
| **Target** | Technical labs with DevOps capacity, evaluators |
| **Studies** | Unlimited |
| **Projects** | Unlimited |
| **Storage** | Customer-managed |
| **Support** | Community (GitHub Issues, Discussions) |
| **Restrictions** | No cloud AI backends (heuristic-only classification/PHI detection), no SLA, no managed upgrades |

**Why offer this:** Builds adoption, generates contributions, establishes credibility in academic community. XNAT's open-source model proved this works — their hosted service (XNAT Central) converts a fraction of self-hosted users into paying customers.

---

### Tier 1: Starter — $0/month (Hosted Free Tier)

| | |
|---|---|
| **Price** | Free |
| **Target** | Individual researchers, small pilot studies, evaluators |
| **Studies** | 200/month |
| **Projects** | 1 |
| **Storage** | 25 GB |
| **Users** | 2 admin seats |
| **Pipeline** | Full automated pipeline (heuristic backends only) |
| **Export shares** | 10 active shares |
| **Support** | Community |
| **DIMSE** | Not included |
| **API keys** | 1 |

**Conversion trigger:** Storage or study limit hit → upgrade prompt with one-click plan change.

---

### Tier 2: Research — $499/month (annual) or $599/month (monthly)

| | |
|---|---|
| **Price** | $499/mo annual ($5,988/yr) or $599/mo monthly |
| **Target** | Single-site research labs, small multi-site studies |
| **Studies** | 3,000/month |
| **Projects** | 5 |
| **Storage** | 500 GB (included), +$0.10/GB/month overage |
| **Users** | 10 admin seats |
| **Pipeline** | Full automated pipeline with cloud AI backends |
| **Export shares** | Unlimited |
| **BIDS export** | Included (per-study + bulk project export) |
| **Protocol templates** | 20 per project |
| **Webhooks** | 5 subscriptions |
| **Support** | Email (2 business day response) |
| **DIMSE** | Not included |
| **API keys** | 5 |
| **Add-ons** | DIMSE receiver (+$200/mo), priority support (+$150/mo) |

**Why $499:** Fits within a typical NIH R01 "other direct costs" budget line. Annual pricing generates a quote letter for grant applications. Competitive with Flywheel's entry tier (~$500–800/mo estimated).

---

### Tier 3: Consortium — $1,999/month (annual) or $2,499/month (monthly)

| | |
|---|---|
| **Price** | $1,999/mo annual ($23,988/yr) or $2,499/mo monthly |
| **Target** | Multi-site research consortia, CROs, academic medical centers |
| **Studies** | 15,000/month |
| **Projects** | 25 |
| **Storage** | 3 TB (included), +$0.08/GB/month overage |
| **Users** | 50 admin seats |
| **Pipeline** | Full automated pipeline with cloud AI backends |
| **Institutions** | 20 (with IP/AE title attribution) |
| **DIMSE receiver** | Included (1 endpoint) |
| **Protocol templates** | Unlimited |
| **Webhooks** | 25 subscriptions |
| **Federation peers** | 3 (when federation launches) |
| **Routing rules** | Unlimited with import/export |
| **Support** | Email (1 business day response) + quarterly review call |
| **API keys** | 20 |
| **SLA** | 99.5% uptime |

**Why $1,999:** Multi-site studies (ADNI, ENIGMA, ABCD-style) typically budget $20K–50K/yr for data infrastructure. This tier captures that budget with room for add-ons.

---

### Tier 4: Enterprise — Custom pricing (starting ~$5,000/month)

| | |
|---|---|
| **Price** | Custom (starting $5,000/mo, typical $5K–15K/mo) |
| **Target** | Hospital networks, pharma sponsors, large CROs, government agencies |
| **Studies** | Unlimited |
| **Projects** | Unlimited |
| **Storage** | Custom (multi-TB, negotiated) |
| **Users** | Unlimited admin seats |
| **Pipeline** | Full pipeline + custom sidecar integration |
| **DIMSE receiver** | Included (multiple endpoints) |
| **SSO** | IAP, Azure AD, Cognito (customer IdP) |
| **Deployment** | Shared multi-tenant or dedicated single-tenant |
| **Data residency** | Customer choice of cloud region (GCP, AWS, Azure) |
| **Support** | Dedicated account manager, 4-hour critical response |
| **SLA** | 99.9% uptime, BAA execution |
| **Compliance** | SOC 2 Type II report, HIPAA BAA, optional GDPR DPA |
| **Custom features** | HL7 FHIR notifications, custom routing integrations, branded upload portal |

---

## Usage-Based Pricing Components

In addition to tier base pricing, the following usage components apply:

| Metric | Free | Starter | Research | Consortium | Enterprise |
|--------|------|---------|----------|------------|------------|
| **Studies/month** | N/A | 200 | 3,000 | 15,000 | Unlimited |
| **Overage per study** | N/A | Blocked | $0.25 | $0.15 | Negotiated |
| **Storage included** | N/A | 25 GB | 500 GB | 3 TB | Custom |
| **Storage overage** | N/A | Blocked | $0.10/GB/mo | $0.08/GB/mo | Negotiated |
| **Cloud AI calls** | N/A | N/A | Pass-through + 20% | Pass-through + 15% | Pass-through + 10% |

**Cloud AI pass-through pricing** (approximate per-study costs):
- Gemini multimodal (classification/PHI): ~$0.002–0.01/study
- Google Cloud Vision: ~$0.001–0.005/study
- AWS Textract: ~$0.002–0.008/study
- Azure Computer Vision: ~$0.001–0.005/study

Most studies use heuristic backends (free) and only fall back to cloud AI when DICOM tags are missing.

---

## Add-On Modules

| Add-On | Price | Available on |
|--------|-------|-------------|
| **DIMSE receiver** | $200/mo | Research |
| **Priority support** (1-day response) | $150/mo | Research |
| **Dedicated support** (4-hour critical) | $500/mo | Consortium |
| **Additional DIMSE endpoints** | $100/mo each | Consortium, Enterprise |
| **On-premises deployment license** | $25,000–75,000/yr | Enterprise (contact sales) |
| **Protocol template pack** (ADNI4, HCP, ABCD, UK Biobank, ENIGMA) | $1,500 one-time | Research, Consortium |
| **White-label upload portal** | $500/mo | Consortium, Enterprise |
| **Custom BAA execution** | Included | Enterprise |
| **PACS integration consulting** | $250/hr | All tiers |

---

## Discounts & Programs

### Academic Discount
- **20% off** any paid tier with valid `.edu` email and institutional purchase order
- Stackable with annual billing discount

### Grant Budget Letter
- On request: formal letter quoting AEGIS costs for NIH/NSF/DOD grant budget justification
- Includes: annual cost, description of services, HIPAA compliance attestation

### Startup / Pilot Program
- **3 months free** on Research tier for new imaging startups (<$5M funding)
- Requires case study participation at end of pilot

### Volume Discount (Enterprise)
- 5+ projects: 10% discount
- 10+ projects: 15% discount
- Multi-year (2–3 yr commitment): additional 10%

### Non-Profit / Government
- **15% off** for 501(c)(3) organizations and government agencies (VA, DoD, NIH intramural)

---

## Revenue Streams Summary

| Stream | Margin | Scalability | Notes |
|--------|--------|-------------|-------|
| **SaaS subscriptions** | 70–85% | High | Core recurring revenue |
| **Storage overage** | 60–70% | High | Cloud storage cost + margin |
| **Cloud AI pass-through** | 15–20% | High | Low margin but high volume |
| **Add-on modules** | 80–90% | Medium | DIMSE, support, template packs |
| **Professional services** | 40–60% | Low | Consulting, integration, onboarding |
| **On-premises licenses** | 90%+ | Low | High-value, low-volume |

---

## Billing & Payment

- **Payment methods:** Credit card (Stripe), ACH/wire, purchase order (Consortium + Enterprise)
- **Billing cycle:** Monthly or annual (prepaid annual = 2 months free)
- **Invoicing:** Net-30 for institutional POs
- **Currency:** USD (EUR and GBP available for Enterprise)
- **Cancellation:** Monthly plans cancel anytime; annual plans prorated refund for remaining months

---

## Competitive Pricing Context

| Competitor | Model | Approximate Price | Notes |
|------------|-------|-------------------|-------|
| **Flywheel** | SaaS subscription | $500–5,000+/mo (estimated) | Research-focused, includes viewer and analysis |
| **XNAT Central** | Hosted service | Free (limited) / custom enterprise | Open-source self-hosted is free |
| **ENCOG (Enlitic)** | Enterprise subscription | Not public (est. $3K–10K/mo) | Requires PACS integration |
| **MIRC CTP** | Open source | Free (self-hosted only) | No hosted option, no support |
| **Ambra/Intelerad** | Per-study or subscription | $2–5/study or custom | Cloud PACS, limited de-identification |
| **AEGIS (proposed)** | Freemium + tiers | $0–15,000/mo | Only platform covering both research + enterprise |

AEGIS's free tier and $499 entry point undercut all commercial alternatives while offering a broader feature set. The open-source option provides a floor that prevents vendor lock-in concerns.

---

## Key Metrics to Track

| Metric | Target (Year 1) | Target (Year 3) |
|--------|-----------------|-----------------|
| Free tier signups | 100+ | 500+ |
| Paid conversions | 10–15% of free users | 15–20% |
| Average revenue per account (ARPA) | $800/mo | $1,500/mo |
| Net revenue retention | 100%+ | 120%+ |
| Monthly churn | <5% | <3% |
| CAC payback period | <6 months | <4 months |

---

## Implementation Notes

### Metering Infrastructure
- Study count: already tracked in `studies` table (`created_at` per month)
- Storage: already tracked via `GET /api/storage/stats` and per-project `GET /api/stats/storage-usage`
- Cloud AI calls: add metering counter in classification/PHI detection service callbacks
- API key usage: already tracked (`last_used_at` on `api_keys`)

### Billing Integration
- Stripe for credit card + subscription management
- Stripe Billing Portal for self-service plan changes
- Custom invoicing via Go service for PO-based enterprise customers
- Usage metering → Stripe Usage Records for overage billing

### Plan Enforcement
- Middleware layer checking project count, study count, storage against plan limits
- Soft limits (warning at 80%) → hard limits (block at 100%) for studies and storage
- Grace period (7 days) for storage overage before blocking new uploads
