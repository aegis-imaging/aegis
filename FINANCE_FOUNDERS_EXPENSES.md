# AEGIS — Founders Paid Expenses Tracker

Use this sheet to log all personal payments for business software, services, and infrastructure until business bank account is live.

## Template

| Date | Vendor | Category | Amount (USD) | Tax | Total | Card Used | Business Purpose | Invoice # | Status | Notes |
|------|--------|----------|--------------|-----|-------|-----------|------------------|-----------|--------|-------|
| 2026-02-20 | Microsoft 365 | Software/Email | $120.00 | $0.00 | $120.00 | Personal Amex | Business email + Azure AD setup | INV-M365-12345 | Paid | Premium tier, annual commit |
| 2026-02-20 | Claude.ai | AI/Tooling | $20.00 | $0.00 | $20.00 | Personal Amex | AEGIS org account, Pro plan | INV-CLAUDE-67890 | Paid | Monthly subscription |
| | | | | | | | | | | |

## Key Fields

- **Date**: Transaction/invoice date (YYYY-MM-DD)
- **Vendor**: Company name (Microsoft, Anthropic/Claude, GCP, AWS, Azure, etc.)
- **Category**: Type of expense (Software/Email, Cloud Infrastructure, AI/Tooling, Database, DevOps, etc.)
- **Amount**: Subscription/service cost (USD, pre-tax)
- **Tax**: Sales tax or VAT if applicable
- **Total**: Amount + Tax (what you paid)
- **Card Used**: Which personal card (to reconcile with bank statement)
- **Business Purpose**: Why this is for AEGIS (e.g., "Cloud Run deployment", "de-identification service infrastructure")
- **Invoice #**: Vendor invoice number or reference (for matching to vendor portal)
- **Status**: `Paid` (charged to personal account) | `Reimbursed` (moved to business card/account) | `Pending`
- **Notes**: Any special terms (e.g., annual commitment, free tier upgrade, trial period, discount code used)

## Reimbursement Workflow

When business bank account is live:

1. **Update payment methods** in each vendor portal to business card/ACH.
2. **Batch reimbursement** (monthly or quarterly):
   - Total all `Paid` rows in a given period.
   - Create one business check or ACH transfer to personal account with memo: `"Founders expenses reimbursement — software (Feb 2026)" or similar`.
   - Mark all rows in that batch as `Reimbursed` and add reimbursement date + check/ACH reference number in Notes column.
3. **Keep supporting docs**:
   - Vendor invoice PDFs in folder: `docs/finance/invoices/`
   - Bank statement screenshots showing the reimbursement
   - This spreadsheet log

## Tax & Accounting Notes

- **Consult your accountant** before reimbursing yourself — treatment depends on entity type (LLC, S-Corp, C-Corp, Sole Prop).
- For S-Corps: reimbursements are typically tax-free but must be reasonable and documented.
- For LLCs/Pass-throughs: reimbursements reduce owner draws or are treated as basis/capital contributions.
- **Sales tax/VAT**: SaaS is often exempt depending on your state/country. Keep vendor tax statements.
- **Documentation**: IRS expects you to have vendor invoices, payment proof, business purpose, and reimbursement records tied together.

## Example: Feb 2026 Setup Costs

| Date | Vendor | Category | Amount | Tax | Total | Card Used | Business Purpose | Invoice # | Status | Notes |
|------|--------|----------|--------|-----|-------|-----------|------------------|-----------|--------|-------|
| 2026-02-20 | Microsoft 365 | Software/Email | $120.00 | $0.00 | $120.00 | Personal Amex | Business email tenant + Entra/Azure AD identity platform | M365-PREM-FY26 | Paid | Premium, annual commit, domain verification complete |
| 2026-02-20 | Claude.ai | AI/Tooling | $20.00 | $0.00 | $20.00 | Personal Amex | AEGIS org Claude account (ai@aegisimaging.ai), Pro plan, separate from personal | CLAUDE-ORG-001 | Paid | Monthly, auto-renew, used for AEGIS research & documentation |
| 2026-02-21 | Google Cloud | Cloud Infrastructure | $0.00 | $0.00 | $0.00 | N/A | GCP project bootstrap (free tier / credits) | — | Free Trial | $300 free credits; project: aegis-prod-main |
| 2026-02-21 | Azure | Cloud Infrastructure | $0.00 | $0.00 | $0.00 | N/A | Azure subscription (free tier / credits) | — | Free Trial | $200 free credits; tenant linked to M365 Entra |
| | **Feb 2026 Total** | | **$140.00** | **$0.00** | **$140.00** | | | | | |

---

## Files to Save (One Folder: `docs/finance/`)

```
docs/finance/
├── invoices/
│   ├── 2026-02/
│   │   ├── Microsoft_365_Invoice_M365-PREM-FY26.pdf
│   │   ├── Claude_Pro_Receipt_CLAUDE-ORG-001.pdf
│   │   └── [other vendor invoices]
│   └── 2026-03/
│   └── [future months]
├── bank-statements/
│   ├── Personal_Amex_Feb2026.pdf
│   └── [reconciliation docs after reimbursement]
└── reimbursement-records/
    ├── Reimbursement_Check_002_Feb2026.pdf
    └── [ACH confirmation after business account live]
```

---

## Next Steps

1. **Print or bookmark this sheet** in your preferred tool (Google Sheets, Airtable, Excel, Notion).
2. **Start logging today** — first entries: M365 Premium, Claude Pro, GCP free tier, Azure free tier.
3. **Save vendor invoices** as PDFs to `docs/finance/invoices/{YYYY-MM}/`.
4. **Reconcile monthly** — match charges to bank statement + vendor portal.
5. **Consult accountant** before first reimbursement batch (Feb/Mar 2026).

---

**Last Updated:** 2026-02-20  
**Owner:** AEGIS Finance Lead  
**Status:** Active (live tracking)
