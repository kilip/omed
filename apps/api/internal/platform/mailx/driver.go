package mailx

import (
	"context"
)

// Driver specifies the low-level contract for sending messages across physical transports.
type Driver interface {
	// Send transmits a message over the underlying transport.
	Send(ctx context.Context, msg Message) error
	// Close releases any persistent connections held by the driver.
	Close() error
}
