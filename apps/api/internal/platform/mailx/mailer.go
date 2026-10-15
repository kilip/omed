package mailx

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Mailer provides high-level email dispatch capabilities.
type Mailer interface {
	// Send transmits a message synchronously.
	Send(ctx context.Context, msg Message) error
	// SendAsync enqueues a message for background transmission.
	SendAsync(ctx context.Context, msg Message) error
	// SendTemplate renders templates and transmits the message synchronously.
	SendTemplate(ctx context.Context, tmplName string, data any, msg Message) error
	// SendTemplateAsync renders templates and enqueues the message for background transmission.
	SendTemplateAsync(ctx context.Context, tmplName string, data any, msg Message) error
	// Driver returns the underlying transport driver.
	Driver() Driver
	// Close shuts down background queues and releases driver resources.
	Close() error
}

type defaultMailer struct {
	cfg    Config
	driver Driver
	queue  Queue
	tmpl   *TemplateEngine
	logger *slog.Logger
}

// New creates and initializes a ready-to-use Mailer instance based on Config.
func New(cfg Config) (Mailer, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	var driver Driver
	var err error

	switch cfg.Driver {
	case "smtp":
		driver, err = NewSMTPDriver(cfg.SMTP)
		if err != nil {
			return nil, fmt.Errorf("mailx: init smtp driver: %w", err)
		}
	default:
		driver = NewMemoryDriver()
	}

	cfg.Queue.Logger = cfg.Logger
	q := NewMemoryQueue(driver, cfg.Queue)
	if err := q.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("mailx: start memory queue: %w", err)
	}

	return &defaultMailer{
		cfg:    cfg,
		driver: driver,
		queue:  q,
		tmpl:   NewTemplateEngine(),
		logger: cfg.Logger,
	}, nil
}

// NewWithComponents builds a Mailer directly with explicit components (useful for testing).
func NewWithComponents(cfg Config, driver Driver, queue Queue, tmpl *TemplateEngine) Mailer {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if tmpl == nil {
		tmpl = NewTemplateEngine()
	}
	return &defaultMailer{
		cfg:    cfg,
		driver: driver,
		queue:  queue,
		tmpl:   tmpl,
		logger: cfg.Logger,
	}
}

func (m *defaultMailer) prepareMessage(msg *Message) {
	if msg.From.Address == "" && m.cfg.FromAddress != "" {
		msg.From = m.cfg.DefaultFrom()
	}
}

// Send validates and transmits a message synchronously.
func (m *defaultMailer) Send(ctx context.Context, msg Message) error {
	m.prepareMessage(&msg)
	if err := msg.Validate(); err != nil {
		return err
	}
	return m.driver.Send(ctx, msg)
}

// SendAsync validates and enqueues a message for background transmission.
func (m *defaultMailer) SendAsync(ctx context.Context, msg Message) error {
	m.prepareMessage(&msg)
	if err := msg.Validate(); err != nil {
		return err
	}
	return m.queue.Enqueue(ctx, msg)
}

// SendTemplate renders template files and transmits synchronously.
func (m *defaultMailer) SendTemplate(ctx context.Context, tmplName string, data any, msg Message) error {
	htmlBody, textBody, err := m.tmpl.Render(tmplName, data)
	if err != nil {
		return err
	}
	msg.HTMLBody = htmlBody
	msg.TextBody = textBody
	return m.Send(ctx, msg)
}

// SendTemplateAsync renders template files and enqueues for background delivery.
func (m *defaultMailer) SendTemplateAsync(ctx context.Context, tmplName string, data any, msg Message) error {
	htmlBody, textBody, err := m.tmpl.Render(tmplName, data)
	if err != nil {
		return err
	}
	msg.HTMLBody = htmlBody
	msg.TextBody = textBody
	return m.SendAsync(ctx, msg)
}

// Driver returns the underlying transport driver.
func (m *defaultMailer) Driver() Driver {
	return m.driver
}

// Close gracefully stops worker queues and closes the driver.
func (m *defaultMailer) Close() error {
	var err error
	if m.queue != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if qErr := m.queue.Stop(ctx); qErr != nil {
			err = qErr
		}
	}
	if m.driver != nil {
		if dErr := m.driver.Close(); dErr != nil && err == nil {
			err = dErr
		}
	}
	return err
}
