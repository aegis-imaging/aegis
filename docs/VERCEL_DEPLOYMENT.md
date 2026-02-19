---
pdf_options:
  format: Letter
  margin: 20mm
  printBackground: true
css: |
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; font-size: 14px; line-height: 1.6; color: #24292e; }
  h1 { font-size: 2em; margin: 0.67em 0; }
  h2 { font-size: 1.5em; margin: 1.2em 0 0.6em; border-bottom: 1px solid #e5e7eb; padding-bottom: 0.4em; }
  h3 { font-size: 1.2em; margin: 1em 0 0.4em; }
  p { margin: 0.8em 0; }
  ul, ol { padding-left: 2em; margin: 0.8em 0; }
  li { margin: 0.3em 0; }
  table { border-collapse: collapse; width: 100%; margin: 1em 0; font-size: 13px; }
  th { background: #f6f8fa; font-weight: 600; text-align: left; padding: 8px 12px; border: 1px solid #d0d7de; }
  td { padding: 7px 12px; border: 1px solid #d0d7de; vertical-align: top; }
  code { background: #f6f8fa; padding: 2px 5px; border-radius: 4px; font-size: 85%; }
  strong { font-weight: 600; }
---

# AEGIS Landing Page — Vercel + GoDaddy Deployment

**Site:** aegisimaging.ai | **Source:** `frontend/landing/` | **Framework:** Vite + React

---

## Step 1: Set Up Vercel Project

- [ ] Go to [vercel.com](https://vercel.com) and sign up (free Hobby plan is fine)
- [ ] Click **Add New → Project**
- [ ] Connect your GitHub account and select the **AEGIS** repo
- [ ] **Configure Project:**

| Setting | Value |
|---------|-------|
| **Root Directory** | `frontend/landing` |
| **Framework Preset** | Vite (should auto-detect) |
| **Build Command** | `npm run build` (default) |
| **Output Directory** | `dist` (default) |

- [ ] Click **Deploy** — wait for the first build to succeed
- [ ] Note the preview URL Vercel gives you (e.g. `aegis-abc123.vercel.app`)

---

## Step 2: Configure Contact Form Email (Brevo SMTP)

The contact form uses a Vercel serverless function (`api/contact.ts`) that sends email via Brevo SMTP. Without these env vars, submissions are logged to the console but not emailed.

- [ ] Sign up at [brevo.com](https://www.brevo.com) (free tier: 300 emails/day)
- [ ] Go to **Settings → SMTP & API → SMTP** and note your credentials
- [ ] In your Vercel project: **Settings → Environment Variables**
- [ ] Add the following:

| Key | Value | Environments |
|-----|-------|-------------|
| `BREVO_SMTP_HOST` | `smtp-relay.brevo.com` | Production, Preview |
| `BREVO_SMTP_PORT` | `587` | Production, Preview |
| `BREVO_SMTP_USER` | *(your Brevo SMTP login)* | Production, Preview |
| `BREVO_SMTP_PASS` | *(your Brevo SMTP password)* | Production, Preview |
| `BREVO_SMTP_FROM` | `AEGIS <noreply@aegisimaging.ai>` | Production, Preview |

- [ ] Redeploy (Settings → Deployments → click **⋮** on latest → **Redeploy**)
- [ ] Test the contact form — check your inbox at `contact@aegisimaging.ai`

*If you skip this step, the contact form still works — submissions are logged server-side but not emailed. You can configure it later.*

---

## Step 3: Add Custom Domain in Vercel

- [ ] In your Vercel project: **Settings → Domains**
- [ ] Type `aegisimaging.ai` and click **Add**
- [ ] Vercel will show the DNS records you need — keep this page open

---

## Step 4: Configure DNS at GoDaddy

- [ ] Log in to [godaddy.com](https://godaddy.com)
- [ ] Go to **My Products → aegisimaging.ai → DNS → Manage DNS**
- [ ] Delete any GoDaddy parking/forwarding records if present
- [ ] Add or edit these records:

| Type | Name | Value | TTL |
|------|------|-------|-----|
| `A` | `@` | `76.76.21.21` | 1 Hour |
| `CNAME` | `www` | `cname.vercel-dns.com` | 1 Hour |

- [ ] Save changes
- [ ] Go back to the Vercel Domains page — wait for the green checkmark (5–30 min, sometimes up to 48 hours)
- [ ] Vercel provisions a free SSL certificate automatically

---

## Step 5: Set Up aegisimaging.org Redirect (Optional)

- [ ] **Option A — Via Vercel:** Add `aegisimaging.org` as a domain in the same Vercel project. Vercel will redirect it to the primary domain.
- [ ] **Option B — Via GoDaddy:** On the `.org` domain, set up a domain forward: GoDaddy → My Products → aegisimaging.org → Manage → Forwarding → Forward to `https://aegisimaging.ai`

---

## Step 6: Verify Everything Works

- [ ] Visit `https://aegisimaging.ai` — page loads with SSL
- [ ] Visit `https://www.aegisimaging.ai` — redirects to apex
- [ ] Scroll through all sections — animations trigger
- [ ] Test mobile layout (resize browser or use phone)
- [ ] Submit the contact form — check inbox at `contact@aegisimaging.ai`
- [ ] Visit `https://aegisimaging.org` — redirects to `.ai` (if configured)

---

## Ongoing: Auto-Deploy

Vercel auto-deploys on every push to the branch it's connected to (typically `develop` or `main`). No manual deploys needed after initial setup. Every PR also gets a preview URL.

---

## Architecture: Contact Form

The contact form (`POST /api/contact`) works via two independent paths:

| Path | Backend | When |
|------|---------|------|
| **Vercel serverless function** | `frontend/landing/api/contact.ts` — nodemailer + Brevo SMTP | Landing page hosted on Vercel (standalone) |
| **Go API endpoint** | `api/handler/contact.go` — existing SMTP config | Landing page proxied to Go backend |

Both accept the same JSON payload `{ name, email, organization, role, message }` and return `{ sent: true }`.

The Vercel function is self-contained — it doesn't depend on the Go API. This means the landing page and contact form work even before the backend is deployed. Once the backend is live, the Go API endpoint provides the same functionality using the platform's existing SMTP infrastructure.
