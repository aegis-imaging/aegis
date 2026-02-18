package email

import (
	"context"
	"net/smtp"

	"github.com/msenjem/aegis/api/config"
)

// Client sends plain-text emails via SMTP. It is a no-op when EmailEnabled is false,
// so services without SMTP configured continue to work normally.
type Client struct {
	host     string
	port     string
	from     string
	username string
	password string
	enabled  bool
}

func New(cfg *config.Config) *Client {
	return &Client{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		from:     cfg.SMTPFrom,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		enabled:  cfg.SMTPHost != "",
	}
}

// Send delivers a plain-text email. Returns nil immediately if email is disabled.
func (c *Client) Send(_ context.Context, to, subject, body string) error {
	if !c.enabled {
		return nil
	}
	addr := c.host + ":" + c.port
	msg := []byte(
		"From: " + c.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body + "\r\n",
	)
	var auth smtp.Auth
	if c.username != "" {
		auth = smtp.PlainAuth("", c.username, c.password, c.host)
	}
	return smtp.SendMail(addr, auth, c.from, []string{to}, msg)
}
