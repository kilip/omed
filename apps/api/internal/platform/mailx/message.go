// Package mailx provides an extensible mailing platform supporting multiple drivers,
// background queue delivery, and template rendering.
package mailx

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	// ErrNoRecipient is returned when a message has no To/Cc/Bcc recipients.
	ErrNoRecipient = errors.New("mailx: message has no recipients")
	// ErrNoSubject is returned when a message subject is empty.
	ErrNoSubject = errors.New("mailx: message subject is empty")
	// ErrNoBody is returned when both TextBody and HTMLBody are empty.
	ErrNoBody = errors.New("mailx: message text and html bodies are both empty")
	// ErrInvalidAddress is returned when an email address format is invalid.
	ErrInvalidAddress = errors.New("mailx: invalid email address")
)

// Address represents an RFC 5322 email address with an optional display name.
type Address struct {
	Name    string
	Address string
}

// String returns formatted "Name <address@domain>" or "address@domain".
func (a Address) String() string {
	addr := mail.Address{Name: a.Name, Address: a.Address}
	return addr.String()
}

// Attachment holds email attachment data.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Message represents an outgoing email message.
type Message struct {
	From        Address
	To          []Address
	Cc          []Address
	Bcc         []Address
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
}

// NewMessage initializes an empty Message with defaults.
func NewMessage() Message {
	return Message{
		Headers: make(map[string]string),
	}
}

// AddTo appends a recipient to the To field.
func (m *Message) AddTo(email, name string) {
	m.To = append(m.To, Address{Name: name, Address: strings.TrimSpace(email)})
}

// AddCc appends a recipient to the Cc field.
func (m *Message) AddCc(email, name string) {
	m.Cc = append(m.Cc, Address{Name: name, Address: strings.TrimSpace(email)})
}

// AddBcc appends a recipient to the Bcc field.
func (m *Message) AddBcc(email, name string) {
	m.Bcc = append(m.Bcc, Address{Name: name, Address: strings.TrimSpace(email)})
}

// AddAttachment appends an attachment to the message.
func (m *Message) AddAttachment(filename, contentType string, data []byte) {
	m.Attachments = append(m.Attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	})
}

// Validate verifies that the message has required fields and valid addresses.
func (m Message) Validate() error {
	if len(m.To) == 0 && len(m.Cc) == 0 && len(m.Bcc) == 0 {
		return ErrNoRecipient
	}
	if strings.TrimSpace(m.Subject) == "" {
		return ErrNoSubject
	}
	if strings.TrimSpace(m.TextBody) == "" && strings.TrimSpace(m.HTMLBody) == "" {
		return ErrNoBody
	}

	for _, addr := range append(append(m.To, m.Cc...), m.Bcc...) {
		if _, err := mail.ParseAddress(addr.Address); err != nil {
			return ErrInvalidAddress
		}
	}

	if m.From.Address != "" {
		if _, err := mail.ParseAddress(m.From.Address); err != nil {
			return ErrInvalidAddress
		}
	}

	return nil
}
