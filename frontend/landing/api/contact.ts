import type { VercelRequest, VercelResponse } from '@vercel/node'
import nodemailer from 'nodemailer'

const SMTP_HOST = process.env.BREVO_SMTP_HOST || ''
const SMTP_PORT = Number(process.env.BREVO_SMTP_PORT || 587)
const SMTP_USER = process.env.BREVO_SMTP_USER || ''
const SMTP_PASS = process.env.BREVO_SMTP_PASS || ''
const SMTP_FROM = process.env.BREVO_SMTP_FROM || 'AEGIS <noreply@aegisimaging.ai>'
const CONTACT_TO = 'contact@aegisimaging.ai'

const canSendEmail = SMTP_HOST && SMTP_USER && SMTP_PASS

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' })
  }

  const { name, email, organization, role, message } = req.body || {}

  if (!name || typeof name !== 'string' || !name.trim()) {
    return res.status(400).json({ error: 'Name is required' })
  }
  if (!email || typeof email !== 'string' || !email.trim()) {
    return res.status(400).json({ error: 'Email is required' })
  }
  if (!message || typeof message !== 'string' || !message.trim()) {
    return res.status(400).json({ error: 'Message is required' })
  }
  if (message.length > 5000) {
    return res.status(400).json({ error: 'Message too long (max 5000 characters)' })
  }

  const trimmedName = name.trim()
  const trimmedEmail = email.trim()
  const trimmedOrg = typeof organization === 'string' ? organization.trim() : ''
  const trimmedRole = typeof role === 'string' ? role.trim() : ''
  const trimmedMessage = message.trim()

  if (!canSendEmail) {
    console.log('SMTP not configured. Contact form submission:', {
      name: trimmedName,
      email: trimmedEmail,
      organization: trimmedOrg,
      role: trimmedRole,
      message: trimmedMessage,
    })
    return res.status(200).json({ sent: true })
  }

  try {
    const subject = `[AEGIS Contact] ${trimmedName} — ${trimmedMessage.slice(0, 60)}${trimmedMessage.length > 60 ? '...' : ''}`

    const transporter = nodemailer.createTransport({
      host: SMTP_HOST,
      port: SMTP_PORT,
      secure: SMTP_PORT === 465,
      auth: { user: SMTP_USER, pass: SMTP_PASS },
    })

    await transporter.sendMail({
      from: SMTP_FROM,
      to: CONTACT_TO,
      replyTo: trimmedEmail,
      subject,
      text: [
        `Name: ${trimmedName}`,
        `Email: ${trimmedEmail}`,
        trimmedOrg ? `Organization: ${trimmedOrg}` : null,
        trimmedRole ? `Role: ${trimmedRole}` : null,
        '',
        'Message:',
        trimmedMessage,
      ]
        .filter((line) => line !== null)
        .join('\n'),
      html: `
        <div style="font-family: sans-serif; max-width: 600px;">
          <h2 style="color: #1a365d;">AEGIS Contact Form</h2>
          <table style="border-collapse: collapse; width: 100%; margin-bottom: 16px;">
            <tr><td style="padding: 8px; border-bottom: 1px solid #eee; color: #666; width: 100px;">Name</td><td style="padding: 8px; border-bottom: 1px solid #eee;">${trimmedName}</td></tr>
            <tr><td style="padding: 8px; border-bottom: 1px solid #eee; color: #666;">Email</td><td style="padding: 8px; border-bottom: 1px solid #eee;"><a href="mailto:${trimmedEmail}">${trimmedEmail}</a></td></tr>
            ${trimmedOrg ? `<tr><td style="padding: 8px; border-bottom: 1px solid #eee; color: #666;">Organization</td><td style="padding: 8px; border-bottom: 1px solid #eee;">${trimmedOrg}</td></tr>` : ''}
            ${trimmedRole ? `<tr><td style="padding: 8px; border-bottom: 1px solid #eee; color: #666;">Role</td><td style="padding: 8px; border-bottom: 1px solid #eee;">${trimmedRole}</td></tr>` : ''}
          </table>
          <div style="background: #f7fafc; padding: 16px; border-radius: 8px; white-space: pre-wrap;">${trimmedMessage}</div>
        </div>
      `,
    })

    return res.status(200).json({ sent: true })
  } catch (err) {
    console.error('Contact email error:', err)
    return res.status(500).json({ error: 'Failed to send message' })
  }
}
