package mailx

import "log/slog"

// Config encapsulates configuration for the mailx subsystem.
type Config struct {
	Driver      string // "smtp", "memory"
	FromAddress string
	FromName    string
	SMTP        SMTPConfig
	Queue       MemoryQueueConfig
	Logger      *slog.Logger
}

// DefaultFrom returns the default From Address.
func (c Config) DefaultFrom() Address {
	return Address{
		Name:    c.FromName,
		Address: c.FromAddress,
	}
}
