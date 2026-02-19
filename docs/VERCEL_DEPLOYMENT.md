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

## Step 1: Create a Formspree Form

- [ ] Go to [formspree.io](https://formspree.io) and sign up (free tier)
- [ ] Create a new form — name it "AEGIS Contact"
- [ ] Copy the form ID from the endpoint URL (the part after `https://formspree.io/f/`)
- [ ] Save it — you'll need it in Step 3

*If you skip this, the contact form falls back to a `mailto:contact@aegisimaging.ai` link.*

---

## Step 2: Set Up Vercel Project

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

## Step 3: Set Environment Variable

- [ ] In your Vercel project: **Settings → Environment Variables**
- [ ] Add:

| Key | Value | Environments |
|-----|-------|-------------|
| `VITE_FORMSPREE_ID` | *(your form ID from Step 1)* | Production, Preview, Development |

- [ ] Redeploy (Settings → Deployments → click **⋮** on latest → **Redeploy**)

---

## Step 4: Add Custom Domain in Vercel

- [ ] In your Vercel project: **Settings → Domains**
- [ ] Type `aegisimaging.ai` and click **Add**
- [ ] Vercel will show the DNS records you need — keep this page open

---

## Step 5: Configure DNS at GoDaddy

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

## Step 6: Set Up aegisimaging.org Redirect (Optional)

- [ ] **Option A — Via Vercel:** Add `aegisimaging.org` as a domain in the same Vercel project. Vercel will redirect it to the primary domain.
- [ ] **Option B — Via GoDaddy:** On the `.org` domain, set up a domain forward: GoDaddy → My Products → aegisimaging.org → Manage → Forwarding → Forward to `https://aegisimaging.ai`

---

## Step 7: Verify Everything Works

- [ ] Visit `https://aegisimaging.ai` — page loads with SSL
- [ ] Visit `https://www.aegisimaging.ai` — redirects to apex
- [ ] Scroll through all 13 sections — animations trigger
- [ ] Test mobile layout (resize browser or use phone)
- [ ] Submit the contact form — check Formspree dashboard for the submission
- [ ] Visit `https://aegisimaging.org` — redirects to `.ai` (if configured)

---

## Ongoing: Auto-Deploy

Vercel auto-deploys on every push to the branch it's connected to (typically `develop` or `main`). No manual deploys needed after initial setup. Every PR also gets a preview URL.
