package mailx

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"

	"github.com/wneessen/go-mail"
)

// SMTPConfig contains connection settings for an SMTP server.
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	InsecureTLS bool
}

// SMTPDriver sends email via standard SMTP using github.com/wneessen/go-mail.
type SMTPDriver struct {
	cfg    SMTPConfig
	client *mail.Client
}

// NewSMTPDriver initializes a new SMTPDriver with the provided configuration.
func NewSMTPDriver(cfg SMTPConfig) (*SMTPDriver, error) {
	if cfg.Port <= 0 {
		cfg.Port = 25
	}

	opts := []mail.Option{
		mail.WithPort(cfg.Port),
	}

	if cfg.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	}

	if cfg.InsecureTLS {
		opts = append(opts,
			mail.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec
			mail.WithTLSPolicy(mail.TLSOpportunistic),
		)
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mailx: create smtp client: %w", err)
	}

	return &SMTPDriver{
		cfg:    cfg,
		client: client,
	}, nil
}

// Send transmits the message using the SMTP client.
func (d *SMTPDriver) Send(ctx context.Context, msg Message) error {
	m := mail.NewMsg()

	if err := m.From(msg.From.String()); err != nil {
		return fmt.Errorf("mailx: set from: %w", err)
	}

	for _, to := range msg.To {
		if err := m.AddTo(to.String()); err != nil {
			return fmt.Errorf("mailx: set to %q: %w", to.String(), err)
		}
	}

	for _, cc := range msg.Cc {
		if err := m.AddCc(cc.String()); err != nil {
			return fmt.Errorf("mailx: set cc %q: %w", cc.String(), err)
		}
	}

	for _, bcc := range msg.Bcc {
		if err := m.AddBcc(bcc.String()); err != nil {
			return fmt.Errorf("mailx: set bcc %q: %w", bcc.String(), err)
		}
	}

	m.Subject(msg.Subject)

	for k, v := range msg.Headers {
		m.SetHeader(mail.Header(k), v)
	}

	hasText := msg.TextBody != ""
	hasHTML := msg.HTMLBody != ""

	switch {
	case hasText && hasHTML:
		m.SetBodyString(mail.TypeTextPlain, msg.TextBody)
		m.AddAlternativeString(mail.TypeTextHTML, msg.HTMLBody)
	case hasHTML:
		m.SetBodyString(mail.TypeTextHTML, msg.HTMLBody)
	case hasText:
		m.SetBodyString(mail.TypeTextPlain, msg.TextBody)
	}

	for _, att := range msg.Attachments {
		m.AttachReader(att.Filename, bytes.NewReader(att.Data))
	}

	if err := d.client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("mailx: send smtp: %w", err)
	}

	return nil
}

// Close closes any open SMTP connection pool.
func (d *SMTPDriver) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}
