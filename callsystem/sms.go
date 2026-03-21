package callsystem

import "context"

// SMSMessage represents a sent or received SMS message.
type SMSMessage struct {
	// ID is the provider-specific message identifier.
	ID string

	// To is the recipient phone number (E.164 format).
	To string

	// From is the sender phone number (E.164 format).
	From string

	// Body is the message content.
	Body string

	// Status is the message delivery status.
	Status string
}

// SMSProvider defines the interface for sending SMS messages.
// CallSystem implementations that support SMS should also implement this interface.
type SMSProvider interface {
	// SendSMS sends an SMS message.
	SendSMS(ctx context.Context, to, body string) (*SMSMessage, error)

	// SendSMSFrom sends an SMS message from a specific number.
	SendSMSFrom(ctx context.Context, to, from, body string) (*SMSMessage, error)
}
